package crush

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliHamza-Coder/crush/internal/analyse"
	"github.com/AliHamza-Coder/crush/internal/backup"
	"github.com/AliHamza-Coder/crush/internal/compress"
	"github.com/AliHamza-Coder/crush/internal/favicon"
	"github.com/AliHamza-Coder/crush/internal/fileutil"
	"github.com/AliHamza-Coder/crush/internal/install"
	"github.com/AliHamza-Coder/crush/internal/ui"
	"github.com/AliHamza-Coder/crush/internal/update"
)

type Config struct {
	Input     string
	OutputDir string
	Format    string
	Quality   int
	Lossless  bool
	Backup    bool
	BackupDir string
	Parallel  int
	Filter    string
	DryRun    bool
	Verbose   bool
}

func Run() int {
	if len(os.Args) > 1 {
		arg := strings.ToLower(os.Args[1])
		switch arg {
		case "install":
			install.Run()
			return 0
		case "uninstall":
			install.Uninstall()
			return 0
		case "update":
			update.Run()
			return 0
		case "analyse", "analyze":
			dir := "."
			if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
				dir = os.Args[2]
			}
			jsonMode := false
			for _, a := range os.Args[2:] {
				if a == "--json" || a == "-j" {
					jsonMode = true
				}
			}
			runAnalyse(dir, jsonMode)
			return 0
		case "help", "--help", "-h":
			printUsage()
			return 0
		case "version", "--version", "-v":
			fmt.Printf("CRUSH %s\n", fileutil.Version)
			return 0
		}
	}

	if hasAnyFlags() {
		cfg := parseFlags()
		return directMode(cfg)
	}

	return interactiveMode()
}

func hasAnyFlags() bool {
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "-") {
			return true
		}
		if a == "." || a == ".." {
			continue
		}
		if _, err := os.Stat(a); err == nil {
			return true
		}
		if strings.Contains(a, "*") || strings.Contains(a, "?") {
			return true
		}
	}
	return false
}

func parseFlags() Config {
	var cfg Config
	cfg.Backup = true
	cfg.Quality = 85
	cfg.Parallel = runtime.NumCPU()
	cfg.Input = "."
	cfg.Filter = "all"

	flag.StringVar(&cfg.Input, "i", ".", "Input file, directory, or glob")
	flag.StringVar(&cfg.Input, "input", ".", "Input file, directory, or glob")
	flag.StringVar(&cfg.OutputDir, "o", "", "Output directory")
	flag.StringVar(&cfg.OutputDir, "output", "", "Output directory")
	flag.StringVar(&cfg.Format, "f", "", "Target format")
	flag.StringVar(&cfg.Format, "format", "", "Target format")
	flag.IntVar(&cfg.Quality, "q", 85, "Quality 1-100")
	flag.IntVar(&cfg.Quality, "quality", 85, "Quality 1-100")
	flag.BoolVar(&cfg.Lossless, "lossless", false, "Lossless mode (preserves original quality)")
	flag.BoolVar(&cfg.Backup, "b", true, "Backup originals")
	flag.BoolVar(&cfg.Backup, "backup", true, "Backup originals")
	flag.StringVar(&cfg.BackupDir, "backup-dir", "", "Custom backup directory")
	flag.IntVar(&cfg.Parallel, "p", runtime.NumCPU(), "Parallel workers")
	flag.IntVar(&cfg.Parallel, "parallel", runtime.NumCPU(), "Parallel workers")
	flag.StringVar(&cfg.Filter, "t", "all", "Filter: image, video, audio")
	flag.StringVar(&cfg.Filter, "type", "all", "Filter: image, video, audio")
	flag.BoolVar(&cfg.DryRun, "n", false, "Dry run")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Dry run")
	flag.BoolVar(&cfg.Verbose, "v", false, "Verbose")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose")
	_ = flag.Bool("h", false, "Help")
	_ = flag.Bool("help", false, "Help")

	flag.Usage = printUsage

	// Re-arrange args so flags come before positional args.
	// Go's flag.Parse stops at the first non-flag arg, so
	// "crush ./dir/ -f webp" would silently ignore -f.
	var flagArgs, positional []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		// Handle --no-backup before flag.Parse (not a registered flag)
		if a == "--no-backup" {
			cfg.Backup = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
			// Value-taking flags — grab the next arg if it's not a flag
			needsVal := false
			switch a {
			case "-i", "--input", "-o", "--output", "-f", "--format",
				"-q", "--quality", "-p", "--parallel", "-t", "--type",
				"-b", "--backup", "--backup-dir":
				needsVal = true
			}
			if needsVal && i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, os.Args[i])
			}
		} else {
			positional = append(positional, a)
		}
	}

	// Parse the re-ordered flags
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	if err := flag.CommandLine.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if cfg.Quality < 1 {
		cfg.Quality = 1
	}
	if cfg.Quality > 100 {
		cfg.Quality = 100
	}
	if cfg.Parallel < 1 {
		cfg.Parallel = 1
	}
	if len(positional) > 0 {
		cfg.Input = positional[0]
	}

	return cfg
}

