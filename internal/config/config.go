package config

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application configuration
type Config struct {
	WorkingDir      string
	DatabasePath    string
	GhostscriptPath string
	AppDataDir      string
	BundledAssets   embed.FS
}

// New creates a new configuration instance
func New(bundledAssets embed.FS) *Config {
	cfg := &Config{
		BundledAssets: bundledAssets,
	}

	cfg.setupDirectories()
	cfg.setupGhostscriptPath()

	return cfg
}

func (c *Config) setupDirectories() {
	// Set up working directory (temp files)
	tempDir := os.TempDir()
	c.WorkingDir = filepath.Join(tempDir, "kleinpdf")

	// Ensure working directory exists
	os.MkdirAll(c.WorkingDir, 0755)

	// Set up app data directory (database, settings)
	c.AppDataDir = getAppDataDir()
	os.MkdirAll(c.AppDataDir, 0755)

	// Database path
	c.DatabasePath = filepath.Join(c.AppDataDir, "database.sqlite3")
}

func (c *Config) setupGhostscriptPath() {
	// Always use embedded Ghostscript
	extractDir := filepath.Join(os.TempDir(), "kleinpdf-ghostscript")
	gsPath := filepath.Join(extractDir, "ghostscript", "bin", "gs")

	// Check if already extracted and valid
	if c.isValidGhostscriptInstallation(extractDir) {
		c.GhostscriptPath = gsPath
		log.Printf("Using cached Ghostscript: %s", gsPath)
		return
	}

	// Clean and extract from embedded assets
	os.RemoveAll(extractDir)
	log.Printf("Extracting embedded Ghostscript to: %s", extractDir)

	if err := c.extractGhostscriptFromEmbed(extractDir); err != nil {
		log.Printf("Failed to extract Ghostscript: %v", err)
		return
	}

	if c.isValidGhostscriptInstallation(extractDir) {
		c.GhostscriptPath = gsPath
		log.Printf("Successfully setup embedded Ghostscript: %s", gsPath)
	} else {
		log.Printf("Ghostscript setup failed")
		os.RemoveAll(extractDir)
	}
}

// isValidGhostscriptInstallation checks if the extracted Ghostscript installation is complete
func (c *Config) isValidGhostscriptInstallation(extractDir string) bool {
	gsPath := filepath.Join(extractDir, "ghostscript", "bin", "gs")

	// Check if binary exists and is executable
	if stat, err := os.Stat(gsPath); err != nil || stat.Mode()&0111 == 0 {
		return false
	}

	// Check if required directories exist
	requiredDirs := []string{
		filepath.Join(extractDir, "ghostscript", "lib"),
		filepath.Join(extractDir, "ghostscript", "share", "ghostscript"),
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); err != nil {
			return false
		}
	}

	return true
}

// extractGhostscriptFromEmbed extracts the embedded Ghostscript archive to the filesystem
func (c *Config) extractGhostscriptFromEmbed(extractDir string) error {
	// Ensure extract directory exists
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	// Read the embedded tar.gz archive
	const archivePath = "bundled/ghostscript.tar.gz"
	data, err := c.BundledAssets.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded archive %s: %w", archivePath, err)
	}

	// Set up gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Set up tar reader
	tarReader := tar.NewReader(gzReader)

	baseExtractDir := filepath.Clean(extractDir) + string(os.PathSeparator)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading tar entry: %w", err)
		}

		// Sanitize and build destination path
		cleanName := filepath.Clean(header.Name)
		destPath := filepath.Join(extractDir, cleanName)

		// Prevent path traversal
		if !strings.HasPrefix(destPath, baseExtractDir) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", destPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", destPath, err)
			}
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", destPath, err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", destPath, err)
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", destPath, err)
			}

			// Adjust permissions for executables and shared libraries
			filename := filepath.Base(destPath)
			perm := os.FileMode(header.Mode)
			if filename == "gs" || strings.HasSuffix(filename, ".exe") ||
				strings.HasSuffix(filename, ".dylib") || strings.HasSuffix(filename, ".so") {
				perm = 0755
			}
			if err := os.Chmod(destPath, perm); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", destPath, err)
			}

		case tar.TypeSymlink:
			// Best effort: create symlink if possible; if not, skip
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for symlink %s: %w", destPath, err)
			}
			if err := os.Symlink(header.Linkname, destPath); err != nil {
				// Non-fatal: log and continue
				log.Printf("Warning: failed to create symlink %s -> %s: %v", destPath, header.Linkname, err)
			}

		default:
			// Ignore other types (hard links, etc.)
		}
	}

	return nil
}

func getAppDataDir() string {
	// macOS application support directory
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "KleinPDF")
}
