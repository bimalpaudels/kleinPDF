package database

import (
	"encoding/json"
	"time"
)

// UserPreferences database model
type UserPreferences struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PreferencesJSON string    `gorm:"type:text" json:"preferences_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserPreferencesData represents user preferences data
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

// DefaultPreferences returns default user preferences
func DefaultPreferences() UserPreferencesData {
	return UserPreferencesData{
		DefaultCompressionLevel: "good_enough",
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

// GetPreferences returns the user preferences data
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

// SetPreferences sets the user preferences data
func (up *UserPreferences) SetPreferences(prefs UserPreferencesData) error {
	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	up.PreferencesJSON = string(data)
	return nil
}