func printUsage() {
	c := func(s string) string { return fileutil.Cyan + s + fileutil.Reset }
	g := func(s string) string { return fileutil.Green + s + fileutil.Reset }

	fmt.Printf("%sCRUSH %s — Lightning-fast media compressor%s\n\n", fileutil.Bold, fileutil.Version, fileutil.Reset)
	fmt.Printf("%sUSAGE%s\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  crush                        Interactive mode (analyse + menu)\n")
	fmt.Printf("  crush [flags] [input]        Direct CLI mode\n")
	fmt.Printf("  crush install                Install FFmpeg + add to PATH\n")
	fmt.Printf("  crush update                 Check for updates and self-update\n")
	fmt.Printf("  crush uninstall              Remove CRUSH from your system\n")
	fmt.Printf("  crush analyse                Analyse directory only\n")
	fmt.Printf("  crush analyse --json         Analyse as JSON\n\n")
	fmt.Printf("%sFLAGS%s\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  -i, --input <path>    Input (%s)\n", c("default: ."))
	fmt.Printf("  -o, --output <dir>    Output directory (%s)\n", c("default: same as input"))
	fmt.Printf("  -f, --format <fmt>    Target format (%s)\n", c("webp, mp4, mp3, avif, webm..."))
	fmt.Printf("  -q, --quality <1-100> Quality (%s)\n", c("default: 85"))
	fmt.Printf("  -b, --backup          Backup originals (%s)\n", g("default: true"))
	fmt.Printf("  --no-backup           Skip backup\n")
	fmt.Printf("  -p, --parallel <n>    Workers (%s)\n", c(fmt.Sprintf("default: %d", runtime.NumCPU())))
	fmt.Printf("  -t, --type <type>     Filter: image, video, audio (%s)\n", c("default: all"))
	fmt.Printf("  --lossless            Preserve original quality (%s)\n", c("no compression"))
	fmt.Printf("  -n, --dry-run         Preview only\n")
	fmt.Printf("  -v, --verbose         Show ffmpeg output\n")
	fmt.Println()
}

// ─── INTERACTIVE MODE ────────────────────────────────────────────

