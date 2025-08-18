package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"kleinpdf/internal/binary"
)

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	cfg := &Config{
		Logger: slog.Default(),
	}

	cfg.setupDirectories()
	cfg.setupGhostscriptPath()

	return cfg
}

func (c *Config) setupDirectories() {
	// Set up app data directory (database, settings)
	appDataDir := getAppDataDir()
	os.MkdirAll(appDataDir, 0755)

	// Database path
	c.DatabasePath = filepath.Join(appDataDir, "database.sqlite3")
}

func (c *Config) setupGhostscriptPath() {
	// Use embedded binary directly in app data directory for persistence
	appDataDir := getAppDataDir()
	extractDir := filepath.Join(appDataDir, "bin")
	gsPath := filepath.Join(extractDir, "ghostscript")

	// Check if already extracted and valid
	if c.isValidGhostscriptBinary(gsPath) {
		c.GhostscriptPath = gsPath
		c.Logger.Info("Using cached Ghostscript", "path", gsPath)
		return
	}

	// Create directory and extract binary
	os.MkdirAll(extractDir, 0755)
	c.Logger.Info("Extracting embedded Ghostscript binary", "path", gsPath)

	if err := c.extractGhostscriptBinary(gsPath); err != nil {
		c.Logger.Error("Failed to extract Ghostscript binary", "error", err)
		return
	}

	if c.isValidGhostscriptBinary(gsPath) {
		c.GhostscriptPath = gsPath
		c.Logger.Info("Successfully setup embedded Ghostscript", "path", gsPath)
	} else {
		c.Logger.Error("Ghostscript binary setup failed")
		os.Remove(gsPath)
	}
}

// isValidGhostscriptBinary checks if the Ghostscript binary exists and is executable
func (c *Config) isValidGhostscriptBinary(gsPath string) bool {
	// Check if binary exists and is executable
	if stat, err := os.Stat(gsPath); err != nil || stat.Mode()&0111 == 0 {
		return false
	}
	return true
}

// extractGhostscriptBinary extracts the embedded Ghostscript binary to the filesystem
func (c *Config) extractGhostscriptBinary(gsPath string) error {
	// Write the embedded binary directly to the filesystem
	file, err := os.OpenFile(gsPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("failed to create binary file %s: %w", gsPath, err)
	}
	defer file.Close()

	_, err = file.Write(binary.GhostscriptBinary)
	if err != nil {
		return fmt.Errorf("failed to write binary data: %w", err)
	}

	return nil
}

func getAppDataDir() string {
	// macOS application support directory
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "KleinPDF")
}