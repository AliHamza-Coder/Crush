use clap::{Parser, Subcommand};
use std::path::PathBuf;

use crush_core::core::config;
use crush_core::core::fileutil;
use crush_core::core::deps;
use crush_core::core::backup::BackupManager;
use crush_core::engine::ffmpeg::FfmpegEngine;
use crush_core::engine::native::NativeEngine;
use crush_core::engine;
use crush_core::core::queue::{TaskQueue, TaskType, Engine, TaskStatus};

#[derive(Parser)]
#[command(name = "crush", version = "3.0.0", about = "Multimedia Mission Control", long_version = "CRUSH v3.0.0", disable_version_flag = true)]
pub struct Args {
    #[arg(short, long)]
    pub input: Option<PathBuf>,

    #[arg(short, long)]
    pub output: Option<PathBuf>,

    #[arg(short, long)]
    pub format: Option<String>,

    #[arg(short, long, default_value = "85")]
    pub quality: u8,

    #[arg(long)]
    pub lossless: bool,

    #[arg(short, long)]
    pub r#type: Option<String>,

    #[arg(short, long)]
    pub parallel: Option<usize>,

    #[arg(long)]
    pub no_backup: bool,

    #[arg(long)]
    pub backup_dir: Option<PathBuf>,

    #[arg(short = 'd', long)]
    pub dry_run: bool,

    #[arg(short = 'V', long)]
    pub verbose: bool,

    #[arg(short = 'v', long = "version")]
    pub version_flag: bool,

    #[command(subcommand)]
    pub command: Option<Command>,
}

#[derive(Subcommand)]
pub enum Command {
    Analyse {
        #[arg(default_value = ".")]
        dir: String,
        #[arg(short, long)]
        json: bool,
    },
    Install,
    Uninstall,
    Update,
    CheckDeps,
    Setup,
    #[command(name = "ai-check", hide = true)]
    AiCheck,
    #[command(alias = "v")]
    Version,
}

pub fn run_analyse(dir: &str, json: bool) -> anyhow::Result<()> {
    let path = std::path::Path::new(dir);
    let (files, stats) = fileutil::scan_directory(path);

    if json {
        println!("{}", serde_json::json!({
            "dir": stats.dir,
            "total": stats.total,
            "total_size": stats.total_size,
            "images": stats.images,
            "videos": stats.videos,
            "audio": stats.audio,
            "formats": stats.formats,
        }));
    } else {
        println!("\n  Directory: {}", stats.dir);
        println!("  Total: {} files | {}", stats.total, fileutil::format_size(stats.total_size));
        if stats.images > 0 { println!("  Images: {} ({})", stats.images, fileutil::format_size(stats.image_size)); }
        if stats.videos > 0 { println!("  Videos: {} ({})", stats.videos, fileutil::format_size(stats.video_size)); }
        if stats.audio > 0 { println!("  Audio:  {} ({})", stats.audio, fileutil::format_size(stats.audio_size)); }
        println!();
        for f in &files {
            let (icon, _) = match f.file_type {
                fileutil::FileType::Image => ("🖼", "green"),
                fileutil::FileType::Video => ("🎬", "cyan"),
                fileutil::FileType::Audio => ("🎵", "yellow"),
                _ => ("📄", "gray"),
            };
            println!("  {:<4} {} {:<8} {:<10} {:<6}  {}",
                f.index, icon, f.type_name, f.size_str,
                f.ext.trim_start_matches('.').to_uppercase(), f.name);
        }

        println!("\n  Formats:");
        let mut fmt_parts: Vec<String> = stats.formats.iter()
            .map(|(ext, count)| format!("{} x{}", ext.trim_start_matches('.'), count))
            .collect();
        fmt_parts.sort();
        println!("    {}", fmt_parts.join("  "));
    }

    Ok(())
}

