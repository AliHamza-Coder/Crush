# CRUSH v3.0.0 — Development Guide

## Build & Run

```bash
# Build
cargo build

# Run TUI
cargo run

# Run CLI
cargo run -- analyse .
cargo run -- -i ./images -f webp -q 85
cargo run -- check-deps
cargo run -- install
cargo run -- version
```

## Project Structure

```
F:\Crush\
├── Cargo.toml                    # Workspace root
├── ARCHITECTURE_V3.md            # Architecture docs
├── AGENTS.md                     # This file
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
    │       │   └── deps.rs       # Dependency checker
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

## TUI Controls

| Key | Panel | Action |
|-----|-------|--------|
| `↑↓` / `jk` | All | Navigate |
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

## CLI Commands

```bash
crush                              # Launch TUI
crush analyse [dir] [--json]       # Analyze directory
crush install                      # Check dependencies
crush check-deps                   # Same as install
crush uninstall                    # Remove from PATH
crush update                       # Self-update from GitHub
crush version                      # Print version

# Direct mode
crush -i <dir> -f <format> [-q <quality>] [--type image|video|audio]
crush -i <dir> -f webp --dry-run   # Preview without processing
crush -i <dir> -f mp3 -v           # Verbose output
crush -i <dir> --lossless          # Lossless mode
crush -i <dir> --no-backup         # Skip backup
crush -i <dir> --backup-dir <dir>  # Custom backup directory
```

## Testing

Test files are in:
- `F:\Crush\Images test\` — 3 PNG files (2.4MB, 2.2MB, 2.2MB)
- `F:\Crush\video test\` — 1 MP4 file (29MB)

### Test Results (verified)

| Test | Input | Output | Size Reduction |
|------|-------|--------|----------------|
| PNG→WebP (q85) | 1.png (2.4MB) | 1.webp (200KB) | 92% |
| PNG→JPG (q85) | 1.png (2.4MB) | 1.jpg (185KB) | 92% |
| Video skip | video.mp4 | (skipped, same format) | — |
| Dry run | All | Shows what would happen | — |
| Check deps | — | FFmpeg ✓, ONNX ✗, Native ✓ | — |

## Dependencies

- **FFmpeg** — Video/audio processing (auto-detected via `where`/`which`)
- **ONNX Model** — AI upscaling (optional, place `realesrgan-x4plus.onnx` in `./models/`)
- **Rust Native** — Image encoding (webp, avif, png, jpg, bmp — always available)

## Key Patterns

- All engines implement `Clone` for use in `tokio::spawn`
- `TaskQueue` uses `Arc<RwLock<Vec<Task>>>` for thread-safe state
- `AiUpscaleEngine` uses `Arc<Mutex<Session>>` for ONNX inference
- Quality overflow is handled with `u16` intermediate calculations
- Backup runs before each file is processed
- Files already in target format are automatically skipped
