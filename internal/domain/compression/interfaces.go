package compression

import (
	"context"
)

// PDFProcessor defines the interface for PDF compression operations
type PDFProcessor interface {
	CompressPDF(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error
	GetGhostscriptPath() string
	IsGhostscriptAvailable() bool
}

// Service defines the domain service for compression operations
type Service interface {
	CompressPDF(ctx context.Context, request CompressionRequest) CompressionResponse
	ProcessFileData(ctx context.Context, fileData []FileUpload) CompressionResponse
}

// FileManager defines file operation capabilities
type FileManager interface {
	CopyFile(src, dst string) error
	SaveFileToDownloadFolder(result FileResult, downloadFolder string) (string, error)
}

// ProgressNotifier defines progress notification capabilities  
type ProgressNotifier interface {
	EmitFileProgress(fileID, filename, status string, progress float64, workerID int, err error)
	EmitFileCompleted(result FileResult)
	EmitCompressionProgress(percent float64, current, total int)
}