# CRUSH — Lightning-fast Media Compressor

> **v2.0** — Parallel batch compression for developers.  
> Converts & compresses **images, videos, and audio** into web-optimized formats.  
> Built with Go, powered by FFmpeg.

---

## Features

| Capability | Description |
|------------|-------------|
| **Interactive mode** | Run `crush` with no args — analyse + menu-driven conversion |
| **Direct CLI mode** | `crush -f webp -q 90` — one command, done |
| **Analyse subcommand** | `crush analyse` — see what's in a directory |
| **Parallel** | Uses all CPU cores by default |
| **Auto-backup** | Originals saved to `backup/originals_<timestamp>/` before any change |
| **Skip smartly** | Files already in target format are automatically skipped |
| **Range selection** | `1-4,7,9-11` or `all` — pick specific files by number |
| **Any-to-any** | Convert between all major media formats |
| **Glob support** | `crush *.jpg -f webp` |
| **Directory walk** | Recursively finds all media in subfolders |
| **JSON output** | `crush analyse --json` for scripts |
| **Portable mode** | Drop `ffmpeg.exe` next to `crush.exe`, no install |

### Supported formats

| Type | Input extensions | Output formats |
|------|-----------------|----------------|
| **Image** | jpg, jpeg, png, webp, bmp, tiff, avif, gif, ico, heic, svg | webp, avif, jpg, png, gif, bmp |
| **Video** | mp4, mov, avi, mkv, wmv, flv, webm, m4v, mpg, 3gp, ts | mp4, webm, avi, mov, gif |
| **Audio** | mp3, wav, flac, ogg, aac, wma, m4a, opus, aiff, alac | mp3, ogg, wav, flac, aac, opus, m4a |

---

## Quick install

```powershell
crush install
```

Auto-installs FFmpeg via package manager and adds crush to PATH.  
After that, `crush` works from any terminal.

| OS | Command used |
|----|-------------|
| Windows | `winget install -e --id Gyan.FFmpeg` |
| macOS | `brew install ffmpeg` |
| Linux | `sudo apt install ffmpeg` |

---

## Usage

### Interactive mode (no flags)

```powershell
crush
```

Shows directory analysis + menu:

```
╔══════════════════════════════════════════════╗
║     CRUSH v2.0.0                             ║
║     Interactive Media Compressor             ║
╚══════════════════════════════════════════════╝

Directory: C:\projects\site\assets

  Images █████████████████████████    12 (63%)  24.5 MB
  Videos ████████                      5 (26%)  120.1 MB
  Audio  ███                            2 (11%)  8.5 MB

  Total: 19 files  |  153.1 MB
  Breakdown: JPEG x6  PNG x4  WebP x2  MP4 x3  WebM x2  MP3 x2

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

After choosing a conversion option, you're asked for:
- Target format (or press Enter to keep original)
- Quality
- Backup preference
- Parallel workers

### Direct CLI mode

```powershell
crush -f webp -q 90              # all images -> WebP
crush video.mp4 -q 80            # compress single video
crush ./media/ -f webm -p 6      # folder -> WebM, 6 workers
crush . -t audio -f mp3          # all audio -> MP3
crush ./raw/ -o ./opt/           # custom output dir
crush -n                         # dry run (preview only)
```

### Select specific files by range

```powershell
# In interactive mode:
[S] Select specific files by number
Enter selection (e.g. 1-4,7,9-11 or 'all'): 1-4,7,9-11
```

Parses ranges, commas, mixed, reversed (`5-1` → `1-5`), and `all`.

### Analyse subcommand

```powershell
crush analyse                    # detailed analysis
crush analyse ./media/           # specific directory
crush analyse --json             # JSON output for scripts
```

---

## Flags

```
Input/Output:
  -i, --input <path>       Input file, dir, or glob (default: .)
  -o, --output <dir>       Output directory (default: same as input)
  -f, --format <fmt>       Target format: webp, mp4, mp3, avif, ...
  -q, --quality <1-100>    Quality (default: 85)

Backup:
  -b, --backup             Backup originals (default: true)
  --no-backup              Skip backup
  --backup-dir <path>      Custom backup folder

Performance:
  -p, --parallel <n>       Parallel workers (default: CPU cores)

Filters:
  -t, --type <type>        image, video, audio, all (default: all)

Other:
  -n, --dry-run            Preview without processing
  -v, --verbose            Show ffmpeg output
  --version                Show version
  -h, --help               Show help
```

---

## How it works

```
crush (no args)
  │
  ├─ analyse ./ → show bar chart + numbered file list
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
  3. Process N files in parallel (N = CPU cores)
  4. Show summary: ✓ OK | ✗ FAIL | ⏭ SKIP | ⏱ time
```

---

## Quality reference

### Video (CRF scale — lower = better)

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
| 100  | 1        | 100          | Lossless / max quality |
| 90   | 4        | 90           | Web — high quality |
| 80   | 7        | 80           | Web — good quality |
| 60   | 13       | 60           | Thumbnails / preview |

### Audio

| `-q` | MP3 VBR | AAC bitrate | Use case |
|------|---------|-------------|----------|
| 100  | V0 (~245k) | 320k     | Archival |
| 85   | V2 (~190k) | 256k     | Music |
| 70   | V4 (~160k) | 192k     | Podcast |
| 50   | V6 (~130k) | 128k     | Speech |

---

## Edge cases handled

| Scenario | Behavior |
|----------|----------|
| Empty directory | Shows "No media files found", prompts for another dir |
| No ffmpeg | Clear error + instructions |
| Already target format | Skipped automatically |
| Invalid range input | Error message, re-prompt |
| Reversed range (`5-1`) | Treated as `1-5` |
| Out-of-bounds numbers | Clamped silently |
| Quality out of range | Clamped to 1-100 |
| Ctrl+C during batch | Finishes current files gracefully |
| Duplicate file names | Backup uses timestamp dirs to avoid collision |

---

## Installation

### Portable (recommended)

1. Drop `crush.exe` and `ffmpeg.exe` in the same folder
2. Run — no PATH setup needed

### Installed

1. Add `ffmpeg.exe` to system `PATH`
2. Put `crush.exe` anywhere on `PATH`
3. Run `crush` from any terminal

### Build

```powershell
git clone https://github.com/AliHamza-Coder/crush.git
cd crush
go build -o crush.exe .
```

---

## Tech

- **Go** — single static binary, zero runtime dependencies
- **FFmpeg** — handles all actual encoding
- **MIT License** — free to use, modify, distribute

---

Made with by [AliHamza-Coder](https://github.com/AliHamza-Coder)
