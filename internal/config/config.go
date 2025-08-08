package config

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	// Check if bundled Ghostscript already exists in filesystem
	candidates := []string{
		"./bundled/ghostscript/bin/gs",
		"./bundled/ghostscript/gs",
	}
	if runtime.GOOS == "windows" {
		candidates = []string{
			"./bundled/ghostscript/bin/gswin64c.exe",
			"./bundled/ghostscript/gswin64c.exe",
		}
	}

	// First check if already extracted to filesystem
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			c.GhostscriptPath = abs
			return
		}
	}

	// If not found on filesystem, try to extract from embedded assets
	if err := c.extractGhostscriptFromEmbed(); err != nil {
		return // Failed to extract, leave GhostscriptPath empty
	}

	// Try candidates again after extraction
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			c.GhostscriptPath = abs
			return
		}
	}
}

// extractGhostscriptFromEmbed extracts the embedded Ghostscript to the filesystem
func (c *Config) extractGhostscriptFromEmbed() error {
	// Walk through the embedded bundled directory
	return fs.WalkDir(c.BundledAssets, "bundled", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create the corresponding filesystem path
		localPath := path
		
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

		// Write file
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return err
		}

		// Make executables (gs binary) executable
		if filepath.Base(localPath) == "gs" || filepath.Ext(localPath) == ".exe" {
			os.Chmod(localPath, 0755)
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
