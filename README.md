# CRUSH v3.0.0

Blazing-fast media compressor and converter written in Rust.

## Install

### Windows
```powershell
irm https://raw.githubusercontent.com/AliHamza-Coder/Crush/main/scripts/install.ps1 | iex
```

### Linux / macOS
```bash
git clone https://github.com/AliHamza-Coder/Crush.git && cd Crush && cargo install --path crates/crush-cli
```

### From Source (Any OS)
```bash
git clone https://github.com/AliHamza-Coder/Crush.git
cd Crush
cargo build --release
```

## Usage

```bash
crush                              # Launch TUI
crush analyse .                    # Analyse directory
crush -i ./images -f webp -q 85    # Compress images
crush -i ./videos -f mp4           # Convert videos
crush setup                        # Setup dependencies
crush -v                           # Version
```

## Features

- 3 engines: FFmpeg (video/audio), Native Rust (image), ONNX AI (4x upscale)
- Professional TUI with mouse support
- 20+ formats: webp, avif, mp4, webm, mp3, flac, ogg, wav, and more
- Quality presets with custom quality option
- Automatic backup system
- Batch processing with parallel workers

## Dependencies

- **Rust 1.75+** — Required
- **FFmpeg** — Optional (for video/audio)
- **ONNX Model** — Optional (auto-downloaded for AI upscale)

## License

MIT
