# CRUSH v3.0 — Rust Rewrite Architecture

**Date:** 2026-09-08
**Status:** IMPLEMENTED
**From:** Go (v2.5.0) → Rust (v3.0.0)

---

## 1. Executive Summary

CRUSH v3.0 is a complete rewrite from Go to Rust. It transforms a single-engine FFmpeg wrapper into a **multi-engine multimedia processing TUI** with three distinct processing backends:

| Engine | Responsibility | Backend |
|--------|---------------|---------|
| **FFmpeg Engine** | Video compression, format conversion, audio extraction | FFmpeg subprocess |
| **Native Image Engine** | Static image compression & conversion (PNG→WebP, PNG→AVIF) | Rust `image` + `webp` crates |
| **AI Core Engine** | Neural-network image upscaling (4x) | ONNX Runtime via `ort` crate (Real-ESRGAN) |

The result: a single zero-dependency binary with a rich Ratatui TUI dashboard showing a live processing queue, per-file progress, and engine attribution.

---

## 2. Project Structure

```
crush/
├── Cargo.toml                    # Workspace root
├── crates/
│   ├── crush-core/               # Library crate — all business logic
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs            # Public API re-exports
│   │       ├── config.rs         # Config struct, flag parsing
│   │       ├── fileutil/
│   │       │   ├── mod.rs        # File type detection, extensions, helpers
│   │       │   └── analyze.rs    # Directory scanner + stats
│   │       ├── engine/
│   │       │   ├── mod.rs        # Engine trait + dispatcher
│   │       │   ├── ffmpeg.rs     # FFmpeg engine (video, audio)
│   │       │   ├── native.rs     # Native image engine (image crate)
│   │       │   └── ai_upscale.rs # ONNX AI upscaling engine
│   │       ├── backup.rs         # Backup creation/management
│   │       ├── queue.rs          # Task queue, progress tracking
│   │       └── error.rs          # Unified error types
│   │
│   └── crush-cli/                # Binary crate — CLI + TUI
│       ├── Cargo.toml
│       └── src/
│           ├── main.rs           # Entry point
│           ├── cli.rs            # Clap argument parsing
│           ├── tui/
│           │   ├── mod.rs        # TUI app state machine
│           │   ├── app.rs        # Main App struct, event loop
│           │   ├── ui.rs         # Ratatui rendering functions
│           │   ├── dashboard.rs  # Dashboard panel (stats, file list)
│           │   ├── queue.rs      # Live processing queue panel
│           │   ├── settings.rs   # Settings/compression config panel
│           │   └── input.rs      # User input handling
│           └── interactive.rs    # Interactive mode (non-TUI fallback)
│
├── models/                       # ONNX model files (gitignored, user-downloaded)
│   └── README.md                 # Instructions for downloading Real-ESRGAN
│
├── scripts/
│   ├── install.ps1               # Windows installer (updated for Rust binary)
│   └── build_release.sh          # Cross-platform release build script
│
└── README.md
```

---

## 3. Crate Architecture

### 3.1 Workspace `Cargo.toml`

```toml
[workspace]
members = ["crates/crush-core", "crates/crush-cli"]
resolver = "2"
```

### 3.2 `crush-core` (Library)

**Purpose:** Pure business logic, no TUI, no terminal I/O. Reusable by other tools.

**Key Dependencies:**

```toml
[dependencies]
# Image processing (native engine)
image = { version = "0.25", features = ["webp", "avif", "png", "jpeg"] }
webp = "0.7"

# AI upscaling
ort = { version = "2.0", features = ["load-dynamic"] }
ndarray = "0.16"

# Async runtime
tokio = { version = "1", features = ["full"] }

# Error handling
thiserror = "2"
anyhow = "1"

# Serialization
serde = { version = "1", features = ["derive"] }

# Utilities
pathdiff = "0.2"
chrono = "0.4"
```

**Key Types:**

