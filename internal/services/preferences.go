package services

import (
	"os"
	"path/filepath"

	"pdf-compressor-wails/internal/models"

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
	if val, ok := data["default_download_folder"]; ok {
		if folder, ok := val.(string); ok && folder != "" {
			// Validate and create the path if needed
			folderPath := filepath.Clean(folder)
			if err := os.MkdirAll(folderPath, 0755); err == nil {
				currentPrefs.DefaultDownloadFolder = folderPath
			}
		} else {
			currentPrefs.DefaultDownloadFolder = ""
		}
	}
	
	if val, ok := data["default_compression_level"]; ok {
		if level, ok := val.(string); ok {
			currentPrefs.DefaultCompressionLevel = level
		}
	}
	
	if val, ok := data["auto_download_enabled"]; ok {
		if enabled, ok := val.(bool); ok {
			currentPrefs.AutoDownloadEnabled = enabled
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

// GetDownloadFolder gets the default download folder, creating it if needed
func (s *PreferencesService) GetDownloadFolder() (string, error) {
	prefs, err := models.GetOrCreatePreferences(s.db)
	if err != nil {
		return "", err
	}
	
	prefsData := prefs.GetPreferences()
	
	var downloadDir string
	if prefsData.DefaultDownloadFolder != "" {
		downloadDir = prefsData.DefaultDownloadFolder
	} else {
		// Use default Downloads folder
		homeDir, _ := os.UserHomeDir()
		downloadDir = filepath.Join(homeDir, "Downloads")
	}
	
	// Ensure the directory exists
	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return "", err
	}
	
	return downloadDir, nil
}