## v2.3.1 (2026-07-12)

### Added
- **Quality presets**: Choose `90` (high quality) or `Custom` (enter any value 1-100) in the quality selector
- **Export Audio from Video**: Rewritten with dedicated format picker — pick from mp3, wav, flac, ogg, aac, opus, m4a, alac. Original video is always preserved.

### Fixed
- **Audio encoder bug**: `flac` and `alac` formats no longer silently encode as MP3 — correct codecs are used
- **Video deletion fixed**: Audio extraction no longer deletes the original video file

## v2.3.0 (2026-07-12)

### Changed
- **`promptui` → `charmbracelet/huh`**: Replaced the unmaintained `promptui` library with actively-maintained `huh` (built on Bubble Tea). Arrow-key menus now work on ALL terminals — Windows cmd, PowerShell, Windows Terminal, Linux, macOS. No more silent "Goodbye!" when the terminal doesn't support the old library.
- **Go 1.21 → 1.23** required by `huh`.

### Added
- **Numbered fallback**: If the interactive menu fails for any reason (non-interactive terminal, CI, etc.), automatically falls back to a simple number selection (`1) All files`, `2) Images`, etc.).
- **Missing `backup` package**: Backup functions (`CreateDir`, `CopyFile`) now included in the source tree.

### Fixed
- **Banner visual bug**: Version line was missing the closing `║` border — now properly aligned.
- **Consistent input handling**: `install.go` now uses `ui.ReadInput()` instead of raw `fmt.Scanln`.
