package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Input     string
	OutputDir string
	Format    string
	Quality   int
	Backup    bool
	BackupDir string
	Parallel  int
	Filter    string
	DryRun    bool
	Verbose   bool
	Analyse   bool
	AnalyseJSON bool
}

type FileType int

const (
	TypeImage FileType = iota
	TypeVideo
	TypeAudio
	TypeUnknown
)

type FileInfo struct {
	Index    int
	Path     string
	Name     string
	Ext      string
	Size     int64
	SizeStr  string
	Type     FileType
	TypeName string
}

type AnalyseStats struct {
	Total     int
	TotalSize int64
	Images    int
	Videos    int
	Audio     int
	ImageSize int64
	VideoSize int64
	AudioSize int64
	Formats   map[string]int
	Dir       string
}

var (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"

	videoExts = map[string]bool{
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".mpg": true, ".mpeg": true, ".3gp": true, ".ts": true,
	}
	imageExts = map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
		".bmp": true, ".tiff": true, ".tif": true, ".avif": true,
		".gif": true, ".svg": true, ".ico": true, ".heic": true,
		".heif": true,
	}
	audioExts = map[string]bool{
		".mp3": true, ".wav": true, ".flac": true, ".ogg": true,
		".aac": true, ".wma": true, ".m4a": true, ".opus": true,
		".aiff": true, ".alac": true,
	}

	version   = "v2.0.0"
	typeNames = map[FileType]string{
		TypeImage: "Image", TypeVideo: "Video", TypeAudio: "Audio",
	}
	formatNames = map[string]string{
		".jpg": "JPEG", ".jpeg": "JPEG", ".png": "PNG", ".webp": "WebP",
		".avif": "AVIF", ".gif": "GIF", ".bmp": "BMP", ".svg": "SVG",
		".ico": "ICO", ".tiff": "TIFF", ".tif": "TIFF", ".heic": "HEIC",
		".heif": "HEIF",
		".mp4": "MP4", ".mov": "MOV", ".avi": "AVI", ".mkv": "MKV",
		".wmv": "WMV", ".flv": "FLV", ".webm": "WebM", ".m4v": "M4V",
		".mpg": "MPG", ".mpeg": "MPEG", ".3gp": "3GP", ".ts": "TS",
		".mp3": "MP3", ".wav": "WAV", ".flac": "FLAC", ".ogg": "OGG",
		".aac": "AAC", ".wma": "WMA", ".m4a": "M4A", ".opus": "Opus",
		".aiff": "AIFF", ".alac": "ALAC",
	}

	scan *bufio.Scanner
)

func main() {
	if len(os.Args) > 1 {
		arg := strings.ToLower(os.Args[1])
		switch arg {
		case "install":
			runInstall()
			return
		case "analyse", "analyze":
			jsonMode := false
			for _, a := range os.Args[2:] {
				if a == "--json" || a == "-j" {
					jsonMode = true
				}
			}
			runAnalyse(jsonMode)
			return
		}
	}

	cfg := parseFlags()

	ffmpeg, exeDir := findFFmpeg()
	if ffmpeg == "" {
		fmt.Fprintf(os.Stderr, "%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Red, Reset)
		fmt.Fprintf(os.Stderr, "%s  FFMPEG NOT INSTALLED%s\n", Bold+Red, Reset)
		fmt.Fprintf(os.Stderr, "%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Red, Reset)
		fmt.Fprintf(os.Stderr, "\nQuick fix:\n")
		fmt.Fprintf(os.Stderr, "  %scrush install%s\n\n", Bold+Cyan, Reset)
		fmt.Fprintf(os.Stderr, "Or install FFmpeg manually:\n")
		fmt.Fprintf(os.Stderr, "  %swinget install -e --id Gyan.FFmpeg%s   (Windows)\n", Cyan, Reset)
		fmt.Fprintf(os.Stderr, "  %sbrew install ffmpeg%s                  (macOS)\n", Cyan, Reset)
		fmt.Fprintf(os.Stderr, "  %ssudo apt install ffmpeg%s               (Linux)\n", Cyan, Reset)
		fmt.Fprintf(os.Stderr, "\nOr download ffmpeg.exe and place in:\n")
		fmt.Fprintf(os.Stderr, "  %s%s\n", Yellow, exeDir)
		fmt.Fprintf(os.Stderr, "%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", Red, Reset)
		pause()
		os.Exit(1)
	}

	if hasAnyFlags() {
		directMode(ffmpeg, cfg)
	} else {
		interactiveMode(ffmpeg, cfg)
	}
}

func hasAnyFlags() bool {
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "-") {
			return true
		}
		if a == "." || a == ".." {
			continue
		}
		if fi, err := os.Stat(a); err == nil {
			_ = fi
			return true
		}
		if strings.Contains(a, "*") || strings.Contains(a, "?") {
			return true
		}
	}
	return false
}

