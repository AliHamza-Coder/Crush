package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
)

var scan = bufio.NewScanner(os.Stdin)

func PrintDeveloperCredit() {
	fmt.Printf("\n  %s✦ Developed by %sAli Hamza Coder%s ✦%s\n",
		fileutil.Dim, fileutil.Cyan, fileutil.Dim, fileutil.Reset)
}

func PrintBanner() {
	ClearScreen()

	lines := []string{
		"",
		fmt.Sprintf("  %s╔══════════════════════════════════════════════╗%s", fileutil.Cyan, fileutil.Reset),
		fmt.Sprintf("  %s║                                             ║%s", fileutil.Cyan, fileutil.Reset),
		fmt.Sprintf("  %s║     %s✦ CRUSH %s%s%s%s%s", fileutil.Cyan, fileutil.Bold, fileutil.Reset, fileutil.Green, fileutil.Version, fileutil.Reset, strings.Repeat(" ", 32-len(fileutil.Version))),
		fmt.Sprintf("  %s║     %sMedia Compressor%s                     ║%s", fileutil.Cyan, fileutil.Dim, fileutil.Reset, fileutil.Reset),
		fmt.Sprintf("  %s║                                             ║%s", fileutil.Cyan, fileutil.Reset),
		fmt.Sprintf("  %s╚══════════════════════════════════════════════╝%s", fileutil.Cyan, fileutil.Reset),
	}

	for _, line := range lines {
		fmt.Println(line)
		time.Sleep(25 * time.Millisecond)
	}

	PrintDeveloperCredit()
	fmt.Println()
}

func PrintSection(title string) {
	fmt.Printf("\n  %s%s%s\n", fileutil.Bold, title, fileutil.Reset)
	fmt.Printf("  %s%s%s\n", fileutil.Dim, strings.Repeat("─", len(title)), fileutil.Reset)
}

func PrintInteractiveMenu() {
	fmt.Printf("\n  %s[A]%s ALL files  — images + videos + audio\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[I]%s Images    — jpg, png, webp, avif, gif...\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[V]%s Videos    — mp4, mov, webm, avi, mkv...\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[O]%s Audio     — mp3, wav, flac, ogg, aac...\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[X]%s Extract audio from video  — e.g., mp4 → mp3\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[S]%s Select specific files by number\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[D]%s Change directory\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[Q]%s Quit\n", fileutil.Bold, fileutil.Reset)
}

func PrintQualityTable(filter string) {
	fmt.Printf("\n")
	fmt.Printf("  %sRecommended quality values:%s\n", fileutil.Bold, fileutil.Reset)
	switch filter {
	case "image", "all":
		fmt.Printf("    %s85%s → balanced (good quality, ~50-70%% smaller)  %s★ recommended%s\n", fileutil.Green, fileutil.Reset, fileutil.Dim, fileutil.Reset)
		fmt.Printf("    %s75%s → smaller file, slightly lower quality\n", fileutil.Yellow, fileutil.Reset)
		fmt.Printf("    %s100%s → lossless / best quality (largest file)\n", fileutil.Cyan, fileutil.Reset)
	case "video":
		fmt.Printf("    %s85%s → CRF 23 (balanced, ~50%% smaller)  %s★ recommended%s\n", fileutil.Green, fileutil.Reset, fileutil.Dim, fileutil.Reset)
		fmt.Printf("    %s70%s → CRF 28 (smaller, some quality loss)\n", fileutil.Yellow, fileutil.Reset)
		fmt.Printf("    %s100%s → CRF 18 (near-lossless, larger)\n", fileutil.Cyan, fileutil.Reset)
	case "audio":
		fmt.Printf("    %s85%s → VBR ~192kbps (excellent quality)  %s★ recommended%s\n", fileutil.Green, fileutil.Reset, fileutil.Dim, fileutil.Reset)
		fmt.Printf("    %s60%s → VBR ~128kbps (smaller, good for podcasts)\n", fileutil.Yellow, fileutil.Reset)
		fmt.Printf("    %s100%s → maximum quality (largest file)\n", fileutil.Cyan, fileutil.Reset)
	}
	fmt.Printf("\n")
}

