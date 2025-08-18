package app

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitDatabase initializes and returns a GORM database instance
func (a *App) InitDatabase() error {
	db, err := gorm.Open(sqlite.Open(a.config.DatabasePath), &gorm.Config{})
	if err != nil {
		return err
	}

	a.db = db

	// Auto-migrate the schema
	err = a.db.AutoMigrate(&UserPreferences{})
	if err != nil {
		return err
	}

	return nil
}

// GetPreferences gets the current user preferences
func (a *App) GetPreferences() (*UserPreferencesData, error) {
	prefs, err := a.getOrCreatePreferences()
	if err != nil {
		return nil, err
	}

	prefsData := prefs.GetPreferences()
	return &prefsData, nil
}

// UpdatePreferences updates user preferences
func (a *App) UpdatePreferences(data map[string]interface{}) error {
	prefs, err := a.getOrCreatePreferences()
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

	return a.db.Save(prefs).Error
}

// getOrCreatePreferences gets existing preferences or creates default ones
func (a *App) getOrCreatePreferences() (*UserPreferences, error) {
	var prefs UserPreferences

	// Try to get existing preferences with ID = 1
	result := a.db.First(&prefs, 1)

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

			if err := a.db.Create(&prefs).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}

	return &prefs, nil
}