package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
	"github.com/AliHamza-Coder/crush/internal/ui"
)

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func Run() {
	ui.ClearScreen()
	ui.PrintBanner()
	ui.PrintSection("Check for Updates")

	current := fileutil.Version
	ui.PrintStep(fmt.Sprintf("Current version: %s", current))

	latest, url, err := fetchLatest()
	if err != nil {
		ui.PrintFail(fmt.Sprintf("Cannot check updates: %s", err))
		ui.Pause()
		return
	}

	ui.PrintStep(fmt.Sprintf("Latest version:  %s", latest))

	if !isNewer(latest, current) {
		ui.PrintOK("You already have the latest version!")
		ui.Pause()
		return
	}

	fmt.Printf("\n  A new version (%s) is available!\n", latest)
	resp := ui.ReadInput("  Download and update? (Y/n): ")
	if strings.ToLower(resp) == "n" {
		ui.PrintWarn("Update skipped")
		ui.Pause()
		return
	}

	ui.PrintStep("Downloading...")
	if err := downloadAndReplace(url); err != nil {
		ui.PrintFail(fmt.Sprintf("Update failed: %s", err))
		ui.Pause()
		return
	}
}

func fetchLatest() (tag, downloadURL string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/AliHamza-Coder/Crush/releases/latest")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", "", err
	}

	if rel.TagName == "" {
		return "", "", fmt.Errorf("no releases found")
	}

	for _, a := range rel.Assets {
		if a.Name == "crush.exe" {
			return rel.TagName, a.DownloadURL, nil
		}
	}

	return "", "", fmt.Errorf("no crush.exe asset found in latest release")
}

func isNewer(latest, current string) bool {
	l := strings.TrimPrefix(latest, "v")
	c := strings.TrimPrefix(current, "v")

	lp := strings.Split(l, ".")
	cp := strings.Split(c, ".")

	for i := 0; i < 3; i++ {
		var lv, cv int
		if i < len(lp) {
			fmt.Sscanf(lp[i], "%d", &lv)
		}
		if i < len(cp) {
			fmt.Sscanf(cp[i], "%d", &cv)
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return false
}

func downloadAndReplace(url string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current executable: %s", err)
	}

	tmpPath := exePath + ".new"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %s", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("download failed: %s", err)
	}
	defer resp.Body.Close()

	size, err := io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download incomplete: %s", err)
	}

	ui.PrintOK(fmt.Sprintf("Downloaded %.1f MB", float64(size)/1024/1024))

	return replaceSelf(exePath, tmpPath)
}

func replaceSelf(exePath, tmpPath string) error {
	ui.PrintStep("Installing update...")

	if runtime.GOOS == "windows" {
		batPath := filepath.Join(filepath.Dir(exePath), "_crush_update.bat")
		oldPath := exePath + ".old"
		batContent := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
ren "%s" "crush.exe.old" >nul 2>&1
move /y "%s" "%s" >nul 2>&1
del "%s" >nul 2>&1
del "%%~f0" >nul 2>&1
`, exePath, tmpPath, exePath, oldPath)
		if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot create update script: %s", err)
		}
		exec.Command("cmd", "/c", "start", "/b", batPath).Start()
		ui.PrintOK("Update applied! Restart CRUSH to use the new version.")
		ui.Pause()
		os.Exit(0)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace executable: %s", err)
	}

	if err := os.Chmod(exePath, 0755); err != nil {
		return fmt.Errorf("cannot set permissions: %s", err)
	}

	ui.PrintOK("Update applied! Restart CRUSH to use the new version.")
	ui.Pause()
	return nil
}
