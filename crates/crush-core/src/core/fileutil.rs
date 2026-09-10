use std::path::Path;

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum FileType {
    Image,
    Video,
    Audio,
    Unknown,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct FileInfo {
    pub index: usize,
    pub path: String,
    pub name: String,
    pub ext: String,
    pub size: u64,
    pub size_str: String,
    pub file_type: FileType,
    pub type_name: String,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AnalyseStats {
    pub dir: String,
    pub total: usize,
    pub total_size: u64,
    pub images: usize,
    pub videos: usize,
    pub audio: usize,
    pub image_size: u64,
    pub video_size: u64,
    pub audio_size: u64,
    pub formats: std::collections::HashMap<String, usize>,
}

const VIDEO_EXTS: &[&str] = &[
    ".mp4", ".mov", ".avi", ".mkv", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg", ".3gp", ".ts",
];

const IMAGE_EXTS: &[&str] = &[
    ".jpg", ".jpeg", ".png", ".webp", ".bmp", ".tiff", ".tif", ".avif", ".gif", ".svg", ".ico",
    ".heic", ".heif",
];

const AUDIO_EXTS: &[&str] = &[
    ".mp3", ".wav", ".flac", ".ogg", ".aac", ".wma", ".m4a", ".opus", ".aiff", ".alac",
];

const FORMAT_NAMES: &[(&str, &str)] = &[
    (".jpg", "JPEG"),
    (".jpeg", "JPEG"),
    (".png", "PNG"),
    (".webp", "WebP"),
    (".avif", "AVIF"),
    (".gif", "GIF"),
    (".bmp", "BMP"),
    (".svg", "SVG"),
    (".ico", "ICO"),
    (".tiff", "TIFF"),
    (".heic", "HEIC"),
    (".mp4", "MP4"),
    (".mov", "MOV"),
    (".avi", "AVI"),
    (".mkv", "MKV"),
    (".wmv", "WMV"),
    (".flv", "FLV"),
    (".webm", "WebM"),
    (".m4v", "M4V"),
    (".mp3", "MP3"),
    (".wav", "WAV"),
    (".flac", "FLAC"),
    (".ogg", "OGG"),
    (".aac", "AAC"),
    (".m4a", "M4A"),
    (".opus", "Opus"),
];

pub fn detect_type(ext: &str) -> FileType {
    let lower = ext.to_lowercase();
    if VIDEO_EXTS.iter().any(|&e| e == lower) {
        FileType::Video
    } else if IMAGE_EXTS.iter().any(|&e| e == lower) {
        FileType::Image
    } else if AUDIO_EXTS.iter().any(|&e| e == lower) {
        FileType::Audio
    } else {
        FileType::Unknown
    }
}

pub fn format_display_name(ext: &str) -> String {
    let lower = ext.to_lowercase();
    for &(e, name) in FORMAT_NAMES {
        if e == lower {
            return name.to_string();
        }
    }
    ext.trim_start_matches('.').to_uppercase()
}

pub fn format_size(bytes: u64) -> String {
    if bytes < 1024 {
        return format!("{} B", bytes);
    }
    if bytes < 1024 * 1024 {
        return format!("{:.0} KB", bytes as f64 / 1024.0);
    }
    if bytes < 1024 * 1024 * 1024 {
        return format!("{:.1} MB", bytes as f64 / 1024.0 / 1024.0);
    }
    format!("{:.2} GB", bytes as f64 / 1024.0 / 1024.0 / 1024.0)
}

pub fn file_name_without_ext(name: &str) -> &str {
    Path::new(name)
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or(name)
}

pub fn can_convert_to(file_type: FileType, format: &str) -> bool {
    let f = format.to_lowercase();
    match file_type {
        FileType::Image => matches!(
            f.as_str(),
            "webp" | "avif" | "jpg" | "jpeg" | "png" | "gif" | "bmp" | "tiff"
        ),
        FileType::Video => matches!(
            f.as_str(),
            "mp4"
                | "webm"
                | "avi"
                | "mov"
                | "mkv"
                | "gif"
                | "mp3"
                | "ogg"
                | "wav"
                | "flac"
                | "aac"
                | "opus"
                | "m4a"
                | "alac"
        ),
        FileType::Audio => matches!(
            f.as_str(),
            "mp3" | "ogg" | "wav" | "flac" | "aac" | "opus" | "m4a" | "alac"
        ),
        FileType::Unknown => false,
    }
}

pub fn native_supports_format(input_ext: &str, target_ext: &str) -> bool {
    let inp = input_ext.to_lowercase();
    let tgt = target_ext.to_lowercase();

    let supported_input = matches!(
        inp.as_str(),
        ".png" | ".jpg" | ".jpeg" | ".bmp" | ".tiff" | ".tif" | ".gif"
    );
    let supported_output = matches!(
        tgt.as_str(),
        "webp" | "avif" | "png" | "jpg" | "jpeg" | "bmp"
    );

    supported_input && supported_output
}

pub fn scan_directory(dir: &Path) -> (Vec<FileInfo>, AnalyseStats) {
    let mut stats = AnalyseStats {
        dir: dir.display().to_string(),
        total: 0,
        total_size: 0,
        images: 0,
        videos: 0,
        audio: 0,
        image_size: 0,
        video_size: 0,
        audio_size: 0,
        formats: std::collections::HashMap::new(),
    };
    let mut files = Vec::new();

    let Ok(entries) = std::fs::read_dir(dir) else {
        return (files, stats);
    };

    let mut index = 0;
    for entry in entries.flatten() {
        let metadata = match entry.metadata() {
            Ok(m) => m,
            Err(_) => continue,
        };
        if metadata.is_dir() {
            continue;
        }

        let path = entry.path();
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_lowercase();

        if ext.is_empty() {
            continue;
        }

        let file_type = detect_type(&format!(".{}", ext));
        if file_type == FileType::Unknown {
            continue;
        }

        index += 1;
        let size = metadata.len();
        let name = entry.file_name().to_string_lossy().to_string();
        let type_name = format_display_name(&format!(".{}", ext));

        stats.total += 1;
        stats.total_size += size;
        *stats.formats.entry(ext.clone()).or_insert(0) += 1;

        match file_type {
            FileType::Image => {
                stats.images += 1;
                stats.image_size += size;
            }
            FileType::Video => {
                stats.videos += 1;
                stats.video_size += size;
            }
            FileType::Audio => {
                stats.audio += 1;
                stats.audio_size += size;
            }
            _ => {}
        }

        files.push(FileInfo {
            index,
            path: path.display().to_string(),
            name,
            ext,
            size,
            size_str: format_size(size),
            file_type,
            type_name,
        });
    }

    files.sort_by(|a, b| a.name.cmp(&b.name));
    for (i, f) in files.iter_mut().enumerate() {
        f.index = i + 1;
    }

    (files, stats)
}

pub fn filter_by_type(files: &[FileInfo], filter: &str) -> Vec<FileInfo> {
    let ft = match filter {
        "image" => FileType::Image,
        "video" => FileType::Video,
        "audio" => FileType::Audio,
        _ => return files.to_vec(),
    };
    files
        .iter()
        .filter(|f| f.file_type == ft)
        .cloned()
        .collect()
}
