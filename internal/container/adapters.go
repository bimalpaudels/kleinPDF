package container

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
	compressionDomain "kleinpdf/internal/domain/compression"
	preferencesDomain "kleinpdf/internal/domain/preferences"
	statisticsDomain "kleinpdf/internal/domain/statistics"
	"kleinpdf/internal/config"
	"kleinpdf/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// PDFProcessorAdapter adapts services.PDFService to compressionDomain.PDFProcessor
type PDFProcessorAdapter struct {
	service *services.PDFService
}

func (a *PDFProcessorAdapter) CompressPDF(inputPath, outputPath, compressionLevel string, options *compressionDomain.CompressionOptions) error {
	// Convert domain options to service options
	var serviceOptions *services.CompressionOptions
	if options != nil {
		serviceOptions = &services.CompressionOptions{
			ImageDPI:           options.ImageDPI,
			ImageQuality:       options.ImageQuality,
			PDFVersion:         options.PDFVersion,
			RemoveMetadata:     options.RemoveMetadata,
			EmbedFonts:         options.EmbedFonts,
			GenerateThumbnails: options.GenerateThumbnails,
			ConvertToGrayscale: options.ConvertToGrayscale,
		}
	}
	
	return a.service.CompressPDF(inputPath, outputPath, compressionLevel, serviceOptions)
}

func (a *PDFProcessorAdapter) GetGhostscriptPath() string {
	return a.service.GetGhostscriptPath()
}

func (a *PDFProcessorAdapter) IsGhostscriptAvailable() bool {
	return a.service.IsGhostscriptAvailable()
}

// PreferencesRepositoryAdapter adapts services.PreferencesService to preferencesDomain.Repository
type PreferencesRepositoryAdapter struct {
	service *services.PreferencesService
}

func (a *PreferencesRepositoryAdapter) GetPreferences() (*preferencesDomain.UserPreferencesData, error) {
	prefs, err := a.service.GetPreferences()
	if err != nil {
		return nil, err
	}
	
	// Convert service model to domain model
	return &preferencesDomain.UserPreferencesData{
		DefaultDownloadFolder:     prefs.DefaultDownloadFolder,
		DefaultCompressionLevel:   prefs.DefaultCompressionLevel,
		AutoDownloadEnabled:       prefs.AutoDownloadEnabled,
		ImageDPI:                  prefs.ImageDPI,
		ImageQuality:              prefs.ImageQuality,
		RemoveMetadata:            prefs.RemoveMetadata,
		EmbedFonts:                prefs.EmbedFonts,
		GenerateThumbnails:        prefs.GenerateThumbnails,
		ConvertToGrayscale:        prefs.ConvertToGrayscale,
		PDFVersion:                prefs.PDFVersion,
		AdvancedOptionsExpanded:   prefs.AdvancedOptionsExpanded,
	}, nil
}

func (a *PreferencesRepositoryAdapter) UpdatePreferences(data map[string]any) error {
	return a.service.UpdatePreferences(data)
}

func (a *PreferencesRepositoryAdapter) GetDownloadFolder() (string, error) {
	return a.service.GetDownloadFolder()
}

// CompressionServiceImpl implements the compression domain service
type CompressionServiceImpl struct {
	processor compressionDomain.PDFProcessor
	prefsRepo preferencesDomain.Repository
	config    *config.Config
	ctx       context.Context
}

