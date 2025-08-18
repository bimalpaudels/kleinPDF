package application

import (
	"context"
	"fmt"
	"log"
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
	// Validate input
	if len(request.Files) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "No files provided",
		}
	}

	// Clean up old temp files
	CleanupOldTempFiles(h.config.WorkingDir)

	// Use compression level from preferences if not specified
	compressionLevel := request.CompressionLevel
	if compressionLevel == "" {
		prefs, err := h.prefsService.GetPreferences()
		if err == nil && prefs != nil {
			compressionLevel = prefs.DefaultCompressionLevel
		} else {
			compressionLevel = "good_enough"
		}
	}

	totalFiles := len(request.Files)
	// Use available CPU cores, but cap at reasonable limit for I/O intensive operations
	maxConcurrency := runtime.NumCPU()
	if maxConcurrency > 8 {
		maxConcurrency = 8 
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
		wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
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
				result, err := h.processSingleFileWithProgress(work.ID, work.FilePath, compressionLevel, request.AdvancedOptions, workerID)
				if err != nil {
					log.Printf("Error processing file %s: %v", work.FilePath, err)

					// Emit error status for this file
					wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
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
						Error:            err.Error(),
					}
					resultChan <- errorResult
				} else {
					// Emit completion status
					wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "completed",
						Progress: 100,
						WorkerID: workerID,
					})

					result.Status = "completed"
					resultChan <- result

					// Stream individual file result immediately
					wailsruntime.EventsEmit(h.ctx, "file:completed", result)
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
		wailsruntime.EventsEmit(h.ctx, "compression:progress", map[string]any{
			"percent":   overallProgress,
			"current":   completed,
			"total":     totalFiles,
			"completed": completed,
		})
	}

	// Final progress update
	wailsruntime.EventsEmit(h.ctx, "compression:progress", map[string]interface{}{
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
				log.Printf("Error saving file %s: %v", result.OriginalFilename, err)
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

func (h *CompressionHandler) processSingleFileWithProgress(fileID, filePath, compressionLevel string, advancedOptions *services.CompressionOptions, workerID int) (*FileResult, error) {
	filename := filepath.Base(filePath)

	// Emit copying status
	wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "copying",
		Progress: 10,
		WorkerID: workerID,
	})

	// Generate temp directory
	tempDir := filepath.Join(h.config.WorkingDir, fileID)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	// Create timestamp-based filename for compressed file
	timestamp := time.Now().UTC().Format("20060102_150405")
	baseName := strings.TrimSuffix(filename, ".pdf")
	compressedFilename := fmt.Sprintf("%s_%s.pdf", baseName, timestamp)

	// Copy original file to temp directory
	originalTempPath := filepath.Join(tempDir, filename)
	if err := CopyFile(filePath, originalTempPath); err != nil {
		return nil, fmt.Errorf("failed to copy file to temp directory: %v", err)
	}

	// Emit compression status
	wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "compressing",
		Progress: 30,
		WorkerID: workerID,
	})

	// Compress the PDF
	compressedPath := filepath.Join(tempDir, compressedFilename)

	err := h.pdfService.CompressPDF(originalTempPath, compressedPath, compressionLevel, advancedOptions)
	if err != nil {
		return nil, err
	}

	// Emit finishing status
	wailsruntime.EventsEmit(h.ctx, "file:progress", FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "finalizing",
		Progress: 90,
		WorkerID: workerID,
	})

	// Get file sizes
	originalInfo, err := os.Stat(originalTempPath)
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
		TempPath:           compressedPath,
	}, nil
}

// ProcessFileData handles PDF compression from file data instead of file paths
func (h *CompressionHandler) ProcessFileData(fileData []FileUpload) CompressionResponse {
	// Validate input
	if len(fileData) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "No files provided",
		}
	}

	// Write files to temp directory first
	filePaths, err := h.filesHandler.WriteFilesToTemp(fileData)
	if err != nil {
		return CompressionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to prepare files: %v", err),
		}
	}

	// Use the regular CompressPDF logic but adjust progress to account for preparation phase (20%)
	request := CompressionRequest{
		Files:            filePaths,
		CompressionLevel: "good_enough",
		AutoDownload:     false,
		DownloadFolder:   "",
		AdvancedOptions:  nil,
	}

	// Load preferences for compression level
	prefs, err := h.prefsService.GetPreferences()
	if err == nil && prefs != nil {
		request.CompressionLevel = prefs.DefaultCompressionLevel
	}

	// Process using the regular compression logic
	return h.CompressPDF(request)
}