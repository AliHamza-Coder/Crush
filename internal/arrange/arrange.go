package arrange

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
	"github.com/AliHamza-Coder/crush/internal/ui"
)

// FolderSuffix is the prefix used for generated folders, e.g. "All webp".
const FolderPrefix = "All "

// Run organizes files in dir into per-extension folders.
// Example: all .webp files move to "All webp", all .mp4 to "All mp4".
func Run(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		ui.PrintFail(fmt.Sprintf("Cannot read directory: %s", err))
		ui.Pause()
		return
	}

	// Group file paths by extension.
	groups := make(map[string][]string)
	var total int64
	for _, entry := range files {
		if entry.IsDir() {
			continue
		}
		// Skip our own generated folders and the backup folder.
		name := entry.Name()
		if strings.HasPrefix(name, FolderPrefix) || strings.HasPrefix(name, "backup") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext == "" {
			ext = "no_extension"
		} else {
			ext = strings.TrimPrefix(ext, ".")
		}
		groups[ext] = append(groups[ext], filepath.Join(dir, name))
		total++
	}

	if total == 0 {
		ui.PrintWarn("No files to arrange")
		ui.Pause()
		return
	}

	exts := make([]string, 0, len(groups))
	for ext := range groups {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	fmt.Printf("\n  %s%d file(s) found%s\n\n", fileutil.Bold, total, fileutil.Reset)

	for _, ext := range exts {
		list := groups[ext]
		folderName := FolderPrefix + ext
		folderPath := filepath.Join(dir, folderName)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			ui.PrintFail(fmt.Sprintf("Cannot create %s: %s", folderName, err))
			continue
		}
		moved := 0
		for _, path := range list {
			dst := filepath.Join(folderPath, filepath.Base(path))
			if _, err := os.Stat(dst); err == nil {
				continue // destination already exists — skip
			}
			if err := os.Rename(path, dst); err != nil {
				ui.PrintWarn(fmt.Sprintf("Cannot move %s: %s", filepath.Base(path), err))
				continue
			}
			moved++
		}
		ui.PrintOK(fmt.Sprintf("%s → %d file(s) moved", folderName, moved))
	}

	ui.Pause()
}