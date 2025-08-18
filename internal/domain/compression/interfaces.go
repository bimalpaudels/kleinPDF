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
