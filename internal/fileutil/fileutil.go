package fileutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const Version = "v2.1.0"

var (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
)

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
	VideoExts = map[string]bool{
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".mpg": true, ".mpeg": true, ".3gp": true, ".ts": true,
	}
	ImageExts = map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
		".bmp": true, ".tiff": true, ".tif": true, ".avif": true,
		".gif": true, ".svg": true, ".ico": true, ".heic": true,
		".heif": true,
	}
	AudioExts = map[string]bool{
		".mp3": true, ".wav": true, ".flac": true, ".ogg": true,
		".aac": true, ".wma": true, ".m4a": true, ".opus": true,
		".aiff": true, ".alac": true,
	}
	FormatNames = map[string]string{
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
)

func DetectType(ext string) FileType {
	if VideoExts[ext] {
		return TypeVideo
	}
	if ImageExts[ext] {
		return TypeImage
	}
	if AudioExts[ext] {
		return TypeAudio
	}
	return TypeUnknown
}

func CanConvertTo(ft FileType, format string) bool {
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

func FormatToType(format string) string {
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

func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	if bytes < 1024*1024 {
		return strconv.FormatFloat(float64(bytes)/1024, 'f', 0, 64) + " KB"
	}
	if bytes < 1024*1024*1024 {
		return strconv.FormatFloat(float64(bytes)/1024/1024, 'f', 1, 64) + " MB"
	}
	return strconv.FormatFloat(float64(bytes)/1024/1024/1024, 'f', 2, 64) + " GB"
}

func FormatDisplayName(ext string) string {
	if n, ok := FormatNames[ext]; ok {
		return n
	}
	return strings.ToUpper(strings.TrimPrefix(ext, "."))
}

func FormatColor(ext string) string {
	if ImageExts[ext] {
		return Green
	}
	if VideoExts[ext] {
		return Cyan
	}
	if AudioExts[ext] {
		return Yellow
	}
	return Reset
}

func SortFiles(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
}

func FindFFmpeg() (string, string) {
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

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FileNameWithoutExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

func IsValidTargetFormat(format string) bool {
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

func FilterByType(files []FileInfo, filter string) []FileInfo {
	if filter == "all" || filter == "custom" {
		return files
	}
	var ft FileType
	switch filter {
	case "image":
		ft = TypeImage
	case "video":
		ft = TypeVideo
	case "audio":
		ft = TypeAudio
	default:
		return files
	}
	var result []FileInfo
	for _, f := range files {
		if f.Type == ft {
			result = append(result, f)
		}
	}
	return result
}

func FilterByFormat(files []FileInfo, format string) []FileInfo {
	var result []FileInfo
	for _, f := range files {
		if CanConvertTo(f.Type, format) {
			result = append(result, f)
		}
	}
	return result
}

func IsInPATH(dir string) bool {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
