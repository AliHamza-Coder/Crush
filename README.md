# CRUSH v3.0.0 — Multimedia Mission Control

A blazing-fast media compressor and converter written in Rust, with a professional terminal UI. Compress images, videos, and audio with a single command.

## Features

- **3 Processing Engines**: FFmpeg (video/audio), Native Rust (image), ONNX AI (4x upscale)
- **Terminal TUI**: Professional Ratatui interface with mouse support
- **Quality Presets**: Recommended quality levels with descriptions
- **Backup System**: Automatic backup before processing with timestamped folders
- **Format Support**: 20+ formats including webp, avif, mp4, webm, mp3, flac, ogg, wav, and more
- **Batch Processing**: Process entire directories with parallel workers
- **Smart Skip**: Automatically skips files already in target format

## Quick Start

```bash
# Build
cargo build --release

# Run TUI
cargo run --release

# Run CLI
cargo run --release -- analyse .
cargo run --release -- -i ./images -f webp -q 85
```

## Installation

### Windows (Recommended)

```bash
# Install via cargo
cargo install --path crates/crush-cli

# Or copy the binary to a folder in your PATH
copy target\release\crush.exe C:\Users\YourName\.cargo\bin\
```

### Linux / macOS

```bash
# Install via cargo
cargo install --path crates/crush-cli

# Or copy to /usr/local/bin
cp target/release/crush /usr/local/bin/
```

## Dependencies

### Required
- **Rust 1.75+** — Build toolchain

### Optional
- **FFmpeg** — Video/audio processing (auto-detected via `where`/`which`)
  - Install: `winget install -e --id Gyan.FFmpeg` (Windows)
  - Install: `brew install ffmpeg` (macOS)
  - Install: `sudo apt install ffmpeg` (Linux)
- **ONNX Model** — AI upscaling (optional)
  - Download: `realesrgan-x4plus.onnx` to `./models/` directory
  - Or let CRUSH download it automatically on first use

## CLI Commands

### Analyse Directory
```bash
crush analyse .                    # Analyse current directory
crush analyse ./images --json      # JSON output for scripting
```

### Direct Mode (No TUI)
```bash
# Compress images to WebP
crush -i ./images -f webp -q 85

# Convert videos to MP4
crush -i ./videos -f mp4 -q 90

# Extract audio from video
crush -i ./videos -f mp3 --type video

# Dry run (preview without processing)
crush -i ./images -f webp --dry-run

# Custom quality
crush -i ./images -f webp -q 95

# Lossless mode
crush -i ./images --lossless

# Verbose output
crush -i ./images -f webp -v

# Skip backup
crush -i ./images -f webp --no-backup

# Custom backup directory
crush -i ./images -f webp --backup-dir ./my-backups
```

### Management
```bash
crush install                      # Install crush globally + add to PATH
crush setup                        # Doctor + auto-fix (install FFmpeg/model, fix PATH)
crush check-deps                   # Same as setup
crush uninstall                    # Remove crush + all data (models, backups, PATH)
crush update                       # Check GitHub for updates + auto-fix setup
crush -v / --version / version     # Print version
```

### Setup & Auto-Fix (like `flutter doctor`)

`crush setup` checks everything. If something is missing it asks
`[Y/n]` and installs it automatically on `Y` or `Enter`:

