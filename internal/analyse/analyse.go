package analyse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
)

func Directory(dir string) ([]fileutil.FileInfo, *fileutil.AnalyseStats) {
	stats := &fileutil.AnalyseStats{Dir: dir, Formats: make(map[string]int)}
	var files []fileutil.FileInfo

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), "backup") || strings.HasPrefix(info.Name(), "originals_") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		ft := fileutil.DetectType(ext)
		if ft == fileutil.TypeUnknown {
			return nil
		}

		stats.Total++
		stats.TotalSize += info.Size()
		stats.Formats[ext]++

		switch ft {
		case fileutil.TypeImage:
			stats.Images++
			stats.ImageSize += info.Size()
		case fileutil.TypeVideo:
			stats.Videos++
			stats.VideoSize += info.Size()
		case fileutil.TypeAudio:
			stats.Audio++
			stats.AudioSize += info.Size()
		}

		files = append(files, fileutil.FileInfo{
			Path: path, Name: info.Name(), Ext: ext,
			Size: info.Size(), SizeStr: fileutil.FormatSize(info.Size()),
			Type: ft, TypeName: fileutil.FormatDisplayName(ext),
		})
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot read directory %s: %s\n", dir, err)
	}

	fileutil.SortFiles(files)
	for i := range files {
		files[i].Index = i + 1
	}

	return files, stats
}

func PrintAnalysis(stats *fileutil.AnalyseStats) {
	fmt.Printf("  %sDirectory:%s %s\n", fileutil.Bold, fileutil.Reset, stats.Dir)

	bar := func(label string, count int, total int, size int64, totalSize int64, color string) {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		barW := 20
		filled := int(pct * float64(barW) / 100)
		if filled > barW {
			filled = barW
		}
		barStr := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		fmt.Printf("  %s%-6s %s%s  %3d  %s\n", color, label, barStr, fileutil.Reset, count, fileutil.FormatSize(size))
	}

	all := stats.Images + stats.Videos + stats.Audio
	if all == 0 {
		all = 1
	}

	bar("Images", stats.Images, all, stats.ImageSize, stats.TotalSize, fileutil.Green)
	bar("Videos", stats.Videos, all, stats.VideoSize, stats.TotalSize, fileutil.Cyan)
	bar("Audio ", stats.Audio, all, stats.AudioSize, stats.TotalSize, fileutil.Yellow)

	var parts []string
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif", ".svg",
		".mp4", ".mov", ".webm", ".mkv", ".avi",
		".mp3", ".wav", ".flac", ".ogg", ".m4a", ".aac"} {
		if c := stats.Formats[ext]; c > 0 {
			parts = append(parts, fmt.Sprintf("%s%s%s x%d", fileutil.FormatColor(ext), strings.TrimPrefix(ext, "."), fileutil.Reset, c))
		}
	}

	fmt.Printf("\n  %sTotal:%s %d files  |  %s\n", fileutil.Bold, fileutil.Reset, stats.Total, fileutil.FormatSize(stats.TotalSize))
	if len(parts) > 0 {
		fmt.Printf("  %sFormats:%s %s\n", fileutil.Bold, fileutil.Reset, strings.Join(parts, "  "))
	}
}

func PrintFileList(files []fileutil.FileInfo) {
	if len(files) == 0 {
		return
	}
	if len(files) > 100 {
		fmt.Printf("  (%d files — use 'Select specific files')\n", len(files))
		return
	}

	header := fmt.Sprintf("  %-4s %-8s %-10s %-6s  %s", "#", "Type", "Size", "Format", "Filename")
	fmt.Printf("  %s%s%s\n", fileutil.Dim, strings.Repeat("─", len(header)-2), fileutil.Reset)

	for _, f := range files {
		col := ""
		switch f.Type {
		case fileutil.TypeImage:
			col = fileutil.Green
		case fileutil.TypeVideo:
			col = fileutil.Cyan
		case fileutil.TypeAudio:
			col = fileutil.Yellow
		}
		fmt.Printf("  %-4d %s%-8s%s %-10s %-6s  %s\n",
			f.Index, col, f.TypeName, fileutil.Reset, f.SizeStr, strings.ToUpper(strings.TrimPrefix(f.Ext, ".")), f.Name)
	}
	fmt.Printf("  %s%s%s\n", fileutil.Dim, strings.Repeat("─", len(header)-2), fileutil.Reset)
}
