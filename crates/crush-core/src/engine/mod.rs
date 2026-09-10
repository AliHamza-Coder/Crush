pub mod ffmpeg;
pub mod native;
pub mod ai_upscale;

use crate::core::queue::{Engine, TaskType};
use crate::core::fileutil;

pub fn select_engine(task_type: &TaskType, file_ext: &str) -> Engine {
    match task_type {
        TaskType::VideoCompress { .. }
        | TaskType::VideoConvert { .. }
        | TaskType::AudioExtract { .. }
        | TaskType::AudioConvert { .. } => Engine::Ffmpeg,

        TaskType::ImageCompress { format, .. }
        | TaskType::ImageConvert { target: format, .. } => {
            if fileutil::native_supports_format(
                &format!(".{}", file_ext),
                format,
            ) {
                Engine::Native
            } else {
                Engine::Ffmpeg
            }
        }

        TaskType::ImageUpscale { .. } => Engine::OnnxAi,
    }
}
