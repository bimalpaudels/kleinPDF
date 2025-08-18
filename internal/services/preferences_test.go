package services

import (
	"testing"

	"kleinpdf/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.UserPreferences{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestNewPreferencesService(t *testing.T) {
	db := setupTestDB(t)
	service := NewPreferencesService(db)

	if service == nil {
		t.Fatal("Expected PreferencesService instance, got nil")
	}

	if service.db != db {
		t.Error("Expected database to be set correctly")
	}
}

func TestGetPreferences_CreatesDefault(t *testing.T) {
	db := setupTestDB(t)
	service := NewPreferencesService(db)

	prefs, err := service.GetPreferences()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if prefs == nil {
		t.Fatal("Expected preferences, got nil")
	}

	// Should have default values
	expectedLevel := "good_enough"
	if prefs.DefaultCompressionLevel != expectedLevel {
		t.Errorf("Expected default compression level %s, got %s", expectedLevel, prefs.DefaultCompressionLevel)
	}

	expectedDPI := 150
	if prefs.ImageDPI != expectedDPI {
		t.Errorf("Expected default ImageDPI %d, got %d", expectedDPI, prefs.ImageDPI)
	}
}

func TestUpdatePreferences(t *testing.T) {
	db := setupTestDB(t)
	service := NewPreferencesService(db)

	// First get initial preferences to create the record
	_, err := service.GetPreferences()
	if err != nil {
		t.Fatalf("Failed to initialize preferences: %v", err)
	}

	// Update some preferences
	updateData := map[string]interface{}{
		"default_compression_level": "ultra",
		"image_dpi":                 float64(300),
		"auto_download_enabled":     true,
	}

	err = service.UpdatePreferences(updateData)
	if err != nil {
		t.Fatalf("Expected no error updating preferences, got %v", err)
	}

	// Verify the updates
	prefs, err := service.GetPreferences()
	if err != nil {
		t.Fatalf("Failed to get updated preferences: %v", err)
	}

	if prefs.DefaultCompressionLevel != "ultra" {
		t.Errorf("Expected compression level to be updated to 'ultra', got %s", prefs.DefaultCompressionLevel)
	}

	if prefs.ImageDPI != 300 {
		t.Errorf("Expected ImageDPI to be updated to 300, got %d", prefs.ImageDPI)
	}

	if !prefs.AutoDownloadEnabled {
		t.Error("Expected AutoDownloadEnabled to be true")
	}
}

func TestUpdatePreferences_InvalidFolder(t *testing.T) {
	db := setupTestDB(t)
	service := NewPreferencesService(db)

	// Initialize preferences
	_, err := service.GetPreferences()
	if err != nil {
		t.Fatalf("Failed to initialize preferences: %v", err)
	}

	// Try to update with invalid folder path
	updateData := map[string]interface{}{
		"default_download_folder": "/invalid/path/that/cannot/be/created\x00", // Invalid path with null byte
	}

	err = service.UpdatePreferences(updateData)
	// This should not fail the entire operation, just skip the invalid folder
	if err != nil {
		t.Fatalf("Expected no error even with invalid folder, got %v", err)
	}

	// Verify the folder was not updated
	prefs, err := service.GetPreferences()
	if err != nil {
		t.Fatalf("Failed to get preferences: %v", err)
	}

	// Should still be empty since the invalid path couldn't be created
	if prefs.DefaultDownloadFolder != "" {
		t.Errorf("Expected folder to remain empty due to invalid path, got %s", prefs.DefaultDownloadFolder)
	}
}

func TestGetDownloadFolder_Default(t *testing.T) {
	db := setupTestDB(t)
	service := NewPreferencesService(db)

	folder, err := service.GetDownloadFolder()
	if err != nil {
		t.Fatalf("Expected no error getting download folder, got %v", err)
	}

	if folder == "" {
		t.Error("Expected non-empty download folder path")
	}

	// Should contain "Downloads" for default behavior
	if !contains(folder, "Downloads") {
		t.Errorf("Expected default folder to contain 'Downloads', got %s", folder)
	}
}
