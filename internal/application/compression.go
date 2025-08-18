package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"kleinpdf/internal/config"
	"kleinpdf/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type CompressionHandler struct {
	ctx          context.Context
	config       *config.Config
	pdfService   *services.PDFService
	prefsService *services.PreferencesService
	filesHandler *FilesHandler
	statsManager *StatsManager
}

func NewCompressionHandler(
	ctx context.Context,
	config *config.Config,
	pdfService *services.PDFService,
	prefsService *services.PreferencesService,
	filesHandler *FilesHandler,
	statsManager *StatsManager,
) *CompressionHandler {
	return &CompressionHandler{
		ctx:          ctx,
		config:       config,
		pdfService:   pdfService,
		prefsService: prefsService,
		filesHandler: filesHandler,
		statsManager: statsManager,
	}
}

func (h *CompressionHandler) CompressPDF(request CompressionRequest) CompressionResponse {
	return h.CompressPDFWithContext(h.ctx, request)
}

func (h *CompressionHandler) CompressPDFWithContext(ctx context.Context, request CompressionRequest) CompressionResponse {
	// Validate input
	if len(request.Files) == 0 {
		h.config.Logger.Error("Compression request validation failed", "error", "no files provided")
		return CompressionResponse{
			Success: false,
			Error:   ErrNoFilesProvided.Error(),
		}
	}


	// Use compression level from preferences if not specified
	compressionLevel, err := h.resolveCompressionLevel(request.CompressionLevel)
	if err != nil {
		h.config.Logger.Error("Failed to resolve compression level", "error", err)
		return CompressionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to resolve compression level: %v", err),
		}
	}

	totalFiles := len(request.Files)
	// Use available CPU cores, but cap at reasonable limit for I/O intensive operations
	maxConcurrency := runtime.NumCPU()
	if maxConcurrency > MaxConcurrencyLimit {
		maxConcurrency = MaxConcurrencyLimit 
	}

	// Create file work items with unique IDs
	type fileWork struct {
		ID       string
		FilePath string
	}

	var fileWorkItems []fileWork
	for _, filePath := range request.Files {
		fileWorkItems = append(fileWorkItems, fileWork{
			ID:       GenerateUUID(),
			FilePath: filePath,
		})
	}

	// Use channels to coordinate concurrent processing
	workChan := make(chan fileWork, totalFiles)
	resultChan := make(chan *FileResult, totalFiles)
	completedCount := make(chan int, totalFiles)

	// Fill the work channel
	for _, work := range fileWorkItems {
		workChan <- work

		// Emit initial file status
		wailsruntime.EventsEmit(h.ctx, EventFileProgress, FileProgressUpdate{
			FileID:   work.ID,
			Filename: filepath.Base(work.FilePath),
			Status:   "queued",
			Progress: 0,
		})
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
				case <-ctx.Done():
					h.config.Logger.Info("Compression cancelled by context", "worker_id", workerID)
					return
				default:
				}
				
				result, err := h.processSingleFileWithProgress(ctx, work.ID, work.FilePath, compressionLevel, request.AdvancedOptions, workerID)
				if err != nil {
					compressionErr := NewCompressionError("processing", work.FilePath, err)
					h.config.Logger.Error("Error processing file", 
						"file", work.FilePath, 
						"worker_id", workerID,
						"error", compressionErr)

					// Emit error status for this file
					wailsruntime.EventsEmit(h.ctx, EventFileProgress, FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "error",
						Progress: 0,
						WorkerID: workerID,
						Error:    err.Error(),
					})

					// Send error result
					errorResult := &FileResult{
						FileID:           work.ID,
						OriginalFilename: filepath.Base(work.FilePath),
						Status:           "error",
						Error:            compressionErr.Error(),
					}
					resultChan <- errorResult
				} else {
					// Emit completion status
					wailsruntime.EventsEmit(h.ctx, EventFileProgress, FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "completed",
						Progress: 100,
						WorkerID: workerID,
					})

					result.Status = "completed"
					resultChan <- result

					// Stream individual file result immediately
					wailsruntime.EventsEmit(h.ctx, EventFileCompleted, result)
				}
				completedCount <- 1
			}
		}(i)
	}

	// Wait for all workers and close channels
	go func() {
		wg.Wait()
		close(resultChan)
		close(completedCount)
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
		// Emit overall progress
		overallProgress := float64(completed) / float64(totalFiles) * 100
		wailsruntime.EventsEmit(h.ctx, EventCompressionProgress, map[string]any{
			"percent":   overallProgress,
			"current":   completed,
			"total":     totalFiles,
			"completed": completed,
		})
	}

	// Final progress update
	wailsruntime.EventsEmit(h.ctx, EventCompressionProgress, map[string]interface{}{
		"percent": 100.0,
		"current": totalFiles,
		"total":   totalFiles,
		"file":    "Complete",
	})

	// Calculate overall compression ratio
	overallCompressionRatio := float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100
	dataSaved := totalOriginalSize - totalCompressedSize

	// Update stats
	h.statsManager.UpdateStats(len(results), dataSaved)

	response := CompressionResponse{
		Success:                 true,
		Files:                   results,
		TotalFiles:              len(results),
		TotalOriginalSize:       totalOriginalSize,
		TotalCompressedSize:     totalCompressedSize,
		OverallCompressionRatio: overallCompressionRatio,
		CompressionLevel:        compressionLevel,
		AutoDownload:            request.AutoDownload,
	}

	// Handle auto-download if enabled
	if request.AutoDownload {
		var downloadPaths []string
		for i, result := range results {
			downloadPath, err := h.filesHandler.SaveFileToDownloadFolder(result, request.DownloadFolder)
			if err != nil {
				h.config.Logger.Error("Error saving file", 
					"filename", result.OriginalFilename,
					"error", err)
				continue
			}
			downloadPaths = append(downloadPaths, downloadPath)
			// Update the result with saved path
			results[i].SavedPath = &downloadPath
		}
		response.Files = results
		response.DownloadPaths = downloadPaths
	}

	return response
}

