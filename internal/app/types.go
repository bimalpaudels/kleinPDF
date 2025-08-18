package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// App represents the main application structure
type App struct {
	ctx         context.Context
	config      *Config
	db          *gorm.DB
	preferences *UserPreferences
	stats       *AppStats
}

// Config holds application configuration
type Config struct {
	DatabasePath    string
	GhostscriptPath string
	Logger          *slog.Logger
}

// CompressionOptions holds advanced compression options for PDF processing
type CompressionOptions struct {
	ImageDPI           int    `json:"image_dpi"`
	ImageQuality       int    `json:"image_quality"`
	PDFVersion         string `json:"pdf_version"`
	RemoveMetadata     bool   `json:"remove_metadata"`
	EmbedFonts         bool   `json:"embed_fonts"`
	GenerateThumbnails bool   `json:"generate_thumbnails"`
	ConvertToGrayscale bool   `json:"convert_to_grayscale"`
}

// DefaultCompressionOptions returns default compression options
func DefaultCompressionOptions() CompressionOptions {
	return CompressionOptions{
		ImageDPI:           150,
		ImageQuality:       85,
		PDFVersion:         "1.4",
		RemoveMetadata:     false,
		EmbedFonts:         true,
		GenerateThumbnails: false,
		ConvertToGrayscale: false,
	}
}

// CompressionRequest represents a PDF compression request
type CompressionRequest struct {
	Files            []string            `json:"files"`
	CompressionLevel string              `json:"compressionLevel"`
	AdvancedOptions  *CompressionOptions `json:"advancedOptions"`
}

// CompressionResponse represents the result of a compression operation
type CompressionResponse struct {
	Success                 bool         `json:"success"`
	Files                   []FileResult `json:"files"`
	TotalFiles              int          `json:"total_files"`
	TotalOriginalSize       int64        `json:"total_original_size"`
	TotalCompressedSize     int64        `json:"total_compressed_size"`
	OverallCompressionRatio float64      `json:"overall_compression_ratio"`
	CompressionLevel        string       `json:"compression_level"`
	Error                   string       `json:"error,omitempty"`
}

// FileResult represents the result of compressing a single file
type FileResult struct {
	FileID             string  `json:"file_id"`
	OriginalFilename   string  `json:"original_filename"`
	CompressedFilename string  `json:"compressed_filename"`
	OriginalSize       int64   `json:"original_size"`
	CompressedSize     int64   `json:"compressed_size"`
	CompressionRatio   float64 `json:"compression_ratio"`
	CompressedPath     string  `json:"compressed_path"`
	Status             string  `json:"status"`
	Error              string  `json:"error,omitempty"`
}

// FileUpload represents uploaded file data
type FileUpload struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
	Size int64  `json:"size"`
}

// AppStats holds application statistics
type AppStats struct {
	TotalFilesCompressed   int64 `json:"total_files_compressed"`
	TotalDataSaved         int64 `json:"total_data_saved"`
	SessionFilesCompressed int   `json:"session_files_compressed"`
	SessionDataSaved       int64 `json:"session_data_saved"`
}

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