package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
	"github.com/AliHamza-Coder/crush/internal/ui"
)

func Run() {
	ui.ClearScreen()
	ui.PrintBanner()
	ui.PrintStep("Checking system...")

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	ffmpegPath, _ := fileutil.FindFFmpeg()
	if ffmpegPath != "" {
		ui.PrintOK(fmt.Sprintf("FFmpeg found at %s", ffmpegPath))
	} else {
		ui.PrintWarn("FFmpeg not found")

		var pkgName, pkgCmd string
		switch runtime.GOOS {
		case "windows":
			pkgName = "winget"
			pkgCmd = "winget install -e --id Gyan.FFmpeg"
		case "darwin":
			pkgName = "brew"
			pkgCmd = "brew install ffmpeg"
		default:
			pkgName = "apt"
			pkgCmd = "sudo apt install ffmpeg"
		}

		fmt.Printf("\n  Install via %s? [Y/n]: ", pkgName)
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "n" {
			ui.PrintStep(fmt.Sprintf("Running: %s", pkgCmd))
			parts := strings.Fields(pkgCmd)
			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				ui.PrintFail(fmt.Sprintf("Installation failed: %s", err))
			} else {
				ui.PrintOK("FFmpeg installed")
			}
		} else {
			ui.PrintWarn("Skipping FFmpeg install")
			fmt.Printf("\n  Install manually:\n")
			fmt.Printf("    %s\n", pkgCmd)
		}
	}

	ui.PrintStep("Setting up PATH...")
	if !fileutil.IsInPATH(exeDir) {
		addToPATH(exeDir)
		ui.PrintOK(fmt.Sprintf("Added %s to PATH", exeDir))
	} else {
		ui.PrintOK("Already in PATH")
	}

	fmt.Printf("\n  %s%s%s\n", fileutil.Bold+fileutil.Green, strings.Repeat("━", 40), fileutil.Reset)
	ui.PrintOK("CRUSH installation complete!")
	fmt.Printf("  Run %scrush%s from any terminal\n", fileutil.Cyan, fileutil.Reset)
	if ffmpegPath == "" {
		fmt.Printf("  Then run %scrush install%s to install FFmpeg\n", fileutil.Cyan, fileutil.Reset)
	}
	fmt.Printf("  %s%s%s\n", fileutil.Bold+fileutil.Green, strings.Repeat("━", 40), fileutil.Reset)
	ui.Pause()
}

func addToPATH(dir string) {
	if runtime.GOOS == "windows" {
		ps := fmt.Sprintf(
			`[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';%s', 'User')`,
			dir,
		)
		exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
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
}

func Uninstall() {
	ui.ClearScreen()
	ui.PrintBanner()
	ui.PrintSection("Uninstall CRUSH")

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	fmt.Printf("  This will:\n")
	fmt.Printf("    • Remove %s from your PATH\n", exeDir)
	fmt.Printf("    • Delete %s\n\n", exePath)

	resp := ui.ReadInput("  Uninstall CRUSH? (y/N): ")
	if strings.ToLower(resp) != "y" {
		ui.PrintWarn("Uninstall cancelled")
		ui.Pause()
		return
	}

	ui.PrintStep("Removing from PATH...")
	if fileutil.IsInPATH(exeDir) {
		removeFromPATH(exeDir)
		ui.PrintOK("Removed from PATH")
	} else {
		ui.PrintOK("Not in PATH")
	}

	ui.PrintStep("Deleting executable...")
	if runtime.GOOS == "windows" {
		psCmd := fmt.Sprintf(
			`Start-Sleep -Seconds 2; Remove-Item -LiteralPath '%s' -Force`,
			exePath,
		)
		exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psCmd).Start()

		fmt.Printf("\n  %sCRUSH has been uninstalled.%s\n", fileutil.Green, fileutil.Reset)
		fmt.Printf("  Close this terminal and open a new one for PATH changes.\n")
		os.Exit(0)
	}

	if err := os.Remove(exePath); err != nil {
		ui.PrintFail(fmt.Sprintf("Could not delete %s (delete manually)", exePath))
	} else {
		ui.PrintOK("Deleted successfully")
	}

	fmt.Printf("\n  %sCRUSH has been uninstalled.%s\n", fileutil.Green, fileutil.Reset)
	fmt.Printf("  Close and reopen your terminal for PATH changes.\n")
	ui.Pause()
}

func removeFromPATH(dir string) {
	if runtime.GOOS == "windows" {
		ps := fmt.Sprintf(
			`[Environment]::SetEnvironmentVariable('Path', ([Environment]::GetEnvironmentVariable('Path', 'User').Split(';') -ne '%s') -join ';', 'User')`,
			dir,
		)
		exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	} else {
		home, _ := os.UserHomeDir()
		rcFile := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
			rcFile = filepath.Join(home, ".zshrc")
		}
		input, err := os.ReadFile(rcFile)
		if err == nil {
			lines := strings.Split(string(input), "\n")
			var out []string
			for _, line := range lines {
				if !strings.Contains(line, dir) {
					out = append(out, line)
				}
			}
			os.WriteFile(rcFile, []byte(strings.Join(out, "\n")), 0644)
		}
	}
}
