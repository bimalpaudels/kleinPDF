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

	"github.com/panjf2000/ants/v2"
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

	// Calculate optimal worker count
	maxConcurrency := runtime.NumCPU()
	if maxConcurrency > common.MaxConcurrencyLimit {
		maxConcurrency = common.MaxConcurrencyLimit
	}

	// Create ants pool
	pool, err := ants.NewPool(maxConcurrency)
	if err != nil {
		a.config.Logger.Error("Failed to create worker pool", "error", err)
		return CompressionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create worker pool: %v", err),
		}
	}
	defer pool.Release()

	// Prepare for concurrent processing
	totalFiles := len(request.Files)
	results := make([]*FileResult, totalFiles)
	var wg sync.WaitGroup
	
	// Process files concurrently using ants
	for i, filePath := range request.Files {
		wg.Add(1)
		
		// Capture variables for goroutine
		index := i
		file := filePath
		
		err := pool.Submit(func() {
			defer wg.Done()
			
			// Check for context cancellation
			select {
			case <-a.ctx.Done():
				a.config.Logger.Info("Compression cancelled by context", "file", file)
				return
			default:
			}

			fileID := common.GenerateUUID()
			result, err := a.processSingleFile(fileID, file, compressionLevel, request.AdvancedOptions, index)
			
			if err != nil {
				a.config.Logger.Error("Error processing file", "file", file, "worker_id", index, "error", err)
				// Create error result
				results[index] = &FileResult{
					FileID:           fileID,
					OriginalFilename: filepath.Base(file),
					Status:           "error",
					Error:            err.Error(),
				}
			} else {
				result.Status = "completed"
				results[index] = result
			}
		})
		
		if err != nil {
			wg.Done() // Decrement since Submit failed
			a.config.Logger.Error("Failed to submit task", "file", filePath, "error", err)
			results[i] = &FileResult{
				FileID:           common.GenerateUUID(),
				OriginalFilename: filepath.Base(filePath),
				Status:           "error",
				Error:            err.Error(),
			}
		}
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Collect and aggregate results
	var finalResults []FileResult
	var totalOriginalSize, totalCompressedSize int64
	completed := 0

	for _, result := range results {
		if result != nil {
			finalResults = append(finalResults, *result)
			if result.Status == "completed" {
				totalOriginalSize += result.OriginalSize
				totalCompressedSize += result.CompressedSize
			}
			completed++
		}
	}

	// Calculate overall compression ratio
	var overallCompressionRatio float64
	if totalOriginalSize > 0 {
		overallCompressionRatio = float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100
	}

	// Update statistics
	dataSaved := totalOriginalSize - totalCompressedSize
	a.stats.SessionFilesCompressed += completed
	a.stats.SessionDataSaved += dataSaved
	a.stats.TotalFilesCompressed += int64(completed)
	a.stats.TotalDataSaved += dataSaved

	return CompressionResponse{
		Success:                 true,
		Files:                   finalResults,
		TotalFiles:              len(finalResults),
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
func (a *App) processSingleFile(fileID, filePath, compressionLevel string, advancedOptions *compression.CompressionOptions, workerID int) (*FileResult, error) {
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
		a.config.Logger.Error("Error processing file",
			"file", filePath,
			"worker_id", workerID,
			"error", err)
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