package favicon

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AliHamza-Coder/crush/internal/fileutil"
	"github.com/AliHamza-Coder/crush/internal/ui"
)

// Generate creates 16×16 and 32×32 SVG favicons from an input image.
// It uses ffmpeg to resize the image to PNG at the target sizes, then
// wraps each resized PNG in an inline SVG.
func Generate(ffmpeg, inputPath string) error {
	base := fileutil.FileNameWithoutExt(filepath.Base(inputPath))
	outDir := "."

	sizes := []int{16, 32}

	for _, size := range sizes {
		pngFile := filepath.Join(outDir, fmt.Sprintf("%s_%dx%d_tmp.png", base, size, size))
		svgFile := filepath.Join(outDir, fmt.Sprintf("favicon_%dx%d.svg", size, size))

		// Step 1: resize to PNG via ffmpeg
		args := []string{"-i", inputPath, "-vf", fmt.Sprintf("scale=%d:%d:flags=lanczos", size, size), "-y", pngFile}
		cmd := exec.Command(ffmpeg, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			os.Remove(pngFile)
			return fmt.Errorf("ffmpeg resize %dx%d failed: %s\n%s", size, size, err, string(output))
		}

		// Step 2: read the resized PNG
		pngData, err := os.ReadFile(pngFile)
		if err != nil {
			os.Remove(pngFile)
			return fmt.Errorf("cannot read resized PNG: %s", err)
		}

		// Step 3: base64-encode it for embedding inside SVG
		b64 := base64.StdEncoding.EncodeToString(pngData)

		// Step 4: generate the SVG (use regular string for escaping)
		svg := fmt.Sprintf(
			"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
				"<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\">\n"+
				"  <image width=\"%d\" height=\"%d\" href=\"data:image/png;base64,%s\"/>\n"+
				"</svg>", size, size, size, size, size, size, b64)

		if err := os.WriteFile(svgFile, []byte(svg), 0644); err != nil {
			os.Remove(pngFile)
			return fmt.Errorf("cannot write SVG: %s", err)
		}

		// Clean up temp PNG
		os.Remove(pngFile)
	}

	return nil
}

// RunFaviconMenu handles the interactive favicon generation flow.
func RunFaviconMenu(ffmpeg string, files []fileutil.FileInfo) {
	// Only show image files
	var images []fileutil.FileInfo
	for _, f := range files {
		if f.Type == fileutil.TypeImage {
			images = append(images, f)
		}
	}
	if len(images) == 0 {
		ui.PrintWarn("No image files found — need an image to generate a favicon")
		ui.Pause()
		return
	}

	ui.PrintSection("Generate Favicon")
	fmt.Printf("  Select an image to generate 16×16 and 32×32 SVG favicons\n\n")

	// Build a list of image file choices with index prefix
	imageChoices := make([]string, len(images))
	for i, img := range images {
		imageChoices[i] = fmt.Sprintf("%d. %s  (%s)", i+1, img.Name, img.SizeStr)
	}

	choice := ui.SelectFromList(imageChoices, "Choose an image (↑↓ to choose, Enter to confirm):")
	if choice == "" {
		return
	}

	// Extract the index from "N. filename  (size)"
	parts := strings.SplitN(choice, ".", 2)
	idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || idx < 1 || idx > len(images) {
		ui.PrintFail("Invalid selection")
		ui.Pause()
		return
	}
	selectedImg := images[idx-1]

	ui.PrintStep(fmt.Sprintf("Generating favicons from %s...", selectedImg.Name))

	if err := Generate(ffmpeg, selectedImg.Path); err != nil {
		ui.PrintFail(fmt.Sprintf("Favicon generation failed: %s", err))
		ui.Pause()
		return
	}

	ui.PrintOK("favicon_16x16.svg created")
	ui.PrintOK("favicon_32x32.svg created")
	ui.PrintOK("Place these in your website root for browser favicons")
	ui.Pause()
}