func interactiveMode() int {
	ffmpeg, _ := getFFmpeg()
	if ffmpeg == "" {
		return 1
	}

	dir := "."

	for {
		ui.ClearScreen()
		ui.PrintBanner()

		files, stats := analyse.Directory(dir)
		analyse.PrintAnalysis(stats)
		analyse.PrintFileList(files)

		if len(files) == 0 {
			ui.PrintWarn(fmt.Sprintf("No media files found in: %s", dir))
			d := ui.ReadInput("  Enter directory to analyse (or 'q' to quit): ")
			if strings.ToLower(d) == "q" || strings.ToLower(d) == "quit" {
				return 0
			}
			if d != "" {
				dir = d
			}
			continue
		}

		choice := ui.SelectFromList([]string{
			"ALL files  — images + videos + audio",
			"Images     — jpg, png, webp, avif, gif...",
			"Videos     — mp4, mov, webm, avi, mkv...",
			"Audio      — mp3, wav, flac, ogg, aac...",
			"Extract audio from video — e.g., mp4 → mp3",
			"Select specific files by number",
			"Change directory",
			"Generate Favicon — 16×16 + 32×32 SVG from image",
			"Quit",
		}, "Choose Action (↑↓ to choose, Enter to confirm):")

		switch choice {
		case "ALL files  — images + videos + audio":
			pickMode(ffmpeg, fileutil.FilterByType(files, "all"), "all")
		case "Images     — jpg, png, webp, avif, gif...":
			pickMode(ffmpeg, fileutil.FilterByType(files, "image"), "image")
		case "Videos     — mp4, mov, webm, avi, mkv...":
			pickMode(ffmpeg, fileutil.FilterByType(files, "video"), "video")
		case "Audio      — mp3, wav, flac, ogg, aac...":
			pickMode(ffmpeg, fileutil.FilterByType(files, "audio"), "audio")
		case "Extract audio from video — e.g., mp4 → mp3":
			pickExtract(ffmpeg, fileutil.FilterByType(files, "video"), "video")
		case "Select specific files by number":
			selectSpecific(ffmpeg, files)
		case "Change directory":
			d := ui.ReadInput("  Enter directory: ")
			if d != "" {
				dir = d
			}
		case "Generate Favicon — 16×16 + 32×32 SVG from image":
			favicon.RunFaviconMenu(ffmpeg, files)
		case "Quit", "":
			fmt.Println("  Goodbye!")
			return 0
		}
	}
}

// ─── MODE PICKER ─────────────────────────────────────────────────

func pickMode(ffmpeg string, files []fileutil.FileInfo, filter string) {
	if len(files) == 0 {
		ui.PrintWarn("No files to process")
		ui.Pause()
		return
	}

	ui.PrintSection("Choose Mode")
	fmt.Printf("  %d file(s) selected\n\n", len(files))

	mode := ui.SelectFromList([]string{"Compress — shrink file size, keep same format", "Convert — change to another format (e.g., jpg → webp)"}, "Mode (↑↓ to choose, Enter to confirm):")

	switch mode {
	case "Convert — change to another format (e.g., jpg → webp)":
		convertMode(ffmpeg, files, filter)
	default:
		compressMode(ffmpeg, files, filter)
	}
}

// ─── COMPRESS MODE ───────────────────────────────────────────────

func compressMode(ffmpeg string, files []fileutil.FileInfo, filter string) {
	ui.PrintSection("Compress Settings")

	quality, lossless := ui.SelectQuality(filter)

	backupEnabled := ui.SelectFromList([]string{"Yes — backup originals before processing", "No — skip backup"}, "Backup originals? (↑↓ to choose, Enter to confirm):") == "Yes — backup originals before processing"

	bDir := ""
	if backupEnabled {
		bDir = ui.ReadInput("  Backup folder (Enter = ./backup/): ")
	}

	pStr := ui.ReadInput(fmt.Sprintf("  Parallel workers (Enter = %d): ", runtime.NumCPU()))
	parallel := runtime.NumCPU()
	if pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallel = p
		}
	}

	runProcess(ffmpeg, files, "", quality, lossless, backupEnabled, bDir, parallel)
}

// ─── CONVERT MODE ────────────────────────────────────────────────

