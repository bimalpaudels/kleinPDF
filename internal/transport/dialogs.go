package transport

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type dialogsHandler struct {
	ctx context.Context
}

func NewDialogsHandler(ctx context.Context) DialogHandler {
	return &dialogsHandler{
		ctx: ctx,
	}
}

func (h *dialogsHandler) OpenFileDialog() ([]string, error) {
	selection, err := wailsruntime.OpenMultipleFilesDialog(h.ctx, wailsruntime.OpenDialogOptions{
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

func (h *dialogsHandler) OpenDirectoryDialog() (string, error) {
	selection, err := wailsruntime.OpenDirectoryDialog(h.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select download folder",
	})

	if err != nil {
		return "", err
	}

	return selection, nil
}

func (h *dialogsHandler) ShowSaveDialog(filename string) (string, error) {
	selection, err := wailsruntime.SaveFileDialog(h.ctx, wailsruntime.SaveDialogOptions{
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

func (h *dialogsHandler) OpenFile(filePath string) error {
	wailsruntime.BrowserOpenURL(h.ctx, "file://"+filePath)
	return nil
}