// ─── FLAGS ───────────────────────────────────────────────────────

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
	flag.StringVar(&cfg.Format, "f", "", "Target format: webp, mp4, mp3, ...")
	flag.StringVar(&cfg.Format, "format", "", "Target format: webp, mp4, mp3, ...")
	flag.IntVar(&cfg.Quality, "q", 85, "Quality 1-100")
	flag.IntVar(&cfg.Quality, "quality", 85, "Quality 1-100")
	flag.BoolVar(&cfg.Backup, "b", true, "Backup originals")
	flag.BoolVar(&cfg.Backup, "backup", true, "Backup originals")
	flag.StringVar(&cfg.BackupDir, "backup-dir", "", "Custom backup directory")
	flag.IntVar(&cfg.Parallel, "p", runtime.NumCPU(), "Parallel workers")
	flag.IntVar(&cfg.Parallel, "parallel", runtime.NumCPU(), "Parallel workers")
	flag.StringVar(&cfg.Filter, "t", "all", "Filter: image, video, audio, all")
	flag.StringVar(&cfg.Filter, "type", "all", "Filter: image, video, audio, all")
	flag.BoolVar(&cfg.DryRun, "n", false, "Dry run")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Dry run")
	flag.BoolVar(&cfg.Verbose, "v", false, "Verbose")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose")

	showHelp := flag.Bool("h", false, "Help")
	flag.BoolVar(showHelp, "help", false, "Help")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Usage = printUsage
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}
	if *showVersion {
		fmt.Printf("CRUSH %s\n", version)
		os.Exit(0)
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
	if cfg.Format != "" && cfg.Filter == "all" {
		cfg.Filter = formatToType(cfg.Format)
	}
	for _, a := range os.Args[1:] {
		if a == "--no-backup" {
			cfg.Backup = false
			break
		}
	}

	// Positional arg = input
	if flag.NArg() > 0 {
		cfg.Input = flag.Arg(0)
	}

	return cfg
}

func printUsage() {
	c := func(s string) string { return Cyan + s + Reset }
	g := func(s string) string { return Green + s + Reset }

	fmt.Printf("%sCRUSH %s — Lightning-fast media compressor%s\n\n", Bold, version, Reset)
	fmt.Printf("%sUSAGE%s\n", Bold, Reset)
	fmt.Printf("  crush                        Interactive mode (analyse + menu)\n")
	fmt.Printf("  crush [flags] [input]        Direct CLI mode\n")
	fmt.Printf("  crush install                Auto-install FFmpeg + setup PATH\n")
	fmt.Printf("  crush analyse                Analyse directory only\n")
	fmt.Printf("  crush analyse --json         Analyse as JSON\n\n")
	fmt.Printf("%sFLAGS%s\n", Bold, Reset)
	fmt.Printf("  -i, --input <path>    Input (%s)\n", c("default: ."))
	fmt.Printf("  -o, --output <dir>    Output directory (%s)\n", c("default: same as input"))
	fmt.Printf("  -f, --format <fmt>    Target format (%s)\n", c("webp, mp4, mp3, avif, webm..."))
	fmt.Printf("  -q, --quality <1-100> Quality (%s)\n", c("default: 85"))
	fmt.Printf("  -b, --backup          Backup originals (%s)\n", g("default: true"))
	fmt.Printf("  --no-backup           Skip backup\n")
	fmt.Printf("  -p, --parallel <n>    Workers (%s)\n", c(fmt.Sprintf("default: %d", runtime.NumCPU())))
	fmt.Printf("  -t, --type <type>     Filter: image, video, audio (%s)\n", c("default: all"))
	fmt.Printf("  -n, --dry-run         Preview only\n")
	fmt.Printf("  -v, --verbose         Show ffmpeg output\n\n")
	fmt.Printf("%sSUBPROGRAMS%s\n", Bold, Reset)
	fmt.Printf("  crush install                Auto-download ffmpeg + add to PATH\n")
	fmt.Printf("  crush analyse                Show directory analysis\n")
	fmt.Printf("  crush analyse --json         Machine-readable JSON output\n\n")
	fmt.Printf("%sEXAMPLES%s\n", Bold, Reset)
	fmt.Printf("  crush                          %s\n", c("# interactive mode"))
	fmt.Printf("  crush install                  %s\n", c("# auto-install ffmpeg"))
	fmt.Printf("  crush -f webp -q 90            %s\n", c("# all images -> WebP"))
	fmt.Printf("  crush video.mp4 -q 80          %s\n", c("# single video"))
	fmt.Printf("  crush . -t audio -f mp3        %s\n", c("# all audio -> MP3"))
	fmt.Println()
}

// ─── INTERACTIVE MODE ────────────────────────────────────────────