func convertMode(ffmpeg string, files []fileutil.FileInfo, filter string) {
	ui.PrintSection("Convert Settings")

	fmt.Printf("  %d file(s) selected\n", len(files))

	format := ui.PrintFormatMenu(filter)
	if format == "" {
		ui.PrintWarn("No format selected — switching to compress mode")
		compressMode(ffmpeg, files, filter)
		return
	}

	quality, lossless := ui.SelectQuality(filter)

	backupEnabled := ui.SelectFromList([]string{"Yes — backup originals before processing", "No — skip backup"}, "Backup originals? (↑↓ to choose, Enter to confirm):") == "Yes — backup originals before processing"

	bDir := ""
	if backupEnabled {
		bDir = ui.ReadInput("  Backup folder (Enter = ./backup/): ")
	}

	pStr := ui.ReadInput(fmt.Sprintf("  Parallel workers (Enter = %d): ", runtime.NumCPU()))
	parallel := runtime.NumCPU()
	if pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallel = p
		}
	}

	runProcess(ffmpeg, files, format, quality, lossless, backupEnabled, bDir, parallel)
}

// ─── EXTRACT AUDIO ───────────────────────────────────────────────

func pickExtract(ffmpeg string, files []fileutil.FileInfo, filter string) {
	files = fileutil.FilterByType(files, "video")
	if len(files) == 0 {
		ui.PrintWarn("No video files found to extract audio from")
		ui.Pause()
		return
	}

	ui.PrintSection("Export Audio from Video")
	fmt.Printf("  %d video file(s) — audio will be extracted to current dir\n\n", len(files))

	// Show all available audio formats the user can export to
	format := ui.SelectFromList([]string{
		"mp3  — MPEG Audio Layer 3 (universal)",
		"wav  — PCM Wave (lossless, uncompressed)",
		"flac — Free Lossless Audio Codec",
		"ogg  — Vorbis (good compression)",
		"aac  — Advanced Audio Codec",
		"opus — Opus (best for low bitrate)",
		"m4a  — MPEG-4 Audio (AAC in M4A container)",
		"alac — Apple Lossless Audio Codec",
	}, "Export format (↑↓ to choose, Enter to confirm):")

	if format == "" {
		format = "mp3"
	}
	// Extract just the format name (before the first space)
	format = strings.Split(format, " ")[0]

	quality, _ := ui.SelectQuality("audio")

	pStr := ui.ReadInput(fmt.Sprintf("  Parallel workers (Enter = %d): ", runtime.NumCPU()))
	parallel := runtime.NumCPU()
	if pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallel = p
		}
	}

	runExtractAudio(ffmpeg, files, format, quality, parallel)
}

func runExtractAudio(ffmpeg string, files []fileutil.FileInfo, format string, quality int, parallel int) {
	fmt.Printf("\n")
	ui.PrintStep("Exporting audio...")

	var success, failed int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	start := time.Now()

	total := len(files)
	for i, f := range files {
		wg.Add(1)
		go func(idx int, file fileutil.FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			outName := fileutil.FileNameWithoutExt(file.Name) + "." + format
			outPath := filepath.Join(".", outName)
			err := compress.Audio(ffmpeg, file.Path, outPath, quality, format, false)

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				ui.PrintFail(fmt.Sprintf("%s — %v", file.Name, err))
				failed++
			} else {
				// Original video is kept — only the audio file is created
				ui.PrintOK(fmt.Sprintf("%s → %s (video kept)", file.Name, outName))
				success++
			}
			mu.Unlock()
		}(i, f)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println()
	ui.PrintResultSummary(success, failed, 0, elapsed)
	ui.PrintOK("Original video files preserved in current directory")
	ui.Pause()
}

func selectSpecific(ffmpeg string, files []fileutil.FileInfo) {
	if len(files) == 0 {
		ui.PrintWarn("No files available")
		ui.Pause()
		return
	}

	fmt.Printf("\n  Available files: 1-%d\n", len(files))
	input := ui.ReadInput("Enter selection (e.g. 1-4,7,9-11 or 'all'): ")

	selected, err := parseRange(input, len(files))
	if err != nil {
		ui.PrintFail(fmt.Sprintf("Invalid selection: %s", err))
		ui.Pause()
		return
	}

	var selectedFiles []fileutil.FileInfo
	for _, idx := range selected {
		selectedFiles = append(selectedFiles, files[idx-1])
	}

	fmt.Printf("\n  %d file(s) selected\n", len(selectedFiles))
	pickMode(ffmpeg, selectedFiles, "custom")
}

