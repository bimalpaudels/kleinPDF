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
}