func interactiveMode(ffmpeg string, cfg Config) {
	scan = bufio.NewScanner(os.Stdin)
	dir := cfg.Input

	for {
		clearScreen()
		printBanner()

		files, stats := analyseDirectory(dir)
		printAnalysis(stats, len(files) <= 30)
		printFileList(files)

		if len(files) == 0 {
			fmt.Printf("%sNo media files found in: %s%s\n", Yellow, dir, Reset)
			fmt.Print("\nEnter directory to analyse (or 'q' to quit): ")
			scan.Scan()
			d := strings.TrimSpace(scan.Text())
			if d == "q" || d == "Q" {
				return
			}
			if d != "" {
				dir = d
			}
			continue
		}

		fmt.Println()
		fmt.Printf("  %s[A]%s Convert ALL files\n", Bold, Reset)
		fmt.Printf("  %s[I]%s Convert Images only\n", Bold, Reset)
		fmt.Printf("  %s[V]%s Convert Videos only\n", Bold, Reset)
		fmt.Printf("  %s[O]%s Convert Audio only\n", Bold, Reset)
		fmt.Printf("  %s[S]%s Select specific files by number\n", Bold, Reset)
		fmt.Printf("  %s[D]%s Change directory\n", Bold, Reset)
		fmt.Printf("  %s[Q]%s Quit\n", Bold, Reset)

		choice := readInput("Choice [A/I/V/O/S/D/Q]: ")

		switch strings.ToUpper(choice) {
		case "A":
			filtered := filterByType(files, "all")
			processInteractive(ffmpeg, filtered, "all")
		case "I":
			filtered := filterByType(files, "image")
			processInteractive(ffmpeg, filtered, "image")
		case "V":
			filtered := filterByType(files, "video")
			processInteractive(ffmpeg, filtered, "video")
		case "O":
			filtered := filterByType(files, "audio")
			processInteractive(ffmpeg, filtered, "audio")
		case "S":
			selectSpecific(ffmpeg, files)
		case "D":
			fmt.Print("Enter directory: ")
			scan.Scan()
			d := strings.TrimSpace(scan.Text())
			if d != "" {
				dir = d
			}
		case "Q":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			beep()
			pause()
		}
	}
}

func printBanner() {
	fmt.Printf(`%s╔══════════════════════════════════════════════╗
║     CRUSH %-15s               ║
║     Interactive Media Compressor            ║
╚══════════════════════════════════════════════╝%s
`, Bold, version, Reset)
}

func processInteractive(ffmpeg string, files []FileInfo, filter string) {
	if len(files) == 0 {
		fmt.Printf("%sNo files to process%s\n", Yellow, Reset)
		pause()
		return
	}

	fmt.Printf("\n%d file(s) selected\n", len(files))

	fmt.Print("Target format (or press Enter to keep original): ")
	scan.Scan()
	format := strings.TrimSpace(scan.Text())

	if format != "" && !isValidTargetFormat(format) {
		fmt.Printf("%sUnsupported format: %s%s\n", Red, format, Reset)
		pause()
		return
	}

	qStr := readInput(fmt.Sprintf("Quality [1-100, default %d]: ", 85))
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

	bStr := readInput("Backup originals? [Y/n]: ")
	backup := strings.ToLower(bStr) != "n"

	bDir := ""
	if backup {
		bDirStr := readInput("Backup directory [default: ./backup/]: ")
		if bDirStr != "" {
			bDir = bDirStr
		}
	}

	pStr := readInput(fmt.Sprintf("Parallel workers [default: %d]: ", runtime.NumCPU()))
	parallel := runtime.NumCPU()
	if pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallel = p
		}
	}

	runInteractiveProcess(ffmpeg, files, format, quality, backup, bDir, parallel)
}

func selectSpecific(ffmpeg string, files []FileInfo) {
	if len(files) == 0 {
		fmt.Printf("%sNo files available%s\n", Yellow, Reset)
		pause()
		return
	}

	fmt.Printf("\nAvailable files: 1-%d\n", len(files))
	fmt.Print("Enter selection (e.g. 1-4,7,9-11 or 'all'): ")
	scan.Scan()
	input := strings.TrimSpace(scan.Text())

	selected, err := parseRange(input, len(files))
	if err != nil {
		fmt.Printf("%sInvalid selection: %s%s\n", Red, err, Reset)
		pause()
		return
	}

	var selectedFiles []FileInfo
	for _, idx := range selected {
		selectedFiles = append(selectedFiles, files[idx-1])
	}

	fmt.Printf("\n%d file(s) selected\n", len(selectedFiles))
	processInteractive(ffmpeg, selectedFiles, "custom")
}

