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
	flag.Parse()

	if cfg.Quality < 1 {
		cfg.Quality = 1
	}
	if cfg.Quality > 100 {
		cfg.Quality = 100
	}
	if cfg.Parallel < 1 {
		cfg.Parallel = 1
	}
	if cfg.Format != "" && cfg.Filter == "all" {
		cfg.Filter = fileutil.FormatToType(cfg.Format)
	}
	for _, a := range os.Args[1:] {
		if a == "--no-backup" {
			cfg.Backup = false
			break
		}
	}
	if flag.NArg() > 0 {
		cfg.Input = flag.Arg(0)
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

		ui.PrintInteractiveMenu()
		choice := ui.ReadInput("Choice [A/I/V/O/X/S/D/Q]: ")

		switch strings.ToUpper(choice) {
		case "A":
			pickMode(ffmpeg, fileutil.FilterByType(files, "all"), "all")
		case "I":
			pickMode(ffmpeg, fileutil.FilterByType(files, "image"), "image")
		case "V":
			pickMode(ffmpeg, fileutil.FilterByType(files, "video"), "video")
		case "O":
			pickMode(ffmpeg, fileutil.FilterByType(files, "audio"), "audio")
		case "X":
			pickExtract(ffmpeg, fileutil.FilterByType(files, "video"), "video")
		case "S":
			selectSpecific(ffmpeg, files)
		case "D":
			d := ui.ReadInput("Enter directory: ")
			if d != "" {
				dir = d
			}
		case "Q":
			fmt.Println("  Goodbye!")
			return 0
		default:
			ui.Beep()
			ui.Pause()
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
	fmt.Printf("  %s[C]%s Compress  — shrink file size, keep same format\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[F]%s Convert   — change to another format (e.g., jpg → webp)\n\n", fileutil.Bold, fileutil.Reset)

	mode := ui.ReadInput("Mode [C/F] (Enter = compress): ")

	switch strings.ToUpper(mode) {
	case "F":
		convertMode(ffmpeg, files, filter)
	default:
		compressMode(ffmpeg, files, filter)
	}
}

// ─── COMPRESS MODE ───────────────────────────────────────────────

func compressMode(ffmpeg string, files []fileutil.FileInfo, filter string) {
	ui.PrintSection("Compress Settings")
	ui.PrintQualityTable(filter)

	qStr := ui.ReadInput("  Quality 1-100 (Enter = lossless/best): ")
	lossless := qStr == ""
	quality := 85
	if qStr != "" {
		if q, err := strconv.Atoi(qStr); err == nil {
			quality = q
			if quality < 1 {
				quality = 1
			}
			if quality > 100 {
				quality = 100
			}
			lossless = false
		}
	}
	if lossless {
		ui.PrintOK("Lossless — original quality preserved")
	}

	bStr := ui.ReadInput("  Backup originals? (Y/n): ")
	backupEnabled := strings.ToLower(bStr) != "n"

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

	fmt.Printf("  %d file(s) selected\n\n", len(files))
	fmt.Printf("  %sTip:%s Type a format to convert to (e.g., webp, mp4, mp3)\n", fileutil.Yellow, fileutil.Reset)
	fmt.Printf("  %s     %s Examples: jpg→webp, mov→mp4, wav→flac, png→avif\n\n", fileutil.Yellow, fileutil.Reset)

	format := ui.ReadInput("  Target format: ")
	if format == "" {
		ui.PrintWarn("No format entered — switching to compress mode")
		compressMode(ffmpeg, files, filter)
		return
	}
	if !fileutil.IsValidTargetFormat(format) {
		ui.PrintFail(fmt.Sprintf("Unsupported format: %s", format))
		ui.Pause()
		return
	}

	ui.PrintQualityTable(filter)
	qStr := ui.ReadInput("  Quality 1-100 (Enter = lossless/conversion only): ")
	lossless := qStr == ""
	quality := 85
	if qStr != "" {
		if q, err := strconv.Atoi(qStr); err == nil {
			quality = q
			if quality < 1 {
				quality = 1
			}
			if quality > 100 {
				quality = 100
			}
			lossless = false
		}
	}
	if lossless {
		ui.PrintOK("Lossless conversion — no quality loss")
	}

	bStr := ui.ReadInput("  Backup originals? (Y/n): ")
	backupEnabled := strings.ToLower(bStr) != "n"

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

	ui.PrintSection("Extract Audio from Video")
	fmt.Printf("  %d video file(s) — audio will be extracted\n\n", len(files))
	fmt.Printf("  Common formats: mp3, flac, ogg, wav, m4a, opus\n\n")

	format := ui.ReadInput("  Audio format (Enter = mp3): ")
	if format == "" {
		format = "mp3"
	}
	if !fileutil.IsValidTargetFormat(format) {
		ui.PrintFail(fmt.Sprintf("Unsupported format: %s", format))
		ui.Pause()
		return
	}

	ui.PrintQualityTable("audio")
	qStr := ui.ReadInput("  Quality 1-100 (Enter = best): ")
	quality := 85
	if qStr != "" {
		if q, err := strconv.Atoi(qStr); err == nil {
			quality = q
			if quality < 1 {
				quality = 1
			}
			if quality > 100 {
				quality = 100
			}
		}
	}

	bStr := ui.ReadInput("  Backup originals? (Y/n): ")
	backupEnabled := strings.ToLower(bStr) != "n"

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

	runExtractAudio(ffmpeg, files, format, quality, backupEnabled, bDir, parallel)
}

func runExtractAudio(ffmpeg string, files []fileutil.FileInfo, format string, quality int, backupEnabled bool, backupDir string, parallel int) {
	fmt.Printf("\n")
	ui.PrintStep("Extracting audio...")

	var backupTarget string
	if backupEnabled {
		backupTarget = backup.CreateDir(backupDir, ".")
		ui.PrintOK(fmt.Sprintf("Originals backed up to: %s", backupTarget))
	}

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

			if backupEnabled {
				dst := filepath.Join(backupTarget, file.Name)
				if err := backup.CopyFile(file.Path, dst); err != nil {
					ui.PrintWarn(fmt.Sprintf("Backup failed: %s", file.Name))
				}
			}

			outName := fileutil.FileNameWithoutExt(file.Name) + "." + format
			outPath := filepath.Join(".", outName)
			err := compress.Audio(ffmpeg, file.Path, outPath, quality, format, false)

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				ui.PrintFail(file.Name)
				failed++
			} else {
				ui.PrintOK(fmt.Sprintf("%s → %s", file.Name, outName))
				success++
			}
			mu.Unlock()
		}(i, f)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println()
	ui.PrintResultSummary(success, failed, 0, elapsed)
	if backupEnabled {
		ui.PrintOK(fmt.Sprintf("Originals: %s", backupTarget))
	}
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
		ui.PrintOK(fmt.Sprintf("Originals backed up to: %s", backupTarget))
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
			err := processFile(ffmpeg, file, cfg, ".")

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				ui.PrintFail(file.Name)
				failed++
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
		ui.PrintOK(fmt.Sprintf("Originals: %s", backupTarget))
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

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	if outputDir != "." {
		os.MkdirAll(outputDir, 0755)
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

			if cfg.Backup {
				dst := filepath.Join(backupDir, file.Name)
				if err := backup.CopyFile(file.Path, dst); err != nil {
					ui.PrintWarn(fmt.Sprintf("Backup failed: %s", file.Name))
				}
			}

			err := processFile(ffmpeg, file, cfg, outputDir)

			mu.Lock()
			ui.PrintProgress(idx+1, total)
			if err != nil {
				ui.PrintFail(file.Name)
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

	switch file.Type {
	case fileutil.TypeImage:
		return compress.Image(ffmpeg, file.Path, outPath, cfg.Quality, cfg.Format, cfg.Lossless)
	case fileutil.TypeVideo:
		return compress.Video(ffmpeg, file.Path, outPath, cfg.Quality, cfg.Format, cfg.Lossless)
	case fileutil.TypeAudio:
		return compress.Audio(ffmpeg, file.Path, outPath, cfg.Quality, cfg.Format, cfg.Lossless)
	}
	return fmt.Errorf("unsupported type")
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
