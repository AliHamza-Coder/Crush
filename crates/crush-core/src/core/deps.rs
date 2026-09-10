use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::Command;

#[derive(Debug, Clone)]
pub struct DepStatus {
    pub name: String,
    pub available: bool,
    pub path: Option<String>,
    pub version: Option<String>,
    pub note: String,
}

// ---------------------------------------------------------------------------
// Standard directories
// ---------------------------------------------------------------------------

/// Where CRUSH is installed so `crush` is available on PATH (home/Crush).
pub fn crush_install_dir() -> PathBuf {
    dirs::home_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join("Crush")
}

/// Where CRUSH keeps its data (models, backups). Prefers Local AppData on
/// Windows since the AI model is large (~64MB).
pub fn crush_data_dir() -> PathBuf {
    dirs::data_local_dir()
        .or_else(dirs::data_dir)
        .unwrap_or_else(|| crush_install_dir())
        .join("crush")
}

/// Directory holding ONNX AI models.
pub fn crush_models_dir() -> PathBuf {
    crush_data_dir().join("models")
}

/// Full path to the Real-ESRGAN ONNX model.
pub fn onnx_model_path() -> PathBuf {
    crush_models_dir().join("realesrgan-x4plus.onnx")
}

/// True when CRUSH has been installed globally (binary present in the install
/// dir). Once installed the launcher is on the user PATH registry; a freshly
/// opened terminal picks it up.
pub fn is_installed_on_path() -> bool {
    let install_dir = crush_install_dir();
    let has_files = install_dir.join("crush.exe").exists() || install_dir.join("crush.bat").exists();
    if !has_files {
        return false;
    }
    // Already installed — the PATH entry is set at install time. Whether the
    // *current* shell sees it only matters until the terminal is restarted.
    true
}

// ---------------------------------------------------------------------------
// Dependency checks
// ---------------------------------------------------------------------------

pub fn check_all() -> Vec<DepStatus> {
    let mut deps = Vec::new();
    deps.push(check_ffmpeg());
    deps.push(check_onnx_model());
    deps.push(check_onnxruntime());
    deps.push(check_rust_native());
    deps
}

pub fn check_ffmpeg() -> DepStatus {
    match super::config::find_ffmpeg() {
        Some(path) => {
            let version = Command::new(&path)
                .arg("-version")
                .output()
                .ok()
                .and_then(|o| String::from_utf8(o.stdout).ok())
                .and_then(|s| s.lines().next().map(|l| l.trim().to_string()));
            DepStatus {
                name: "FFmpeg".into(),
                available: true,
                path: Some(path.display().to_string()),
                version,
                note: "Video/audio processing".into(),
            }
        }
        None => DepStatus {
            name: "FFmpeg".into(),
            available: false,
            path: None,
            version: None,
            note: "NOT FOUND — video features disabled".into(),
        },
    }
}

pub fn check_onnx_model() -> DepStatus {
    let model_path = onnx_model_path();

    if model_path.exists() {
        DepStatus {
            name: "ONNX Model".into(),
            available: true,
            path: Some(model_path.display().to_string()),
            version: Some("Real-ESRGAN x4plus".into()),
            note: "AI upscaling (4x)".into(),
        }
    } else {
        DepStatus {
            name: "ONNX Model".into(),
            available: false,
            path: Some(model_path.display().to_string()),
            version: None,
            note: "NOT FOUND — AI upscale disabled".into(),
        }
    }
}

pub fn check_rust_native() -> DepStatus {
    DepStatus {
        name: "Rust Native".into(),
        available: true,
        path: None,
        version: Some("webp, avif, png, jpg, bmp".into()),
        note: "Image processing (no FFmpeg needed)".into(),
    }
}

// ---------------------------------------------------------------------------
// ONNX Runtime (the .dll used by the AI engine)
// ---------------------------------------------------------------------------

