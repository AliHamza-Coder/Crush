use anyhow::Result;
use std::process::Stdio;
use tokio::process::Command;

use crate::core::queue::{Task, TaskStatus, TaskType};

#[derive(Clone)]
pub struct FfmpegEngine {
    ffmpeg_path: String,
}

impl FfmpegEngine {
    pub fn new(ffmpeg_path: &str) -> Self {
        Self {
            ffmpeg_path: ffmpeg_path.to_string(),
        }
    }

    pub async fn process(&self, task: &mut Task) -> Result<()> {
        task.status = TaskStatus::Running;
        let args = self.build_args(task)?;

        let output = Command::new(&self.ffmpeg_path)
            .args(&args)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output()
            .await?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            let error_msg = self.parse_error(&stderr);
            return Err(anyhow::anyhow!("FFmpeg: {}", error_msg));
        }

        task.status = TaskStatus::Completed;
        task.progress = 100.0;
        Ok(())
    }

    fn build_args(&self, task: &Task) -> Result<Vec<String>> {
        let mut args = vec!["-i".to_string(), task.input_path.clone()];

        match &task.task_type {
            TaskType::VideoCompress { quality, format } => {
                let crf = self.quality_to_crf(*quality);
                let codec = self.video_codec_for_format(format);
                args.extend(["-c:v".into(), codec.into(), "-crf".into(), crf.to_string()]);
                args.extend(["-preset".into(), "fast".into()]);
                args.extend(["-c:a".into(), "copy".into()]);
                args.extend(["-movflags".into(), "+faststart".into()]);
            }
            TaskType::VideoConvert { target, quality } => {
                let crf = self.quality_to_crf(*quality);
                match target.as_str() {
                    "webm" => {
                        args.extend([
                            "-c:v".into(),
                            "libvpx-vp9".into(),
                            "-crf".into(),
                            crf.to_string(),
                        ]);
                        args.extend(["-b:v".into(), "0".into()]);
                        args.extend([
                            "-c:a".into(),
                            "libopus".into(),
                            "-b:a".into(),
                            "128k".into(),
                        ]);
                    }
                    "gif" => {
                        args.extend(["-vf".into(), "fps=10,scale=320:-1:flags=lanczos".into()]);
                    }
                    _ => {
                        args.extend([
                            "-c:v".into(),
                            "libx264".into(),
                            "-crf".into(),
                            crf.to_string(),
                        ]);
                        args.extend(["-preset".into(), "fast".into()]);
                        args.extend(["-c:a".into(), "aac".into(), "-b:a".into(), "192k".into()]);
                        args.extend(["-movflags".into(), "+faststart".into()]);
                    }
                }
            }
            TaskType::ImageCompress { quality, format } => {
                args.extend(self.image_args(format, *quality));
            }
            TaskType::ImageConvert { target, quality } => {
                args.extend(self.image_args(target, *quality));
            }
            TaskType::AudioExtract { format, quality } => {
                args.extend(["-map".into(), "a:0".into()]);
                args.extend(self.audio_args(format, *quality));
            }
            TaskType::AudioConvert { format, quality } => {
                args.extend(["-map".into(), "a:0".into()]);
                args.extend(self.audio_args(format, *quality));
            }
            TaskType::ImageUpscale { .. } => {
                return Err(anyhow::anyhow!("Use AI engine for upscaling"));
            }
        }

        args.extend(["-y".to_string(), task.output_path.clone()]);
        Ok(args)
    }

    fn image_args(&self, format: &str, quality: u8) -> Vec<String> {
        match format {
            "webp" => vec![
                "-c:v".into(),
                "libwebp".into(),
                "-quality".into(),
                quality.to_string(),
                "-compression_level".into(),
                "4".into(),
            ],
            "avif" => {
                let crf = 20u16 + (100u16.saturating_sub(quality as u16)) * 43 / 100;
                vec![
                    "-c:v".into(),
                    "libaom-av1".into(),
                    "-crf".into(),
                    crf.to_string(),
                    "-b:v".into(),
                    "0".into(),
                    "-strict".into(),
                    "experimental".into(),
                ]
            }
            "png" => {
                let level = 9 - (quality / 11);
                vec![
                    "-c:v".into(),
                    "png".into(),
                    "-compression_level".into(),
                    level.to_string(),
                ]
            }
            "gif" => vec!["-vf".into(), "fps=10,scale=320:-1:flags=lanczos".into()],
            _ => vec!["-q:v".into(), self.quality_to_qv(quality).to_string()],
        }
    }

    fn audio_args(&self, format: &str, quality: u8) -> Vec<String> {
        match format {
            "mp3" => {
                let q = (100u16.saturating_sub(quality as u16)) * 9 / 100;
                vec![
                    "-c:a".into(),
                    "libmp3lame".into(),
                    "-q:a".into(),
                    q.to_string(),
                ]
            }
            "flac" => {
                let level = 8 - (quality / 12);
                vec![
                    "-c:a".into(),
                    "flac".into(),
                    "-compression_level".into(),
                    level.to_string(),
                ]
            }
            "aac" | "m4a" => {
                let bitrate = 64 + (quality as u32 * 256 / 100);
                vec![
                    "-c:a".into(),
                    "aac".into(),
                    "-b:a".into(),
                    format!("{}k", bitrate),
                ]
            }
            "opus" => {
                let bitrate = 32 + (quality as u32 * 128 / 100);
                vec![
                    "-c:a".into(),
                    "libopus".into(),
                    "-b:a".into(),
                    format!("{}k", bitrate),
                ]
            }
            "ogg" => {
                let q = (100u16.saturating_sub(quality as u16)) * 10 / 100;
                vec![
                    "-c:a".into(),
                    "libvorbis".into(),
                    "-q:a".into(),
                    q.to_string(),
                ]
            }
            "wav" => vec![],
            "alac" => vec!["-c:a".into(), "alac".into()],
            _ => {
                let q = (100u16.saturating_sub(quality as u16)) * 9 / 100;
                vec![
                    "-c:a".into(),
                    "libmp3lame".into(),
                    "-q:a".into(),
                    q.to_string(),
                ]
            }
        }
    }

    fn quality_to_crf(&self, quality: u8) -> u8 {
        let crf = 10u16 + (100u16.saturating_sub(quality as u16)) * 35 / 100;
        crf.min(51) as u8
    }

    fn quality_to_qv(&self, quality: u8) -> u8 {
        let q = 1u16 + ((100u16.saturating_sub(quality as u16)) * 30) / 100;
        (q as u8).clamp(1, 31)
    }

    fn video_codec_for_format(&self, format: &str) -> &str {
        match format {
            "webm" => "libvpx-vp9",
            _ => "libx264",
        }
    }

    fn parse_error(&self, stderr: &str) -> String {
        for line in stderr.lines() {
            let lower = line.to_lowercase();
            if lower.contains("error")
                || lower.contains("failed")
                || lower.contains("not found")
                || lower.contains("invalid")
            {
                return line.trim().to_string();
            }
        }
        stderr
            .lines()
            .last()
            .unwrap_or("Unknown error")
            .trim()
            .to_string()
    }
}
