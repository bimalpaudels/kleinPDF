package transport

import (
	"context"

	compressionDomain "kleinpdf/internal/domain/compression"
	preferencesDomain "kleinpdf/internal/domain/preferences"  
	statisticsDomain "kleinpdf/internal/domain/statistics"
	"kleinpdf/internal/models"
)

// WailsApp provides the transport layer for Wails application
type WailsApp struct {
	ctx                context.Context
	compressionService compressionDomain.Service
	preferencesRepo    preferencesDomain.Repository
	statisticsService  statisticsDomain.Service
	dialogsHandler     DialogHandler
}

// NewWailsApp creates a new Wails transport adapter
func NewWailsApp(
	ctx context.Context,
	compressionService compressionDomain.Service,
	preferencesRepo preferencesDomain.Repository,
	statisticsService statisticsDomain.Service,
) *WailsApp {
	return &WailsApp{
		ctx:                ctx,
		compressionService: compressionService,
		preferencesRepo:    preferencesRepo,
		statisticsService:  statisticsService,
		dialogsHandler:     NewDialogsHandler(ctx),
	}
}

// CompressPDF handles PDF compression requests from the frontend
func (a *WailsApp) CompressPDF(request CompressionRequest) CompressionResponse {
	// Convert transport request to domain request
	domainRequest := compressionDomain.CompressionRequest{
		Files:            request.Files,
		CompressionLevel: request.CompressionLevel,
		AutoDownload:     request.AutoDownload,
		DownloadFolder:   request.DownloadFolder,
	}

	// Convert advanced options if present
	if request.AdvancedOptions != nil {
		domainRequest.AdvancedOptions = &compressionDomain.CompressionOptions{
			ImageDPI:           request.AdvancedOptions.ImageDPI,
			ImageQuality:       request.AdvancedOptions.ImageQuality,
			PDFVersion:         request.AdvancedOptions.PDFVersion,
			RemoveMetadata:     request.AdvancedOptions.RemoveMetadata,
			EmbedFonts:         request.AdvancedOptions.EmbedFonts,
			GenerateThumbnails: request.AdvancedOptions.GenerateThumbnails,
			ConvertToGrayscale: request.AdvancedOptions.ConvertToGrayscale,
		}
	}

	// Call domain service
	domainResponse := a.compressionService.CompressPDF(a.ctx, domainRequest)

	// Convert domain response to transport response
	transportFiles := make([]FileResult, len(domainResponse.Files))
	for i, file := range domainResponse.Files {
		transportFiles[i] = FileResult{
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
		Success:                 domainResponse.Success,
		Files:                   transportFiles,
		TotalFiles:              domainResponse.TotalFiles,
		TotalOriginalSize:       domainResponse.TotalOriginalSize,
		TotalCompressedSize:     domainResponse.TotalCompressedSize,
		OverallCompressionRatio: domainResponse.OverallCompressionRatio,
		CompressionLevel:        domainResponse.CompressionLevel,
		AutoDownload:            domainResponse.AutoDownload,
		DownloadPaths:           domainResponse.DownloadPaths,
		Error:                   domainResponse.Error,
	}
}

// ProcessFileData handles file upload processing
func (a *WailsApp) ProcessFileData(fileData []FileUpload) CompressionResponse {
	// Convert transport file data to domain file data
	domainFileData := make([]compressionDomain.FileUpload, len(fileData))
	for i, file := range fileData {
		domainFileData[i] = compressionDomain.FileUpload{
			Name: file.Name,
			Data: file.Data,
			Size: file.Size,
		}
	}

	// Call domain service
	domainResponse := a.compressionService.ProcessFileData(a.ctx, domainFileData)

	// Convert response (same as above)
	transportFiles := make([]FileResult, len(domainResponse.Files))
	for i, file := range domainResponse.Files {
		transportFiles[i] = FileResult{
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
		Success:                 domainResponse.Success,
		Files:                   transportFiles,
		TotalFiles:              domainResponse.TotalFiles,
		TotalOriginalSize:       domainResponse.TotalOriginalSize,
		TotalCompressedSize:     domainResponse.TotalCompressedSize,
		OverallCompressionRatio: domainResponse.OverallCompressionRatio,
		CompressionLevel:        domainResponse.CompressionLevel,
		AutoDownload:            domainResponse.AutoDownload,
		DownloadPaths:           domainResponse.DownloadPaths,
		Error:                   domainResponse.Error,
	}
}

// GetPreferences gets user preferences
func (a *WailsApp) GetPreferences() (*models.UserPreferencesData, error) {
	domainPrefs, err := a.preferencesRepo.GetPreferences()
	if err != nil {
		return nil, err
	}

	// Convert domain model to transport model
	return &models.UserPreferencesData{
		DefaultDownloadFolder:     domainPrefs.DefaultDownloadFolder,
		DefaultCompressionLevel:   domainPrefs.DefaultCompressionLevel,
		AutoDownloadEnabled:       domainPrefs.AutoDownloadEnabled,
		ImageDPI:                  domainPrefs.ImageDPI,
		ImageQuality:              domainPrefs.ImageQuality,
		RemoveMetadata:            domainPrefs.RemoveMetadata,
		EmbedFonts:                domainPrefs.EmbedFonts,
		GenerateThumbnails:        domainPrefs.GenerateThumbnails,
		ConvertToGrayscale:        domainPrefs.ConvertToGrayscale,
		PDFVersion:                domainPrefs.PDFVersion,
		AdvancedOptionsExpanded:   domainPrefs.AdvancedOptionsExpanded,
	}, nil
}

// UpdatePreferences updates user preferences
func (a *WailsApp) UpdatePreferences(data map[string]interface{}) error {
	// Convert interface{} to any for domain layer
	anyData := make(map[string]any, len(data))
	for k, v := range data {
		anyData[k] = v
	}
	return a.preferencesRepo.UpdatePreferences(anyData)
}

// OpenFileDialog opens a file selection dialog
func (a *WailsApp) OpenFileDialog() ([]string, error) {
	return a.dialogsHandler.OpenFileDialog()
}

// OpenDirectoryDialog opens a directory selection dialog
func (a *WailsApp) OpenDirectoryDialog() (string, error) {
	return a.dialogsHandler.OpenDirectoryDialog()
}

// ShowSaveDialog shows a save file dialog
func (a *WailsApp) ShowSaveDialog(filename string) (string, error) {
	return a.dialogsHandler.ShowSaveDialog(filename)
}

// OpenFile opens a file in the default application
func (a *WailsApp) OpenFile(filePath string) error {
	return a.dialogsHandler.OpenFile(filePath)
}

// GetAppStatus gets application status information
func (a *WailsApp) GetAppStatus() map[string]interface{} {
	return a.statisticsService.GetAppStatus("")
}

// GetStats gets application statistics
func (a *WailsApp) GetStats() *AppStats {
	domainStats := a.statisticsService.GetStats()
	return &AppStats{
		TotalFilesCompressed:   domainStats.TotalFilesCompressed,
		TotalDataSaved:         domainStats.TotalDataSaved,
		SessionFilesCompressed: domainStats.SessionFilesCompressed,
		SessionDataSaved:       domainStats.SessionDataSaved,
	}
}