func runInteractiveProcess(ffmpeg string, files []FileInfo, format string, quality int, backup bool, backupDir string, parallel int) {
	var backupTarget string
	if backup {
		backupTarget = createBackupDir(backupDir, ".")
		fmt.Printf("\n%sBackup -> %s%s\n", Yellow, backupTarget, Reset)
	}

	var success, failed, skipped int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	start := time.Now()

	for _, f := range files {
		if format != "" && strings.EqualFold(filepath.Ext(f.Name), "."+format) {
			skipped++
			continue
		}

		wg.Add(1)
		go func(file FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if backup {
				dst := filepath.Join(backupTarget, file.Name)
				os.MkdirAll(filepath.Dir(dst), 0755)
				if err := copyFile(file.Path, dst); err != nil {
					fmt.Printf("  %s⚠ Backup failed: %s%s\n", Yellow, file.Name, Reset)
				}
			}

			cfg := Config{Format: format, Quality: quality, Verbose: false}
			err := processFile(ffmpeg, file, cfg, ".")
			mu.Lock()
			if err != nil {
				fmt.Printf("  %s✗ %s: %s%s\n", Red, file.Name, err, Reset)
				failed++
			} else {
				fmt.Printf("  %s✓ %s%s\n", Green, file.Name, Reset)
				success++
			}
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("\n%s━━━━━━━━━━━━━━━━━━━━━%s\n", Cyan, Reset)
	fmt.Printf("%s%d OK  |  %d FAIL  |  %d SKIP  |  ⏱ %s%s\n", Bold, success, failed, skipped, elapsed, Reset)
	if skipped > 0 {
		fmt.Printf("%s(%d already in target format — skipped)%s\n", Yellow, skipped, Reset)
	}
	if backup {
		fmt.Printf("%sOriginals: %s%s\n", Yellow, backupTarget, Reset)
	}
	pause()
}

// ─── ANALYSE ─────────────────────────────────────────────────────

func runAnalyse(jsonMode bool) {
	dir := "."
	if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
		dir = os.Args[2]
	}
	if len(os.Args) > 3 && !strings.HasPrefix(os.Args[3], "-") {
		dir = os.Args[3]
	}

	files, stats := analyseDirectory(dir)
	if jsonMode {
		b, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(b))
		return
	}

	printAnalysis(stats, true)
	printFileList(files)
}

func analyseDirectory(dir string) ([]FileInfo, *AnalyseStats) {
	stats := &AnalyseStats{Dir: dir, Formats: make(map[string]int)}
	var files []FileInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		ft := detectType(ext)
		if ft == TypeUnknown {
			return nil
		}

		stats.Total++
		stats.TotalSize += info.Size()
		stats.Formats[ext]++

		switch ft {
		case TypeImage:
			stats.Images++
			stats.ImageSize += info.Size()
		case TypeVideo:
			stats.Videos++
			stats.VideoSize += info.Size()
		case TypeAudio:
			stats.Audio++
			stats.AudioSize += info.Size()
		}

		files = append(files, FileInfo{
			Path:     path,
			Name:     info.Name(),
			Ext:      ext,
			Size:     info.Size(),
			SizeStr:  formatSize(info.Size()),
			Type:     ft,
			TypeName: formatDisplayName(ext),
		})
		return nil
	})
	if err != nil {
		fmt.Printf("%sError reading directory: %s%s\n", Red, err, Reset)
	}

	// Sort files
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].Name > files[j].Name {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	for i := range files {
		files[i].Index = i + 1
	}

	return files, stats
}

func printAnalysis(stats *AnalyseStats, showFileList bool) {
	fmt.Printf("\n%sDirectory:%s %s\n", Bold, Reset, stats.Dir)

	bar := func(label string, count int, total int, size int64, totalSize int64, color string) {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		barW := 25
		filled := int(pct * float64(barW) / 100)
		if filled > barW {
			filled = barW
		}
		barStr := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		sizeStr := formatSize(size)
		fmt.Printf("  %s%s %s%s  %3d (%.0f%%)  %s\n",
			color, label, barStr, Reset, count, pct, sizeStr)
	}

	maxFiles := stats.Images
	if stats.Videos > maxFiles {
		maxFiles = stats.Videos
	}
	if stats.Audio > maxFiles {
		maxFiles = stats.Audio
	}
	total := maxFiles
	if total == 0 {
		total = 1
	}

	fmt.Println()
	bar("Images", stats.Images, total, stats.ImageSize, stats.TotalSize, Green)
	bar("Videos", stats.Videos, total, stats.VideoSize, stats.TotalSize, Cyan)
	bar("Audio ", stats.Audio, total, stats.AudioSize, stats.TotalSize, Yellow)

	var fmtCounts []string
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif", ".svg", ".mp4", ".mov", ".webm", ".mkv", ".avi", ".mp3", ".wav", ".flac", ".ogg", ".m4a", ".aac"} {
		if c := stats.Formats[ext]; c > 0 {
			fmtCounts = append(fmtCounts, fmt.Sprintf("%s%s x%d%s", formatColor(ext), ext[1:], c, Reset))
		}
	}

	fmt.Printf("\n  %sTotal:%s %d files  |  %s\n", Bold, Reset, stats.Total, formatSize(stats.TotalSize))
	if len(fmtCounts) > 0 {
		fmt.Printf("  %sBreakdown:%s %s\n", Bold, Reset, strings.Join(fmtCounts, "  "))
	}
	fmt.Println()
}

