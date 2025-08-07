package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"pdf-compressor-wails/internal/config"
	"pdf-compressor-wails/internal/database"
	"pdf-compressor-wails/internal/models"
	"pdf-compressor-wails/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	pdfService   *services.PDFService
	prefsService *services.PreferencesService
	config       *config.Config
	stats        *AppStats
}

// AppStats tracks application statistics
type AppStats struct {
	TotalFilesCompressed int64 `json:"total_files_compressed"`
	TotalDataSaved       int64 `json:"total_data_saved"`
	SessionFilesCompressed int `json:"session_files_compressed"`
	SessionDataSaved     int64 `json:"session_data_saved"`
}

// CompressionRequest represents a compression request from the frontend
type CompressionRequest struct {
	Files            []string                     `json:"files"`
	CompressionLevel string                       `json:"compressionLevel"`
	AutoDownload     bool                         `json:"autoDownload"`
	DownloadFolder   string                       `json:"downloadFolder"`
	AdvancedOptions  *services.CompressionOptions `json:"advancedOptions"`
}

// FileResult represents the result of processing a single file
type FileResult struct {
	FileID             string  `json:"file_id"`
	OriginalFilename   string  `json:"original_filename"`
	CompressedFilename string  `json:"compressed_filename"`
	OriginalSize       int64   `json:"original_size"`
	CompressedSize     int64   `json:"compressed_size"`
	CompressionRatio   float64 `json:"compression_ratio"`
	TempPath           string  `json:"temp_path"`
	SavedPath          *string `json:"saved_path,omitempty"`
	Status             string  `json:"status"` // "copying", "compressing", "completed", "error"
	Error              string  `json:"error,omitempty"`
}

// FileProgressUpdate represents progress for a single file
type FileProgressUpdate struct {
	FileID   string  `json:"file_id"`
	Filename string  `json:"filename"`
	Status   string  `json:"status"` // "copying", "compressing", "completed", "error"
	Progress float64 `json:"progress"` // 0-100
	WorkerID int     `json:"worker_id"`
	Error    string  `json:"error,omitempty"`
}

// CompressionResponse represents the response from compression
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

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		stats: &AppStats{},
	}
}

// OnStartup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	cfg := config.New()
	a.config = cfg

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.UserPreferences{})
	if err != nil {
		log.Printf("Failed to migrate database: %v", err)
		return
	}

	// Initialize services
	a.pdfService = services.NewPDFService(cfg)
	a.prefsService = services.NewPreferencesService(db)

	log.Printf("Wails app initialized successfully")
	log.Printf("Working directory: %s", cfg.WorkingDir)
	log.Printf("Database path: %s", cfg.DatabasePath)
	log.Printf("Ghostscript available: %t", a.pdfService.IsGhostscriptAvailable())
}

// CompressPDF handles PDF compression through Wails
func (a *App) CompressPDF(request CompressionRequest) CompressionResponse {
	// Validate input
	if len(request.Files) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "No files provided",
		}
	}

	// Clean up old temp files
	a.cleanupOldTempFiles()

	// Use compression level from preferences if not specified
	compressionLevel := request.CompressionLevel
	if compressionLevel == "" {
		prefs, err := a.prefsService.GetPreferences()
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
		maxConcurrency = 8 // Cap to avoid overwhelming disk I/O
	}
	
	// Create file work items with unique IDs
	type fileWork struct {
		ID       string
		FilePath string
	}
	
	var fileWorkItems []fileWork
	for _, filePath := range request.Files {
		fileWorkItems = append(fileWorkItems, fileWork{
			ID:       a.generateUUID(),
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
		wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
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
				result, err := a.processSingleFileWithProgress(work.ID, work.FilePath, compressionLevel, request.AdvancedOptions, workerID)
				if err != nil {
					log.Printf("Error processing file %s: %v", work.FilePath, err)
					
					// Emit error status for this file
					wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
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
					wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
						FileID:   work.ID,
						Filename: filepath.Base(work.FilePath),
						Status:   "completed",
						Progress: 100,
						WorkerID: workerID,
					})
					
					result.Status = "completed"
					resultChan <- result
					
					// Stream individual file result immediately
					wailsruntime.EventsEmit(a.ctx, "file:completed", result)
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
		wailsruntime.EventsEmit(a.ctx, "compression:progress", map[string]any{
			"percent":   overallProgress,
			"current":   completed,
			"total":     totalFiles,
			"completed": completed,
		})
	}

	// Final progress update
	wailsruntime.EventsEmit(a.ctx, "compression:progress", map[string]interface{}{
		"percent": 100.0,
		"current": totalFiles,
		"total":   totalFiles,
		"file":    "Complete",
	})

	// Calculate overall compression ratio
	overallCompressionRatio := float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100
	dataSaved := totalOriginalSize - totalCompressedSize

	// Update stats
	a.stats.SessionFilesCompressed += len(results)
	a.stats.SessionDataSaved += dataSaved
	a.stats.TotalFilesCompressed += int64(len(results))
	a.stats.TotalDataSaved += dataSaved

	// Emit stats update
	wailsruntime.EventsEmit(a.ctx, "stats:update", a.stats)

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
			downloadPath, err := a.saveFileToDownloadFolder(result, request.DownloadFolder)
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

func (a *App) processSingleFileWithProgress(fileID, filePath, compressionLevel string, advancedOptions *services.CompressionOptions, workerID int) (*FileResult, error) {
	filename := filepath.Base(filePath)
	
	// Emit copying status
	wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "copying",
		Progress: 10,
		WorkerID: workerID,
	})
	
	// Generate temp directory
	tempDir := filepath.Join(a.config.WorkingDir, fileID)
	
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}
	
	// Create timestamp-based filename for compressed file
	timestamp := time.Now().UTC().Format("20060102_150405")
	baseName := strings.TrimSuffix(filename, ".pdf")
	compressedFilename := fmt.Sprintf("%s_%s.pdf", baseName, timestamp)
	
	// Copy original file to temp directory
	originalTempPath := filepath.Join(tempDir, filename)
	if err := a.copyFile(filePath, originalTempPath); err != nil {
		return nil, fmt.Errorf("failed to copy file to temp directory: %v", err)
	}
	
	// Emit compression status
	wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
		FileID:   fileID,
		Filename: filename,
		Status:   "compressing",
		Progress: 30,
		WorkerID: workerID,
	})
	
	// Compress the PDF
	compressedPath := filepath.Join(tempDir, compressedFilename)
	
	err := a.pdfService.CompressPDF(originalTempPath, compressedPath, compressionLevel, advancedOptions)
	if err != nil {
		return nil, err
	}
	
	// Emit finishing status
	wailsruntime.EventsEmit(a.ctx, "file:progress", FileProgressUpdate{
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


func (a *App) saveFileToDownloadFolder(result FileResult, customDownloadFolder string) (string, error) {
	var downloadDir string
	var err error

	if customDownloadFolder != "" {
		downloadDir = customDownloadFolder
	} else {
		downloadDir, err = a.prefsService.GetDownloadFolder()
		if err != nil {
			return "", err
		}
	}

	downloadPath := filepath.Join(downloadDir, result.CompressedFilename)
	err = a.copyFile(result.TempPath, downloadPath)
	if err != nil {
		return "", err
	}

	return downloadPath, nil
}

func (a *App) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dst)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (a *App) cleanupOldTempFiles() {
	workingDir := a.config.WorkingDir
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(workingDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(workingDir, entry.Name())
			os.RemoveAll(dirPath)
		}
	}
}

