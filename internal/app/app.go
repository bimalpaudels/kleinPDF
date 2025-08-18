package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"kleinpdf/internal/common"
	"kleinpdf/internal/compression"
	"kleinpdf/internal/database"
)

// NewApp creates a new application instance
func NewApp() *App {
	return &App{}
}

// OnStartup is called when the app context is ready
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	a.config = NewConfig()

	// Initialize database
	db, err := database.NewDatabase(a.config.DatabasePath)
	if err != nil {
		a.config.Logger.Error("Failed to initialize database", "error", err)
		return
	}
	a.db = db

	// Initialize compressor
	a.compressor = compression.NewCompressor(a.config.GhostscriptPath, a.config.Logger)

	// Initialize stats
	a.stats = &AppStats{}

	a.config.Logger.Info("Wails app initialized successfully")
	a.config.Logger.Info("Application configuration",
		"database_path", a.config.DatabasePath,
		"ghostscript_available", a.compressor.IsAvailable())
}

// CompressPDF handles PDF compression requests
func (a *App) CompressPDF(request CompressionRequest) CompressionResponse {
	// Validate input
	if len(request.Files) == 0 {
		a.config.Logger.Error("Compression request validation failed", "error", "no files provided")
		return CompressionResponse{
			Success: false,
			Error:   "no files provided",
		}
	}

	// Resolve compression level
	compressionLevel, err := a.resolveCompressionLevel(request.CompressionLevel)
	if err != nil {
		a.config.Logger.Error("Failed to resolve compression level", "error", err)
		return CompressionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to resolve compression level: %v", err),
		}
	}

	totalFiles := len(request.Files)
	maxConcurrency := runtime.NumCPU()
	if maxConcurrency > common.MaxConcurrencyLimit {
		maxConcurrency = common.MaxConcurrencyLimit
	}

	// Create file work items with unique IDs
	type fileWork struct {
		ID       string
		FilePath string
	}

	var fileWorkItems []fileWork
	for _, filePath := range request.Files {
		fileWorkItems = append(fileWorkItems, fileWork{
			ID:       common.GenerateUUID(),
			FilePath: filePath,
		})
	}

	// Use channels to coordinate concurrent processing
	workChan := make(chan fileWork, totalFiles)
	resultChan := make(chan *FileResult, totalFiles)

	// Fill the work channel
	for _, work := range fileWorkItems {
		workChan <- work
	}
	close(workChan)

	// Start concurrent workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency && i < totalFiles; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for work := range workChan {
				// Check for context cancellation
				select {
				case <-a.ctx.Done():
					a.config.Logger.Info("Compression cancelled by context", "worker_id", workerID)
					return
				default:
				}

				result, err := a.processSingleFile(work.ID, work.FilePath, compressionLevel, request.AdvancedOptions, workerID)
				if err != nil {
					a.config.Logger.Error("Error processing file",
						"file", work.FilePath,
						"worker_id", workerID,
						"error", err)

					// Send error result
					errorResult := &FileResult{
						FileID:           work.ID,
						OriginalFilename: filepath.Base(work.FilePath),
						Status:           "error",
						Error:            err.Error(),
					}
					resultChan <- errorResult
				} else {
					result.Status = "completed"
					resultChan <- result
				}
			}
		}(i)
	}

	// Wait for all workers and close channels
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results as they stream in
	var results []FileResult
	var totalOriginalSize, totalCompressedSize int64
	completed := 0

	for result := range resultChan {
		results = append(results, *result)
		if result.Status == "completed" {
			totalOriginalSize += result.OriginalSize
			totalCompressedSize += result.CompressedSize
		}
		completed++
	}

	// Calculate overall compression ratio
	overallCompressionRatio := float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100

	// Update statistics
	a.stats.SessionFilesCompressed += completed
	a.stats.SessionDataSaved += totalOriginalSize - totalCompressedSize
	a.stats.TotalFilesCompressed += int64(completed)
	a.stats.TotalDataSaved += totalOriginalSize - totalCompressedSize

	return CompressionResponse{
		Success:                 true,
		Files:                   results,
		TotalFiles:              len(results),
		TotalOriginalSize:       totalOriginalSize,
		TotalCompressedSize:     totalCompressedSize,
		OverallCompressionRatio: overallCompressionRatio,
		CompressionLevel:        compressionLevel,
	}
}

// ProcessFileData handles file data uploads
func (a *App) ProcessFileData(fileData []FileUpload) CompressionResponse {
	if len(fileData) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "no files provided",
		}
	}

	// Extract file paths
	var filePaths []string
	for _, file := range fileData {
		filePaths = append(filePaths, file.Name)
	}

	// Create request
	request := CompressionRequest{
		Files:            filePaths,
		CompressionLevel: common.DefaultCompressionLevel,
		AdvancedOptions:  nil,
	}

	// Load preferences for compression level
	prefs, err := a.db.GetPreferences()
	if err == nil && prefs != nil {
		request.CompressionLevel = prefs.DefaultCompressionLevel
	}

	return a.CompressPDF(request)
}

// GetAppStatus returns application status information
func (a *App) GetAppStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":                "running",
		"framework":             "Wails + Preact",
		"app_name":              "KleinPDF",
		"ghostscript_path":      a.compressor.GetGhostscriptPath(),
		"ghostscript_available": a.compressor.IsAvailable(),
	}
}

// GetStats returns application statistics
func (a *App) GetStats() *AppStats {
	return a.stats
}

// processSingleFile processes a single PDF file
func (a *App) processSingleFile(fileID, filePath, compressionLevel string, advancedOptions *compression.CompressionOptions, _ int) (*FileResult, error) {
	filename := filepath.Base(filePath)

	// Create timestamp-based filename for compressed file
	timestamp := time.Now().UTC().Format("20060102_150405")
	baseName := strings.TrimSuffix(filename, ".pdf")
	compressedFilename := fmt.Sprintf("%s_%s_compressed.pdf", baseName, timestamp)

	// Generate output path in the same directory as input
	inputDir := filepath.Dir(filePath)
	compressedPath := filepath.Join(inputDir, compressedFilename)

	// Check for context cancellation before compression
	select {
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	default:
	}

	// Direct compression
	err := a.compressor.CompressFile(filePath, compressedPath, compressionLevel, advancedOptions)
	if err != nil {
		return nil, err
	}

	// Get file sizes for statistics
	originalInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	compressedInfo, err := os.Stat(compressedPath)
	if err != nil {
		return nil, err
	}

	originalSize := originalInfo.Size()
	compressedSize := compressedInfo.Size()
	compressionRatio := float64(originalSize-compressedSize) / float64(originalSize) * 100

	return &FileResult{
		FileID:             fileID,
		OriginalFilename:   filename,
		CompressedFilename: compressedFilename,
		OriginalSize:       originalSize,
		CompressedSize:     compressedSize,
		CompressionRatio:   compressionRatio,
		CompressedPath:     compressedPath,
	}, nil
}

// resolveCompressionLevel resolves the compression level from request or preferences
func (a *App) resolveCompressionLevel(requestedLevel string) (string, error) {
	if requestedLevel != "" {
		return requestedLevel, nil
	}

	prefs, err := a.db.GetPreferences()
	if err != nil {
		a.config.Logger.Warn("Failed to load preferences, using default compression level", "error", err)
		return common.DefaultCompressionLevel, nil
	}

	if prefs == nil {
		a.config.Logger.Debug("No preferences found, using default compression level")
		return common.DefaultCompressionLevel, nil
	}

	return prefs.DefaultCompressionLevel, nil
}