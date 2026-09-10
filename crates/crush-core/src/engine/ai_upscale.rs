use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use anyhow::Result;
use image::{DynamicImage, GenericImageView, RgbImage, Rgb};
use ort::session::Session;
use ort::value::Tensor;

use crate::core::queue::{Task, TaskStatus, TaskType};
use crate::core::deps;

#[derive(Clone)]
pub struct AiUpscaleEngine {
    model_path: Option<PathBuf>,
    session: Arc<Mutex<Option<Arc<Mutex<Session>>>>>,
}

impl AiUpscaleEngine {
    pub fn new(model_dir: &Path) -> Self {
        let model_path = model_dir.join("realesrgan-x4plus.onnx");
        Self {
            model_path: if model_path.exists() { Some(model_path) } else { None },
            session: Arc::new(Mutex::new(None)),
        }
    }

    /// True when the model file AND a usable ONNX Runtime dll both exist.
    /// Never touches `ort`, so it cannot panic.
    pub fn is_model_available(&self) -> bool {
        self.model_path.is_some() && deps::resolve_onnxruntime_dll().is_some()
    }

    /// Diagnostic: force-load the ONNX Runtime dll + model to confirm the AI
    /// engine can actually initialise (returns an error instead of panicking).
    pub fn test_ready(&self) -> Result<()> {
        self.ensure_session().map(|_| ())
    }

    fn load_model(path: &Path) -> Result<Session> {
        let session = Session::builder()?.commit_from_file(path)?;
        Ok(session)
    }

    /// Lazily initialise the ONNX Runtime session on first use.
    fn ensure_session(&self) -> Result<Arc<Mutex<Session>>> {
        let model_path = self.model_path.as_ref()
            .ok_or_else(|| anyhow::anyhow!("AI model not found. Run `crush setup` to download it."))?;

        // Only initialise ort when we have a trusted runtime dll. Point ort at
        // it so a random onnxruntime.dll in the cwd can never be picked up.
        deps::resolve_onnxruntime_dll()
            .ok_or_else(|| anyhow::anyhow!("ONNX Runtime not found. Run `crush setup` to install it."))?;
        deps::prepare_ort();

        let mut slot = self.session.lock().map_err(|e| anyhow::anyhow!("Lock error: {}", e))?;
        if slot.is_none() {
            let session = Self::load_model(model_path)?;
            *slot = Some(Arc::new(Mutex::new(session)));
        }
        Ok(slot.as_ref().unwrap().clone())
    }

    pub async fn process(&self, task: &mut Task) -> Result<()> {
        task.status = TaskStatus::Running;

        let session_arc = self.ensure_session()?;

        let scale = match &task.task_type {
            TaskType::ImageUpscale { scale } => *scale as u32,
            _ => 4,
        };

        let img = image::open(&task.input_path)?;
        let (width, height) = img.dimensions();
        let rgb = img.to_rgb8();

        task.progress = 20.0;

        let output_w = width * scale;
        let output_h = height * scale;

        let tiles = self.split_tiles(&rgb, 512, 32);
        let total_tiles = tiles.len() as f32;
        let mut upscaled_tiles = Vec::new();

        for (i, tile) in tiles.iter().enumerate() {
            let mut session = session_arc.lock().map_err(|e| anyhow::anyhow!("Lock error: {}", e))?;
            let upscaled = self.upscale_tile(&mut session, tile)?;
            drop(session);
            upscaled_tiles.push(upscaled);
            task.progress = 20.0 + (i as f32 / total_tiles) * 60.0;
        }

        task.progress = 85.0;
        let result = self.stitch_tiles(&upscaled_tiles, output_w, output_h, 512, 32 * scale)?;
        result.save(&task.output_path)?;

        task.status = TaskStatus::Completed;
        task.progress = 100.0;
        Ok(())
    }

