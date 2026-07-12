# Changelog

## v2.3.0 (2026-07-12)

### Changed
- **Replaced `promptui` with `charmbracelet/huh`** — arrow-key menus now work on all terminals (Windows cmd/PowerShell, macOS, Linux), no more silent "Goodbye!" exit on unsupported terminals
- **Go 1.21 → 1.23** required by `huh` dependency

### Added
- **Numbered fallback menu** — if the interactive menu fails (non-interactive terminal, CI, etc.), automatically falls back to a simple `1), 2), 3)...` number pick
- **`internal/backup` package** — backup directory creation and file copying (was missing from source tree)

### Fixed
- **Banner version line missing closing `║` border** — fixed visual artifact in the CRUSH title banner
- **`install.go` uses `ui.ReadInput`** instead of raw `fmt.Scanln` for consistency

## v2.2.6 (2026-07-12)

### Fixed
- **PowerShell installer download failure** — added TLS 1.2 and `-UseBasicParsing` to `Invoke-WebRequest` in `install.ps1`
  - TLS 1.2 is required by GitHub but was not enabled by default on older Windows/PowerShell versions
  - `-UseBasicParsing` suppresses the script-execution security warning

## v2.2.5 (2026-07-12)

### Added
- **Arrow key selection for all prompts** — use ↑/↓ to choose mode, quality, backup, and format (previously only format used arrow keys)
- New `SelectQuality` helper — pick quality from a curated list (Balanced, Smaller, Max, Lossless) with arrow keys instead of typing a number
- **Safe in-place compression** — writes to `.crush_tmp` file first, then renames to target (prevents data corruption if interrupted)
- Backup file auto-cleaned on failure — no wasted disk space from failed processing

### Fixed
- "★ recommended" label duplication bug in Windows arrow key menu
- Backup copies no longer orphaned when compression fails

### Changed
- All interactive prompts now use arrow key selection (mode, quality, backup yes/no, format)
- First list option now shows `(default)` label for clarity
- Code quality improvements in `processFile` and `runProcess`

## v2.2.4 (2026-07-12)

### Fixed
- **Update now works** — batch file renames old exe before moving new one (Windows blocks overwriting a running executable)
- Previously `crush update` downloaded the new version but couldn't replace the running exe

## v2.2.3 (2026-07-12)

### Added
- **Arrow key selection** on Windows — use ↑/↓ to navigate format list, Enter to confirm
- Non-Windows still uses number selection as fallback

### Changed
- **Analyse now scans current dir only** — no longer recurses into subdirectories
- Subdirectory support planned for a future release

## v2.2.2 (2026-07-12)

### Fixed
- **Files no longer duplicated** — backup/ directory excluded from analyse scan (was doubling file count)
- **Originals deleted after successful conversion** — when backup is enabled, original file is removed from current dir (backup has the copy)
- **Format selection menu** — pick format by number (1, 2, 3...) instead of typing name
- **Consistent behavior** in both interactive and direct modes

### Changed
- Analyse now skips `backup*` and `originals_*` directories during walk
- Format list shown with recommended format marked ★

## v2.2.1 (2026-07-12)

### Fixed
- Uninstall now uses PowerShell instead of unreliable batch file — deletes correctly on Windows
- Confirmed `crush -v` and `crush --version` work properly
- Added `_crush_*` temp files to .gitignore

## v2.2.0 (2026-07-12)

### Added
- **Two distinct modes**: Compress (shrink, keep format) vs Convert (change format)
- **Extract audio from video** — `[X]` menu option, e.g., mp4 → mp3
- **Recommended quality table** shown before quality prompt per media type
- Smart defaults: empty quality = lossless, empty format = keep original

### Changed
- Clearer mode picker after selecting files
- Quality prompt now shows recommended values (85★, 75, 100)
- Audio extraction uses proper `-vn` flag

### Fixed
- Empty quality now defaults to lossless/best instead of arbitrary 85

## v2.1.0 (2026-07-12)

### Added
- `crush update` — auto-download and install newer versions from GitHub
- `crush uninstall` — cleanly remove CRUSH (PATH + executable)
- Format hints in interactive menu (shows supported extensions per type)
- Simpler conversion prompts with examples for target format

### Changed
- Clearer menu layout with extension examples
- Smoother setup prompts: Enter = default, n/N = skip

### Fixed
- Version bumped to v2.1.0

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
