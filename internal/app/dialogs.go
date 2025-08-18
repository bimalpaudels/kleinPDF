package app

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenFileDialog opens a file selection dialog for PDF files
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

// OpenDirectoryDialog opens a directory selection dialog
func (a *App) OpenDirectoryDialog() (string, error) {
	selection, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select download folder",
	})

	if err != nil {
		return "", err
	}

	return selection, nil
}

// ShowSaveDialog opens a file save dialog
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

// OpenFile opens a file using the system default application
func (a *App) OpenFile(filePath string) error {
	wailsruntime.BrowserOpenURL(a.ctx, "file://"+filePath)
	return nil
}