    fn split_tiles(&self, img: &RgbImage, tile_size: u32, overlap: u32) -> Vec<Tile> {
        let (w, h) = img.dimensions();
        let mut tiles = Vec::new();
        let step = tile_size.saturating_sub(overlap);
        let mut y = 0;

        while y < h {
            let mut x = 0;
            while x < w {
                let tw = tile_size.min(w - x);
                let th = tile_size.min(h - y);
                let mut data = Vec::with_capacity((tw * th * 3) as usize);
                for py in 0..th {
                    for px in 0..tw {
                        let pixel = img.get_pixel(x + px, y + py);
                        data.extend_from_slice(&pixel.0);
                    }
                }
                tiles.push(Tile { x, y, width: tw, height: th, data });
                if x + tile_size >= w { break; }
                x += step;
            }
            if y + tile_size >= h { break; }
            y += step;
        }
        tiles
    }

    fn upscale_tile(&self, session: &mut Session, tile: &Tile) -> Result<Vec<f32>> {
        let c = 3usize;
        let h = tile.height as usize;
        let w = tile.width as usize;

        let mut input_data = Vec::with_capacity(c * h * w);
        for pixel in tile.data.chunks(3) {
            input_data.push(pixel[0] as f32 / 255.0);
            input_data.push(pixel[1] as f32 / 255.0);
            input_data.push(pixel[2] as f32 / 255.0);
        }

        let input_tensor = Tensor::from_array(([1, c, h, w], input_data))?;
        let input_val = input_tensor.into_dyn();
        let outputs = session.run(ort::inputs![input_val])?;
        let output = &outputs[0];
        let (_shape, data) = output.try_extract_tensor::<f32>()?;
        Ok(data.to_vec())
    }

    fn stitch_tiles(&self, tiles: &[Vec<f32>], out_w: u32, out_h: u32, tile_size: u32, scaled_overlap: u32) -> Result<DynamicImage> {
        let mut output = vec![0.0f32; (out_w * out_h * 3) as usize];
        let mut weight = vec![0.0f32; (out_w * out_h) as usize];
        let step = tile_size.saturating_sub(scaled_overlap);
        let tiles_x = (out_w + step - 1) / step;

        for (i, tile_data) in tiles.iter().enumerate() {
            let ty = (i as u32 / tiles_x) * step;
            let tx = (i as u32 % tiles_x) * step;
            let tw = tile_size.min(out_w.saturating_sub(tx));
            let th = tile_size.min(out_h.saturating_sub(ty));

            for py in 0..th {
                for px in 0..tw {
                    let ox = tx + px;
                    let oy = ty + py;
                    if ox < out_w && oy < out_h {
                        let oi = ((oy * out_w + ox) * 3) as usize;
                        let ti = ((py * tw + px) * 3) as usize;
                        if ti + 2 < tile_data.len() {
                            output[oi] += tile_data[ti];
                            output[oi + 1] += tile_data[ti + 1];
                            output[oi + 2] += tile_data[ti + 2];
                            weight[(oy * out_w + ox) as usize] += 1.0;
                        }
                    }
                }
            }
        }

        let mut rgb = RgbImage::new(out_w, out_h);
        for y in 0..out_h {
            for x in 0..out_w {
                let i = (y * out_w + x) as usize;
                let w_val = weight[i];
                if w_val > 0.0 {
                    let r = (output[i * 3] / w_val * 255.0).clamp(0.0, 255.0) as u8;
                    let g = (output[i * 3 + 1] / w_val * 255.0).clamp(0.0, 255.0) as u8;
                    let b = (output[i * 3 + 2] / w_val * 255.0).clamp(0.0, 255.0) as u8;
                    rgb.put_pixel(x, y, Rgb([r, g, b]));
                }
            }
        }

        Ok(DynamicImage::ImageRgb8(rgb))
    }
}

struct Tile {
    #[allow(dead_code)]
    x: u32,
    #[allow(dead_code)]
    y: u32,
    width: u32,
    height: u32,
    data: Vec<u8>,
}