pub fn run_install() -> anyhow::Result<()> {
    println!("\n  ╔══════════════════════════════════════╗");
    println!("  ║     CRUSH v3.0.0 — Installation      ║");
    println!("  ╚══════════════════════════════════════╝\n");

    install_crush_globally()?;

    println!("\n  ✓ CRUSH installed successfully!\n");
    println!("  Restart your terminal, then run:\n");
    println!("    crush              # Launch TUI");
    println!("    crush setup        # Check + auto-install everything\n");

    let deps_list = deps::check_all();
    let missing: Vec<_> = deps_list.iter().filter(|d| !d.available).collect();

    if !missing.is_empty() {
        println!("  Optional dependencies to install:");
        for dep in &missing {
            println!("    ✗ {} — {}", dep.name, dep.note);
        }
        println!("\n  Tip: run 'crush setup' and press Y/Enter to auto-install them.");
        println!();
    }

    Ok(())
}

pub fn run_check_deps() -> anyhow::Result<()> {
    run_setup()
}

/// Prompt the user Yes/No. Returns default when they just press Enter.
fn confirm(prompt: &str, default: bool) -> bool {
    let hint = if default { "[Y/n]" } else { "[y/N]" };
    print!("    → {} {}: ", prompt, hint);
    let _ = std::io::Write::flush(&mut std::io::stdout());
    let mut line = String::new();
    let _ = std::io::stdin().read_line(&mut line);
    let t = line.trim().to_lowercase();
    if t.is_empty() {
        return default;
    }
    matches!(t.as_str(), "y" | "yes")
}

/// Interactive setup doctor — like `flutter doctor` but with an option to
/// auto-fix each missing item (FFmpeg, model, PATH) by pressing Y / Enter.
pub fn run_setup() -> anyhow::Result<()> {
    println!();
    println!("  ╔═══════════════════════════════════════════════╗");
    println!("  ║    CRUSH {} — Setup & Auto-Fix              ║", crush_core::VERSION);
    println!("  ╚═══════════════════════════════════════════════╝");
    println!();
    println!("  Checking your setup... if something is missing we'll");
    println!("  ask and install it for you automatically.\n");

    // 1. CRUSH binary
    println!("  ● CRUSH binary");
    let exe = std::env::current_exe().unwrap_or_default();
    println!("  [✓] {} installed", crush_core::VERSION);
    println!("        └─ {}", exe.display());
    println!();

    // 2. FFmpeg
    println!("  ● FFmpeg (video/audio)");
    let ffmpeg_ok = {
        let d = deps::check_all().into_iter().find(|d| d.name == "FFmpeg");
        d.map(|d| d.available).unwrap_or(false)
    };
    if ffmpeg_ok {
        let d = deps::check_ffmpeg();
        println!("  [✓] {}", d.version.as_deref().unwrap_or("available"));
        if let Some(p) = &d.path {
            println!("        └─ {}", p);
        }
    } else {
        println!("  [✗] FFmpeg not found — video/audio features disabled");
        if confirm("Install FFmpeg now?", true) {
            match deps::install_ffmpeg() {
                Ok(_) => println!("  [✓] FFmpeg installed!"),
                Err(e) => println!("  [✗] {}", e),
            }
        } else {
            println!("      (skipped — video features will be disabled)");
        }
    }
    println!();

    // 3. Rust Native
    println!("  ● Rust Native (images)");
    let native_ok = deps::check_rust_native().available;
    if native_ok {
        println!("  [✓] webp, avif, png, jpg, bmp (built-in)");
    } else {
        println!("  [⚠] Not available");
    }
    println!();

    // 4. ONNX AI model
    println!("  ● ONNX AI Model (4x upscale)");
    let model_ok = deps::check_onnx_model().available;
    if model_ok {
        println!("  [✓] Real-ESRGAN x4plus");
        println!("        └─ {}", deps::onnx_model_path().display());
    } else {
        println!("  [✗] Model not found — AI upscale disabled (optional)");
        if confirm("Download Real-ESRGAN model (~65MB) now?", true) {
            match deps::download_model() {
                Ok(_) => {
                    if deps::check_onnx_model().available {
                        println!("  [✓] AI model ready!");
                    }
                }
                Err(e) => println!("  [✗] {}", e),
            }
        } else {
            println!("      (skipped — AI upscaling will be disabled)");
        }
    }
    println!();

    // 4b. ONNX Runtime (the dll that powers AI upscaling)
    println!("  ● ONNX Runtime (AI upscaling engine)");
    let rt_ok = deps::check_onnxruntime().available;
    if rt_ok {
        println!("  [✓] onnxruntime found");
        if let Some(p) = deps::resolve_onnxruntime_dll() {
            println!("        └─ {}", p.display());
        }
    } else {
        println!("  [✗] ONNX Runtime not found (needed for AI upscaling)");
        if confirm("Download and install ONNX Runtime (~45MB) now?", true) {
            match deps::install_onnxruntime() {
                Ok(_) => println!("  [✓] ONNX Runtime installed!"),
                Err(e) => println!("  [✗] {}", e),
            }
        } else {
            println!("      (skipped — AI upscaling will be disabled)");
        }
    }
    println!();

    // 5. PATH / global install
    println!("  ● Global install (run 'crush' anywhere)");
    if deps::is_installed_on_path() {
        println!("  [✓] crush is installed globally");
        let on_path = std::env::var("PATH").unwrap_or_default()
            .split(';')
            .any(|p| p.trim_end_matches('\\') == deps::crush_install_dir().display().to_string().trim_end_matches('\\'));
        if !on_path {
            println!("        └─ restart your terminal to use 'crush' from anywhere");
        }
    } else {
        println!("  [✗] crush is not installed globally");
        if confirm("Install crush globally now (adds to PATH)?", true) {
            match install_crush_globally() {
                Ok(_) => println!("  [✓] crush installed globally — restart your terminal"),
                Err(e) => println!("  [✗] {}", e),
            }
        } else {
            println!("      (skipped — run 'crush install' later)");
        }
    }
    println!();

    // 6. Terminal
    println!("  ● Terminal TUI support");
    #[cfg(windows)]
    println!("  [✓] Windows terminal detected");
    #[cfg(not(windows))]
    {
        let term = std::env::var("TERM").unwrap_or_default();
        let color = std::env::var("COLORTERM").unwrap_or_default();
        if !term.is_empty() || !color.is_empty() {
            println!("  [✓] Terminal detected");
        } else {
            println!("  [⚠] TERM not set");
        }
    }
    println!();

    println!("  ───────────────────────────────────────────────────────");
    let problems = {
        let mut n = 0;
        if !deps::check_ffmpeg().available { n += 1; }
        if !deps::check_onnx_model().available { n += 1; }
        if !deps::check_onnxruntime().available { n += 1; }
        if !deps::is_installed_on_path() { n += 1; }
        n
    };
    if problems == 0 {
        println!("  ✅ Everything looks good! You're ready to crush.");
    } else {
        println!("  ⚠ {} item(s) still need attention — see messages above.", problems);
    }
    println!("  ───────────────────────────────────────────────────────\n");

    Ok(())
}

