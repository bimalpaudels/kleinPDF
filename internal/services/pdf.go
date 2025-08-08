package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

// ProgressCallback is a function type for progress updates
type ProgressCallback func(percent float64, message string)

// CompressPDF compresses a PDF file using Ghostscript
func (s *PDFService) CompressPDF(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error {
	return s.CompressPDFWithProgress(inputPath, outputPath, compressionLevel, options, nil)
}

// CompressPDFWithProgress compresses a PDF file using Ghostscript with progress callbacks
func (s *PDFService) CompressPDFWithProgress(inputPath, outputPath, compressionLevel string, options *CompressionOptions, progressCallback ProgressCallback) error {
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

	// Execute Ghostscript command directly without progress tracking
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

// buildGhostscriptEnv prepares environment variables so that the bundled
// Ghostscript binary can reliably find its dynamic libraries and resources
// when executed from the application bundle.
func (s *PDFService) buildGhostscriptEnv(baseEnv []string) []string {
	env := append([]string{}, baseEnv...)

	gsPath := s.config.GhostscriptPath
	if gsPath == "" {
		return env
	}

	baseDir := s.discoverBundledGhostscriptBase(gsPath)
	if baseDir == "" {
		return env
	}

	libDir := filepath.Join(baseDir, "lib")
	shareRoot := filepath.Join(baseDir, "share", "ghostscript")

	// Compose GS_LIB search paths
	gsLibPaths := s.discoverGhostscriptResourcePaths(shareRoot)
	if len(gsLibPaths) > 0 {
		env = setEnv(env, "GS_LIB", strings.Join(gsLibPaths, pathListSeparator()))
	}

	// Ensure dynamic loader can locate libgs and friends
	switch runtime.GOOS {
	case "darwin":
		// macOS uses DYLD_LIBRARY_PATH
		env = prependPathLikeEnv(env, "DYLD_LIBRARY_PATH", libDir)
	case "linux":
		// Linux uses LD_LIBRARY_PATH
		env = prependPathLikeEnv(env, "LD_LIBRARY_PATH", libDir)
	case "windows":
		// On Windows, extend PATH so DLLs can be located
		env = prependPathLikeEnv(env, "PATH", libDir)
		env = prependPathLikeEnv(env, "PATH", filepath.Join(baseDir, "bin"))
	}

	return env
}

// discoverBundledGhostscriptBase returns the base directory of the bundled
// Ghostscript distribution given an absolute path to the Ghostscript binary.
// Examples:
// - .../bundled/ghostscript/bin/gs -> .../bundled/ghostscript
// - .../bundled/ghostscript/gs     -> .../bundled/ghostscript
func (s *PDFService) discoverBundledGhostscriptBase(gsPath string) string {
	abs := gsPath
	if !filepath.IsAbs(abs) {
		if resolved, err := filepath.Abs(abs); err == nil {
			abs = resolved
		}
	}
	parent := filepath.Dir(abs)
	if filepath.Base(parent) == "bin" {
		return filepath.Dir(parent)
	}
	return parent
}

// discoverGhostscriptResourcePaths builds a prioritized list of resource
// directories that Ghostscript will search via GS_LIB.
func (s *PDFService) discoverGhostscriptResourcePaths(shareRoot string) []string {
	var paths []string
	// Add the root to allow Ghostscript to search beneath
	if stat, err := os.Stat(shareRoot); err == nil && stat.IsDir() {
		paths = append(paths, shareRoot)

		// Find the highest version directory, e.g., share/ghostscript/10.05.1
		entries, err := os.ReadDir(shareRoot)
		if err == nil {
			var versions []string
			for _, e := range entries {
				if e.IsDir() {
					name := e.Name()
					// simple heuristic: name contains a dot version
					if strings.Contains(name, ".") {
						versions = append(versions, name)
					}
				}
			}
			if len(versions) > 0 {
				sort.Strings(versions)
				best := versions[len(versions)-1]
				versionDir := filepath.Join(shareRoot, best)
				// Common subdirs used by Ghostscript search
				candidateDirs := []string{
					filepath.Join(versionDir, "Resource", "Init"),
					filepath.Join(versionDir, "lib"),
				}
				for _, d := range candidateDirs {
					if st, err := os.Stat(d); err == nil && st.IsDir() {
						paths = append(paths, d)
					}
				}
			}
		}
	}
	return paths
}

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

func pathListSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}