/// Path to the bundled ONNX Runtime dynamic library in the crush data dir.
pub fn onnxruntime_path() -> PathBuf {
    crush_data_dir().join("onnxruntime.dll")
}

/// Best-effort locate a usable ONNX Runtime dll. Prefers the bundled copy in
/// the crush data dir, then one sitting next to the current executable.
pub fn resolve_onnxruntime_dll() -> Option<PathBuf> {
    let candidates = [
        onnxruntime_path(),
        std::env::current_exe()
            .ok()
            .and_then(|e| e.parent().map(|p| p.join("onnxruntime.dll")))
            .unwrap_or_default(),
    ];
    candidates.into_iter().find(|p| p.exists())
}

/// Sets `ORT_DYLIB_PATH` to the bundled ONNX Runtime dll so `ort` loads a
/// known-good binary instead of whatever random `onnxruntime.dll` happens to
/// be in the current directory. Safe to call repeatedly; no-op if not found.
pub fn prepare_ort() {
    if let Some(dll) = resolve_onnxruntime_dll() {
        if std::env::var("ORT_DYLIB_PATH").map(|v| v.trim().is_empty()).unwrap_or(true) {
            unsafe {
                std::env::set_var("ORT_DYLIB_PATH", dll.display().to_string());
            }
        }
    }
}

pub fn check_onnxruntime() -> DepStatus {
    if let Some(path) = resolve_onnxruntime_dll() {
        DepStatus {
            name: "ONNX Runtime".into(),
            available: true,
            path: Some(path.display().to_string()),
            version: Some("dynamic library".into()),
            note: "AI upscaling runtime".into(),
        }
    } else {
        DepStatus {
            name: "ONNX Runtime".into(),
            available: false,
            path: Some(onnxruntime_path().display().to_string()),
            version: None,
            note: "NOT FOUND — AI upscale disabled".into(),
        }
    }
}

/// Downloads and extracts `onnxruntime.dll` into the crush data dir so the AI
/// engine has a reliable runtime. Works on Windows x64.
pub fn install_onnxruntime() -> anyhow::Result<()> {
    let target = onnxruntime_path();
    if target.exists() {
        println!("  ✓ ONNX Runtime already installed: {}", target.display());
        return Ok(());
    }

    let arch = std::env::consts::ARCH;
    if cfg!(not(target_os = "windows")) {
        anyhow::bail!("ONNX Runtime auto-install currently supports Windows. Install onnxruntime manually on your platform.");
    }
    if arch != "x86_64" {
        anyhow::bail!("ONNX Runtime auto-install supports x86_64 only (found {arch}).");
    }

    // ort with api-17 needs onnxruntime >= 1.17.x; 1.17.3 matches exactly.
    let url = "https://github.com/microsoft/onnxruntime/releases/download/v1.17.3/onnxruntime-win-x64-1.17.3.zip";
    let zip_path = std::env::temp_dir().join("onnxruntime-win-x64-1.17.3.zip");

    println!("\n  Downloading ONNX Runtime 1.17.3 ...");
    println!("  From: {}", url);

    // Try curl first (ships with Windows 10+, macOS and most Linux distros) —
    // it handles redirects and large downloads more reliably than reqwest here.
    let mut downloaded = false;
    if let Ok(mut child) = Command::new("curl")
        .args(["-L", "--fail", "--silent", "--show-error", "-o"])
        .arg(&zip_path)
        .arg(url)
        .spawn()
    {
        let _ = child.wait();
        downloaded = zip_path.exists() && zip_path.metadata().map(|m| m.len() > 1_000_000).unwrap_or(false);
    }

    if !downloaded {
        let client = reqwest::blocking::Client::new();
        let resp = client.get(url).header("User-Agent", "crush-updater").send()?;
        if !resp.status().is_success() {
            return Err(anyhow::anyhow!("Download failed: HTTP {}", resp.status()));
        }
        let bytes = resp.bytes()?;
        std::fs::write(&zip_path, &bytes)?;
    }

    let size_mb = zip_path.metadata().map(|m| m.len() as f64 / (1024.0 * 1024.0)).unwrap_or(0.0);
    println!("  Downloaded {:.1} MB, extracting...", size_mb);

    if let Some(parent) = target.parent() {
        std::fs::create_dir_all(parent)?;
    }

    let file = std::fs::File::open(&zip_path)?;
    let mut archive = zip::ZipArchive::new(file)?;
    let mut extracted = false;
    for i in 0..archive.len() {
        let mut entry = archive.by_index(i)?;
        let name = entry.name().to_string();
        if !entry.is_dir() && name.ends_with("onnxruntime.dll") {
            let out = std::fs::File::create(&target)?;
            let mut out = std::io::BufWriter::new(out);
            std::io::copy(&mut entry, &mut out)?;
            out.flush()?;
            extracted = true;
            println!("  ✓ ONNX Runtime saved to: {}", target.display());
            break;
        }
    }

    let _ = std::fs::remove_file(&zip_path);

    if !extracted {
        anyhow::bail!("Could not find onnxruntime.dll inside the downloaded archive");
    }

    prepare_ort();
    Ok(())
}

