package compression

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