```rust
// crates/crush-core/src/engine/mod.rs

pub enum Engine {
    Ffmpeg,
    Native,
    AiUpscale,
}

pub struct ProcessingTask {
    pub id: u64,
    pub input_path: PathBuf,
    pub output_path: PathBuf,
    pub task_type: TaskType,
    pub status: TaskStatus,
    pub progress: f32,          // 0.0 - 100.0
    pub engine: Engine,
    pub error: Option<String>,
}

pub enum TaskType {
    VideoCompress { quality: u8, format: String },
    VideoConvert { target: String, quality: u8 },
    ImageCompress { quality: u8, format: String },
    ImageConvert { target: String, quality: u8 },
    ImageUpscale { scale: u8, model: String },
    AudioExtract { format: String, quality: u8 },
    AudioConvert { format: String, quality: u8 },
}

pub enum TaskStatus {
    Pending,
    Running,
    Completed,
    Failed,
    Skipped,
}

// Engine trait
#[async_trait::async_trait]
pub trait Processor: Send + Sync {
    fn engine(&self) -> Engine;
    async fn process(&self, task: &mut ProcessingTask) -> anyhow::Result<()>;
    fn can_handle(&self, task: &TaskType) -> bool;
}
```

### 3.3 `crush-cli` (Binary)

**Key Dependencies:**

```toml
[dependencies]
crush-core = { path = "../crush-core" }

# TUI
ratatui = "0.29"
crossterm = { version = "0.28", features = ["event-stream"] }

# CLI
clap = { version = "4", features = ["derive"] }

# Async
tokio = { version = "1", features = ["full"] }

# Logging
tracing = "0.1"
tracing-subscriber = "0.3"
```

---

## 4. Feature-to-Engine Mapping

### 4.1 Video Compression & Version Conversion

**Engine:** FFmpeg (subprocess)
**Module:** `engine::ffmpeg`

```rust
// Flow:
// 1. User selects video files
// 2. Dispatcher checks file type → routes to Ffmpeg engine
// 3. Ffmpeg engine spawns `ffmpeg -i input -c:v libx264 -crf N ... -y output`
// 4. Progress parsed from stderr (time= field)
// 5. Task status updated in queue
```

**Supported operations:**
- Compress video (H.264/H.265/AV1 codec selection)
- Convert format (MP4↔WebM↔AVI↔MOV↔MKV)
- Extract audio from video
- Quality mapping: user quality (1-100) → CRF (10-44)

### 4.2 Static Image Compression & Conversion

**Engine:** Native Rust
**Module:** `engine::native`

```rust
// Flow:
// 1. User selects image files
// 2. Dispatcher checks file type → routes to Native engine
// 3. Native engine loads image via `image` crate
// 4. Encodes to target format (webp/avif/png/jpg) with quality settings
// 5. Writes output directly — no subprocess, no FFmpeg needed
```

**Supported operations:**
- PNG → WebP (lossy/lossless)
- PNG/JPG → AVIF
- JPG → PNG
- Any format → BMP
- Quality mapping: user quality (1-100) → encoder-specific params

**Key advantage:** Instant conversion for images. No FFmpeg dependency for this path.

### 4.3 AI Image Upscaling

**Engine:** ONNX Runtime (local)
**Module:** `engine::ai_upscale`

```rust
// Flow:
// 1. User selects image + chooses "AI Upscale 4x"
// 2. Dispatcher routes to AiUpscale engine
// 3. Engine loads ONNX model (Real-ESRGAN x4plus) from models/ dir
// 4. Pre-processes image (normalize, CHW format, tile splitting)
// 5. Runs inference via ort::Session
// 6. Post-processes output (denormalize, stitch tiles, RGB output)
// 7. Saves upscaled image (4x dimensions)
```

**Model loading:**
- Models stored in `~/.config/crush/models/` or `./models/`
- First run: prompt user to download Real-ESRGAN x4plus ONNX
- Model format: ONNX (optimized, quantized optional)

**Tiling strategy for large images:**
- Split into 512x512 tiles with 32px overlap
- Process each tile through the model
- Stitch output tiles seamlessly
- Memory-bounded: handles images of any size

---

## 5. TUI Dashboard Layout