var FormatChoices = map[string][]string{
	"image": {"webp", "avif", "jpg", "png", "gif"},
	"video": {"mp4", "webm", "mov", "avi", "mkv", "gif"},
	"audio": {"mp3", "flac", "ogg", "wav", "aac", "opus", "m4a", "alac"},
}

func PrintFormatMenu(filter string) string {
	formats, ok := FormatChoices[filter]
	if !ok {
		formats = FormatChoices["image"]
	}
	return SelectFromList(formats, "Target format (↑↓ to choose, Enter to confirm):")
}

func PrintResultSummary(success, failed, skipped int64, elapsed time.Duration) {
	fmt.Printf("\n  %s%s%s\n", fileutil.Cyan, strings.Repeat("━", 40), fileutil.Reset)
	fmt.Printf("  %s%d OK%s  |  %s%d FAIL%s  |  %s%d SKIP%s  |  ⏱ %s%s\n",
		fileutil.Green, success, fileutil.Reset,
		fileutil.Red, failed, fileutil.Reset,
		fileutil.Yellow, skipped, fileutil.Reset,
		elapsed.Round(time.Millisecond).String(), fileutil.Reset)
}

func PrintStep(step string) {
	fmt.Printf("  %s▶%s %s\n", fileutil.Cyan, fileutil.Reset, step)
}

func PrintOK(msg string) {
	fmt.Printf("  %s✓%s %s\n", fileutil.Green, fileutil.Reset, msg)
}

func PrintFail(msg string) {
	fmt.Printf("  %s✗%s %s\n", fileutil.Red, fileutil.Reset, msg)
}

func PrintWarn(msg string) {
	fmt.Printf("  %s⚠%s %s\n", fileutil.Yellow, fileutil.Reset, msg)
}

func PrintProgress(current, total int) {
	pct := float64(current) / float64(total) * 100
	barW := 30
	filled := int(pct * float64(barW) / 100)
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	fmt.Printf("\r  %s [%s] %d/%d (%.0f%%)%s", fileutil.Cyan, bar, current, total, pct, fileutil.Reset)
}

func ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func ReadInput(prompt string) string {
	fmt.Print(prompt)
	scan.Scan()
	return strings.TrimSpace(scan.Text())
}

func Pause() {
	fmt.Print("Press Enter to continue...")
	scan.Scan()
}

func Beep() {
	fmt.Print("\a")
}

func SelectQuality(filter string) (int, bool) {
	PrintQualityTable(filter)

	var items []string
	switch filter {
	case "all", "custom":
		items = []string{
			"85  — balanced (good quality, ~50-70% smaller)  ★ recommended",
			"75  — smaller file, slightly lower quality",
			"100 — maximum quality, largest file",
			"Lossless — original quality preserved",
		}
	case "video":
		items = []string{
			"100 — CRF 18 (near-lossless, largest)",
			"85  — CRF 23 (balanced, ~50% smaller)  ★ recommended",
			"70  — CRF 28 (smaller, some quality loss)",
			"Lossless — original quality, largest file",
		}
	case "audio":
		items = []string{
			"100 — VBR ~320kbps (maximum quality)",
			"85  — VBR ~192kbps (excellent quality)  ★ recommended",
			"60  — VBR ~128kbps (smaller, good for podcasts)",
			"Lossless — original quality, largest file",
		}
	default:
		items = []string{
			"85  — balanced (good quality, ~50-70% smaller)  ★ recommended",
			"75  — smaller file, slightly lower quality",
			"100 — maximum quality, largest file",
			"Lossless — original quality preserved",
		}
	}
	fmt.Printf("\n")

	choice := SelectFromList(items, "Quality (↑↓ to choose, Enter to confirm):")
	if choice == "" {
		return 85, false
	}

	switch {
	case strings.Contains(choice, "Lossless"):
		return 0, true
	case strings.HasPrefix(choice, "100"):
		return 100, false
	case strings.HasPrefix(choice, "85"):
		return 85, false
	case strings.HasPrefix(choice, "75"):
		return 75, false
	case strings.HasPrefix(choice, "70"):
		return 70, false
	case strings.HasPrefix(choice, "60"):
		return 60, false
	}
	return 85, false
}
