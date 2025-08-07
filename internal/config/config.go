package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Config holds application configuration
type Config struct {
	Port         string
	WorkingDir   string
	DatabasePath string
	GhostscriptPath string
	AppDataDir   string
}

// New creates a new configuration instance
func New() *Config {
	cfg := &Config{
		Port: getEnv("PORT", "8000"),
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
	// Try to find Ghostscript executable
	candidates := []string{
		"./bundled/ghostscript/gs",
		"./bundled/ghostscript/gswin64c.exe", 
		"./bundled/ghostscript/bin/gs",
		"./bundled/ghostscript/bin/gswin64c.exe",
	}
	
	// Check bundled versions first
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			c.GhostscriptPath = abs
			return
		}
	}
	
	// Check system PATH
	systemCandidates := []string{"gs"}
	if runtime.GOOS == "windows" {
		systemCandidates = []string{
			"gswin64c.exe",
			"gswin32c.exe", 
			"C:\\Program Files\\gs\\gs10.05.1\\bin\\gswin64c.exe",
			"C:\\Program Files\\gs\\gs10.02.1\\bin\\gswin64c.exe",
		}
	}
	
	for _, candidate := range systemCandidates {
		if path, err := exec.LookPath(candidate); err == nil {
			c.GhostscriptPath = path
			return
		}
	}
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