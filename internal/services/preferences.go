package services

import (
	"kleinpdf/internal/models"

	"gorm.io/gorm"
)

// PreferencesService handles user preferences operations
type PreferencesService struct {
	db *gorm.DB
}

// NewPreferencesService creates a new preferences service
func NewPreferencesService(db *gorm.DB) *PreferencesService {
	return &PreferencesService{db: db}
}

// GetPreferences gets the current user preferences
func (s *PreferencesService) GetPreferences() (*models.UserPreferencesData, error) {
	prefs, err := models.GetOrCreatePreferences(s.db)
	if err != nil {
		return nil, err
	}
	
	prefsData := prefs.GetPreferences()
	return &prefsData, nil
}

// UpdatePreferences updates user preferences
func (s *PreferencesService) UpdatePreferences(data map[string]interface{}) error {
	prefs, err := models.GetOrCreatePreferences(s.db)
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
	
	return s.db.Save(prefs).Error
}

