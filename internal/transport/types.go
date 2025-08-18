package transport

import "kleinpdf/internal/services"

// Transport layer types for Wails API

type CompressionRequest struct {
	Files            []string                     `json:"files"`
	CompressionLevel string                       `json:"compressionLevel"`
	AutoDownload     bool                         `json:"autoDownload"`
	DownloadFolder   string                       `json:"downloadFolder"`
	AdvancedOptions  *services.CompressionOptions `json:"advancedOptions"`
}

type CompressionResponse struct {
	Success                 bool         `json:"success"`
	Files                   []FileResult `json:"files"`
	TotalFiles              int          `json:"total_files"`
	TotalOriginalSize       int64        `json:"total_original_size"`
	TotalCompressedSize     int64        `json:"total_compressed_size"`
	OverallCompressionRatio float64      `json:"overall_compression_ratio"`
	CompressionLevel        string       `json:"compression_level"`
	AutoDownload            bool         `json:"auto_download"`
	DownloadPaths           []string     `json:"download_paths,omitempty"`
	Error                   string       `json:"error,omitempty"`
}

type FileResult struct {
	FileID             string  `json:"file_id"`
	OriginalFilename   string  `json:"original_filename"`
	CompressedFilename string  `json:"compressed_filename"`
	OriginalSize       int64   `json:"original_size"`
	CompressedSize     int64   `json:"compressed_size"`
	CompressionRatio   float64 `json:"compression_ratio"`
	TempPath           string  `json:"temp_path"`
	SavedPath          *string `json:"saved_path,omitempty"`
	Status             string  `json:"status"`
	Error              string  `json:"error,omitempty"`
}

type FileUpload struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
	Size int64  `json:"size"`
}

type AppStats struct {
	TotalFilesCompressed   int64 `json:"total_files_compressed"`
	TotalDataSaved         int64 `json:"total_data_saved"`
	SessionFilesCompressed int   `json:"session_files_compressed"`
	SessionDataSaved       int64 `json:"session_data_saved"`
}

// Dialog interface for system dialogs
type DialogHandler interface {
	OpenFileDialog() ([]string, error)
	OpenDirectoryDialog() (string, error)
	ShowSaveDialog(filename string) (string, error)
	OpenFile(filePath string) error
}