func runProcess(ffmpeg string, files []fileutil.FileInfo, format string, quality int, lossless bool, backupEnabled bool, backupDir string, parallel int) {
	fmt.Printf("\n")
	ui.PrintStep("Starting...")

	var backupTarget string
	if backupEnabled {
		backupTarget = backup.CreateDir(backupDir, ".")
		ui.PrintOK(fmt.Sprintf("Backup created at: %s", backupTarget))
	}

	var success, failed, skipped int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	start := time.Now()

	total := len(files)
	for i, f := range files {
		if format != "" && strings.EqualFold(filepath.Ext(f.Name), "."+format) {
			skipped++
			continue
		}

		wg.Add(1)
		go func(idx int, file fileutil.FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if backupEnabled {
				dst := filepath.Join(backupTarget, file.Name)
				if err := backup.CopyFile(file.Path, dst); err != nil {
					ui.PrintWarn(fmt.Sprintf("Backup failed: %s", file.Name))
				}
			}

			cfg := Config{Format: format, Quality: quality, Lossless: lossless}
			// Output dir: for compress (same format) go to file's own directory;
			// for convert (different format) go to current directory.
			outDir := filepath.Dir(file.Path)
			if format != "" {
				outDir = "."
			}
			err := processFile(ffmpeg, file, cfg, outDir)

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				if backupEnabled {
					dst := filepath.Join(backupTarget, file.Name)
					os.Remove(dst)
				}
				ui.PrintFail(fmt.Sprintf("%s — %v", file.Name, err))
				failed++
			} else if backupEnabled && format == "" {
				// Compress in-place: original backed up, compressed file in same dir
				ui.PrintOK(fmt.Sprintf("%s compressed ✓", file.Name))
				success++
			} else if backupEnabled && format != "" {
				// Convert: original backed up (and deleted), new format in current dir
				ui.PrintOK(fmt.Sprintf("%s → %s ✓", file.Name, fileutil.FileNameWithoutExt(file.Name)+"."+format))
				success++
			} else {
				ui.PrintOK(file.Name)
				success++
			}
			mu.Unlock()
		}(i, f)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println()
	ui.PrintResultSummary(success, failed, skipped, elapsed)
	if skipped > 0 {
		ui.PrintWarn(fmt.Sprintf("%d already in target format — skipped", skipped))
	}
	if backupEnabled {
		if success > 0 {
			ui.PrintOK(fmt.Sprintf("Backup saved to: %s", backupTarget))
		} else {
			// All files failed — remove empty backup dir
			os.RemoveAll(backupTarget)
			ui.PrintWarn("All files failed — backup directory removed")
		}
	}
	ui.Pause()
}

// ─── DIRECT MODE ─────────────────────────────────────────────────