// ---------------------------------------------------------------------------
// Installers
// ---------------------------------------------------------------------------

pub fn install_ffmpeg() -> anyhow::Result<()> {
    if cfg!(target_os = "windows") {
        println!("\n  Installing FFmpeg via winget...");
        let status = Command::new("winget")
            .args(["install", "-e", "--id", "Gyan.FFmpeg", "--accept-package-agreements", "--accept-source-agreements"])
            .status();
        match status {
            Ok(s) if s.success() => {
                println!("  ✓ FFmpeg installed successfully!");
                println!("  Restart your terminal to use it.");
                Ok(())
            }
            Ok(_) => Err(anyhow::anyhow!("winget install failed. Try manually:\n  winget install -e --id Gyan.FFmpeg")),
            Err(e) => Err(anyhow::anyhow!("winget not available: {}. Install manually:\n  winget install -e --id Gyan.FFmpeg", e)),
        }
    } else if cfg!(target_os = "macos") {
        println!("\n  Installing FFmpeg via brew...");
        let status = Command::new("brew").args(["install", "ffmpeg"]).status()?;
        if status.success() {
            println!("  ✓ FFmpeg installed successfully!");
            Ok(())
        } else {
            Err(anyhow::anyhow!("brew install ffmpeg failed"))
        }
    } else {
        println!("\n  Installing FFmpeg via apt...");
        let status = Command::new("sudo").args(["apt", "install", "-y", "ffmpeg"]).status()?;
        if status.success() {
            println!("  ✓ FFmpeg installed successfully!");
            Ok(())
        } else {
            Err(anyhow::anyhow!("apt install ffmpeg failed"))
        }
    }
}

