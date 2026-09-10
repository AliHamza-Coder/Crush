use thiserror::Error;

#[derive(Error, Debug)]
pub enum CrushError {
    #[error("FFmpeg not found: {0}")]
    FfmpegNotFound(String),

    #[error("AI model not found: {0}")]
    ModelNotFound(String),

    #[error("Unsupported format: {0}")]
    UnsupportedFormat(String),

    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Image error: {0}")]
    Image(#[from] image::ImageError),

    #[error("ONNX error: {0}")]
    Onnx(String),

    #[error("Task error: {0}")]
    Task(String),
}