func (h *CompressionHandler) processSingleFileWithProgress(ctx context.Context, fileID, filePath, compressionLevel string, advancedOptions *services.CompressionOptions, workerID int) (*FileResult, error) {
	filename := filepath.Base(filePath)

	// Emit compression status - no copying needed, go straight to compression
	wailsruntime.EventsEmit(h.ctx, EventFileProgress, FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "compressing",
		Progress: DefaultProgressPercent,
		WorkerID: workerID,
	})

	// Create timestamp-based filename for compressed file
	timestamp := time.Now().UTC().Format("20060102_150405")
	baseName := strings.TrimSuffix(filename, ".pdf")
	compressedFilename := fmt.Sprintf("%s_%s_compressed.pdf", baseName, timestamp)

	// Generate output path in the same directory as input (for direct processing)
	inputDir := filepath.Dir(filePath)
	compressedPath := filepath.Join(inputDir, compressedFilename)

	// Check for context cancellation before compression
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Direct compression: GS reads from original file, writes to output path
	err := h.pdfService.CompressPDF(filePath, compressedPath, compressionLevel, advancedOptions)
	if err != nil {
		return nil, err
	}

	// Emit completion status
	wailsruntime.EventsEmit(h.ctx, EventFileProgress, FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "completed",
		Progress: CompletedProgressPercent,
		WorkerID: workerID,
	})

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
		TempPath:           compressedPath, // Now points to the final output location
	}, nil
}

// ProcessFileData handles PDF compression from file data (drag & drop with actual file paths)
func (h *CompressionHandler) ProcessFileData(fileData []FileUpload) CompressionResponse {
	// Validate input
	if len(fileData) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "No files provided",
		}
	}

	// Extract file paths directly from the uploaded data (assuming fileData contains actual paths)
	var filePaths []string
	for _, file := range fileData {
		// Use the file name as the path (for drag & drop, this should be the actual file path)
		filePaths = append(filePaths, file.Name)
	}

	// Use the regular CompressPDF logic with direct paths
	request := CompressionRequest{
		Files:            filePaths,
		CompressionLevel: DefaultCompressionLevel,
		AutoDownload:     false,
		DownloadFolder:   "",
		AdvancedOptions:  nil,
	}

	// Load preferences for compression level
	prefs, err := h.prefsService.GetPreferences()
	if err == nil && prefs != nil {
		request.CompressionLevel = prefs.DefaultCompressionLevel
	}

	// Process using the direct compression logic
	return h.CompressPDF(request)
}

// resolveCompressionLevel determines the appropriate compression level
func (h *CompressionHandler) resolveCompressionLevel(requestedLevel string) (string, error) {
	if requestedLevel != "" {
		return requestedLevel, nil
	}
	
	prefs, err := h.prefsService.GetPreferences()
	if err != nil {
		h.config.Logger.Warn("Failed to load preferences, using default compression level", "error", err)
		return DefaultCompressionLevel, nil
	}
	
	if prefs == nil {
		h.config.Logger.Debug("No preferences found, using default compression level")
		return DefaultCompressionLevel, nil
	}
	
	return prefs.DefaultCompressionLevel, nil
}