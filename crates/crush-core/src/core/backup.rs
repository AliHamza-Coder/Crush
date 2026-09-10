use std::fs;
use std::path::{Path, PathBuf};

pub struct BackupManager {
    backup_dir: PathBuf,
}

impl BackupManager {
    pub fn new(custom_dir: &Path, base_dir: &Path) -> Self {
        let backup_dir = if custom_dir.as_os_str().is_empty() {
            base_dir.join(format!(
                "backup_{}",
                chrono::Utc::now().format("%Y%m%d_%H%M%S")
            ))
        } else {
            custom_dir.to_path_buf()
        };
        Self { backup_dir }
    }

    pub fn init(&self) -> Result<(), std::io::Error> {
        fs::create_dir_all(&self.backup_dir)
    }

    pub fn backup_file(&self, src: &Path) -> Result<PathBuf, std::io::Error> {
        let file_name = src.file_name().ok_or_else(|| {
            std::io::Error::new(std::io::ErrorKind::InvalidInput, "Invalid file name")
        })?;

        let dst = self.backup_dir.join(file_name);
        fs::copy(src, &dst)?;
        Ok(dst)
    }

    pub fn cleanup_if_empty(&self) {
        if self.backup_dir.exists() {
            let _ = fs::read_dir(&self.backup_dir).map(|mut entries| {
                if entries.next().is_none() {
                    fs::remove_dir(&self.backup_dir).ok();
                }
            });
        }
    }

    pub fn dir(&self) -> &Path {
        &self.backup_dir
    }
}
