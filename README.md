<div align="center">
  <img src="https://img.shields.io/badge/version-v2.4.0-22c55e?style=flat-square" alt="Version">
  <img src="https://img.shields.io/badge/go-1.23-00ADD8?style=flat-square" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-22c55e?style=flat-square" alt="License">
  <br><br>
  <h1>✦ CRUSH ✦</h1>
  <p><strong>Lightning-fast media compressor for developers</strong></p>
  <p>Built with Go • Powered by FFmpeg</p>
  <br>
  <p>
    <code>crush</code> • <code>crush -f webp -q 90</code> • <code>crush install</code>
  </p>
  <br>
</div>

---

## ✦ One-command install

```powershell
iex "& {$(iwr -Uri https://raw.githubusercontent.com/AliHamza-Coder/crush/main/scripts/install.ps1)}"
```

Downloads `crush.exe`, adds to PATH. Then run `crush install` to install FFmpeg.

### Portable (no install)

Download `crush.exe` from [releases](https://github.com/AliHamza-Coder/crush/releases), place `ffmpeg.exe` next to it, run.

---

## ✦ What it does

| Mode | Command | Description |
|------|---------|-------------|
| **Interactive** | `crush` | Analyse directory + arrow-key menu-driven conversion |
| **Direct CLI** | `crush -f webp -q 90` | One command, no prompts |
| **Install** | `crush install` | Auto-install FFmpeg via winget/brew/apt |
| **Analyse** | `crush analyse` | Show directory breakdown with bar charts |
| **Analyse JSON** | `crush analyse --json` | Machine-readable output |
| **Favicon** | `crush` → menu option | Generate 16×16 + 32×32 SVG favicons from images |

### Formats

| Type | Input | Output |
|------|-------|--------|
| **Image** | jpg, jpeg, png, webp, bmp, tiff, avif, gif, ico, heic, svg | webp, avif, jpg, png, gif, bmp |
| **Video** | mp4, mov, avi, mkv, wmv, flv, webm, m4v, mpg, 3gp, ts | mp4, webm, avi, mov, gif |
| **Audio** | mp3, wav, flac, ogg, aac, wma, m4a, opus, aiff, alac | mp3, ogg, wav, flac, aac, opus, m4a |
| **Export from video** | mp4, mov, webm, avi, mkv + audio codec | **→** mp3, wav, flac, ogg, aac, opus, m4a, alac |
| **Favicon** | png, jpg, webp, avif, ... | **→** 16×16 + 32×32 SVG |

---

## ✦ Quick start

```powershell
# Interactive mode — analyse + arrow-key menu
crush

# Convert everything in current dir to WebP
crush -f webp -q 90

# Compress a single video
crush video.mp4 -q 80

# All videos to WebM, 6 parallel workers
crush . -t video -f webm -p 6

# Batch convert audio to MP3
crush . -t audio -f mp3

# Process folder to custom output, no backup
crush ./raw/ -o ./optimised/ --no-backup

# Preview only (dry run)
crush -n

# Analyse directory
crush analyse ./assets/
```

### Interactive mode walkthrough

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│     ✦ CRUSH v2.4.0 — Lightning-fast media compressor │
│                                                     │
└─────────────────────────────────────────────────────┘

  ✦ Developed by Ali Hamza Coder ✦

  Directory: C:\projects\site\assets

  Images ████████████████████  12  24.5 MB
  Videos ████████               5  120.1 MB
  Audio  ████                   3   8.5 MB

  Total: 20 files | 153.1 MB
  Formats: JPEG x6  PNG x4  WebP x2  MP4 x3  WebM x2  MP3 x3

  #    Type      Size       Format   Filename
  1    JPEG      4.5 MB     JPG      photo1.jpg
  2    PNG       2.1 MB     PNG      hero.png
  3    WebP      1.1 MB     WebP     banner.webp
  4    MP4       45.2 MB    MP4      intro.mp4
  ...

  ▼ Choose Action (↑↓ to choose, Enter to confirm)
     ALL files  — images + videos + audio
     Images     — jpg, png, webp, avif, gif...
     Videos     — mp4, mov, webm, avi, mkv...
     Audio      — mp3, wav, flac, ogg, aac...
     Export Audio from Video — mp4, mov → mp3, wav, flac...
     Select specific files by number
     Change directory
     Generate Favicon — 16×16 + 32×32 SVG from image  🆕
     Quit
```

### Quality selection (with 90 + Custom presets)

When compressing or extracting, you can choose:

```
  ▼ Quality (↑↓ to choose, Enter to confirm)
     85  — balanced  ★ recommended
     90  — high quality
     75  — smaller file
     100 — maximum quality
     Lossless — original quality preserved
     Custom — enter any value (1-100)
```

Custom lets you type any number from 1–100 (e.g. `92`, `67`, `45`) for fine-grained control.

---

## ✦ Flags

```
  -i, --input <path>    Input file, directory, or glob  (default: .)
  -o, --output <dir>    Output directory                (default: same as input)
  -f, --format <fmt>    Target format                    (webp, mp4, mp3, ...)
  -q, --quality <1-100> Quality                          (default: 85)
  -b, --backup          Backup originals                 (default: true)
  --no-backup           Skip backup
  -p, --parallel <n>    Parallel workers                 (default: CPU cores)
  -t, --type <type>     Filter: image, video, audio      (default: all)
  -n, --dry-run         Preview without processing
  -v, --verbose         Show ffmpeg output
```

### Subcommands

```
  crush install      Auto-install FFmpeg + setup PATH
  crush analyse      Show directory analysis
  crush analyse --json   JSON output for scripts
```

---

## ✦ How it works

```
crush (no args)
  │
  ├─ analyse ./ → bar chart + numbered file list
  │
  └─ interactive menu (arrow keys or numbered fallback)
       ├─ All        → prompt format/quality → parallel convert
       ├─ Images     → prompt format/quality → parallel convert
       ├─ Videos     → prompt format/quality → parallel convert
       ├─ Audio      → prompt format/quality → parallel convert
       ├─ Export Audio → pick format (mp3/wav/flac/...) → quality → extract
       ├─ Select     → parse "1-4,7,9-11" → prompt → convert
       ├─ Change dir → re-analyse
       └─ Quit

Each conversion:
  1. Backup originals → backup/originals_<timestamp>/
  2. Skip files already in target format
  3. Process N files in parallel
  4. Summary: ✓ OK | ✗ FAIL | ⏭ SKIP | ⏱ time
```

---

## ✦ Edge cases

| Scenario | Behaviour |
|----------|-----------|
| Empty directory | Prompts for another directory |
| No FFmpeg | Shows install instructions |
| Already target format | Skipped automatically |
| Invalid range (`abc`) | Error with re-prompt |
| Reversed range (`5-1`) | Treated as `1-5` |
| Quality out of range | Clamped to 1-100 |
| Non-interactive terminal | Falls back to numbered menu instead of arrow keys |
| Ctrl+C during batch | Finishes current files gracefully |
| Duplicate filenames | Backup uses timestamp directories |
| **Audio extraction** | Original video is always preserved |

---

## ✦ Installation

### Windows (recommended)

```powershell
# One-liner (also works without Git)
iex "& {$(iwr -Uri https://raw.githubusercontent.com/AliHamza-Coder/crush/main/scripts/install.ps1)}"

# Then install FFmpeg
crush install
```

### Portable

```powershell
# Download crush.exe + ffmpeg.exe to same folder
# Run from that folder — no PATH needed
./crush
```

### Build from source

```powershell
# Requires Go 1.23+
git clone https://github.com/AliHamza-Coder/crush.git
cd crush
go build -o crush.exe ./cmd/crush/
```

### Linux / macOS

```bash
# Requires Go 1.23+
git clone https://github.com/AliHamza-Coder/crush.git
cd crush
go build -o crush ./cmd/crush/

# Install FFmpeg if needed
sudo apt install ffmpeg       # Debian/Ubuntu
brew install ffmpeg           # macOS
```

---

## ✦ Project structure

```
crush/
├── cmd/
│   └── crush/
│       └── main.go              # Entry point
├── internal/
│   ├── crush/
│   │   └── crush.go             # Config, flags, orchestrator
│   ├── analyse/
│   │   └── analyse.go           # Directory scanner + stats
│   ├── compress/
│   │   └── compress.go          # Image/video/audio encoding
│   ├── backup/
│   │   └── backup.go            # Backup creation (added v2.3.0)
│   ├── favicon/
│   │   └── favicon.go           # SVG favicon generator (added v2.4.0)
│   ├── install/
│   │   └── install.go           # Install subcommand
│   ├── ui/
│   │   ├── ui.go                # Colours, banners, helpers
│   │   └── term.go             # Arrow-key + fallback menus (huh)
│   └── fileutil/
│       └── fileutil.go          # Types, enums, file utils
├── scripts/
│   └── install.ps1              # One-line PowerShell installer
├── go.mod
├── README.md
├── CHANGELOG.md
└── .gitignore
```

---

## ✦ v2.4.0 Full Release Notes

See [CHANGELOG.md](CHANGELOG.md) for the complete history.

## ✦ v2.4.0 — What's New

### 🆕 Favicon Generator
Generate 16×16 and 32×32 SVG favicons from any image — right from the interactive menu. Uses ffmpeg to resize and embeds the result as base64 inline SVG.

### 🎬 Dramatically Better Video Quality
- **CRF 15 at default q85** (was CRF 21) — eliminates blocky artifacts
- **`-c:a copy`** preserves original audio during in-place compression (no more re-encoding to 128k AAC)
- **`-map a:0`** ensures full audio duration when extracting (no more truncated audio)
- **192k AAC** for format conversions (was 128k) — noticeably clearer audio

### 🔧 Bug Fixes
- **Flags after path** — `crush ./dir/ -f webp -q 90` now works correctly
- **Windows rename** — `os.Remove` before rename fixes "Access is denied"
- **CLI audio extraction** — `crush . -f mp3` now works on video files
- **`--help` cleanup** — exits cleanly without spurious error messages

---

### v2.3.0 — `promptui` → `charmbracelet/huh`

CRUSH v2.3.0 replaces the old `promptui` library with [`huh`](https://github.com/charmbracelet/huh) (built on Bubble Tea):

- ✅ **Arrow-key menus work on ALL terminals** — Windows cmd, PowerShell, Windows Terminal, Linux, macOS
- ✅ **No more silent "Goodbye!"** — the old library failed silently on some Windows terminals
- ✅ **Numbered fallback** — if the interactive menu fails for any reason, falls back to `1), 2), 3)...` number input

---

## ✦ GitHub releases

| File | Description |
|------|-------------|
| `crush_windows_amd64.zip` | Windows x86_64 binary + install script |
| `crush_linux_amd64.tar.gz` | Linux x86_64 binary |
| `crush_darwin_amd64.tar.gz` | macOS Intel binary |
| `crush_darwin_arm64.tar.gz` | macOS Apple Silicon binary |

Create a release with any of these methods:

```powershell
# Tag and push
git tag v2.4.0
git push origin v2.4.0

# GitHub CLI
gh release create v2.4.0 ./crush.exe --title "v2.4.0" --notes "See CHANGELOG.md"
```

---

<div align="center">
  <p>
    <strong>✦ Developed by <a href="https://github.com/AliHamza-Coder">Ali Hamza Coder</a> ✦</strong>
  </p>
  <p>
    <sub>MIT License — free to use, modify, distribute</sub>
  </p>
</div>