/// Copies the current binary to ~/Crush, writes a launcher and updates PATH.
fn install_crush_globally() -> anyhow::Result<()> {
    let exe_path = std::env::current_exe()?;
    let install_dir = deps::crush_install_dir();

    if !install_dir.exists() {
        println!("      Creating {} ...", install_dir.display());
        std::fs::create_dir_all(&install_dir)?;
    }

    // Copy the actual binary (best-effort; wrapper also works).
    let dest_exe = install_dir.join("crush.exe");
    let _ = std::fs::copy(&exe_path, &dest_exe);

    // Launcher wrapper for the current binary.
    let bat_path = install_dir.join("crush.bat");
    std::fs::write(&bat_path, format!("@echo off\r\n\"{}\" %*\r\n", exe_path.display()))?;

    if cfg!(target_os = "windows") {
        println!("      Adding {} to PATH ...", install_dir.display());
        deps::add_to_path_windows(&install_dir)?;
    } else if cfg!(target_os = "macos") {
        let _ = std::process::Command::new("bash")
            .args(["-c", &format!("mkdir -p /usr/local/bin && cp '{}' /usr/local/bin/crush", exe_path.display())])
            .status();
    } else {
        let _ = std::process::Command::new("bash")
            .args(["-c", &format!("mkdir -p ~/.local/bin && cp '{}' ~/.local/bin/crush", exe_path.display())])
            .status();
    }

    Ok(())
}

pub fn run_uninstall() -> anyhow::Result<()> {
    deps::uninstall()
}

pub fn run_update() -> anyhow::Result<()> {
    println!("Checking for updates...");
    if let Err(e) = deps::self_update() {
        println!("  ✗ {}", e);
    }
    println!("\n  Now checking your setup and dependencies...\n");
    run_setup()
}

