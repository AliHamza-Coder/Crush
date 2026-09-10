use std::path::PathBuf;
use std::process::Command;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct Config {
    pub input: PathBuf,
    pub output_dir: PathBuf,
    pub quality: u8,
    pub format: String,
    pub lossless: bool,
    pub backup: bool,
    pub backup_dir: PathBuf,
    pub parallel: usize,
    pub filter: String,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            input: PathBuf::from("."),
            output_dir: PathBuf::from("."),
            quality: 85,
            format: String::new(),
            lossless: false,
            backup: true,
            backup_dir: PathBuf::from("./backup"),
            parallel: std::thread::available_parallelism()
                .map(|n| n.get())
                .unwrap_or(4),
            filter: String::from("all"),
        }
    }
}

pub fn find_ffmpeg() -> Option<PathBuf> {
    if cfg!(target_os = "windows") {
        if let Ok(output) = Command::new("where").arg("ffmpeg").output() {
            if output.status.success() {
                let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
                if let Some(first) = path.lines().next() {
                    return Some(PathBuf::from(first));
                }
            }
        }
    } else {
        if let Ok(output) = Command::new("which").arg("ffmpeg").output() {
            if output.status.success() {
                let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
                return Some(PathBuf::from(path));
            }
        }
    }

    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|p| p.to_path_buf()))
        .unwrap_or_else(|| PathBuf::from("."));

    let local = if cfg!(target_os = "windows") {
        exe_dir.join("ffmpeg.exe")
    } else {
        exe_dir.join("ffmpeg")
    };
    if local.exists() {
        return Some(local);
    }

    None
}
