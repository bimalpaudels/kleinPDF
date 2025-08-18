package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type UserPreferences struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PreferencesJSON string    `gorm:"type:text" json:"preferences_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserPreferencesData struct {
	DefaultCompressionLevel string `json:"default_compression_level"`
	ImageDPI                int    `json:"image_dpi"`
	ImageQuality            int    `json:"image_quality"`
	RemoveMetadata          bool   `json:"remove_metadata"`
	EmbedFonts              bool   `json:"embed_fonts"`
	GenerateThumbnails      bool   `json:"generate_thumbnails"`
	ConvertToGrayscale      bool   `json:"convert_to_grayscale"`
	PDFVersion              string `json:"pdf_version"`
	AdvancedOptionsExpanded bool   `json:"advanced_options_expanded"`
}

func DefaultPreferences() UserPreferencesData {
	return UserPreferencesData{
		DefaultCompressionLevel: "good_enough", // Keep string literal here as it's part of the model
		ImageDPI:                150,
		ImageQuality:            85,
		RemoveMetadata:          false,
		EmbedFonts:              true,
		GenerateThumbnails:      false,
		ConvertToGrayscale:      false,
		PDFVersion:              "1.4",
		AdvancedOptionsExpanded: false,
	}
}

func (up *UserPreferences) GetPreferences() UserPreferencesData {
	if up.PreferencesJSON == "" {
		return DefaultPreferences()
	}

	var prefs UserPreferencesData
	if err := json.Unmarshal([]byte(up.PreferencesJSON), &prefs); err != nil {
		return DefaultPreferences()
	}

	return prefs
}

func (up *UserPreferences) SetPreferences(prefs UserPreferencesData) error {
	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	up.PreferencesJSON = string(data)
	return nil
}

func GetOrCreatePreferences(db *gorm.DB) (*UserPreferences, error) {
	var prefs UserPreferences

	// Try to get existing preferences with ID = 1
	result := db.First(&prefs, 1)

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

			if err := db.Create(&prefs).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	}

	return &prefs, nil
}