func printFileList(files []FileInfo) {
	if len(files) == 0 {
		return
	}
	if len(files) > 100 {
		fmt.Printf("(%d files — use 'Select specific files' to see numbered list)\n", len(files))
		return
	}

	header := fmt.Sprintf("%s  %-4s %-8s %-10s %-6s  %s%s", Bold, "#", "Type", "Size", "Format", "Filename", Reset)
	fmt.Println(header)
	fmt.Printf("  %s%s%s\n", Dim, strings.Repeat("─", len(header)-10), Reset)

	for _, f := range files {
		idx := fmt.Sprintf("%d", f.Index)
		col := ""
		switch f.Type {
		case TypeImage:
			col = Green
		case TypeVideo:
			col = Cyan
		case TypeAudio:
			col = Yellow
		}
		fmt.Printf("  %-4s %s%-8s%s %-10s %-6s  %s\n",
			idx, col, f.TypeName, Reset, f.SizeStr, strings.ToUpper(f.Ext[1:]), f.Name)
	}
	fmt.Printf("  %s%s%s\n", Dim, strings.Repeat("─", len(header)-10), Reset)
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.0f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/1024/1024/1024)
}

func formatDisplayName(ext string) string {
	if n, ok := formatNames[ext]; ok {
		return n
	}
	return strings.ToUpper(ext[1:])
}

func formatColor(ext string) string {
	if imageExts[ext] {
		return Green
	}
	if videoExts[ext] {
		return Cyan
	}
	if audioExts[ext] {
		return Yellow
	}
	return Reset
}

