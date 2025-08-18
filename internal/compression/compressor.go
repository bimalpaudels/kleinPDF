package compression

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// Compressor handles PDF compression operations
type Compressor struct {
	ghostscriptPath string
	logger          *slog.Logger
}

// NewCompressor creates a new compressor instance
func NewCompressor(ghostscriptPath string, logger *slog.Logger) *Compressor {
	return &Compressor{
		ghostscriptPath: ghostscriptPath,
		logger:          logger,
	}
}

// CompressFile compresses a PDF file using Ghostscript
func (c *Compressor) CompressFile(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error {
	if c.ghostscriptPath == "" {
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

		err := c.ConvertToGrayscale(inputPath, tempGrayscalePath)
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
	cmd := exec.Command(c.ghostscriptPath, args...)
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

// ConvertToGrayscale converts a PDF to grayscale
func (c *Compressor) ConvertToGrayscale(inputPath, outputPath string) error {
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

	cmd := exec.Command(c.ghostscriptPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("grayscale conversion failed: %v, output: %s", err, string(output))
	}

	return nil
}

// IsAvailable checks if Ghostscript is available
func (c *Compressor) IsAvailable() bool {
	return c.ghostscriptPath != ""
}

// GetGhostscriptPath returns the path to Ghostscript executable
func (c *Compressor) GetGhostscriptPath() string {
	return c.ghostscriptPath
}