func directMode(cfg Config) int {
	ffmpeg, _ := getFFmpeg()
	if ffmpeg == "" {
		return 1
	}

	files, _ := analyse.Directory(cfg.Input)
	if cfg.Format != "" {
		files = fileutil.FilterByFormat(files, cfg.Format)
	}
	files = fileutil.FilterByType(files, cfg.Filter)

	if len(files) == 0 {
		ui.PrintWarn("No files to process")
		return 0
	}

	fmt.Printf("  %d file(s)\n", len(files))

	var backupDir string
	if cfg.Backup {
		backupDir = backup.CreateDir(cfg.BackupDir, cfg.OutputDir)
		ui.PrintOK(fmt.Sprintf("Backup -> %s", backupDir))
	}

	// Create custom output directory if specified by user
	if cfg.OutputDir != "" {
		os.MkdirAll(cfg.OutputDir, 0755)
	}

	start := time.Now()
	var success, failed, skipped int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Parallel)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		<-stop
		fmt.Printf("\n")
		ui.PrintWarn("Interrupted. Finishing current files...")
	}()

	total := len(files)
	for i, f := range files {
		if cfg.Format != "" && strings.EqualFold(filepath.Ext(f.Name), "."+cfg.Format) {
			skipped++
			continue
		}

		wg.Add(1)
		go func(idx int, file fileutil.FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Output dir: for compress (same format) go to file's directory;
			// for convert (different format) go to current dir (or user-specified -o).
			outDir := cfg.OutputDir
			if outDir == "" {
				if cfg.Format == "" {
					outDir = filepath.Dir(file.Path)
				} else {
					outDir = "."
				}
			}

			if cfg.Backup {
				dst := filepath.Join(backupDir, file.Name)
				if err := backup.CopyFile(file.Path, dst); err != nil {
					ui.PrintWarn(fmt.Sprintf("Backup failed: %s", file.Name))
				}
			}

			err := processFile(ffmpeg, file, cfg, outDir)

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				if cfg.Backup {
					dst := filepath.Join(backupDir, file.Name)
					os.Remove(dst)
				}
				ui.PrintFail(fmt.Sprintf("%s — %v", file.Name, err))
				failed++
			} else {
				ui.PrintOK(file.Name)
				success++
			}
			mu.Unlock()
		}(i, f)
	}

	wg.Wait()
	signal.Stop(stop)

	elapsed := time.Since(start)
	fmt.Println()
	ui.PrintResultSummary(success, failed, skipped, elapsed)
	if skipped > 0 {
		ui.PrintWarn(fmt.Sprintf("%d already in target format", skipped))
	}
	if cfg.DryRun {
		ui.PrintWarn("Dry run — no files modified")
	} else if cfg.Backup && success > 0 {
		ui.PrintOK(fmt.Sprintf("Backup saved to: %s", backupDir))
	} else if cfg.Backup && success == 0 {
		os.RemoveAll(backupDir)
		ui.PrintWarn("All files failed — backup directory removed")
	}

	return 0
}

func processFile(ffmpeg string, file fileutil.FileInfo, cfg Config, outputDir string) error {
	if cfg.DryRun {
		return nil
	}

	outName := file.Name
	if cfg.Format != "" {
		outName = fileutil.FileNameWithoutExt(file.Name) + "." + cfg.Format
	}
	outPath := filepath.Join(outputDir, outName)

	samePath := strings.EqualFold(filepath.Clean(outPath), filepath.Clean(file.Path))

	var compressFn func(string, string, string, Config) error
	switch file.Type {
	case fileutil.TypeImage:
		compressFn = func(ffmpeg, src, dst string, c Config) error {
			return compress.Image(ffmpeg, src, dst, c.Quality, c.Format, c.Lossless)
		}
	case fileutil.TypeVideo:
		// Route video→audio conversions (audio extraction) to compress.Audio
		if fileutil.FormatToType(cfg.Format) == "audio" {
			compressFn = func(ffmpeg, src, dst string, c Config) error {
				return compress.Audio(ffmpeg, src, dst, c.Quality, c.Format, c.Lossless)
			}
		} else {
			compressFn = func(ffmpeg, src, dst string, c Config) error {
				return compress.Video(ffmpeg, src, dst, c.Quality, c.Format, c.Lossless)
			}
		}
	case fileutil.TypeAudio:
		compressFn = func(ffmpeg, src, dst string, c Config) error {
			return compress.Audio(ffmpeg, src, dst, c.Quality, c.Format, c.Lossless)
		}
	default:
		return fmt.Errorf("unsupported type")
	}

	if samePath {
		tmpPath := outPath + ".crush_tmp" + filepath.Ext(outPath)
		if err := compressFn(ffmpeg, file.Path, tmpPath, cfg); err != nil {
			os.Remove(tmpPath)
			return err
		}
		// Windows: os.Rename fails with "Access is denied" if target exists.
		// Remove it first, then rename.
		os.Remove(outPath)
		return os.Rename(tmpPath, outPath)
	}
	return compressFn(ffmpeg, file.Path, outPath, cfg)
}