func (a *App) generateUUID() string {
	// Simple UUID generation for file IDs
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GetPreferences returns current user preferences
func (a *App) GetPreferences() (*models.UserPreferencesData, error) {
	return a.prefsService.GetPreferences()
}

// UpdatePreferences updates user preferences
func (a *App) UpdatePreferences(data map[string]interface{}) error {
	return a.prefsService.UpdatePreferences(data)
}

// OpenFileDialog opens a file dialog for selecting PDF files
func (a *App) OpenFileDialog() ([]string, error) {
	selection, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select PDF files to compress",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "PDF Files (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return selection, nil
}

// OpenDirectoryDialog opens a directory dialog for selecting download folder
func (a *App) OpenDirectoryDialog() (string, error) {
	selection, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select download folder",
	})

	if err != nil {
		return "", err
	}

	return selection, nil
}

// ShowSaveDialog shows a save dialog for individual file
func (a *App) ShowSaveDialog(filename string) (string, error) {
	selection, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save compressed PDF",
		DefaultFilename: filename,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "PDF Files (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	})

	if err != nil {
		return "", err
	}

	return selection, nil
}

// OpenFile opens a file using the system's default application
func (a *App) OpenFile(filePath string) error {
	wailsruntime.BrowserOpenURL(a.ctx, "file://"+filePath)
	return nil
}

// GetAppStatus returns the current app status
func (a *App) GetAppStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":                "running",
		"framework":             "Wails + Preact",
		"app_name":              "KleinPDF",
		"ghostscript_path":      a.pdfService.GetGhostscriptPath(),
		"ghostscript_available": a.pdfService.IsGhostscriptAvailable(),
		"working_directory":     a.config.WorkingDir,
	}
}

// GetStats returns the current application statistics
func (a *App) GetStats() *AppStats {
	return a.stats
}

// WriteFilesToTemp writes uploaded files to temp directory and returns their paths
func (a *App) WriteFilesToTemp(fileData []FileUpload) ([]string, error) {
	var filePaths []string
	
	for i, file := range fileData {
		// Generate unique temp directory for this batch
		batchID := a.generateUUID()
		tempDir := filepath.Join(a.config.WorkingDir, "upload_"+batchID)
		
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %v", err)
		}
		
		// Write file to temp location
		tempPath := filepath.Join(tempDir, file.Name)
		if err := os.WriteFile(tempPath, file.Data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %v", file.Name, err)
		}
		
		filePaths = append(filePaths, tempPath)
		
		// Emit progress for file writing
		progress := float64(i+1) / float64(len(fileData)) * 20 // First 20% for file writing
		wailsruntime.EventsEmit(a.ctx, "compression:progress", map[string]interface{}{
			"percent": progress,
			"current": i + 1,
			"total":   len(fileData),
			"file":    fmt.Sprintf("Preparing %s...", file.Name),
			"stage":   "preparation",
		})
	}
	
	return filePaths, nil
}

// ProcessFileData handles PDF compression from file data instead of file paths
func (a *App) ProcessFileData(fileData []FileUpload) CompressionResponse {
	// Validate input
	if len(fileData) == 0 {
		return CompressionResponse{
			Success: false,
			Error:   "No files provided",
		}
	}

	// Write files to temp directory first
	filePaths, err := a.WriteFilesToTemp(fileData)
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
	prefs, err := a.prefsService.GetPreferences()
	if err == nil && prefs != nil {
		request.CompressionLevel = prefs.DefaultCompressionLevel
	}

	// Process using the regular compression logic
	return a.CompressPDF(request)
}

// FileUpload represents uploaded file data
type FileUpload struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
	Size int64  `json:"size"`
}

