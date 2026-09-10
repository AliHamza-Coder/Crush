use crush_core::core::backup::BackupManager;
use crush_core::core::config;
use crush_core::core::fileutil::{self, AnalyseStats, FileInfo};
use crush_core::core::queue::{Engine, QueueState, TaskQueue, TaskType};
use crush_core::engine;
use crush_core::engine::ai_upscale::AiUpscaleEngine;
use crush_core::engine::ffmpeg::FfmpegEngine;
use crush_core::engine::native::NativeEngine;
use std::path::PathBuf;

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Panel {
    Menu,
    Files,
    Quality,
    Format,
    Queue,
    QualitySubmenu,
    BackupConfirm,
    Help,
}

/// Geometry of the interactive list captured on the last frame, so mouse
/// clicks can be mapped reliably even when the terminal is resized.
#[derive(Clone, Copy, Default)]
pub struct UiGeom {
    /// X column where the menu list begins (Panel::Menu only).
    pub menu_x: u16,
    /// Width of the interactive list column.
    pub menu_w: u16,
    /// Absolute Y of the first selectable row for the active panel.
    pub item_y: u16,
}

impl Panel {
    #[allow(dead_code)]
    pub fn all() -> &'static [Panel] {
        &[
            Panel::Menu,
            Panel::Files,
            Panel::Quality,
            Panel::Format,
            Panel::Queue,
            Panel::QualitySubmenu,
            Panel::BackupConfirm,
        ]
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum ProcessMode {
    Compress,
    Convert,
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum SubmenuFlow {
    None,
    QualityFirst,
    FormatFirst,
}

impl SubmenuFlow {
    #[allow(dead_code)]
    pub fn is_none(&self) -> bool {
        matches!(self, SubmenuFlow::None)
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum MenuAction {
    CompressAll,
    CompressImages,
    CompressVideos,
    CompressAudio,
    ConvertAll,
    ConvertImages,
    ConvertVideos,
    ConvertAudio,
    ExtractAudio,
    SelectFiles,
    Arrange,
    ChangeDir,
    Favicon,
    CheckDeps,
    Quit,
}

impl MenuAction {
    pub fn all() -> Vec<Self> {
        vec![
            Self::CompressAll,
            Self::CompressImages,
            Self::CompressVideos,
            Self::CompressAudio,
            Self::ConvertAll,
            Self::ConvertImages,
            Self::ConvertVideos,
            Self::ConvertAudio,
            Self::ExtractAudio,
            Self::SelectFiles,
            Self::Arrange,
            Self::ChangeDir,
            Self::Favicon,
            Self::CheckDeps,
            Self::Quit,
        ]
    }

    pub fn label(&self) -> &str {
        match self {
            Self::CompressAll => "Compress ALL — best quality at smaller size",
            Self::CompressImages => "Compress Images — webp/avif/jpg (q85 = 90% smaller)",
            Self::CompressVideos => "Compress Videos — mp4/webm (CRF 18-28 = great)",
            Self::CompressAudio => "Compress Audio — mp3/flac/ogg (q85 = transparent)",
            Self::ConvertAll => "Convert ALL — change format, keep quality",
            Self::ConvertImages => "Convert Images — png→webp, jpg→avif, etc.",
            Self::ConvertVideos => "Convert Videos — mp4→webm, mov→mp4, etc.",
            Self::ConvertAudio => "Convert Audio — wav→mp3, flac→ogg, etc.",
            Self::ExtractAudio => "Extract Audio — mp4→mp3, video→wav",
            Self::SelectFiles => "Select Files — pick specific files to process",
            Self::Arrange => "Arrange Files — sort into 'All webp', 'All mp4' folders",
            Self::ChangeDir => "Change Directory — browse to a folder",
            Self::Favicon => "Generate Favicon — 16×16 + 32×32 SVG from image",
            Self::CheckDeps => "Check Dependencies — FFmpeg, ONNX model status",
            Self::Quit => "Quit",
        }
    }

    #[allow(dead_code)]
    pub fn icon(&self) -> &str {
        match self {
            Self::CompressAll => "◆",
            Self::CompressImages => "🖼",
            Self::CompressVideos => "🎬",
            Self::CompressAudio => "🎵",
            Self::ConvertAll => "◇",
            Self::ConvertImages => "🖼",
            Self::ConvertVideos => "🎬",
            Self::ConvertAudio => "🎵",
            Self::ExtractAudio => "🔊",
            Self::SelectFiles => "#",
            Self::Arrange => "📂",
            Self::ChangeDir => "📁",
            Self::Favicon => "🌐",
            Self::CheckDeps => "🔍",
            Self::Quit => "✕",
        }
    }
}

pub struct App {
    pub dir: PathBuf,
    pub files: Vec<FileInfo>,
    pub stats: Option<AnalyseStats>,
    pub active_panel: Panel,
    pub menu_idx: usize,
    pub selected_file: usize,
    pub quality: u8,
    pub quality_idx: usize,
    pub format: String,
    pub format_idx: usize,
    pub backup: bool,
    #[allow(dead_code)]
    pub parallel: usize,
    pub processing: bool,
    pub queue: TaskQueue,
    pub queue_state: Option<QueueState>,
    pub ffmpeg: Option<FfmpegEngine>,
    pub native: NativeEngine,
    pub ai: AiUpscaleEngine,
    pub status_msg: String,
    pub input_mode: bool,
    pub input_buffer: String,
    pub input_prompt: String,
    pub filter_type: String,
    pub selected_indices: Vec<usize>,
    pub select_mode: bool,
    pub lossless: bool,
    pub mode: ProcessMode,
    pub elapsed_ms: u64,
    pub deps_status: Vec<crush_core::core::deps::DepStatus>,
    pub should_quit: bool,
    pub submenu_flow: SubmenuFlow,
    pub submenu_idx: usize,
    pub submenu_items: Vec<String>,
    pub geom: UiGeom,
}

impl App {
    pub fn new() -> Self {
        let ffmpeg = config::find_ffmpeg().map(|p| FfmpegEngine::new(&p.display().to_string()));
        let native = NativeEngine::new();
        let ai = AiUpscaleEngine::new(&crush_core::core::deps::crush_models_dir());

        Self {
            dir: std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")),
            files: Vec::new(),
            stats: None,
            active_panel: Panel::Menu,
            menu_idx: 0,
            selected_file: 0,
            quality: 85,
            quality_idx: 2,
            format: "webp".into(),
            format_idx: 0,
            backup: true,
            parallel: 4,
            processing: false,
            queue: TaskQueue::new(),
            queue_state: None,
            ffmpeg,
            native,
            ai,
            status_msg: String::new(),
            input_mode: false,
            input_buffer: String::new(),
            input_prompt: String::new(),
            filter_type: "all".into(),
            selected_indices: Vec::new(),
            select_mode: false,
            lossless: false,
            mode: ProcessMode::Compress,
            elapsed_ms: 0,
            deps_status: Vec::new(),
            should_quit: false,
            submenu_flow: SubmenuFlow::None,
            submenu_idx: 0,
            submenu_items: Vec::new(),
            geom: UiGeom::default(),
        }
    }

    pub fn init(&mut self) -> anyhow::Result<()> {
        self.scan()?;
        self.deps_status = crush_core::core::deps::check_all();

        let mut msgs = Vec::new();
        if self.ffmpeg.is_some() {
            msgs.push("FFmpeg ✓");
        } else {
            msgs.push("FFmpeg ✗");
        }
        if self.ai.is_model_available() {
            msgs.push("ONNX ✓");
        } else {
            msgs.push("ONNX ✗");
        }
        msgs.push("Native ✓");
        self.status_msg = msgs.join("  |  ");
        Ok(())
    }

    pub fn scan(&mut self) -> anyhow::Result<()> {
        let (files, stats) = fileutil::scan_directory(&self.dir);
        self.files = files;
        self.stats = Some(stats);
        self.selected_file = 0;
        self.selected_indices.clear();
        Ok(())
    }

    pub fn get_filtered_files(&self) -> Vec<FileInfo> {
        fileutil::filter_by_type(&self.files, &self.filter_type)
    }

    pub fn menu_items() -> Vec<MenuAction> {
        MenuAction::all()
    }

    pub fn next_menu(&mut self) {
        let len = Self::menu_items().len();
        self.menu_idx = (self.menu_idx + 1) % len;
    }

    pub fn prev_menu(&mut self) {
        let len = Self::menu_items().len();
        self.menu_idx = if self.menu_idx == 0 {
            len - 1
        } else {
            self.menu_idx - 1
        };
    }

    pub fn next_file(&mut self) {
        let len = self.get_filtered_files().len();
        if len > 0 {
            self.selected_file = (self.selected_file + 1) % len;
        }
    }

    pub fn prev_file(&mut self) {
        let len = self.get_filtered_files().len();
        if len > 0 {
            self.selected_file = if self.selected_file == 0 {
                len - 1
            } else {
                self.selected_file - 1
            };
        }
    }

    pub fn toggle_select_file(&mut self) {
        let files = self.get_filtered_files();
        if let Some(file) = files.get(self.selected_file) {
            let idx = file.index;
            if let Some(pos) = self.selected_indices.iter().position(|&i| i == idx) {
                self.selected_indices.remove(pos);
            } else {
                self.selected_indices.push(idx);
            }
        }
    }

    pub fn next_quality(&mut self) {
        self.quality_idx = (self.quality_idx + 1) % 6;
        self.quality = match self.quality_idx {
            0 => 100,
            1 => 90,
            2 => 85,
            3 => 75,
            4 => 60,
            _ => 85,
        };
    }

    pub fn prev_quality(&mut self) {
        self.quality_idx = if self.quality_idx == 0 {
            5
        } else {
            self.quality_idx - 1
        };
        self.quality = match self.quality_idx {
            0 => 100,
            1 => 90,
            2 => 85,
            3 => 75,
            4 => 60,
            _ => 85,
        };
    }

    pub fn set_custom_quality(&mut self, val: u8) {
        self.quality = val.clamp(1, 100);
        self.quality_idx = 5;
    }

    pub fn next_format(&mut self) {
        let formats = self.available_formats();
        self.format_idx = (self.format_idx + 1) % formats.len();
        self.format = formats[self.format_idx].clone();
    }

    pub fn prev_format(&mut self) {
        let formats = self.available_formats();
        self.format_idx = if self.format_idx == 0 {
            formats.len() - 1
        } else {
            self.format_idx - 1
        };
        self.format = formats[self.format_idx].clone();
    }

    pub fn available_formats(&self) -> Vec<String> {
        let has_image = self
            .files
            .iter()
            .any(|f| f.file_type == fileutil::FileType::Image);
        let has_video = self
            .files
            .iter()
            .any(|f| f.file_type == fileutil::FileType::Video);
        let has_audio = self
            .files
            .iter()
            .any(|f| f.file_type == fileutil::FileType::Audio);

        let mut formats = Vec::new();
        if has_image {
            formats.extend(
                ["webp", "avif", "png", "jpg", "bmp"]
                    .iter()
                    .map(|s| s.to_string()),
            );
        }
        if has_video {
            formats.extend(
                ["mp4", "webm", "mkv", "mov", "avi"]
                    .iter()
                    .map(|s| s.to_string()),
            );
        }
        if has_audio {
            formats.extend(
                ["mp3", "flac", "ogg", "wav", "aac", "opus", "m4a", "alac"]
                    .iter()
                    .map(|s| s.to_string()),
            );
        }
        if formats.is_empty() {
            formats.extend(
                ["webp", "avif", "png", "jpg", "mp4", "mp3"]
                    .iter()
                    .map(|s| s.to_string()),
            );
        }
        formats
    }

    #[allow(dead_code)]
    pub fn quality_label(&self) -> &str {
        match self.quality {
            100 => "Maximum",
            90 => "High",
            85 => "Balanced ★",
            75 => "Smaller",
            60 => "Compact",
            _ => "Custom",
        }
    }

    pub fn quality_options_for(filter: &str) -> Vec<(u8, &'static str, &'static str)> {
        match filter {
            "video" => vec![
                (100, "Maximum", "CRF 18 — near-lossless, largest"),
                (90, "High", "CRF 20 — high quality"),
                (85, "Balanced ★", "CRF 23 — good quality, ~50% smaller"),
                (75, "Smaller", "CRF 28 — smaller, some quality loss"),
                (60, "Compact", "CRF 32 — very small, lower quality"),
            ],
            "audio" => vec![
                (100, "Maximum", "VBR ~320kbps — best quality"),
                (90, "High", "VBR ~256kbps — high quality"),
                (85, "Balanced ★", "VBR ~192kbps — excellent, recommended"),
                (75, "Smaller", "VBR ~160kbps — smaller file"),
                (60, "Compact", "VBR ~128kbps — good for podcasts"),
            ],
            _ => vec![
                (100, "Maximum", "best quality, largest file"),
                (90, "High", "slightly larger"),
                (85, "Balanced ★", "good quality, ~50-70% smaller"),
                (75, "Smaller", "slightly lower quality"),
                (60, "Compact", "good for web sharing"),
            ],
        }
    }

    pub fn quality_submenu_items(filter: &str) -> Vec<String> {
        let opts = Self::quality_options_for(filter);
        let mut items: Vec<String> = opts
            .iter()
            .map(|(q, label, desc)| format!("{:>3}% — {:<10} {}", q, label, desc))
            .collect();
        items.push("Lossless — original quality preserved".into());
        items.push("Custom — enter any value (1-100)".into());
        items
    }

    pub fn confirm_submenu_quality(&mut self) {
        let opts = Self::quality_options_for(&self.filter_type);
        if self.submenu_idx < opts.len() {
            self.quality = opts[self.submenu_idx].0;
            self.lossless = false;
        } else if self.submenu_idx == opts.len() {
            self.lossless = true;
            self.quality = 0;
        } else {
            self.input_mode = true;
            self.input_buffer.clear();
            self.input_prompt = "Enter quality (1-100):".into();
            self.status_msg = "Type quality number, press Enter".into();
            return;
        }
        self.proceed_after_quality();
    }

    pub fn proceed_after_quality(&mut self) {
        match self.submenu_flow {
            SubmenuFlow::QualityFirst => {
                self.show_backup_confirm();
            }
            SubmenuFlow::FormatFirst => {
                self.show_backup_confirm();
            }
            SubmenuFlow::None => {}
        }
    }

    pub fn show_backup_confirm(&mut self) {
        self.submenu_items = vec![
            "Yes — backup originals before processing".into(),
            "No — skip backup".into(),
        ];
        self.submenu_idx = 0;
        self.active_panel = Panel::BackupConfirm;
        self.status_msg = "Backup originals? (↑↓ to choose, Enter to confirm)".into();
    }

    pub fn confirm_backup(&mut self) {
        self.backup = self.submenu_idx == 0;
        self.start_processing().ok();
    }

    pub fn start_submenu_flow(&mut self, flow: SubmenuFlow) {
        self.submenu_flow = flow;
        self.submenu_idx = 2;
        self.submenu_items = Self::quality_submenu_items(&self.filter_type);
        self.active_panel = Panel::QualitySubmenu;
        self.status_msg = "Select quality (↑↓ to choose, Enter to confirm)".into();
    }

    pub fn execute_menu_action(&mut self) -> anyhow::Result<()> {
        let action = Self::menu_items()[self.menu_idx];
        match action {
            MenuAction::CompressAll => {
                self.mode = ProcessMode::Compress;
                self.filter_type = "all".into();
                self.start_submenu_flow(SubmenuFlow::QualityFirst);
            }
            MenuAction::CompressImages => {
                self.mode = ProcessMode::Compress;
                self.filter_type = "image".into();
                self.start_submenu_flow(SubmenuFlow::QualityFirst);
            }
            MenuAction::CompressVideos => {
                self.mode = ProcessMode::Compress;
                self.filter_type = "video".into();
                self.start_submenu_flow(SubmenuFlow::QualityFirst);
            }
            MenuAction::CompressAudio => {
                self.mode = ProcessMode::Compress;
                self.filter_type = "audio".into();
                self.start_submenu_flow(SubmenuFlow::QualityFirst);
            }
            MenuAction::ConvertAll => {
                self.mode = ProcessMode::Convert;
                self.filter_type = "all".into();
                self.active_panel = Panel::Format;
                self.status_msg = "Select target format, then press Enter".into();
            }
            MenuAction::ConvertImages => {
                self.mode = ProcessMode::Convert;
                self.filter_type = "image".into();
                self.active_panel = Panel::Format;
                self.status_msg = "Select target format".into();
            }
            MenuAction::ConvertVideos => {
                self.mode = ProcessMode::Convert;
                self.filter_type = "video".into();
                self.active_panel = Panel::Format;
                self.status_msg = "Select target format".into();
            }
            MenuAction::ConvertAudio => {
                self.mode = ProcessMode::Convert;
                self.filter_type = "audio".into();
                self.active_panel = Panel::Format;
                self.status_msg = "Select target format".into();
            }
            MenuAction::ExtractAudio => {
                self.filter_type = "video".into();
                self.format = "mp3".into();
                self.start_audio_extract()?;
            }
            MenuAction::SelectFiles => {
                self.select_mode = true;
                self.active_panel = Panel::Files;
                self.status_msg = "Space: toggle | Enter: confirm | Esc: cancel".into();
            }
            MenuAction::Arrange => {
                self.arrange_files()?;
            }
            MenuAction::ChangeDir => {
                let dialog = rfd::FileDialog::new()
                    .set_title("Select Directory")
                    .set_directory(&self.dir);
                if let Some(path) = dialog.pick_folder() {
                    self.dir = path;
                    self.scan()?;
                    self.status_msg = format!("Directory: {}", self.dir.display());
                } else {
                    self.status_msg = "Directory selection cancelled".into();
                }
            }
            MenuAction::Favicon => {
                self.generate_favicon()?;
            }
            MenuAction::CheckDeps => {
                self.deps_status = crush_core::core::deps::check_all();
                self.status_msg = self
                    .deps_status
                    .iter()
                    .map(|d| format!("{}: {}", d.name, if d.available { "✓" } else { "✗" }))
                    .collect::<Vec<_>>()
                    .join("  |  ");
            }
            MenuAction::Quit => {
                self.should_quit = true;
            }
        }
        Ok(())
    }

    pub fn confirm_select_files(&mut self) -> anyhow::Result<()> {
        if self.selected_indices.is_empty() {
            self.status_msg = "No files selected".into();
            self.select_mode = false;
            self.active_panel = Panel::Menu;
            return Ok(());
        }

        self.filter_type = "custom".into();
        self.start_processing()?;
        self.select_mode = false;
        self.active_panel = Panel::Menu;
        Ok(())
    }

    pub fn parse_range_selection(&mut self, input: &str) -> anyhow::Result<()> {
        let files = self.get_filtered_files();
        if files.is_empty() {
            self.status_msg = "No files available".into();
            return Ok(());
        }

        self.selected_indices.clear();
        let trimmed = input.trim().to_lowercase();

        if trimmed == "all" {
            for f in &files {
                self.selected_indices.push(f.index);
            }
            self.status_msg = format!("Selected all {} files", files.len());
            return Ok(());
        }

        for part in trimmed.split(',') {
            let part = part.trim();
            if let Some((start_s, end_s)) = part.split_once('-') {
                let start: usize = start_s
                    .trim()
                    .parse()
                    .map_err(|_| anyhow::anyhow!("Invalid range: {}", part))?;
                let end: usize = end_s
                    .trim()
                    .parse()
                    .map_err(|_| anyhow::anyhow!("Invalid range: {}", part))?;
                for f in &files {
                    if f.index >= start && f.index <= end {
                        self.selected_indices.push(f.index);
                    }
                }
            } else {
                let num: usize = part
                    .parse()
                    .map_err(|_| anyhow::anyhow!("Invalid number: {}", part))?;
                if let Some(f) = files.iter().find(|f| f.index == num) {
                    self.selected_indices.push(f.index);
                }
            }
        }

        self.selected_indices.sort();
        self.selected_indices.dedup();
        self.status_msg = format!("Selected {} files", self.selected_indices.len());
        Ok(())
    }

    pub fn start_processing(&mut self) -> anyhow::Result<()> {
        let files = if self.filter_type == "custom" {
            self.files
                .iter()
                .filter(|f| self.selected_indices.contains(&f.index))
                .cloned()
                .collect::<Vec<_>>()
        } else {
            self.get_filtered_files()
        };

        if files.is_empty() {
            self.status_msg = "No files to process".into();
            return Ok(());
        }

        self.processing = true;
        self.active_panel = Panel::Queue;
        self.status_msg = format!("Processing {} files...", files.len());

        let quality = self.quality;
        let format = self.format.clone();
        let filter = self.filter_type.clone();
        let mode = self.mode;
        let _lossless = self.lossless;
        let backup = self.backup;
        let dir = self.dir.clone();
        let queue = self.queue.clone();
        let ffmpeg = self.ffmpeg.clone();
        let native = self.native.clone();
        let ai = self.ai.clone();

        let backup_mgr = if backup {
            let mgr = BackupManager::new(&dir.join("backup"), &dir);
            mgr.init().ok();
            Some(mgr)
        } else {
            None
        };

        tokio::spawn(async move {
            let mut q = queue;
            q.start().await;

            for file in &files {
                if backup {
                    if let Some(ref mgr) = backup_mgr {
                        let _ = mgr.backup_file(std::path::Path::new(&file.path));
                    }
                }

                let task_type = match filter.as_str() {
                    "video" => {
                        if matches!(mode, ProcessMode::Convert) {
                            if is_audio_format(&format) {
                                TaskType::AudioExtract {
                                    format: format.clone(),
                                    quality,
                                }
                            } else {
                                TaskType::VideoConvert {
                                    target: format.clone(),
                                    quality,
                                }
                            }
                        } else {
                            TaskType::VideoCompress {
                                quality,
                                format: format.clone(),
                            }
                        }
                    }
                    "audio" => TaskType::AudioConvert {
                        format: format.clone(),
                        quality,
                    },
                    "image" => {
                        if matches!(mode, ProcessMode::Convert) {
                            TaskType::ImageConvert {
                                target: format.clone(),
                                quality,
                            }
                        } else {
                            TaskType::ImageCompress {
                                quality,
                                format: format.clone(),
                            }
                        }
                    }
                    "custom" => match file.file_type {
                        fileutil::FileType::Video => {
                            if matches!(mode, ProcessMode::Convert) {
                                if is_audio_format(&format) {
                                    TaskType::AudioExtract {
                                        format: format.clone(),
                                        quality,
                                    }
                                } else {
                                    TaskType::VideoConvert {
                                        target: format.clone(),
                                        quality,
                                    }
                                }
                            } else {
                                TaskType::VideoCompress {
                                    quality,
                                    format: format.clone(),
                                }
                            }
                        }
                        fileutil::FileType::Image => {
                            if matches!(mode, ProcessMode::Convert) {
                                TaskType::ImageConvert {
                                    target: format.clone(),
                                    quality,
                                }
                            } else {
                                TaskType::ImageCompress {
                                    quality,
                                    format: format.clone(),
                                }
                            }
                        }
                        fileutil::FileType::Audio => TaskType::AudioConvert {
                            format: format.clone(),
                            quality,
                        },
                        _ => continue,
                    },
                    _ => match file.file_type {
                        fileutil::FileType::Video => {
                            if matches!(mode, ProcessMode::Convert) {
                                if is_audio_format(&format) {
                                    TaskType::AudioExtract {
                                        format: format.clone(),
                                        quality,
                                    }
                                } else {
                                    TaskType::VideoConvert {
                                        target: format.clone(),
                                        quality,
                                    }
                                }
                            } else {
                                TaskType::VideoCompress {
                                    quality,
                                    format: format.clone(),
                                }
                            }
                        }
                        fileutil::FileType::Image => {
                            if matches!(mode, ProcessMode::Convert) {
                                TaskType::ImageConvert {
                                    target: format.clone(),
                                    quality,
                                }
                            } else {
                                TaskType::ImageCompress {
                                    quality,
                                    format: format.clone(),
                                }
                            }
                        }
                        fileutil::FileType::Audio => TaskType::AudioConvert {
                            format: format.clone(),
                            quality,
                        },
                        _ => continue,
                    },
                };

                let engine_type = engine::select_engine(&task_type, &file.ext);
                let output_name =
                    format!("{}.{}", fileutil::file_name_without_ext(&file.name), format);
                let output_path = dir.join(&output_name);

                q.add_task(
                    file.path.clone(),
                    output_path.display().to_string(),
                    file.name.clone(),
                    file.size,
                    file.size_str.clone(),
                    task_type,
                    engine_type,
                )
                .await;
            }

            let state = q.get_state().await;
            for task in &state.tasks {
                let mut task = task.clone();

                // Only skip if input and output are the same file (prevent self-overwrite)
                let input_path = std::path::Path::new(&task.input_path);
                let output_path = std::path::Path::new(&task.output_path);
                let same_file = input_path.canonicalize().ok() == output_path.canonicalize().ok();
                if same_file {
                    q.update_status(task.id, crush_core::core::queue::TaskStatus::Skipped)
                        .await;
                    continue;
                }

                let result = match &task.engine {
                    Engine::Ffmpeg => {
                        if let Some(ref eng) = ffmpeg {
                            eng.process(&mut task).await
                        } else {
                            Err(anyhow::anyhow!("FFmpeg not found"))
                        }
                    }
                    Engine::Native => native.process(&mut task).await,
                    Engine::OnnxAi => ai.process(&mut task).await,
                };

                if let Err(e) = result {
                    q.set_error(task.id, e.to_string()).await;
                } else {
                    // Engines update a local task copy; write status + progress
                    // back into the queue so the UI sees completion.
                    q.update_status(task.id, crush_core::core::queue::TaskStatus::Completed)
                        .await;
                    q.update_progress(task.id, task.progress).await;
                }
            }
        });

        Ok(())
    }

    pub fn start_audio_extract(&mut self) -> anyhow::Result<()> {
        let videos: Vec<FileInfo> = self
            .files
            .iter()
            .filter(|f| f.file_type == fileutil::FileType::Video)
            .cloned()
            .collect();

        if videos.is_empty() {
            self.status_msg = "No video files to extract audio from".into();
            return Ok(());
        }

        self.processing = true;
        self.active_panel = Panel::Queue;

        let quality = self.quality;
        let format = self.format.clone();
        let dir = self.dir.clone();
        let queue = self.queue.clone();
        let ffmpeg = self.ffmpeg.clone();

        tokio::spawn(async move {
            let mut q = queue;
            q.start().await;

            for file in &videos {
                let task_type = TaskType::AudioExtract {
                    format: format.clone(),
                    quality,
                };
                let output_name =
                    format!("{}.{}", fileutil::file_name_without_ext(&file.name), format);
                let output_path = dir.join(&output_name);

                q.add_task(
                    file.path.clone(),
                    output_path.display().to_string(),
                    file.name.clone(),
                    file.size,
                    file.size_str.clone(),
                    task_type,
                    Engine::Ffmpeg,
                )
                .await;
            }

            let state = q.get_state().await;
            for task in &state.tasks {
                let mut task = task.clone();
                let result = if let Some(ref eng) = ffmpeg {
                    eng.process(&mut task).await
                } else {
                    Err(anyhow::anyhow!("FFmpeg not found"))
                };
                if let Err(e) = result {
                    q.set_error(task.id, e.to_string()).await;
                } else {
                    q.update_status(task.id, crush_core::core::queue::TaskStatus::Completed)
                        .await;
                    q.update_progress(task.id, task.progress).await;
                }
            }
        });

        Ok(())
    }

    pub fn arrange_files(&mut self) -> anyhow::Result<()> {
        use std::collections::HashMap;
        use std::fs;

        let mut groups: HashMap<String, Vec<String>> = HashMap::new();
        for entry in fs::read_dir(&self.dir)? {
            let entry = entry?;
            if entry.file_type()?.is_dir() {
                continue;
            }
            let name = entry.file_name().to_string_lossy().to_string();
            if name.starts_with("All ") || name.starts_with("backup") {
                continue;
            }
            let ext = std::path::Path::new(&name)
                .extension()
                .and_then(|e| e.to_str())
                .unwrap_or("no_extension")
                .to_lowercase();
            groups
                .entry(ext)
                .or_default()
                .push(entry.path().display().to_string());
        }

        let mut moved = 0;
        for (ext, paths) in &groups {
            let folder = self.dir.join(format!("All {}", ext));
            fs::create_dir_all(&folder)?;
            for path in paths {
                let src = std::path::Path::new(path);
                let dst = folder.join(src.file_name().unwrap());
                if !dst.exists() && fs::rename(src, &dst).is_ok() {
                    moved += 1;
                }
            }
        }

        self.status_msg = format!("Arranged {} files into 'All <ext>' folders", moved);
        self.scan()?;
        Ok(())
    }

    pub fn generate_favicon(&mut self) -> anyhow::Result<()> {
        let images: Vec<&FileInfo> = self
            .files
            .iter()
            .filter(|f| f.file_type == fileutil::FileType::Image)
            .collect();

        if images.is_empty() {
            self.status_msg = "No image files for favicon generation".into();
            return Ok(());
        }

        let img = &images[0];
        let base = fileutil::file_name_without_ext(&img.name);
        let out_dir = &self.dir;

        if let Some(ref ffmpeg_path) = config::find_ffmpeg() {
            for &size in &[16u32, 32] {
                let png_tmp = out_dir.join(format!("{}_{}x{}_tmp.png", base, size, size));
                let svg_out = out_dir.join(format!("favicon_{}x{}.svg", size, size));

                let status = std::process::Command::new(ffmpeg_path)
                    .args([
                        "-i",
                        &img.path,
                        "-vf",
                        &format!("scale={}:{}:flags=lanczos", size, size),
                        "-y",
                        &png_tmp.display().to_string(),
                    ])
                    .status()?;

                if status.success() {
                    let png_data = std::fs::read(&png_tmp)?;
                    let b64 = base64::Engine::encode(
                        &base64::engine::general_purpose::STANDARD,
                        &png_data,
                    );
                    let svg = format!(
                        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"{}\" height=\"{}\" viewBox=\"0 0 {} {}\">\n  <image width=\"{}\" height=\"{}\" href=\"data:image/png;base64,{}\"/>\n</svg>",
                        size, size, size, size, size, size, b64
                    );
                    std::fs::write(&svg_out, svg)?;
                    let _ = std::fs::remove_file(&png_tmp);
                }
            }
            self.status_msg = "Favicons generated: favicon_16x16.svg, favicon_32x32.svg".into();
        } else {
            self.status_msg = "FFmpeg required for favicon generation".into();
        }

        Ok(())
    }

    pub fn cancel(&mut self) {
        self.processing = false;
        self.status_msg = "Cancelled".into();
    }

    pub fn clear_queue(&mut self) {
        let queue = self.queue.clone();
        tokio::spawn(async move {
            queue.clear().await;
        });
        self.status_msg = "Queue cleared".into();
    }

    pub fn tick(&mut self) {
        if self.processing {
            let queue = self.queue.clone();
            let rt = tokio::runtime::Handle::current();
            if let Ok(state) = rt.block_on(async { Ok::<_, ()>(queue.get_state().await) }) {
                self.elapsed_ms = state.elapsed_ms;
                if state.running == 0 && state.pending == 0 && state.total > 0 {
                    self.processing = false;
                    let elapsed = self.elapsed_ms as f64 / 1000.0;
                    self.status_msg = format!(
                        "Done: {} OK, {} FAIL, {} SKIP in {:.1}s",
                        state.completed,
                        state.failed,
                        state
                            .tasks
                            .iter()
                            .filter(|t| t.status == crush_core::core::queue::TaskStatus::Skipped)
                            .count(),
                        elapsed
                    );
                    print!("\x07");
                }
                self.queue_state = Some(state);
            }
        }
    }
}

fn is_audio_format(format: &str) -> bool {
    matches!(
        format.to_lowercase().as_str(),
        "mp3" | "flac" | "ogg" | "wav" | "aac" | "opus" | "m4a" | "alac"
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    // Regression test: pressing Enter on the "Backup Originals?" confirm used to
    // panic with "there is no reactor running" because `start_processing` calls
    // `tokio::spawn` outside any Tokio runtime. This test drives the same code
    // path from inside a runtime (as `tui::run` now does).
    #[test]
    fn start_processing_runs_under_runtime() {
        let src = std::path::Path::new("F:/Crush/Images test/1.png");
        if !src.exists() {
            eprintln!("skipping: fixture F:/Crush/Images test/1.png not found");
            return;
        }

        let rt = tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .expect("runtime");
        let _guard = rt.enter();

        let dir = std::env::temp_dir().join("crush_tui_processing_test");
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).expect("create temp dir");
        let _ = std::fs::copy(src, dir.join("1.png"));

        let mut app = App::new();
        app.dir = dir.clone();
        app.filter_type = "image".into();
        app.quality = 85;
        app.format = "webp".into();
        app.mode = ProcessMode::Compress;
        app.backup = false;
        app.scan().expect("scan");
        assert_eq!(
            app.get_filtered_files().len(),
            1,
            "should see the copied png"
        );

        app.start_processing()
            .expect("start_processing must not panic");

        let mut done = false;
        for _ in 0..300 {
            std::thread::sleep(std::time::Duration::from_millis(100));
            app.tick();
            if !app.processing {
                done = true;
                break;
            }
        }

        let _ = std::fs::remove_dir_all(&dir);
        assert!(done, "processing should have completed");
        if let Some(state) = &app.queue_state {
            assert!(state.completed >= 1, "at least one task completed");
        }
    }
}
