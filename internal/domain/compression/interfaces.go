package compression

import (
	"context"
)

type PDFProcessor interface {
	CompressPDF(inputPath, outputPath, compressionLevel string, options *CompressionOptions) error
	GetGhostscriptPath() string
	IsGhostscriptAvailable() bool
}

type Service interface {
	CompressPDF(ctx context.Context, request CompressionRequest) CompressionResponse
	ProcessFileData(ctx context.Context, fileData []FileUpload) CompressionResponse
}

type FileManager interface {
	CopyFile(src, dst string) error
	SaveFileToDownloadFolder(result FileResult, downloadFolder string) (string, error)
}

type ProgressNotifier interface {
	EmitFileProgress(fileID, filename, status string, progress float64, workerID int, err error)
	EmitFileCompleted(result FileResult)
	EmitCompressionProgress(percent float64, current, total int)
}
