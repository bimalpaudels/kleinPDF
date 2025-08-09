package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"pdf-compressor-wails/internal/binary"
)

// Config holds application configuration
type Config struct {
	WorkingDir      string
	DatabasePath    string
	GhostscriptPath string
	AppDataDir      string
}

// New creates a new configuration instance
func New() *Config {
	cfg := &Config{}

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
	// Use embedded binary directly
	extractDir := filepath.Join(os.TempDir(), "kleinpdf-ghostscript")
	gsPath := filepath.Join(extractDir, "gs")

	// Check if already extracted and valid
	if c.isValidGhostscriptBinary(gsPath) {
		c.GhostscriptPath = gsPath
		log.Printf("Using cached Ghostscript: %s", gsPath)
		return
	}

	// Create directory and extract binary
	os.MkdirAll(extractDir, 0755)
	log.Printf("Extracting embedded Ghostscript binary to: %s", gsPath)

	if err := c.extractGhostscriptBinary(gsPath); err != nil {
		log.Printf("Failed to extract Ghostscript binary: %v", err)
		return
	}

	if c.isValidGhostscriptBinary(gsPath) {
		c.GhostscriptPath = gsPath
		log.Printf("Successfully setup embedded Ghostscript: %s", gsPath)
	} else {
		log.Printf("Ghostscript binary setup failed")
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