```
┌──────────────────────────────────────────────────────────────────────┐
│  CRUSH v3.0 — MULTIMEDIA MISSION CONTROL              [?] Help [Q] │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─ INPUT FILES ──────────────┐  ┌─ COMPRESSION SETTINGS ─────────┐ │
│  │ 📁 /path/to/project        │  │ Engine: Auto (best per file)    │ │
│  │                            │  │ Quality: 85 (balanced)          │ │
│  │  holiday_video.mov  1.2 GB │  │ Video: H.264 / CRF 15          │ │
│  │  poster_art.png     15 MB  │  │ Image: WebP / q=85              │ │
│  │  hero.jpg           4.5 MB │  │ AI: Real-ESRGAN 4x              │ │
│  │  background.png     8 MB   │  │ Backup: ON (./backup/)          │ │
│  │  intro.mp4         45 MB   │  │ Workers: 8                      │ │
│  └────────────────────────────┘  └────────────────────────────────┘ │
│                                                                      │
│  ┌─ LIVE PROCESSING QUEUE ─────────────────────────────────────────┐ │
│  │                                                                  │ │
│  │  #1 holiday_video.mov  [██████████████░░░░░░░░] 68%  FFmpeg     │ │
│  │  #2 poster_art.png     [████░░░░░░░░░░░░░░░░░░] 20%  ONNX AI   │ │
│  │  #3 hero.jpg           [░░░░░░░░░░░░░░░░░░░░░░]  0%  Native    │ │
│  │  #4 background.png     [░░░░░░░░░░░░░░░░░░░░░░]  --  Queued    │ │
│  │  #5 intro.mp4          [░░░░░░░░░░░░░░░░░░░░░░]  --  Queued    │ │
│  │                                                                  │ │
│  └──────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌─ SUMMARY ────────────────────────────────────────────────────────┐ │
│  │  ⏱ 00:42  ✓ 2/5  ⚙ FFmpeg: 1  Native: 1  ONNX: 1  Pending: 2 │ │
│  └──────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

**TUI Panels:**

| Panel | Widget | Content |
|-------|--------|---------|
| Input Files | `List` | Files detected, sizes, types |
| Settings | `Block<List>` | Current config, quality, format targets |
| Queue | `List` + custom | Per-task progress bars, engine badge, status |
| Summary | `Paragraph` | Elapsed time, counts, engine distribution |

**Key bindings:**

| Key | Action |
|-----|--------|
| `Tab` | Cycle between panels |
| `Enter` | Start processing / select item |
| `Esc` | Back / cancel |
| `Space` | Toggle file selection |
| `?` | Help overlay |
| `q` | Quit |

---

## 6. Engine Dispatcher Logic

```rust
// crates/crush-core/src/engine/mod.rs

pub struct EngineDispatcher {
    ffmpeg: FfmpegEngine,
    native: NativeEngine,
    ai_upscale: AiUpscaleEngine,
}

