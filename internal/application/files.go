package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"kleinpdf/internal/config"
	"kleinpdf/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type FilesHandler struct {
	ctx          context.Context
	config       *config.Config
	prefsService *services.PreferencesService
}

func NewFilesHandler(ctx context.Context, config *config.Config, prefsService *services.PreferencesService) *FilesHandler {
	return &FilesHandler{
		ctx:          ctx,
		config:       config,
		prefsService: prefsService,
	}
}

func (h *FilesHandler) SaveFileToDownloadFolder(result FileResult, customDownloadFolder string) (string, error) {
	var downloadDir string
	var err error

	if customDownloadFolder != "" {
		downloadDir = customDownloadFolder
	} else {
		downloadDir, err = h.prefsService.GetDownloadFolder()
		if err != nil {
			return "", err
		}
	}

	downloadPath := filepath.Join(downloadDir, result.CompressedFilename)
	err = CopyFile(result.TempPath, downloadPath)
	if err != nil {
		return "", err
	}

	return downloadPath, nil
}

func (h *FilesHandler) WriteFilesToTemp(fileData []FileUpload) ([]string, error) {
	var filePaths []string

	for i, file := range fileData {
		// Generate unique temp directory for this batch
		batchID := GenerateUUID()
		tempDir := filepath.Join(h.config.WorkingDir, "upload_"+batchID)

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
		wailsruntime.EventsEmit(h.ctx, "compression:progress", map[string]interface{}{
			"percent": progress,
			"current": i + 1,
			"total":   len(fileData),
			"file":    fmt.Sprintf("Preparing %s...", file.Name),
			"stage":   "preparation",
		})
	}

	return filePaths, nil
}