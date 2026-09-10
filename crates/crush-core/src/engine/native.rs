use std::path::Path;
use anyhow::Result;
use image::{DynamicImage, ImageEncoder};

use crate::core::queue::{Task, TaskStatus, TaskType};

#[derive(Clone)]
pub struct NativeEngine;

impl NativeEngine {
    pub fn new() -> Self {
        Self
    }

    pub async fn process(&self, task: &mut Task) -> Result<()> {
        task.status = TaskStatus::Running;
        let img = image::open(&task.input_path)?;
        let output = Path::new(&task.output_path);

        match &task.task_type {
            TaskType::ImageCompress { quality, format } => {
                self.encode(&img, output, format, *quality)?;
            }
            TaskType::ImageConvert { target, quality } => {
                self.encode(&img, output, target, *quality)?;
            }
            _ => return Err(anyhow::anyhow!("Native engine: unsupported task")),
        }

        task.status = TaskStatus::Completed;
        task.progress = 100.0;
        Ok(())
    }

    fn encode(&self, img: &DynamicImage, output: &Path, format: &str, quality: u8) -> Result<()> {
        match format {
            "webp" => self.encode_webp(img, output, quality),
            "avif" => self.encode_avif(img, output),
            "png" => self.encode_png(img, output),
            "jpg" | "jpeg" => self.encode_jpeg(img, output, quality),
            "bmp" => self.encode_bmp(img, output),
            _ => Err(anyhow::anyhow!("Unsupported native format: {}", format)),
        }
    }

    fn encode_webp(&self, img: &DynamicImage, output: &Path, quality: u8) -> Result<()> {
        let rgba = img.to_rgba8();
        let (w, h) = rgba.dimensions();
        let encoder = webp::Encoder::from_rgba(&rgba, w, h);
        let data = if quality == 0 {
            encoder.encode_lossless()
        } else {
            encoder.encode(quality as f32)
        };
        let bytes: &[u8] = &data;
        std::fs::write(output, bytes)?;
        Ok(())
    }

    fn encode_avif(&self, img: &DynamicImage, output: &Path) -> Result<()> {
        let rgb = img.to_rgb8();
        let (w, h) = rgb.dimensions();
        let mut buf = std::fs::File::create(output)?;
        let encoder = image::codecs::avif::AvifEncoder::new(&mut buf);
        encoder.write_image(&rgb, w, h, image::ColorType::Rgb8.into())?;
        Ok(())
    }

    fn encode_png(&self, img: &DynamicImage, output: &Path) -> Result<()> {
        img.save(output)?;
        Ok(())
    }

    fn encode_jpeg(&self, img: &DynamicImage, output: &Path, quality: u8) -> Result<()> {
        let rgb = img.to_rgb8();
        let (w, h) = rgb.dimensions();
        let mut file = std::fs::File::create(output)?;
        let encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut file, quality);
        encoder.write_image(&rgb, w, h, image::ColorType::Rgb8.into())?;
        Ok(())
    }

    fn encode_bmp(&self, img: &DynamicImage, output: &Path) -> Result<()> {
        img.to_rgb8().save(output)?;
        Ok(())
    }
}
