package application

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func GenerateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func CopyFile(src, dst string) error {
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

func CleanupOldTempFiles(workingDir string) {
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
