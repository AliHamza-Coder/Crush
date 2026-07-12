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
	fmt.Printf("\n  %s[A]%s Convert ALL files\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[I]%s Convert Images only\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[V]%s Convert Videos only\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[O]%s Convert Audio only\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[S]%s Select specific files by number\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[D]%s Change directory\n", fileutil.Bold, fileutil.Reset)
	fmt.Printf("  %s[Q]%s Quit\n", fileutil.Bold, fileutil.Reset)
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
