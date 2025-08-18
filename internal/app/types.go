package app

import (
	"context"
	"log/slog"

	"kleinpdf/internal/compression"
	"kleinpdf/internal/database"
)

// App represents the main application structure
type App struct {
	ctx        context.Context
	config     *Config
	db         *database.Database
	compressor *compression.Compressor
	stats      *AppStats
}

// Config holds application configuration
type Config struct {
	DatabasePath    string
	GhostscriptPath string
	Logger          *slog.Logger
}


// CompressionRequest represents a PDF compression request
type CompressionRequest struct {
	Files            []string                     `json:"files"`
	CompressionLevel string                       `json:"compressionLevel"`
	AdvancedOptions  *compression.CompressionOptions `json:"advancedOptions"`
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