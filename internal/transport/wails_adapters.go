package transport

import (
	"context"

	compressionDomain "kleinpdf/internal/domain/compression"
	preferencesDomain "kleinpdf/internal/domain/preferences"
	statisticsDomain "kleinpdf/internal/domain/statistics"
	"kleinpdf/internal/models"
)

type WailsApp struct {
	ctx                context.Context
	compressionService compressionDomain.Service
	preferencesRepo    preferencesDomain.Repository
	statisticsService  statisticsDomain.Service
	dialogsHandler     DialogHandler
}

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

func (a *WailsApp) CompressPDF(request CompressionRequest) CompressionResponse {
	// Convert transport request to domain request
	domainRequest := compressionDomain.CompressionRequest{
		Files:            request.Files,
		CompressionLevel: request.CompressionLevel,
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
			CompressedPath:     file.CompressedPath,
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
		Error:                   domainResponse.Error,
	}
}

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
			CompressedPath:     file.CompressedPath,
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
		Error:                   domainResponse.Error,
	}
}

func (a *WailsApp) GetPreferences() (*models.UserPreferencesData, error) {
	domainPrefs, err := a.preferencesRepo.GetPreferences()
	if err != nil {
		return nil, err
	}

	// Convert domain model to transport model
	return &models.UserPreferencesData{
		DefaultCompressionLevel: domainPrefs.DefaultCompressionLevel,
		ImageDPI:                domainPrefs.ImageDPI,
		ImageQuality:            domainPrefs.ImageQuality,
		RemoveMetadata:          domainPrefs.RemoveMetadata,
		EmbedFonts:              domainPrefs.EmbedFonts,
		GenerateThumbnails:      domainPrefs.GenerateThumbnails,
		ConvertToGrayscale:      domainPrefs.ConvertToGrayscale,
		PDFVersion:              domainPrefs.PDFVersion,
		AdvancedOptionsExpanded: domainPrefs.AdvancedOptionsExpanded,
	}, nil
}

func (a *WailsApp) UpdatePreferences(data map[string]interface{}) error {
	// Convert interface{} to any for domain layer
	anyData := make(map[string]any, len(data))
	for k, v := range data {
		anyData[k] = v
	}
	return a.preferencesRepo.UpdatePreferences(anyData)
}

func (a *WailsApp) OpenFileDialog() ([]string, error) {
	return a.dialogsHandler.OpenFileDialog()
}

func (a *WailsApp) OpenDirectoryDialog() (string, error) {
	return a.dialogsHandler.OpenDirectoryDialog()
}

func (a *WailsApp) ShowSaveDialog(filename string) (string, error) {
	return a.dialogsHandler.ShowSaveDialog(filename)
}

func (a *WailsApp) OpenFile(filePath string) error {
	return a.dialogsHandler.OpenFile(filePath)
}

func (a *WailsApp) GetAppStatus() map[string]interface{} {
	return a.statisticsService.GetAppStatus()
}

func (a *WailsApp) GetStats() *AppStats {
	domainStats := a.statisticsService.GetStats()
	return &AppStats{
		TotalFilesCompressed:   domainStats.TotalFilesCompressed,
		TotalDataSaved:         domainStats.TotalDataSaved,
		SessionFilesCompressed: domainStats.SessionFilesCompressed,
		SessionDataSaved:       domainStats.SessionDataSaved,
	}
}
