<div align="center">
  <img src="https://img.shields.io/badge/version-v2.0.0-22c55e?style=flat-square" alt="Version">
  <img src="https://img.shields.io/badge/go-1.21-00ADD8?style=flat-square" alt="Go">
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
| **Interactive** | `crush` | Analyse directory + menu-driven conversion |
| **Direct CLI** | `crush -f webp -q 90` | One command, no prompts |
| **Install** | `crush install` | Auto-install FFmpeg via winget/brew/apt |
| **Analyse** | `crush analyse` | Show directory breakdown with bar charts |
| **Analyse JSON** | `crush analyse --json` | Machine-readable output |

### Formats

| Type | Input | Output |
|------|-------|--------|
| **Image** | jpg, jpeg, png, webp, bmp, tiff, avif, gif, ico, heic, svg | webp, avif, jpg, png, gif, bmp |
| **Video** | mp4, mov, avi, mkv, wmv, flv, webm, m4v, mpg, 3gp, ts | mp4, webm, avi, mov, gif |
| **Audio** | mp3, wav, flac, ogg, aac, wma, m4a, opus, aiff, alac | mp3, ogg, wav, flac, aac, opus, m4a |

---

## ✦ Quick start

```powershell
# Interactive mode — analyse + menu
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
╔══════════════════════════════════════════════╗
║     ✦ CRUSH v2.0.0                           ║
║     Media Compressor                         ║
╚══════════════════════════════════════════════╝

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

  [A] Convert ALL files
  [I] Convert Images only
  [V] Convert Videos only
  [O] Convert Audio only
  [S] Select specific files by number
  [D] Change directory
  [Q] Quit
```

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
  └─ interactive menu
       ├─ [A] All     → prompt format/quality → parallel convert
       ├─ [I] Images  → prompt format/quality → parallel convert
       ├─ [V] Videos  → prompt format/quality → parallel convert
       ├─ [O] Audio   → prompt format/quality → parallel convert
       ├─ [S] Select  → parse "1-4,7,9-11" → prompt → convert
       └─ [D] Change dir → re-analyse

Each conversion:
  1. Backup originals → backup/originals_<timestamp>/
  2. Skip files already in target format
  3. Process N files in parallel
  4. Summary: ✓ OK | ✗ FAIL | ⏭ SKIP | ⏱ time
```

---

## ✦ Quality reference

### Video (CRF — lower = better)

| `-q` | CRF | Use case |
|------|-----|----------|
| 100  | 18  | Archival / near-lossless |
| 85   | 21  | Web — high quality |
| 70   | 25  | Web — balanced |
| 50   | 29  | Storage saving |
| 30   | 33  | Maximum compression |

### Image

| `-q` | JPEG q:v | WebP quality | Use case |
|------|----------|--------------|----------|
| 100  | 1        | 100          | Lossless |
| 90   | 4        | 90           | Web — high |
| 80   | 7        | 80           | Web — good |
| 60   | 13       | 60           | Thumbnails |

### Audio

| `-q` | MP3 VBR | AAC | Use case |
|------|---------|-----|----------|
| 100  | V0 (245k) | 320k | Archival |
| 85   | V2 (190k) | 256k | Music |
| 70   | V4 (160k) | 192k | Podcast |
| 50   | V6 (130k) | 128k | Speech |

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
| Ctrl+C during batch | Finishes current files gracefully |
| Duplicate filenames | Backup uses timestamp directories |

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
git clone https://github.com/AliHamza-Coder/crush.git
cd crush
go build -o crush.exe ./cmd/crush/
```

### Linux / macOS

```bash
# Build from source
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
│   │   └── backup.go            # Backup creation
│   ├── install/
│   │   └── install.go           # Install subcommand
│   ├── ui/
│   │   └── ui.go                # Colours, banners, helpers
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

## ✦ GitHub releases

| File | Description |
|------|-------------|
| `crush_windows_amd64.zip` | Windows x86_64 binary + install script |
| `crush_linux_amd64.tar.gz` | Linux x86_64 binary |
| `crush_darwin_amd64.tar.gz` | macOS Intel binary |
| `crush_darwin_arm64.tar.gz` | macOS Apple Silicon binary |

Create a release with any of these methods:

```powershell
# 1. Tag and push
git tag v2.0.0
git push origin v2.0.0

# 2. GitHub CLI
gh release create v2.0.0 ./crush.exe --title "v2.0.0" --notes "See CHANGELOG.md"

# 3. GitHub web UI → Releases → Draft a new release
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
