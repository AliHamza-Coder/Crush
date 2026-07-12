# Changelog

## v2.0.0 (2026-07-12)

### Added
- Interactive mode with directory analysis + menu
- `crush install` — auto-install FFmpeg via winget/brew/apt
- `crush analyse` — directory analysis with bar charts
- `crush analyse --json` — JSON output for scripts
- Parallel processing (all CPU cores)
- Auto-backup with timestamp folders
- Range selection: `1-4,7,9-11` or `all`
- Skip files already in target format
- Image, video, and audio support (25+ formats)
- Format conversion: any → any (webp, avif, mp4, webm, mp3, etc.)
- Developer credit banner

### Changed
- Professional package structure with `cmd/` and `internal/` layout
- Pure CLI interface (no interactive prompts in flag mode)
- Quality mapping: `-q 1-100` → proper CRF/q:v/bitrate per codec
- Single-line PowerShell installer

### Fixed
- `pause()` eating input on Windows
- Batch errors being silently ignored
- Quality value clamping

## v1.0.0 (2026-06-01)

- Initial release with interactive menu
- Basic video and image compression via FFmpeg
- Portable mode support
