package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pdf-compressor-wails/internal/config"
)

// PDFService handles PDF compression operations
type PDFService struct {
	config *config.Config
}

// NewPDFService creates a new PDF service
func NewPDFService(cfg *config.Config) *PDFService {
	return &PDFService{config: cfg}
}

// CompressionOptions holds advanced compression options
type CompressionOptions struct {
	ImageDPI           int    `json:"image_dpi"`
	ImageQuality       int    `json:"image_quality"`
	PDFVersion         string `json:"pdf_version"`
	RemoveMetadata     bool   `json:"remove_metadata"`
	EmbedFonts         bool   `json:"embed_fonts"`
	GenerateThumbnails bool   `json:"generate_thumbnails"`
	ConvertToGrayscale bool   `json:"convert_to_grayscale"`
}

// DefaultCompressionOptions returns default compression options
func DefaultCompressionOptions() CompressionOptions {
	return CompressionOptions{
		ImageDPI:           150,
		ImageQuality:       85,
		PDFVersion:         "1.4",
		RemoveMetadata:     false,
		EmbedFonts:         true,
		GenerateThumbnails: false,
		ConvertToGrayscale: false,
	}
}

// CompressPDF compresses a PDF file using Ghostscript
func (s *PDFService) CompressPDF(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error {
	if s.config.GhostscriptPath == "" {
		return fmt.Errorf("Ghostscript not found. Please install Ghostscript to use this application")
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

		err := s.convertToGrayscale(inputPath, tempGrayscalePath)
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
	cmd := exec.Command(s.config.GhostscriptPath, args...)
	// Ensure bundled libraries/resources are discoverable
	cmd.Env = s.buildGhostscriptEnv(os.Environ())
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
func (s *PDFService) convertToGrayscale(inputPath, outputPath string) error {
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

	cmd := exec.Command(s.config.GhostscriptPath, args...)
	cmd.Env = s.buildGhostscriptEnv(os.Environ())
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("grayscale conversion failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetGhostscriptPath returns the path to Ghostscript executable
func (s *PDFService) GetGhostscriptPath() string {
	return s.config.GhostscriptPath
}

// IsGhostscriptAvailable checks if Ghostscript is available
func (s *PDFService) IsGhostscriptAvailable() bool {
	return s.config.GhostscriptPath != ""
}

// buildGhostscriptEnv prepares environment variables for Ghostscript execution
func (s *PDFService) buildGhostscriptEnv(baseEnv []string) []string {
	gsPath := s.config.GhostscriptPath
	if gsPath == "" {
		return baseEnv
	}

	// If using system Ghostscript, no special environment needed
	if !s.isEmbeddedGhostscript(gsPath) {
		return baseEnv
	}

	// For embedded Ghostscript, set up environment
	env := append([]string{}, baseEnv...)
	baseDir := s.getBundledGhostscriptBase(gsPath)
	if baseDir == "" {
		return env
	}

	libDir := filepath.Join(baseDir, "lib")
	shareRoot := filepath.Join(baseDir, "share", "ghostscript")

	// Set GS_LIB for resource discovery - need to include specific paths
	var gsLibPaths []string

	// Add Resource/Init directory (contains gs_init.ps)
	resourceInit := filepath.Join(shareRoot, "Resource", "Init")
	if _, err := os.Stat(resourceInit); err == nil {
		gsLibPaths = append(gsLibPaths, resourceInit)
	}

	// Add Resource directory
	resource := filepath.Join(shareRoot, "Resource")
	if _, err := os.Stat(resource); err == nil {
		gsLibPaths = append(gsLibPaths, resource)
	}

	// Add the share root as fallback
	if _, err := os.Stat(shareRoot); err == nil {
		gsLibPaths = append(gsLibPaths, shareRoot)
	}

	if len(gsLibPaths) > 0 {
		env = setEnv(env, "GS_LIB", strings.Join(gsLibPaths, pathListSeparator()))
	}

	// macOS: set dynamic library path
	env = prependPathLikeEnv(env, "DYLD_LIBRARY_PATH", libDir)

	return env
}

// isEmbeddedGhostscript checks if the path points to embedded Ghostscript
func (s *PDFService) isEmbeddedGhostscript(gsPath string) bool {
	return strings.Contains(gsPath, "kleinpdf-ghostscript")
}

// getBundledGhostscriptBase returns the base directory for embedded Ghostscript
func (s *PDFService) getBundledGhostscriptBase(gsPath string) string {
	if !s.isEmbeddedGhostscript(gsPath) {
		return ""
	}

	// Walk up from /tmp/kleinpdf-ghostscript/ghostscript/bin/gs
	// to find /tmp/kleinpdf-ghostscript/ghostscript
	dir := filepath.Dir(gsPath) // bin
	dir = filepath.Dir(dir)     // ghostscript

	if filepath.Base(dir) == "ghostscript" {
		return dir
	}

	return ""
}

func pathListSeparator() string { return ":" }

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func prependPathLikeEnv(env []string, key, value string) []string {
	if value == "" {
		return env
	}
	prefix := key + "="
	sep := pathListSeparator()
	for i, kv := range env {
		if current, found := strings.CutPrefix(kv, prefix); found {
			if current == "" {
				env[i] = prefix + value
			} else {
				env[i] = prefix + value + sep + current
			}
			return env
		}
	}
	return append(env, prefix+value)
}