// ─── ANALYSE ─────────────────────────────────────────────────────

func runAnalyse(dir string, jsonMode bool) {
	files, stats := analyse.Directory(dir)
	if jsonMode {
		out := struct {
			Dir       string `json:"dir"`
			Total     int    `json:"total"`
			TotalSize int64  `json:"total_size"`
			Images    int    `json:"images"`
			Videos    int    `json:"videos"`
			Audio     int    `json:"audio"`
		}{
			Dir: dir, Total: stats.Total, TotalSize: stats.TotalSize,
			Images: stats.Images, Videos: stats.Videos, Audio: stats.Audio,
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return
	}
	analyse.PrintAnalysis(stats)
	analyse.PrintFileList(files)
}

// ─── RANGE PARSER ────────────────────────────────────────────────

func parseRange(input string, max int) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}
	if strings.ToLower(input) == "all" {
		res := make([]int, max)
		for i := 0; i < max; i++ {
			res[i] = i + 1
		}
		return res, nil
	}

	seen := make(map[int]bool)
	var result []int

	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			s := strings.TrimSpace(bounds[0])
			e := strings.TrimSpace(bounds[1])
			start, err1 := strconv.Atoi(s)
			end, err2 := strconv.Atoi(e)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			if start > end {
				start, end = end, start
			}
			for i := start; i <= end; i++ {
				if i >= 1 && i <= max && !seen[i] {
					seen[i] = true
					result = append(result, i)
				}
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}
			if num >= 1 && num <= max && !seen[num] {
				seen[num] = true
				result = append(result, num)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid selections (1-%d)", max)
	}
	return result, nil
}

// ─── FFMPEG ──────────────────────────────────────────────────────

func getFFmpeg() (string, string) {
	ffmpeg, exeDir := fileutil.FindFFmpeg()
	if ffmpeg != "" {
		return ffmpeg, exeDir
	}

	fmt.Fprintf(os.Stderr, "%s%s%s\n", fileutil.Red, strings.Repeat("━", 45), fileutil.Reset)
	fmt.Fprintf(os.Stderr, "%s  FFMPEG NOT INSTALLED%s\n", fileutil.Bold+fileutil.Red, fileutil.Reset)
	fmt.Fprintf(os.Stderr, "%s%s%s\n", fileutil.Red, strings.Repeat("━", 45), fileutil.Reset)
	fmt.Fprintf(os.Stderr, "\nQuick fix:\n")
	fmt.Fprintf(os.Stderr, "  %scrush install%s\n\n", fileutil.Bold+fileutil.Cyan, fileutil.Reset)
	fmt.Fprintf(os.Stderr, "Or manually:\n")
	fmt.Fprintf(os.Stderr, "  %swinget install -e --id Gyan.FFmpeg%s   (Windows)\n", fileutil.Cyan, fileutil.Reset)
	fmt.Fprintf(os.Stderr, "  %sbrew install ffmpeg%s                  (macOS)\n", fileutil.Cyan, fileutil.Reset)
	fmt.Fprintf(os.Stderr, "  %ssudo apt install ffmpeg%s               (Linux)\n", fileutil.Cyan, fileutil.Reset)
	fmt.Fprintf(os.Stderr, "\nDownload and place in:\n")
	fmt.Fprintf(os.Stderr, "  %s%s\n", fileutil.Yellow, exeDir)
	fmt.Fprintf(os.Stderr, "%s%s%s\n", fileutil.Red, strings.Repeat("━", 45), fileutil.Reset)
	ui.Pause()
	return "", ""
}
