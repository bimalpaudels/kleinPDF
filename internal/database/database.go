package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database handles database operations
type Database struct {
	db *gorm.DB
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}

	// Auto-migrate the schema
	err = db.AutoMigrate(&UserPreferences{})
	if err != nil {
		return nil, err
	}

	return database, nil
}

// GetPreferences gets the current user preferences
func (d *Database) GetPreferences() (*UserPreferencesData, error) {
	prefs, err := d.getOrCreatePreferences()
	if err != nil {
		return nil, err
	}

	prefsData := prefs.GetPreferences()
	return &prefsData, nil
}

// UpdatePreferences updates user preferences
func (d *Database) UpdatePreferences(data map[string]interface{}) error {
	prefs, err := d.getOrCreatePreferences()
	if err != nil {
		return err
	}

	currentPrefs := prefs.GetPreferences()

	// Update fields from request data
	if val, ok := data["default_compression_level"]; ok {
		if level, ok := val.(string); ok {
			currentPrefs.DefaultCompressionLevel = level
		}
	}

	if val, ok := data["advanced_options_expanded"]; ok {
		if expanded, ok := val.(bool); ok {
			currentPrefs.AdvancedOptionsExpanded = expanded
		}
	}

	if val, ok := data["image_dpi"]; ok {
		if dpi, ok := val.(float64); ok {
			currentPrefs.ImageDPI = int(dpi)
		}
	}

	if val, ok := data["image_quality"]; ok {
		if quality, ok := val.(float64); ok {
			currentPrefs.ImageQuality = int(quality)
		}
	}

	if val, ok := data["pdf_version"]; ok {
		if version, ok := val.(string); ok {
			currentPrefs.PDFVersion = version
		}
	}

	if val, ok := data["remove_metadata"]; ok {
		if remove, ok := val.(bool); ok {
			currentPrefs.RemoveMetadata = remove
		}
	}

	if val, ok := data["embed_fonts"]; ok {
		if embed, ok := val.(bool); ok {
			currentPrefs.EmbedFonts = embed
		}
	}

	if val, ok := data["generate_thumbnails"]; ok {
		if generate, ok := val.(bool); ok {
			currentPrefs.GenerateThumbnails = generate
		}
	}

	if val, ok := data["convert_to_grayscale"]; ok {
		if convert, ok := val.(bool); ok {
			currentPrefs.ConvertToGrayscale = convert
		}
	}

	// Save updated preferences
	if err := prefs.SetPreferences(currentPrefs); err != nil {
		return err
	}

	return d.db.Save(prefs).Error
}

// getOrCreatePreferences gets existing preferences or creates default ones
func (d *Database) getOrCreatePreferences() (*UserPreferences, error) {
	var prefs UserPreferences

	// Try to get existing preferences with ID = 1
	result := d.db.First(&prefs, 1)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create default preferences
			prefs = UserPreferences{
				ID: 1,
			}

			defaultPrefs := DefaultPreferences()
			if err := prefs.SetPreferences(defaultPrefs); err != nil {
				return nil, err
			}

			if err := d.db.Create(&prefs).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}

	return &prefs, nil
}