| Check | If missing |
|-------|-----------|
| FFmpeg (video/audio) | Installs via winget / brew / apt |
| Rust Native (images) | Always built-in |
| ONNX AI model (4x upscale) | Downloads Real-ESRGAN (~64MB) to `%LOCALAPPDATA%\crush\models\` |
| Global install (PATH) | Installs to `~/Crush` + adds to PATH |

Data locations:
- **Install dir**: `~/Crush` (binary + launcher)
- **Data dir**: `%LOCALAPPDATA%\crush` (models, backups, settings)

`crush uninstall` removes everything — install dir, data dir, models,
backups, PATH entries — and deletes the running binary on exit.


## TUI Controls

### Navigation
| Key | Panel | Action |
|-----|-------|--------|
| `↑↓` / `jk` | All | Navigate items |
| `Enter` | Menu | Execute action |
| `Space` | Files | Toggle selection |
| `Tab` | All | Switch panel / back |
| `L` | Quality | Toggle lossless |
| `0-9` | Quality | Custom quality input |
| `c` | Queue | Cancel processing |
| `x` | Queue | Clear queue |
| `q` / `Esc` | All | Quit / back |
| Mouse click | All | Select item |
| Scroll wheel | All | Navigate up/down |

### Panels
- **Menu**: Choose action (Compress/Convert/Extract)
- **Files**: Browse and select files
- **Quality**: Select quality preset (100/90/85/75/60/Lossless/Custom)
- **Format**: Choose target format
- **Queue**: Live processing progress with status

### Quality Presets

#### Images
| Quality | Label | Description |
|---------|-------|-------------|
| 100% | Maximum | Best quality, largest file |
| 90% | High | Slightly larger |
| 85% | Balanced ★ | Good quality, ~50-70% smaller |
| 75% | Smaller | Slightly lower quality |
| 60% | Compact | Good for web sharing |
| Lossless | Original | Original quality preserved |

#### Videos
| Quality | Label | Description |
|---------|-------|-------------|
| 100% | Maximum | CRF 18 — near-lossless, largest |
| 90% | High | CRF 20 — high quality |
| 85% | Balanced ★ | CRF 23 — good quality, ~50% smaller |
| 75% | Smaller | CRF 28 — smaller, some quality loss |
| 60% | Compact | CRF 32 — very small, lower quality |

#### Audio
| Quality | Label | Description |
|---------|-------|-------------|
| 100% | Maximum | VBR ~320kbps — best quality |
| 90% | High | VBR ~256kbps — high quality |
| 85% | Balanced ★ | VBR ~192kbps — excellent, recommended |
| 75% | Smaller | VBR ~160kbps — smaller file |
| 60% | Compact | VBR ~128kbps — good for podcasts |

## Format Support

### Images
| Format | Compress | Convert | Engine |
|--------|----------|---------|--------|
| WebP | ✓ | ✓ | Native / FFmpeg |
| AVIF | ✓ | ✓ | Native / FFmpeg |
| PNG | ✓ | ✓ | Native / FFmpeg |
| JPG/JPEG | ✓ | ✓ | Native / FFmpeg |
| BMP | ✓ | ✓ | Native / FFmpeg |
| GIF | — | ✓ | FFmpeg |

### Videos
| Format | Compress | Convert | Engine |
|--------|----------|---------|--------|
| MP4 | ✓ | ✓ | FFmpeg |
| WebM | ✓ | ✓ | FFmpeg |
| MKV | ✓ | ✓ | FFmpeg |
| MOV | ✓ | ✓ | FFmpeg |
| AVI | ✓ | ✓ | FFmpeg |

### Audio
| Format | Compress | Convert | Extract | Engine |
|--------|----------|---------|---------|--------|
| MP3 | ✓ | ✓ | ✓ | FFmpeg |
| FLAC | ✓ | ✓ | ✓ | FFmpeg |
| OGG | ✓ | ✓ | ✓ | FFmpeg |
| WAV | ✓ | ✓ | ✓ | FFmpeg |
| AAC | ✓ | ✓ | ✓ | FFmpeg |
| OPUS | ✓ | ✓ | ✓ | FFmpeg |
| M4A | ✓ | ✓ | ✓ | FFmpeg |
| ALAC | ✓ | ✓ | ✓ | FFmpeg |

## Processing Engines

### FFmpeg Engine
- **Used for**: Video/audio processing, all format conversions
- **Requires**: FFmpeg installed and in PATH
- **Auto-detected**: Yes, via `where`/`which`

### Native Rust Engine
- **Used for**: Image compression (WebP, AVIF, PNG, JPG, BMP)
- **Requires**: Nothing extra — always available
- **Benefits**: No external dependencies, fast, memory-safe

### ONNX AI Engine
- **Used for**: AI-powered 4x upscaling (Real-ESRGAN)
- **Requires**: ONNX Runtime + `realesrgan-x4plus.onnx` model
- **Model**: Auto-downloaded on first use (~65MB)

## Project Structure

```
F:\Crush\
├── Cargo.toml                    # Workspace root
├── ARCHITECTURE_V3.md            # Architecture docs
├── AGENTS.md                     # Development guide
├── README.md                     # This file
└── crates/
    ├── crush-core/               # Core library
    │   └── src/
    │       ├── lib.rs            # VERSION constant
    │       ├── core/
    │       │   ├── mod.rs
    │       │   ├── error.rs      # CrushError enum
    │       │   ├── config.rs     # Config, find_ffmpeg()
    │       │   ├── fileutil.rs   # FileType, FileInfo, scan, filter
    │       │   ├── queue.rs      # TaskQueue, Task, TaskType
    │       │   ├── backup.rs     # BackupManager
    │       │   └── deps.rs       # Dependency checker, install, update
    │       └── engine/
    │           ├── mod.rs        # select_engine()
    │           ├── ffmpeg.rs     # FFmpeg subprocess engine
    │           ├── native.rs     # Pure Rust image encoding
    │           └── ai_upscale.rs # ONNX Real-ESRGAN
    └── crush-cli/                # CLI + TUI binary
        └── src/
            ├── main.rs           # Entry point
            ├── cli.rs            # Clap CLI commands
            └── tui/
                ├── mod.rs        # Event loop + mouse handling
                ├── app.rs        # App state + business logic
                └── ui.rs         # Ratatui rendering
```

## Testing

### Test Files
- `F:\Crush\Images test\` — 3 PNG files (2.4MB, 2.2MB, 2.2MB)
- `F:\Crush\video test\` — 1 MP4 file (29MB)

### Test Results
| Test | Input | Output | Size Reduction |
|------|-------|--------|----------------|
| PNG→WebP (q85) | 1.png (2.4MB) | 1.webp (200KB) | 92% |
| PNG→JPG (q85) | 1.png (2.4MB) | 1.jpg (185KB) | 92% |
| Video→MP3 | video.mp4 (29MB) | audio.mp3 (386KB) | 99% |
| Dry run | All | Shows what would happen | — |
| Check deps | — | FFmpeg ✓, ONNX ✗, Native ✓ | — |

## License

MIT

## Credits

- [Ratatui](https://ratatui.rs/) — Terminal UI framework
- [Crossterm](https://crossterm.rs/) — Terminal manipulation
- [image-rs](https://github.com/image-rs/image) — Image processing
- [webp](https://github.com/nicholasgasior/webp) — WebP encoding
- [ort](https://github.com/pyke/ort) — ONNX Runtime bindings
- [Real-ESRGAN](https://github.com/xinntao/Real-ESRGAN) — AI upscaling model
