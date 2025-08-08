package config

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config holds application configuration
type Config struct {
	Port            string
	WorkingDir      string
	DatabasePath    string
	GhostscriptPath string
	AppDataDir      string
	BundledAssets   embed.FS
}

// New creates a new configuration instance
func New(bundledAssets embed.FS) *Config {
	cfg := &Config{
		Port:          getEnv("PORT", "8000"),
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
	// Use a dedicated temp directory for extraction to avoid permission issues
	extractDir := filepath.Join(os.TempDir(), "kleinpdf-ghostscript")
	
	// Check if bundled Ghostscript already exists in temp directory
	gsPath := filepath.Join(extractDir, "ghostscript", "bin", "gs")
	if runtime.GOOS == "windows" {
		gsPath = filepath.Join(extractDir, "ghostscript", "bin", "gswin64c.exe")
	}

	// First check if already extracted and valid
	if c.isValidGhostscriptInstallation(extractDir) {
		c.GhostscriptPath = gsPath
		log.Printf("Found existing Ghostscript at: %s", gsPath)
		return
	}

	// Clean up any incomplete extraction
	os.RemoveAll(extractDir)

	// Extract from embedded assets
	log.Printf("Extracting Ghostscript to temp directory: %s", extractDir)
	if err := c.extractGhostscriptFromEmbed(extractDir); err != nil {
		log.Printf("Failed to extract Ghostscript from embedded assets: %v", err)
		return
	}

	// Verify extraction was successful
	if c.isValidGhostscriptInstallation(extractDir) {
		c.GhostscriptPath = gsPath
		log.Printf("Successfully extracted and validated Ghostscript at: %s", gsPath)
	} else {
		log.Printf("Ghostscript extraction validation failed")
		os.RemoveAll(extractDir) // Clean up failed extraction
	}
}

// isValidGhostscriptInstallation checks if the extracted Ghostscript installation is complete
func (c *Config) isValidGhostscriptInstallation(extractDir string) bool {
	gsPath := filepath.Join(extractDir, "ghostscript", "bin", "gs")
	if runtime.GOOS == "windows" {
		gsPath = filepath.Join(extractDir, "ghostscript", "bin", "gswin64c.exe")
	}

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
		if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
			return false
		}
	}

	return true
}

// extractGhostscriptFromEmbed extracts the embedded Ghostscript to the filesystem
func (c *Config) extractGhostscriptFromEmbed(extractDir string) error {
	// Ensure extract directory exists
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	// Walk through the embedded bundled directory
	return fs.WalkDir(c.BundledAssets, "bundled", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create the corresponding filesystem path in extract directory
		// Remove "bundled/" prefix from path to avoid nested bundled directories
		relPath := strings.TrimPrefix(path, "bundled/")
		localPath := filepath.Join(extractDir, relPath)
		
		if d.IsDir() {
			// Create directory
			return os.MkdirAll(localPath, 0755)
		}

		// Read embedded file
		data, err := c.BundledAssets.ReadFile(path)
		if err != nil {
			return err
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}

		// Determine file permissions
		perm := os.FileMode(0644)
		filename := filepath.Base(localPath)
		
		// Make executables and shared libraries executable
		if filename == "gs" || strings.HasSuffix(filename, ".exe") || 
		   strings.HasSuffix(filename, ".dylib") || strings.HasSuffix(filename, ".so") {
			perm = 0755
		}

		// Write file
		if err := os.WriteFile(localPath, data, perm); err != nil {
			return err
		}

		return nil
	})
}

func getAppDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, "Library", "Application Support", "KleinPDF")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "KleinPDF")
	default: // Linux and others
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".config", "kleinpdf")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