func isValidTargetFormat(format string) bool {
	f := strings.ToLower(format)
	valid := map[string]bool{
		"webp": true, "avif": true, "jpg": true, "jpeg": true,
		"png": true, "gif": true, "bmp": true,
		"mp4": true, "webm": true, "avi": true, "mov": true, "mkv": true,
		"mp3": true, "ogg": true, "wav": true, "flac": true,
		"aac": true, "opus": true, "m4a": true,
	}
	return valid[f]
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

	parts := strings.Split(input, ",")
	for _, part := range parts {
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

// ─── DIRECT MODE ─────────────────────────────────────────────────

func directMode(ffmpeg string, cfg Config) {
	if cfg.Verbose {
		fmt.Printf("%sCRUSH %s%s\n", Bold, version, Reset)
	}

	files, _ := analyseDirectory(cfg.Input)
	if cfg.Format != "" {
		files = filterByFormat(files, cfg.Format)
	}
	files = filterByType(files, cfg.Filter)

	if len(files) == 0 {
		fmt.Printf("%sNo files to process%s\n", Yellow, Reset)
		os.Exit(0)
	}

	fmt.Printf("%s%d file(s)%s\n", Cyan, len(files), Reset)

	var backupDir string
	if cfg.Backup {
		backupDir = createBackupDir(cfg.BackupDir, cfg.OutputDir)
		fmt.Printf("%sBackup -> %s%s\n", Yellow, backupDir, Reset)
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for _, f := range files {
		if cfg.Format != "" && strings.EqualFold(filepath.Ext(f.Name), "."+cfg.Format) {
			skipped++
			continue
		}

		wg.Add(1)
		go func(file FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if cfg.Backup {
				dst := filepath.Join(backupDir, file.Name)
				os.MkdirAll(filepath.Dir(dst), 0755)
				copyFile(file.Path, dst)
			}

			pCfg := Config{
				Format: cfg.Format, Quality: cfg.Quality,
				OutputDir: outputDir, Verbose: cfg.Verbose, DryRun: cfg.DryRun,
			}
			err := processFile(ffmpeg, file, pCfg, outputDir)

			mu.Lock()
			if err != nil {
				fmt.Printf("%s  ✗ %s: %s%s\n", Red, file.Name, err, Reset)
				failed++
			} else {
				fmt.Printf("%s  ✓ %s%s\n", Green, file.Name, Reset)
				success++
			}
			mu.Unlock()
		}(f)
	}

	go func() {
		<-c
		fmt.Printf("\n%sInterrupted. Finishing current files...%s\n", Yellow, Reset)
	}()

	wg.Wait()

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("\n%s━━━━━━━━━%s\n", Cyan, Reset)
	fmt.Printf("%s%d OK  |  %d FAIL  |  %d SKIP  |  ⏱ %s%s\n",
		Bold, success, failed, skipped, elapsed, Reset)
	if skipped > 0 {
		fmt.Printf("%s(%d already in target format)%s\n", Yellow, skipped, Reset)
	}
	if cfg.DryRun {
		fmt.Printf("%sDry run — no files modified%s\n", Yellow, Reset)
	}
}

// ─── FILE RESOLUTION ─────────────────────────────────────────────

func resolveFiles(input, filter, format string) []string {
	var paths []string

	info, err := os.Stat(input)
	if err == nil && info.IsDir() {
		filepath.Walk(input, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
	} else {
		matches, globErr := filepath.Glob(input)
		if globErr == nil && len(matches) > 0 {
			for _, m := range matches {
				if fi, e := os.Stat(m); e == nil && !fi.IsDir() {
					paths = append(paths, m)
				}
			}
		}
	}

	if len(paths) == 0 {
		if fi, e := os.Stat(input); e == nil && !fi.IsDir() {
			paths = append(paths, input)
		}
	}

	sortStrings(paths)

	var filtered []string
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		ft := detectType(ext)
		if ft == TypeUnknown {
			continue
		}
		if !matchFilter(ft, ext, filter, format) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func detectType(ext string) FileType {
	if videoExts[ext] {
		return TypeVideo
	}
	if imageExts[ext] {
		return TypeImage
	}
	if audioExts[ext] {
		return TypeAudio
	}
	return TypeUnknown
}

func matchFilter(ft FileType, ext, filter, format string) bool {
	if format != "" {
		return canConvertTo(ft, format)
	}
	switch filter {
	case "image":
		return ft == TypeImage
	case "video":
		return ft == TypeVideo
	case "audio":
		return ft == TypeAudio
	case "all":
		return true
	}
	return true
}

func canConvertTo(ft FileType, format string) bool {
	f := strings.ToLower(format)
	switch ft {
	case TypeImage:
		switch f {
		case "webp", "avif", "jpg", "jpeg", "png", "gif", "bmp", "tiff":
			return true
		}
	case TypeVideo:
		switch f {
		case "mp4", "webm", "avi", "mov", "mkv", "gif":
			return true
		}
	case TypeAudio:
		switch f {
		case "mp3", "ogg", "wav", "flac", "aac", "opus", "m4a":
			return true
		}
	}
	return false
}

func formatToType(format string) string {
	f := strings.ToLower(format)
	switch f {
	case "webp", "avif", "jpg", "jpeg", "png", "gif", "bmp", "tiff":
		return "image"
	case "mp4", "webm", "avi", "mov", "mkv":
		return "video"
	case "mp3", "ogg", "wav", "flac", "aac", "opus", "m4a":
		return "audio"
	}
	return "all"
}

func filterByType(files []FileInfo, filter string) []FileInfo {
	if filter == "all" || filter == "custom" {
		return files
	}
	var fType FileType
	switch filter {
	case "image":
		fType = TypeImage
	case "video":
		fType = TypeVideo
	case "audio":
		fType = TypeAudio
	default:
		return files
	}
	var result []FileInfo
	for _, f := range files {
		if f.Type == fType {
			result = append(result, f)
		}
	}
	return result
}

func filterByFormat(files []FileInfo, format string) []FileInfo {
	var result []FileInfo
	for _, f := range files {
		if canConvertTo(f.Type, format) {
			result = append(result, f)
		}
	}
	return result
}

// ─── FILE PROCESSING ─────────────────────────────────────────────

func processFile(ffmpeg string, file FileInfo, cfg Config, outputDir string) error {
	if cfg.DryRun {
		return nil
	}

	outName := file.Name
	if cfg.Format != "" {
		outName = fileNameWithoutExt(file.Name) + "." + cfg.Format
	}
	outPath := filepath.Join(outputDir, outName)

	switch file.Type {
	case TypeImage:
		return compressImage(ffmpeg, file.Path, outPath, cfg)
	case TypeVideo:
		return compressVideo(ffmpeg, file.Path, outPath, cfg)
	case TypeAudio:
		return compressAudio(ffmpeg, file.Path, outPath, cfg)
	}
	return fmt.Errorf("unsupported type")
}

func compressImage(ffmpeg, input, output string, cfg Config) error {
	target := strings.ToLower(filepath.Ext(output))
	target = strings.TrimPrefix(target, ".")

	var args []string
	args = append(args, "-i", input)

	switch target {
	case "webp":
		args = append(args, "-c:v", "libwebp")
		args = append(args, "-quality", strconv.Itoa(cfg.Quality))
		args = append(args, "-compression_level", "4")
	case "avif":
		args = append(args, "-c:v", "libaom-av1")
		crf := 20 + (100-cfg.Quality)*43/100
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-b:v", "0")
		args = append(args, "-strict", "experimental")
	case "png":
		args = append(args, "-compression_level", "9")
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		q := mapQualityToQ(cfg.Quality)
		args = append(args, "-q:v", strconv.Itoa(q))
	}

	args = append(args, "-y", output)
	return runFFmpeg(ffmpeg, args)
}

func compressVideo(ffmpeg, input, output string, cfg Config) error {
	target := strings.ToLower(filepath.Ext(output))
	target = strings.TrimPrefix(target, ".")
	if target == "" {
		target = "mp4"
	}

	crf := 18 + (100-cfg.Quality)*22/100
	var args []string
	args = append(args, "-i", input)

	switch target {
	case "mp4", "m4v":
		args = append(args, "-c:v", "libx264")
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-preset", "fast")
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", "128k")
		args = append(args, "-movflags", "+faststart")
	case "webm":
		args = append(args, "-c:v", "libvpx-vp9")
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-b:v", "0")
		args = append(args, "-c:a", "libopus")
	case "avi":
		args = append(args, "-c:v", "libx264")
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-preset", "fast")
	case "mov":
		args = append(args, "-c:v", "libx264")
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-preset", "fast")
		args = append(args, "-c:a", "aac")
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		args = append(args, "-c:v", "libx264")
		args = append(args, "-crf", strconv.Itoa(crf))
		args = append(args, "-preset", "fast")
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", "128k")
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, "-y", output)
	return runFFmpeg(ffmpeg, args)
}

func compressAudio(ffmpeg, input, output string, cfg Config) error {
	target := strings.ToLower(filepath.Ext(output))
	target = strings.TrimPrefix(target, ".")
	if target == "" {
		target = "mp3"
	}

	var args []string
	args = append(args, "-i", input)

	switch target {
	case "mp3":
		args = append(args, "-c:a", "libmp3lame")
		q := 0 + (100-cfg.Quality)*9/100
		args = append(args, "-q:a", strconv.Itoa(q))
	case "ogg":
		args = append(args, "-c:a", "libvorbis")
		q := 0 + (100-cfg.Quality)*10/100
		args = append(args, "-q:a", strconv.Itoa(q))
	case "flac":
		args = append(args, "-c:a", "flac")
		args = append(args, "-compression_level", "8")
	case "aac", "m4a":
		args = append(args, "-c:a", "aac")
		bitrate := 64 + (cfg.Quality * 256 / 100)
		args = append(args, "-b:a", strconv.Itoa(bitrate)+"k")
	case "opus":
		args = append(args, "-c:a", "libopus")
		bitrate := 32 + (cfg.Quality * 128 / 100)
		args = append(args, "-b:a", strconv.Itoa(bitrate)+"k")
	case "wav":
	default:
		args = append(args, "-c:a", "libmp3lame")
		q := 0 + (100-cfg.Quality)*9/100
		args = append(args, "-q:a", strconv.Itoa(q))
	}

	args = append(args, "-y", output)
	return runFFmpeg(ffmpeg, args)
}

func runFFmpeg(ffmpeg string, args []string) error {
	cmd := exec.Command(ffmpeg, args...)
	cmd.Stderr = nil
	cmd.Stdout = nil
	return cmd.Run()
}

func mapQualityToQ(quality int) int {
	q := 1 + ((100-quality)*30)/100
	if q < 1 {
		return 1
	}
	if q > 31 {
		return 31
	}
	return q
}

func fileNameWithoutExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

// ─── FFMPEG LOCATION ─────────────────────────────────────────────

func findFFmpeg() (string, string) {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	local := filepath.Join(exeDir, "ffmpeg.exe")
	if fileExists(local) {
		return local, exeDir
	}
	localBin := filepath.Join(exeDir, "bin", "ffmpeg.exe")
	if fileExists(localBin) {
		return localBin, exeDir
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, exeDir
	}
	return "", exeDir
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ─── BACKUP ──────────────────────────────────────────────────────

func createBackupDir(customDir, outputDir string) string {
	ts := time.Now().Format("2006-01-02_150405")
	base := "backup"
	if customDir != "" {
		base = customDir
	}
	dir := filepath.Join(base, "originals_"+ts)
	os.MkdirAll(dir, 0755)
	return dir
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ─── UTILITIES ───────────────────────────────────────────────────

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	scan.Scan()
	return strings.TrimSpace(scan.Text())
}

func pause() {
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

func beep() {
	fmt.Print("\a")
}

// ─── INSTALL ─────────────────────────────────────────────────────

func runInstall() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	clearScreen()
	fmt.Printf("%s╔═══════════════════════════════════╗%s\n", Bold+Cyan, Reset)
	fmt.Printf("%s║     CRUSH Installer               ║%s\n", Bold+Cyan, Reset)
	fmt.Printf("%s╚═══════════════════════════════════╝%s\n\n", Bold+Cyan, Reset)

	ffmpegPath, _ := findFFmpeg()
	if ffmpegPath != "" {
		fmt.Printf("%s✓ FFmpeg already installed at:%s %s\n", Green, Reset, ffmpegPath)
		ensureCrushOnPATH(exeDir)
		fmt.Printf("\n%sReady! Run 'crush' from any terminal.%s\n", Green, Reset)
		pause()
		return
	}

	fmt.Printf("%sFFmpeg is required but not found.%s\n\n", Yellow, Reset)

	// Try winget on Windows first
	wingetOK := false
	if runtime.GOOS == "windows" {
		fmt.Printf("Try: %swinget install -e --id Gyan.FFmpeg%s\n", Bold, Reset)
		fmt.Print("Run this now? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "n" {
			fmt.Println("\nRunning winget...")
			cmd := exec.Command("winget", "install", "-e", "--id", "Gyan.FFmpeg")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err == nil {
				wingetOK = true
				fmt.Printf("\n%s✓ FFmpeg installed via winget%s\n", Green, Reset)
			} else {
				fmt.Printf("\n%s⚠ winget failed (not available or cancelled)%s\n", Yellow, Reset)
			}
		}
	}

	// Try brew on macOS
	if runtime.GOOS == "darwin" {
		fmt.Print("Try: brew install ffmpeg\n")
		fmt.Print("Run this now? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "n" {
			fmt.Println("\nRunning brew...")
			cmd := exec.Command("brew", "install", "ffmpeg")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err == nil {
				wingetOK = true
				fmt.Printf("\n%s✓ FFmpeg installed via Homebrew%s\n", Green, Reset)
			} else {
				fmt.Printf("\n%s⚠ brew failed%s\n", Yellow, Reset)
			}
		}
	}

	// Try apt on Linux
	if runtime.GOOS == "linux" {
		fmt.Print("Try: sudo apt install ffmpeg\n")
		fmt.Print("Run this now? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "n" {
			fmt.Println("\nRunning apt...")
			cmd := exec.Command("sudo", "apt", "install", "-y", "ffmpeg")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err == nil {
				wingetOK = true
				fmt.Printf("\n%s✓ FFmpeg installed via apt%s\n", Green, Reset)
			} else {
				fmt.Printf("\n%s⚠ apt failed%s\n", Yellow, Reset)
			}
		}
	}

	// If package manager failed, offer download
	if !wingetOK {
		ffmpegPath, _ = findFFmpeg()
		if ffmpegPath != "" {
			wingetOK = true
		} else {
			fmt.Println("\nInstall manually:")
			if runtime.GOOS == "windows" {
				fmt.Printf("  %s1.%s Download ffmpeg.exe:\n", Bold, Reset)
				fmt.Printf("     %shttps://www.gyan.dev/ffmpeg/builds/%s\n", Cyan, Reset)
				fmt.Printf("  %s2.%s Place it next to crush.exe in: %s\n", Bold, Reset, exeDir)
			} else {
				fmt.Println("  Install ffmpeg via your package manager:")
				fmt.Printf("  %sbrew install ffmpeg%s    (macOS)\n", Cyan, Reset)
				fmt.Printf("  %ssudo apt install ffmpeg%s (Linux)\n", Cyan, Reset)
			}
			fmt.Printf("  %s3.%s Run: crush install\n", Bold, Reset)
		}
	}

	ffmpegPath, _ = findFFmpeg()
	if ffmpegPath == "" {
		fmt.Printf("\n%s⚠ FFmpeg not found after install attempt.%s\n", Red, Reset)
		fmt.Println("  Install ffmpeg manually, then run: crush install")
		pause()
		return
	}

	ensureCrushOnPATH(exeDir)
	fmt.Printf("\n%s╔═══════════════════════════════════╗%s\n", Bold+Green, Reset)
	fmt.Printf("%s║  ✓ INSTALL COMPLETE               ║%s\n", Bold+Green, Reset)
	fmt.Printf("%s║  FFmpeg ready                     ║%s\n", Bold+Green, Reset)
	fmt.Printf("%s║  Run 'crush' from any terminal    ║%s\n", Bold+Green, Reset)
	fmt.Printf("%s╚═══════════════════════════════════╝%s\n", Bold+Green, Reset)

	pause()
}

func addToPATH(dir string) {
	if runtime.GOOS == "windows" {
		ps := fmt.Sprintf(
			`[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';%s', 'User')`,
			dir,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		cmd.Run()
	} else {
		home, _ := os.UserHomeDir()
		rcFile := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
			rcFile = filepath.Join(home, ".zshrc")
		}
		line := fmt.Sprintf("\nexport PATH=\"%s:$PATH\"\n", dir)
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString(line)
		}
	}
	fmt.Printf("  Added: %s\n", dir)
}

func ensureCrushOnPATH(exeDir string) {
	if isInPATH(exeDir) {
		return
	}
	addToPATH(exeDir)
}

func isInPATH(dir string) bool {
	path := os.Getenv("PATH")
	separator := ";"
	if runtime.GOOS != "windows" {
		separator = ":"
	}
	for _, p := range strings.Split(path, separator) {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return true
		}
	}
	return false
}