impl EngineDispatcher {
    pub fn select_engine(&self, task: &TaskType, file_ext: &str) -> &dyn Processor {
        match task {
            TaskType::VideoCompress { .. }
            | TaskType::VideoConvert { .. }
            | TaskType::AudioExtract { .. }
            | TaskType::AudioConvert { .. } => &self.ffmpeg,

            TaskType::ImageCompress { format, .. }
            | TaskType::ImageConvert { target: format, .. } => {
                // Use native engine for image→image conversions
                // Fallback to FFmpeg if native doesn't support the format
                if NativeEngine::supports_format(file_ext, format) {
                    &self.native
                } else {
                    &self.ffmpeg
                }
            }

            TaskType::ImageUpscale { .. } => &self.ai_upscale,
        }
    }
}
```

---

## 7. Migration Mapping (Go → Rust)

| Go File | Rust Module | Notes |
|---------|-------------|-------|
| `cmd/crush/main.go` | `crush-cli/src/main.rs` | Entry point |
| `internal/crush/crush.go` | `crush-core/src/config.rs` + `crush-cli/src/cli.rs` | Split: config + CLI args |
| `internal/compress/compress.go` | `crush-core/src/engine/ffmpeg.rs` | FFmpeg engine |
| `internal/analyse/analyse.go` | `crush-core/src/fileutil/analyze.rs` | Directory scanner |
| `internal/fileutil/fileutil.go` | `crush-core/src/fileutil/mod.rs` | File types, helpers |
| `internal/ui/ui.go` | `crush-cli/src/tui/ui.rs` | Terminal output → Ratatui |
| `internal/ui/term.go` | `crush-cli/src/tui/input.rs` | Interactive menus → TUI events |
| `internal/backup/backup.go` | `crush-core/src/backup.rs` | Backup logic |
| `internal/install/install.go` | Not needed | Rust binary is self-contained |
| `internal/update/update.go` | `crush-cli/src/cli.rs` (via clap) | Self-update via cargo-install |
| `internal/arrange/arrange.go` | `crush-core/src/fileutil/arrange.rs` | File organizer |
| `internal/favicon/favicon.go` | `crush-cli/src/interactive.rs` | Favicon generator |
| *(new)* | `crush-core/src/engine/native.rs` | **NEW:** Native image processing |
| *(new)* | `crush-core/src/engine/ai_upscale.rs` | **NEW:** AI upscaling |
| *(new)* | `crush-core/src/queue.rs` | **NEW:** Task queue + progress |
| *(new)* | `crush-cli/src/tui/dashboard.rs` | **NEW:** Dashboard panel |
| *(new)* | `crush-cli/src/tui/queue.rs` | **NEW:** Live queue panel |

---

## 8. Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Initialize Rust workspace + crates
- [ ] Port `fileutil` (types, detection, helpers) → `crush-core/src/fileutil/`
- [ ] Port `analyse` (directory scanner) → `crush-core/src/fileutil/analyze.rs`
- [ ] Port `backup.rs` → `crush-core/src/backup.rs`
- [ ] Implement `config.rs` + `cli.rs` (Clap args)
- [ ] Basic `main.rs` that runs directory analysis
- **Milestone:** `crush-cli analyse ./dir` works

### Phase 2: FFmpeg Engine (Week 2-3)
- [ ] Implement `engine::ffmpeg` (video, audio processing)
- [ ] Port all FFmpeg command building from Go's `compress.go`
- [ ] Implement progress parsing from FFmpeg stderr
- [ ] Implement `EngineDispatcher` with FFmpeg + fallback logic
- [ ] Port interactive mode (non-TUI numbered menu as fallback)
- **Milestone:** `crush-cli compress ./dir -f webp -q 85` works

### Phase 3: Native Image Engine (Week 3-4)
- [ ] Implement `engine::native` using `image` + `webp` crates
- [ ] PNG→WebP (lossy + lossless)
- [ ] PNG/JPG→AVIF encoding
- [ ] Quality parameter mapping for each encoder
- [ ] Integrate into `EngineDispatcher`
- **Milestone:** Image compression works without FFmpeg

### Phase 4: AI Upscaling Engine (Week 4-5)
- [ ] Implement `engine::ai_upscale` with `ort` crate
- [ ] Model download/management system
- [ ] Image pre-processing (normalize, CHW, tiling)
- [ ] ONNX inference pipeline
- [ ] Image post-processing (denormalize, stitch)
- [ ] Integrate into `EngineDispatcher`
- **Milestone:** `crush-cli upscale image.png --scale 4` works

### Phase 5: Task Queue (Week 5-6)
- [ ] Implement `queue.rs` (task scheduling, progress tracking)
- [ ] Async task execution with `tokio`
- [ ] Per-task progress reporting (FFmpeg time parsing, native progress)
- [ ] Cancel/pause support
- [ ] Summary statistics
- **Milestone:** Multi-file batch processing with live progress

### Phase 6: TUI Dashboard (Week 6-8)
- [ ] Initialize Ratatui app scaffold
- [ ] Dashboard panel (input files, settings)
- [ ] Live processing queue panel with progress bars
- [ ] Engine badge display (FFmpeg / Native / ONNX)
- [ ] Keyboard navigation (Tab, Enter, Esc, Space)
- [ ] Help overlay
- [ ] Summary footer
- **Milestone:** Full TUI dashboard with live processing

### Phase 7: Polish & Release (Week 8-9)
- [ ] Cross-platform build scripts (Windows, Linux, macOS)
- [ ] Release binaries (GitHub Releases)
- [ ] Updated README + documentation
- [ ] Updated install script
- [ ] Integration tests
- [ ] Performance benchmarks
- **Milestone:** v3.0.0 release ready

---

## 9. Key Design Decisions

### 9.1 Why lib + bin split?
- **Library crate** (`crush-core`) can be used by other Rust projects as a dependency
- **Binary crate** (`crush-cli`) handles all terminal/TUI concerns
- Clean separation: business logic never touches terminal directly
- Enables future GUI or web interface using the same core

### 9.2 Why Native Image Engine (not just FFmpeg)?
- **Zero-dependency for images:** No FFmpeg needed for image compression
- **Faster startup:** No subprocess overhead for simple image conversions
- **Better quality control:** Direct access to image crate's quality parameters
- **Smaller binary:** Could theoretically ship without FFmpeg for image-only users

### 9.3 Why ONNX (not built-in ML)?
- **Model flexibility:** Users can swap models (ESRGAN, SwinIR, etc.)
- **Performance:** ONNX Runtime is highly optimized (CPU + GPU)
- **Ecosystem:** Huge model zoo available in ONNX format
- **ort crate:** Mature Rust bindings for ONNX Runtime

### 9.4 Why Ratatui over simpler TUI?
- **Rich widgets:** Progress bars, tables, lists built-in
- **Active community:** Most popular Rust TUI framework
- ** Crossterm backend:** Works on Windows, Linux, macOS
- **Composable:** Easy to add panels, overlays, themes

---

## 10. Dependencies Summary

### crush-core
| Crate | Version | Purpose |
|-------|---------|---------|
| `image` | 0.25 | Image loading/encoding (PNG, JPG, BMP, TIFF) |
| `webp` | 0.7 | WebP encoding (lossy + lossless) |
| `ort` | 2.0 | ONNX Runtime for AI inference |
| `ndarray` | 0.16 | N-dimensional arrays for image tensors |
| `tokio` | 1.x | Async runtime for parallel processing |
| `thiserror` | 2.x | Ergonomic error types |
| `anyhow` | 1.x | Error context and propagation |
| `serde` | 1.x | Config serialization |
| `pathdiff` | 0.2 | Relative path computation |
| `chrono` | 0.4 | Timestamps for backups |
| `async-trait` | 0.1 | Async trait methods |

### crush-cli
| Crate | Version | Purpose |
|-------|---------|---------|
| `crush-core` | path dep | Local library dependency |
| `ratatui` | 0.29 | TUI framework |
| `crossterm` | 0.28 | Terminal manipulation |
| `clap` | 4.x | CLI argument parsing |
| `tokio` | 1.x | Async runtime |
| `tracing` | 0.1 | Structured logging |
| `tracing-subscriber` | 0.3 | Log output formatting |

---

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `ort` crate compile issues on Windows | High | Use `load-dynamic` feature, ship ONNX Runtime DLL alongside binary |
| Large image tiling memory usage | Medium | Configurable tile size, streaming processing |
| FFmpeg not found | Medium | Graceful fallback: disable video features, show clear error |
| ONNX model download required | Low | First-run wizard with download instructions, auto-download option |
| Cross-platform TUI rendering | Low | Crossterm handles this; test on all 3 platforms |

---

## 12. Success Criteria

- [ ] Binary compiles on Windows, Linux, macOS
- [ ] FFmpeg video compression matches Go version quality/size
- [ ] Native image compression produces smaller files than FFmpeg for WebP
- [ ] AI upscaling produces visually superior 4x output
- [ ] TUI dashboard renders correctly in Windows Terminal, iTerm2, kitty
- [ ] Processing queue shows real-time per-file progress
- [ ] Engine badge correctly identifies which backend processed each file
- [ ] All existing Go features ported (arrange, favicon, backup, etc.)
- [ ] Binary size < 15MB (excluding ONNX models)
- [ ] Startup time < 500ms