pub fn run_ai_check() -> anyhow::Result<()> {
    println!("\n  Checking AI upscaling engine...\n");

    let model = deps::check_onnx_model();
    println!("  ● Model     {}", if model.available { "✓" } else { "✗" });
    if let Some(p) = &model.path {
        println!("        └─ {}", p);
    }

    let runtime = deps::check_onnxruntime();
    println!("  ● Runtime   {}", if runtime.available { "✓" } else { "✗" });
    if let Some(p) = &runtime.path {
        println!("        └─ {}", p);
    }

    if !(model.available && runtime.available) {
        println!("\n  Run `crush setup` to install what's missing.\n");
        return Ok(());
    }

    let engine = crush_core::engine::ai_upscale::AiUpscaleEngine::new(&deps::crush_models_dir());
    match engine.test_ready() {
        Ok(_) => println!("\n  ✓ AI engine loads ONNX Runtime + model successfully!\n"),
        Err(e) => println!("\n  ✗ Failed to initialise AI engine: {}\n", e),
    }
    Ok(())
}

pub fn run_direct(args: Args) -> anyhow::Result<()> {
    let input = args.input.unwrap_or_else(|| PathBuf::from("."));
    let output_dir = args.output.unwrap_or_else(|| input.clone());
    let format = args.format.unwrap_or_default();
    let quality = if args.lossless { 0 } else { args.quality.clamp(1, 100) };
    let dry_run = args.dry_run;
    let verbose = args.verbose;

    // Validate format if provided
    if !format.is_empty() && !is_valid_format(&format) {
        anyhow::bail!(
            "Invalid format '{}'. Supported: webp, avif, png, jpg, bmp, mp4, webm, mkv, mov, avi, mp3, flac, ogg, wav, aac, opus, m4a, alac",
            format
        );
    }

    let (files, _) = fileutil::scan_directory(&input);
    let files = if let Some(ref filter) = args.r#type {
        fileutil::filter_by_type(&files, filter)
    } else {
        files
    };

    if files.is_empty() {
        println!("No files to process");
        return Ok(());
    }

    if dry_run {
        println!("\n  DRY RUN — {} file(s) would be processed:\n", files.len());
        for f in &files {
            let task_type = if format.is_empty() {
                match f.file_type {
                    fileutil::FileType::Image => TaskType::ImageCompress { quality, format: "webp".into() },
                    fileutil::FileType::Video => TaskType::VideoCompress { quality, format: "mp4".into() },
                    fileutil::FileType::Audio => TaskType::AudioConvert { format: "mp3".into(), quality },
                    _ => continue,
                }
            } else {
                match f.file_type {
                    fileutil::FileType::Image => TaskType::ImageConvert { target: format.clone(), quality },
                    fileutil::FileType::Video => {
                        if is_audio_format(&format) {
                            TaskType::AudioExtract { format: format.clone(), quality }
                        } else {
                            TaskType::VideoConvert { target: format.clone(), quality }
                        }
                    }
                    fileutil::FileType::Audio => TaskType::AudioConvert { format: format.clone(), quality },
                    _ => continue,
                }
            };
            let engine = engine::select_engine(&task_type, &f.ext);
            let engine_name = match engine {
                Engine::Ffmpeg => "FFmpeg",
                Engine::Native => "Native",
                Engine::OnnxAi => "ONNX",
            };
            println!("  {:<4} {:<10} → {:<6} [{}]", f.index, f.name, format, engine_name);
        }
        println!();
        return Ok(());
    }

    let ffmpeg_path = config::find_ffmpeg();
    let ffmpeg_engine = ffmpeg_path.map(|p| FfmpegEngine::new(&p.display().to_string()));
    let native_engine = NativeEngine::new();

    // Setup backup if enabled
    let backup_mgr = if !args.no_backup {
        let backup_path = args.backup_dir.unwrap_or_else(|| input.join("backup"));
        let mgr = BackupManager::new(&backup_path, &input);
        mgr.init().ok();
        Some(mgr)
    } else {
        None
    };

    println!("Processing {} file(s)...", files.len());

    let rt = tokio::runtime::Runtime::new()?;
    rt.block_on(async {
        let mut queue = TaskQueue::new();
        queue.start().await;

        for file in &files {
            let task_type = if format.is_empty() {
                match file.file_type {
                    fileutil::FileType::Image => TaskType::ImageCompress { quality, format: "webp".into() },
                    fileutil::FileType::Video => TaskType::VideoCompress { quality, format: "mp4".into() },
                    fileutil::FileType::Audio => TaskType::AudioConvert { format: "mp3".into(), quality },
                    _ => continue,
                }
            } else {
                match file.file_type {
                    fileutil::FileType::Image => TaskType::ImageConvert { target: format.clone(), quality },
                    fileutil::FileType::Video => {
                        if is_audio_format(&format) {
                            TaskType::AudioExtract { format: format.clone(), quality }
                        } else {
                            TaskType::VideoConvert { target: format.clone(), quality }
                        }
                    }
                    fileutil::FileType::Audio => TaskType::AudioConvert { format: format.clone(), quality },
                    _ => continue,
                }
            };

            let engine_type = engine::select_engine(&task_type, &file.ext);
            let output_name = format!("{}.{}", fileutil::file_name_without_ext(&file.name), &format);
            let output_path = output_dir.join(&output_name);

            queue.add_task(
                file.path.clone(),
                output_path.display().to_string(),
                file.name.clone(),
                file.size,
                file.size_str.clone(),
                task_type,
                engine_type,
            ).await;
        }

        let state = queue.get_state().await;
        let total = state.total;
        println!("Queued {} tasks\n", total);

        let mut ok = 0usize;
        let mut fail = 0usize;
        let mut skip = 0usize;
        let start = std::time::Instant::now();

        for task in &state.tasks {
            let mut task = task.clone();

            // Only skip if input and output are the same file (prevent self-overwrite)
            let input_path = std::path::Path::new(&task.input_path);
            let output_path = std::path::Path::new(&task.output_path);
            let same_file = input_path.canonicalize().ok() == output_path.canonicalize().ok();
            if same_file {
                queue.update_status(task.id, TaskStatus::Skipped).await;
                skip += 1;
                if verbose {
                    println!("  ⏭  {} (same file — would overwrite)", task.file_name);
                }
                continue;
            }

            // Backup original file if enabled
            if let Some(ref mgr) = backup_mgr {
                let _ = mgr.backup_file(std::path::Path::new(&task.input_path));
            }

            let result = match &task.engine {
                Engine::Ffmpeg => {
                    if let Some(ref eng) = ffmpeg_engine {
                        eng.process(&mut task).await
                    } else {
                        Err(anyhow::anyhow!("FFmpeg not found"))
                    }
                }
                Engine::Native => native_engine.process(&mut task).await,
                Engine::OnnxAi => {
                    Err(anyhow::anyhow!("AI engine not available in CLI mode"))
                }
            };

            match result {
                Ok(()) => {
                    ok += 1;
                    println!("  ✓  {}", task.file_name);
                }
                Err(e) => {
                    fail += 1;
                    println!("  ✗  {} — {}", task.file_name, e);
                }
            }
        }

        let elapsed = start.elapsed().as_secs_f64();
        println!("\n  ─────────────────────────────────────");
        println!("  ✓ OK: {}  ✗ FAIL: {}  ⏭ SKIP: {}  Total: {}", ok, fail, skip, total);
        println!("  ⏱  {:.1}s", elapsed);
        println!("  ─────────────────────────────────────\n");

        print!("\x07");

        Ok(())
    })
}

fn is_audio_format(format: &str) -> bool {
    matches!(format.to_lowercase().as_str(),
        "mp3" | "flac" | "ogg" | "wav" | "aac" | "opus" | "m4a" | "alac"
    )
}

fn is_valid_format(format: &str) -> bool {
    let f = format.to_lowercase();
    matches!(f.as_str(),
        // Image formats
        "webp" | "avif" | "png" | "jpg" | "jpeg" | "bmp" |
        // Video formats
        "mp4" | "webm" | "mkv" | "mov" | "avi" |
        // Audio formats
        "mp3" | "flac" | "ogg" | "wav" | "aac" | "opus" | "m4a" | "alac"
    )
}
