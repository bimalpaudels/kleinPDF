package application

import (
	"context"

	"kleinpdf/internal/config"
	"kleinpdf/internal/container"
	"kleinpdf/internal/database"
	model "kleinpdf/internal/models"
	"kleinpdf/internal/transport"
)

type App struct {
	ctx       context.Context
	container *container.Container
	wailsApp  *transport.WailsApp
	config    *config.Config
}

func NewApp() *App {
	return &App{}
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	cfg := config.New()
	a.config = cfg

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		cfg.Logger.Error("Failed to initialize database", "error", err)
		return
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&model.UserPreferences{})
	if err != nil {
		cfg.Logger.Error("Failed to migrate database", "error", err)
		return
	}

	// Initialize dependency container
	a.container = container.New(ctx, cfg, db)
	
	// Initialize transport layer
	a.wailsApp = transport.NewWailsApp(
		ctx,
		a.container.GetCompressionService(),
		a.container.GetPreferencesRepository(),
		a.container.GetStatisticsService(),
	)

	cfg.Logger.Info("Wails app initialized successfully")
	cfg.Logger.Info("Application configuration", 
		"working_directory", cfg.WorkingDir,
		"database_path", cfg.DatabasePath,
		"ghostscript_available", true) // We'll get this from container later
}

func (a *App) CompressPDF(request CompressionRequest) CompressionResponse {
	// Convert application types to transport types
	transportRequest := transport.CompressionRequest{
		Files:            request.Files,
		CompressionLevel: request.CompressionLevel,
		AutoDownload:     request.AutoDownload,
		DownloadFolder:   request.DownloadFolder,
		AdvancedOptions:  request.AdvancedOptions,
	}
	
	transportResponse := a.wailsApp.CompressPDF(transportRequest)
	
	// Convert transport response back to application response
	appFiles := make([]FileResult, len(transportResponse.Files))
	for i, file := range transportResponse.Files {
		appFiles[i] = FileResult{
			FileID:             file.FileID,
			OriginalFilename:   file.OriginalFilename,
			CompressedFilename: file.CompressedFilename,
			OriginalSize:       file.OriginalSize,
			CompressedSize:     file.CompressedSize,
			CompressionRatio:   file.CompressionRatio,
			TempPath:           file.TempPath,
			SavedPath:          file.SavedPath,
			Status:             file.Status,
			Error:              file.Error,
		}
	}
	
	return CompressionResponse{
		Success:                 transportResponse.Success,
		Files:                   appFiles,
		TotalFiles:              transportResponse.TotalFiles,
		TotalOriginalSize:       transportResponse.TotalOriginalSize,
		TotalCompressedSize:     transportResponse.TotalCompressedSize,
		OverallCompressionRatio: transportResponse.OverallCompressionRatio,
		CompressionLevel:        transportResponse.CompressionLevel,
		AutoDownload:            transportResponse.AutoDownload,
		DownloadPaths:           transportResponse.DownloadPaths,
		Error:                   transportResponse.Error,
	}
}

func (a *App) ProcessFileData(fileData []FileUpload) CompressionResponse {
	// Convert application types to transport types
	transportFileData := make([]transport.FileUpload, len(fileData))
	for i, file := range fileData {
		transportFileData[i] = transport.FileUpload{
			Name: file.Name,
			Data: file.Data,
			Size: file.Size,
		}
	}
	
	transportResponse := a.wailsApp.ProcessFileData(transportFileData)
	
	// Convert transport response back to application response (same as above)
	appFiles := make([]FileResult, len(transportResponse.Files))
	for i, file := range transportResponse.Files {
		appFiles[i] = FileResult{
			FileID:             file.FileID,
			OriginalFilename:   file.OriginalFilename,
			CompressedFilename: file.CompressedFilename,
			OriginalSize:       file.OriginalSize,
			CompressedSize:     file.CompressedSize,
			CompressionRatio:   file.CompressionRatio,
			TempPath:           file.TempPath,
			SavedPath:          file.SavedPath,
			Status:             file.Status,
			Error:              file.Error,
		}
	}
	
	return CompressionResponse{
		Success:                 transportResponse.Success,
		Files:                   appFiles,
		TotalFiles:              transportResponse.TotalFiles,
		TotalOriginalSize:       transportResponse.TotalOriginalSize,
		TotalCompressedSize:     transportResponse.TotalCompressedSize,
		OverallCompressionRatio: transportResponse.OverallCompressionRatio,
		CompressionLevel:        transportResponse.CompressionLevel,
		AutoDownload:            transportResponse.AutoDownload,
		DownloadPaths:           transportResponse.DownloadPaths,
		Error:                   transportResponse.Error,
	}
}

func (a *App) GetPreferences() (*model.UserPreferencesData, error) {
	return a.wailsApp.GetPreferences()
}

func (a *App) UpdatePreferences(data map[string]interface{}) error {
	return a.wailsApp.UpdatePreferences(data)
}

func (a *App) OpenFileDialog() ([]string, error) {
	return a.wailsApp.OpenFileDialog()
}

func (a *App) OpenDirectoryDialog() (string, error) {
	return a.wailsApp.OpenDirectoryDialog()
}

func (a *App) ShowSaveDialog(filename string) (string, error) {
	return a.wailsApp.ShowSaveDialog(filename)
}

func (a *App) OpenFile(filePath string) error {
	return a.wailsApp.OpenFile(filePath)
}

func (a *App) GetAppStatus() map[string]interface{} {
	return a.wailsApp.GetAppStatus()
}

func (a *App) GetStats() *AppStats {
	transportStats := a.wailsApp.GetStats()
	return &AppStats{
		TotalFilesCompressed:   transportStats.TotalFilesCompressed,
		TotalDataSaved:         transportStats.TotalDataSaved,
		SessionFilesCompressed: transportStats.SessionFilesCompressed,
		SessionDataSaved:       transportStats.SessionDataSaved,
	}
}