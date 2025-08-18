package preferences

type Repository interface {
	GetPreferences() (*UserPreferencesData, error)
	UpdatePreferences(data map[string]any) error
	GetDownloadFolder() (string, error)
}

type UserPreferencesData struct {
	DefaultDownloadFolder   string `json:"default_download_folder"`
	DefaultCompressionLevel string `json:"default_compression_level"`
	AutoDownloadEnabled     bool   `json:"auto_download_enabled"`
	ImageDPI                int    `json:"image_dpi"`
	ImageQuality            int    `json:"image_quality"`
	RemoveMetadata          bool   `json:"remove_metadata"`
	EmbedFonts              bool   `json:"embed_fonts"`
	GenerateThumbnails      bool   `json:"generate_thumbnails"`
	ConvertToGrayscale      bool   `json:"convert_to_grayscale"`
	PDFVersion              string `json:"pdf_version"`
	AdvancedOptionsExpanded bool   `json:"advanced_options_expanded"`
}

type Service interface {
	GetPreferences() (*UserPreferencesData, error)
	UpdatePreferences(data map[string]any) error
}
