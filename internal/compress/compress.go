package compress

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func RunFFmpeg(ffmpeg string, args []string) error {
	cmd := exec.Command(ffmpeg, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			lines := strings.Split(errMsg, "\n")
			for i := len(lines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(lines[i])
				if line != "" {
					return fmt.Errorf("ffmpeg: %s", line)
				}
			}
		}
		return err
	}
	return nil
}

func Image(ffmpeg, input, output string, quality int, format string, lossless bool) error {
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	switch target {
	case "webp":
		if lossless {
			args = append(args, "-c:v", "libwebp", "-lossless", "1", "-compression_level", "4")
		} else {
			args = append(args, "-c:v", "libwebp", "-quality", strconv.Itoa(quality), "-compression_level", "4")
		}
	case "avif":
		if lossless {
			args = append(args, "-c:v", "libaom-av1", "-crf", "0", "-b:v", "0", "-strict", "experimental")
		} else {
			crf := 20 + (100-quality)*43/100
			args = append(args, "-c:v", "libaom-av1", "-crf", strconv.Itoa(crf), "-b:v", "0", "-strict", "experimental")
		}
	case "png":
		// Use explicit png codec; map quality (0‑100) to compression level 0‑9 (higher = more compression)
		level := 9 - (quality / 11) // roughly 0‑9 range
		if level < 0 {
			level = 0
		}
		args = append(args, "-c:v", "png", "-compression_level", strconv.Itoa(level))
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		if lossless {
			q := 1
			args = append(args, "-q:v", strconv.Itoa(q))
		} else {
			q := mapQuality(quality)
			args = append(args, "-q:v", strconv.Itoa(q))
		}
	}

	args = append(args, "-y", output)
	return RunFFmpeg(ffmpeg, args)
}

func Video(ffmpeg, input, output string, quality int, format string, lossless bool) error {
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	isConversion := format != ""

	if lossless {
		args = append(args, "-c:v", "libx264", "-crf", "0", "-preset", "fast")
		switch target {
		case "mp4", "m4v":
			args = append(args, "-c:a", "copy")
		case "webm":
			args = append(args, "-c:a", "libopus", "-b:a", "320k")
		case "mov":
			args = append(args, "-c:a", "pcm_s16le")
		default:
			if isConversion {
				args = append(args, "-c:a", "aac", "-b:a", "320k")
			} else {
				args = append(args, "-c:a", "copy")
			}
		}
		args = append(args, "-movflags", "+faststart", "-y", output)
		return RunFFmpeg(ffmpeg, args)
	}

	// CRF mapping: quality 100→10 (near-lossless), 85→15 (excellent), 75→19 (good), 50→27 (okay), 1→44 (small)
	crf := 10 + (100-quality)*35/100

	switch target {
	case "mp4", "m4v":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast")
		if isConversion {
			args = append(args, "-c:a", "aac", "-b:a", "192k")
		} else {
			args = append(args, "-c:a", "copy")
		}
		args = append(args, "-movflags", "+faststart")
	case "webm":
		args = append(args, "-c:v", "libvpx-vp9", "-crf", strconv.Itoa(crf), "-b:v", "0", "-c:a", "libopus", "-b:a", "128k")
	case "avi":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast")
		if isConversion {
			args = append(args, "-c:a", "aac", "-b:a", "192k")
		} else {
			args = append(args, "-c:a", "copy")
		}
	case "mov":
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast")
		if isConversion {
			args = append(args, "-c:a", "aac", "-b:a", "192k")
		} else {
			args = append(args, "-c:a", "copy")
		}
	case "gif":
		args = append(args, "-vf", "fps=10,scale=320:-1:flags=lanczos")
	default:
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", "fast")
		if isConversion {
			args = append(args, "-c:a", "aac", "-b:a", "192k")
		} else {
			args = append(args, "-c:a", "copy")
		}
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, "-y", output)
	return RunFFmpeg(ffmpeg, args)
}

func Audio(ffmpeg, input, output string, quality int, format string, lossless bool) error {
	var args []string
	args = append(args, "-i", input)

	target := format
	if target == "" {
		target = strings.TrimPrefix(filepath.Ext(output), ".")
	}

	// Explicitly map first audio stream for reliable extraction from videos.
	// -map a:0 selects only the first audio stream, ignoring video/subtitle streams.
	// This also ensures the full audio duration is preserved even if input has gaps.
	args = append(args, "-map", "a:0")

	if lossless {
		switch target {
		case "flac":
			args = append(args, "-c:a", "flac", "-compression_level", "8")
		case "wav":
		case "alac", "m4a":
			args = append(args, "-c:a", "alac")
		default:
			args = append(args, "-c:a", "flac", "-compression_level", "8")
		}
		args = append(args, "-y", output)
		return RunFFmpeg(ffmpeg, args)
	}

	switch target {
	case "mp3":
		q := 0 + (100-quality)*9/100
		args = append(args, "-c:a", "libmp3lame", "-q:a", strconv.Itoa(q))
	case "flac":
		level := 8 - (quality / 12)
		if level < 0 {
			level = 0
		}
		args = append(args, "-c:a", "flac", "-compression_level", strconv.Itoa(level))
	case "alac":
		args = append(args, "-c:a", "alac")
	case "ogg":
		q := 0 + (100-quality)*10/100
		args = append(args, "-c:a", "libvorbis", "-q:a", strconv.Itoa(q))
	case "png":
		args = append(args, "-c:v", "png", "-compression_level", strconv.Itoa(9-(quality/10)))
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