pub fn download_model() -> anyhow::Result<()> {
    let model_path = onnx_model_path();

    if model_path.exists() {
        println!("  ✓ Model already exists: {}", model_path.display());
        return Ok(());
    }

    println!("\n  Downloading Real-ESRGAN x4plus model...");
    println!("  This is a one-time download (~65MB) for AI upscaling.");

    if let Some(parent) = model_path.parent() {
        std::fs::create_dir_all(parent)?;
    }

    let url = "https://media.axelera.ai/artifacts/model_cards/weights/image_enhancement/superresolution/RealESRGAN_x4plus.onnx";
    let client = reqwest::blocking::Client::new();
    println!("  From: {}", url);
    let resp = client.get(url)
        .header("User-Agent", "crush-updater")
        .send()?;

    if !resp.status().is_success() {
        return Err(anyhow::anyhow!("Download failed: HTTP {}", resp.status()));
    }

    let total = resp.content_length().unwrap_or(0);
    let mut bytes = Vec::new();
    let mut stream = resp;
    use std::io::Read;
    stream.read_to_end(&mut bytes)?;
    std::fs::write(&model_path, &bytes)?;

    let mb = bytes.len() as f64 / (1024.0 * 1024.0);
    println!("  ✓ Model saved ({} MB): {}", format!("{:.1}", mb), model_path.display());
    if total > 0 {
        println!("  (expected ~{:.1} MB)", total as f64 / (1024.0 * 1024.0));
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// PATH helpers (Windows)
// ---------------------------------------------------------------------------

pub fn add_to_path_windows(new_path: &Path) -> anyhow::Result<()> {
    let script = format!(
        "$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User'); \
         if ($currentPath -notlike '*{}*') {{ \
             [Environment]::SetEnvironmentVariable('Path', \"$currentPath;{}\", 'User') \
         }}",
        new_path.display().to_string().replace('\\', "\\\\"),
        new_path.display().to_string().replace('\\', "\\\\")
    );
    let status = Command::new("powershell")
        .args(["-Command", &script])
        .status()?;
    if status.success() { Ok(()) } else { Err(anyhow::anyhow!("Failed to add to PATH")) }
}

pub fn remove_from_path_windows(remove_dir: &Path) -> anyhow::Result<()> {
    let script = format!(
        "$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User'); \
         $newPath = ($currentPath -split ';' | Where-Object {{ $_ -ne '{}' }}) -join ';'; \
         [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')",
        remove_dir.display().to_string().replace('\\', "\\\\")
    );
    let status = Command::new("powershell")
        .args(["-Command", &script])
        .status()?;
    if status.success() { Ok(()) } else { Err(anyhow::anyhow!("Failed to remove from PATH")) }
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

pub fn self_update() -> anyhow::Result<()> {
    println!("\n  ╔══════════════════════════════════════╗");
    println!("  ║       CRUSH — Self Update            ║");
    println!("  ╚══════════════════════════════════════╝\n");

    let current_version = super::super::VERSION.trim_start_matches('v');
    println!("  Current version: v{}", current_version);
    println!("  Checking for updates...\n");

    let client = reqwest::blocking::Client::new();
    let resp = client.get("https://api.github.com/repos/AliHamza-Coder/crush/releases/latest")
        .header("User-Agent", "crush-updater")
        .send();

    let resp = match resp {
        Ok(r) => r,
        Err(e) => {
            println!("  ✗ Failed to check for updates: {}", e);
            println!("\n  To update manually:");
            println!("    cargo install --git https://github.com/AliHamza-Coder/crush");
            return Ok(());
        }
    };

    if !resp.status().is_success() {
        println!("  ✗ No releases found on GitHub");
        println!("\n  To update manually:");
        println!("    cd F:\\Crush && cargo build --release");
        println!("    .\\target\\release\\crush.exe install");
        return Ok(());
    }

    let json: serde_json::Value = resp.json()?;
    let tag = json["tag_name"].as_str().unwrap_or("unknown");
    let latest_version = tag.trim_start_matches('v');

    println!("  Latest version:  v{}", latest_version);

    if latest_version <= current_version {
        println!("\n  ✓ You are up to date!\n");
        return Ok(());
    }

    println!("\n  New version available: v{} → v{}", current_version, latest_version);

    let download_url = json["assets"].as_array()
        .and_then(|assets| assets.iter().find(|a| {
            let name = a["name"].as_str().unwrap_or("");
            cfg!(target_os = "windows") && name.ends_with(".exe")
                || cfg!(target_os = "linux") && name.contains("linux")
                || cfg!(target_os = "macos") && name.contains("darwin")
        }))
        .and_then(|a| a["browser_download_url"].as_str());

    let download_url = match download_url {
        Some(url) => url,
        None => {
            println!("  ✗ No compatible binary found for your platform");
            println!("\n  Download manually from:");
            println!("    https://github.com/AliHamza-Coder/crush/releases");
            return Ok(());
        }
    };

    println!("  Downloading from: {}", download_url);

    let exe_path = std::env::current_exe()?;
    let tmp_path = exe_path.with_extension("tmp");

    let resp_bytes = client.get(download_url)
        .header("User-Agent", "crush-updater")
        .send()?
        .bytes()?;

    std::fs::write(&tmp_path, &resp_bytes)?;

    let install_dir = crush_install_dir();
    if install_dir.exists() {
        let bat_path = install_dir.join("crush.bat");
        let _ = std::fs::remove_file(&bat_path);
        let new_bat = install_dir.join("crush.bat");
        std::fs::write(&new_bat, format!(
            "@echo off\r\n\"{}\" %*\r\n",
            exe_path.display()
        ))?;
    }

    println!("\n  ✓ Downloaded successfully!");
    println!("\n  To complete the update:");
    println!("    1. Close all crush processes");
    println!("    2. Run: crush install\n");

    Ok(())
}

// ---------------------------------------------------------------------------
// Uninstall
// ---------------------------------------------------------------------------

pub fn uninstall() -> anyhow::Result<()> {
    println!("\n  ╔══════════════════════════════════════╗");
    println!("  ║     CRUSH — Uninstall Everything     ║");
    println!("  ╚══════════════════════════════════════╝\n");

    println!("  This will remove:");
    println!("    • crush from your PATH");
    println!("    • {}", crush_install_dir().display());
    println!("    • {}", crush_data_dir().display());
    println!("    • {}", crush_models_dir().display());
    println!("    • All downloaded models, backups and settings\n");

    print!("  Are you sure you want to remove everything? [y/N]: ");
    let _ = std::io::Write::flush(&mut std::io::stdout());
    let mut answer = String::new();
    std::io::stdin().read_line(&mut answer)?;
    let answer = answer.trim().to_lowercase();
    if !matches!(answer.as_str(), "y" | "yes") {
        println!("\n  Uninstall cancelled.\n");
        return Ok(());
    }

    let exe_path = std::env::current_exe()?;
    let exe_dir = exe_path.parent().unwrap_or(Path::new("."));

    if cfg!(target_os = "windows") {
        let install_dir = crush_install_dir();
        let data_dir = crush_data_dir();
        let cargo_bin = dirs::home_dir()
            .unwrap_or_else(|| PathBuf::from("."))
            .join(".cargo")
            .join("bin");

        for dir in [&install_dir, &data_dir, &cargo_bin, &exe_dir.to_path_buf()] {
            let _ = remove_from_path_windows(dir);
        }
    }

    for target in [
        crush_models_dir(),
        crush_data_dir(),
        crush_install_dir(),
    ] {
        if target.exists() {
            println!("  Deleting {} ...", target.display());
            if std::fs::remove_dir_all(&target).is_err() {
                println!("  ⚠ Could not fully remove {}", target.display());
            }
        }
    }

    let cargo_crush = dirs::home_dir()
        .unwrap_or_else(|| PathBuf::from("."))
        .join(".cargo")
        .join("bin")
        .join(if cfg!(target_os = "windows") { "crush.exe" } else { "crush" });
    if cargo_crush.exists() {
        println!("  Deleting {} ...", cargo_crush.display());
        let _ = std::fs::remove_file(&cargo_crush);
    }

    // Schedule deletion of the running binary after this process exits.
    if cfg!(target_os = "windows") {
        let script = format!(
            "Start-Sleep -Milliseconds 1500; Remove-Item -LiteralPath '{}' -Force -ErrorAction SilentlyContinue",
            exe_path.display().to_string().replace('\'', "''")
        );
        let _ = Command::new("powershell")
            .args(["-WindowStyle", "Hidden", "-Command", &script])
            .spawn();
    }

    println!("\n  ✓ CRUSH uninstalled.\n");
    if cfg!(target_os = "windows") {
        println!("  The running crush.exe will delete itself on exit.");
    }
    println!("  Close this terminal and open a new one to refresh PATH.\n");

    Ok(())
}
