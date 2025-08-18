package application

import (
	"context"
	"path/filepath"

	"kleinpdf/internal/config"
	"kleinpdf/internal/services"
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

