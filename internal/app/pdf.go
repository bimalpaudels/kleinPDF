package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// compressPDFFile compresses a PDF file using Ghostscript
func (a *App) compressPDFFile(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error {
	if a.config.GhostscriptPath == "" {
		return fmt.Errorf("ghostscript not found. Please install ghostscript to use this application")
	}

	if options == nil {
		defaultOptions := DefaultCompressionOptions()
		options = &defaultOptions
	}

	// Validate and set defaults for required fields if they are empty
	if options.PDFVersion == "" {
		options.PDFVersion = "1.4"
	}
	if options.ImageDPI <= 0 {
		options.ImageDPI = 150
	}
	if options.ImageQuality <= 0 {
		options.ImageQuality = 85
	}

	// Handle grayscale conversion if needed
	actualInputPath := inputPath
	if options.ConvertToGrayscale {
		tempGrayscalePath := strings.Replace(inputPath, ".pdf", "_grayscale_temp.pdf", 1)

		err := a.convertToGrayscale(inputPath, tempGrayscalePath)
		if err != nil {
			return fmt.Errorf("grayscale conversion failed: %v", err)
		}

		actualInputPath = tempGrayscalePath
		defer os.Remove(tempGrayscalePath) // Clean up temp file
	}

	// Build Ghostscript command based on compression level
	var pdfSettings string
	switch compressionLevel {
	case "ultra":
		pdfSettings = "/screen"
	case "aggressive":
		pdfSettings = "/ebook"
	default: // good_enough
		pdfSettings = "/printer"
	}

	args := []string{
		"-sDEVICE=pdfwrite",
		"-dPDFSETTINGS=" + pdfSettings,
		"-dCompatibilityLevel=" + options.PDFVersion,
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-dAutoRotatePages=/None",
		"-dColorImageDownsampleType=/Bicubic",
		fmt.Sprintf("-dColorImageResolution=%d", options.ImageDPI),
		"-dGrayImageDownsampleType=/Bicubic",
		fmt.Sprintf("-dGrayImageResolution=%d", options.ImageDPI),
		"-dMonoImageDownsampleType=/Bicubic",
		fmt.Sprintf("-dMonoImageResolution=%d", options.ImageDPI),
		"-dColorConversionStrategy=/sRGB",
		fmt.Sprintf("-dEmbedAllFonts=%t", options.EmbedFonts),
		"-dSubsetFonts=true",
		"-dOptimize=true",
		"-dDownsampleColorImages=true",
		"-dDownsampleGrayImages=true",
		"-dDownsampleMonoImages=true",
	}

	// Add ultra-specific options
	if compressionLevel == "ultra" {
		args = append(args, "-dCompressFonts=true", "-dCompressStreams=true")
	}

	// Add metadata removal if enabled
	if options.RemoveMetadata {
		args = append(args, "-dPDFX", "-dUseCIEColor")
	}

	// Add thumbnail generation if enabled
	if options.GenerateThumbnails {
		args = append(args, "-dGenerateThumbnails=true")
	}

	args = append(args, "-sOutputFile="+outputPath, actualInputPath)

	// Execute Ghostscript command
	cmd := exec.Command(a.config.GhostscriptPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ghostscript failed: %v, output: %s", err, string(output))
	}

	// Check if output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("ghostscript did not create output file")
	}

	return nil
}

// convertToGrayscale converts a PDF to grayscale
func (a *App) convertToGrayscale(inputPath, outputPath string) error {
	args := []string{
		"-sDEVICE=pdfwrite",
		"-sProcessColorModel=DeviceGray",
		"-dOverrideICC",
		"-dUseCIEColor",
		"-dCompatibilityLevel=1.4",
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-sOutputFile=" + outputPath,
		inputPath,
	}

	cmd := exec.Command(a.config.GhostscriptPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("grayscale conversion failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetGhostscriptPath returns the path to Ghostscript executable
func (a *App) GetGhostscriptPath() string {
	return a.config.GhostscriptPath
}

// IsGhostscriptAvailable checks if Ghostscript is available
func (a *App) IsGhostscriptAvailable() bool {
	return a.config.GhostscriptPath != ""
}