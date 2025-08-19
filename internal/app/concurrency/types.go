package concurrency

import (
	"context"

	"kleinpdf/internal/compression"
)

// WorkItem represents a single file to be processed
type WorkItem struct {
	ID       string
	FilePath string
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

// ProcessorFunc defines the function signature for processing a single file
type ProcessorFunc func(fileID, filePath, compressionLevel string, advancedOptions *compression.CompressionOptions, workerID int) (*FileResult, error)

// ConcurrentRequest represents a request to process multiple files concurrently
type ConcurrentRequest struct {
	Files            []string
	CompressionLevel string
	AdvancedOptions  *compression.CompressionOptions
}

// ConcurrentResult represents the result of concurrent processing operation
type ConcurrentResult struct {
	Results                 []FileResult `json:"results"`
	TotalFiles              int          `json:"total_files"`
	TotalOriginalSize       int64        `json:"total_original_size"`
	TotalCompressedSize     int64        `json:"total_compressed_size"`
	OverallCompressionRatio float64      `json:"overall_compression_ratio"`
	Success                 bool         `json:"success"`
	Error                   string       `json:"error,omitempty"`
}

// WorkerPool represents a pool of workers for concurrent processing
type WorkerPool struct {
	ctx           context.Context
	maxWorkers    int
	processor     ProcessorFunc
	workChan      chan WorkItem
	resultChan    chan *FileResult
	totalFiles    int
}