func (s *CompressionServiceImpl) CompressPDF(ctx context.Context, request compressionDomain.CompressionRequest) compressionDomain.CompressionResponse {
	// Validate input
	if len(request.Files) == 0 {
		s.config.Logger.Error("Compression request validation failed", "error", "no files provided")
		return compressionDomain.CompressionResponse{
			Success: false,
			Error:   common.ErrNoFilesProvided.Error(),
		}
	}

	// Resolve compression level
	compressionLevel, err := s.resolveCompressionLevel(request.CompressionLevel)
	if err != nil {
		s.config.Logger.Error("Failed to resolve compression level", "error", err)
		return compressionDomain.CompressionResponse{
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
	resultChan := make(chan *compressionDomain.FileResult, totalFiles)

	// Fill the work channel
	for _, work := range fileWorkItems {
		workChan <- work

		// Emit initial file status
		wailsruntime.EventsEmit(s.ctx, common.EventFileProgress, compressionDomain.FileProgressUpdate{
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
					s.config.Logger.Info("Compression cancelled by context", "worker_id", workerID)
					return
				default:
				}

				result, err := s.processSingleFile(ctx, work.ID, work.FilePath, compressionLevel, request.AdvancedOptions, workerID)
				if err != nil {
					compressionErr := common.NewCompressionError("processing", work.FilePath, err)
					s.config.Logger.Error("Error processing file",
						"file", work.FilePath,
						"worker_id", workerID,
						"error", compressionErr)

					// Emit error status for this file
					wailsruntime.EventsEmit(s.ctx, common.EventFileProgress, compressionDomain.FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "error",
						Progress: 0,
						WorkerID: workerID,
						Error:    compressionErr.Error(),
					})

					// Send error result
					errorResult := &compressionDomain.FileResult{
						FileID:           work.ID,
						OriginalFilename: filepath.Base(work.FilePath),
						Status:           "error",
						Error:            compressionErr.Error(),
					}
					resultChan <- errorResult
				} else {
					// Emit completion status
					wailsruntime.EventsEmit(s.ctx, common.EventFileProgress, compressionDomain.FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "completed",
						Progress: common.CompletedProgressPercent,
						WorkerID: workerID,
					})

					result.Status = "completed"
					resultChan <- result

					// Stream individual file result immediately
					wailsruntime.EventsEmit(s.ctx, common.EventFileCompleted, result)
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
	var results []compressionDomain.FileResult
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
		wailsruntime.EventsEmit(s.ctx, common.EventCompressionProgress, map[string]any{
			"percent":   overallProgress,
			"current":   completed,
			"total":     totalFiles,
			"completed": completed,
		})
	}

	// Final progress update
	wailsruntime.EventsEmit(s.ctx, common.EventCompressionProgress, map[string]any{
		"percent": 100.0,
		"current": totalFiles,
		"total":   totalFiles,
		"file":    "Complete",
	})

	// Calculate overall compression ratio
	overallCompressionRatio := float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100

	return compressionDomain.CompressionResponse{
		Success:                 true,
		Files:                   results,
		TotalFiles:              len(results),
		TotalOriginalSize:       totalOriginalSize,
		TotalCompressedSize:     totalCompressedSize,
		OverallCompressionRatio: overallCompressionRatio,
		CompressionLevel:        compressionLevel,
		AutoDownload:            request.AutoDownload,
	}
}

func (s *CompressionServiceImpl) ProcessFileData(ctx context.Context, fileData []compressionDomain.FileUpload) compressionDomain.CompressionResponse {
	if len(fileData) == 0 {
		return compressionDomain.CompressionResponse{
			Success: false,
			Error:   common.ErrNoFilesProvided.Error(),
		}
	}

	// Extract file paths
	var filePaths []string
	for _, file := range fileData {
		filePaths = append(filePaths, file.Name)
	}

	// Create request
	request := compressionDomain.CompressionRequest{
		Files:            filePaths,
		CompressionLevel: common.DefaultCompressionLevel,
		AutoDownload:     false,
		DownloadFolder:   "",
		AdvancedOptions:  nil,
	}

	// Load preferences for compression level
	prefs, err := s.prefsRepo.GetPreferences()
	if err == nil && prefs != nil {
		request.CompressionLevel = prefs.DefaultCompressionLevel
	}

	return s.CompressPDF(ctx, request)
}

func (s *CompressionServiceImpl) processSingleFile(ctx context.Context, fileID, filePath, compressionLevel string, advancedOptions *compressionDomain.CompressionOptions, workerID int) (*compressionDomain.FileResult, error) {
	filename := filepath.Base(filePath)

	// Emit compression status
	wailsruntime.EventsEmit(s.ctx, common.EventFileProgress, compressionDomain.FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "compressing",
		Progress: common.DefaultProgressPercent,
		WorkerID: workerID,
	})

	// Create timestamp-based filename for compressed file
	timestamp := time.Now().UTC().Format("20060102_150405")
	baseName := strings.TrimSuffix(filename, ".pdf")
	compressedFilename := fmt.Sprintf("%s_%s_compressed.pdf", baseName, timestamp)

	// Generate output path in the same directory as input
	inputDir := filepath.Dir(filePath)
	compressedPath := filepath.Join(inputDir, compressedFilename)

	// Check for context cancellation before compression
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Direct compression
	err := s.processor.CompressPDF(filePath, compressedPath, compressionLevel, advancedOptions)
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

	return &compressionDomain.FileResult{
		FileID:             fileID,
		OriginalFilename:   filename,
		CompressedFilename: compressedFilename,
		OriginalSize:       originalSize,
		CompressedSize:     compressedSize,
		CompressionRatio:   compressionRatio,
		TempPath:           compressedPath,
	}, nil
}

func (s *CompressionServiceImpl) resolveCompressionLevel(requestedLevel string) (string, error) {
	if requestedLevel != "" {
		return requestedLevel, nil
	}

	prefs, err := s.prefsRepo.GetPreferences()
	if err != nil {
		s.config.Logger.Warn("Failed to load preferences, using default compression level", "error", err)
		return common.DefaultCompressionLevel, nil
	}

	if prefs == nil {
		s.config.Logger.Debug("No preferences found, using default compression level")
		return common.DefaultCompressionLevel, nil
	}

	return prefs.DefaultCompressionLevel, nil
}

// StatisticsServiceImpl implements the statistics domain service
type StatisticsServiceImpl struct {
	processor compressionDomain.PDFProcessor
	stats     statisticsDomain.AppStats
	ctx       context.Context
}

func (s *StatisticsServiceImpl) UpdateStats(filesCompressed int, dataSaved int64) {
	s.stats.SessionFilesCompressed += filesCompressed
	s.stats.SessionDataSaved += dataSaved
	s.stats.TotalFilesCompressed += int64(filesCompressed)
	s.stats.TotalDataSaved += dataSaved

	// Emit stats update
	wailsruntime.EventsEmit(s.ctx, common.EventStatsUpdate, s.stats)
}

func (s *StatisticsServiceImpl) GetStats() *statisticsDomain.AppStats {
	return &s.stats
}

func (s *StatisticsServiceImpl) GetAppStatus(workingDir string) map[string]interface{} {
	return map[string]interface{}{
		"status":                "running",
		"framework":             "Wails + Preact",
		"app_name":              "KleinPDF",
		"ghostscript_path":      s.processor.GetGhostscriptPath(),
		"ghostscript_available": s.processor.IsGhostscriptAvailable(),
		"working_directory":     workingDir,
	}
}