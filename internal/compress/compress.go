package compress

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func RunFFmpeg(ffmpeg string, args []string) error {
	cmd := exec.Command(ffmpeg, args...)
	cmd.Stderr = nil
	cmd.Stdout = nil
	return cmd.Run()
}

func Image(ffmpeg, input, output string, quality int, format string) error {
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	switch target {
	case "webp":
		args = append(args, "-c:v", "libwebp", "-quality", strconv.Itoa(quality), "-compression_level", "4")
	case "avif":
		crf := 20 + (100-quality)*43/100
		args = append(args, "-c:v", "libaom-av1", "-crf", strconv.Itoa(crf), "-b:v", "0", "-strict", "experimental")
	case "png":
		args = append(args, "-compression_level", "9")
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		q := mapQuality(quality)
		args = append(args, "-q:v", strconv.Itoa(q))
	}

	args = append(args, "-y", output)
	return RunFFmpeg(ffmpeg, args)
}

func Video(ffmpeg, input, output string, quality int, format string) error {
	crf := 18 + (100-quality)*22/100
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	switch target {
	case "mp4", "m4v":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast",
			"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart")
	case "webm":
		args = append(args, "-c:v", "libvpx-vp9", "-crf", strconv.Itoa(crf), "-b:v", "0", "-c:a", "libopus")
	case "avi":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast")
	case "mov":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast", "-c:a", "aac")
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast",
			"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart")
	}

	args = append(args, "-y", output)
	return RunFFmpeg(ffmpeg, args)
}

func Audio(ffmpeg, input, output string, quality int, format string) error {
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	switch target {
	case "mp3":
		q := 0 + (100-quality)*9/100
		args = append(args, "-c:a", "libmp3lame", "-q:a", strconv.Itoa(q))
	case "ogg":
		q := 0 + (100-quality)*10/100
		args = append(args, "-c:a", "libvorbis", "-q:a", strconv.Itoa(q))
	case "flac":
		args = append(args, "-c:a", "flac", "-compression_level", "8")
	case "aac", "m4a":
		bitrate := 64 + (quality * 256 / 100)
		args = append(args, "-c:a", "aac", "-b:a", strconv.Itoa(bitrate)+"k")
	case "opus":
		bitrate := 32 + (quality * 128 / 100)
		args = append(args, "-c:a", "libopus", "-b:a", strconv.Itoa(bitrate)+"k")
	case "wav":
	default:
		q := 0 + (100-quality)*9/100
		args = append(args, "-c:a", "libmp3lame", "-q:a", strconv.Itoa(q))
	}

	args = append(args, "-y", output)
	return RunFFmpeg(ffmpeg, args)
}

func mapQuality(quality int) int {
	q := 1 + ((100-quality)*30)/100
	if q < 1 {
		return 1
	}
	if q > 31 {
		return 31
	}
	return q
}
