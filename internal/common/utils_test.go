package common

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/google/uuid"
)

func TestGenerateUUID(t *testing.T) {
	// Generate multiple UUIDs
	uuid1 := GenerateUUID()
	uuid2 := GenerateUUID()
	
	// Should not be empty
	if uuid1 == "" {
		t.Error("Expected non-empty UUID")
	}
	
	if uuid2 == "" {
		t.Error("Expected non-empty UUID")
	}
	
	// Should be different
	if uuid1 == uuid2 {
		t.Error("Expected different UUIDs")
	}
	
	// Should be valid UUID format
	_, err := uuid.Parse(uuid1)
	if err != nil {
		t.Errorf("Generated UUID is not valid: %v", err)
	}
	
	_, err = uuid.Parse(uuid2)
	if err != nil {
		t.Errorf("Generated UUID is not valid: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary source file
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")
	
	// Create source file with content
	content := "Hello, World!"
	err := os.WriteFile(srcPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Copy the file
	err = CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Expected no error copying file, got %v", err)
	}
	
	// Verify destination file exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Destination file was not created")
	}
	
	// Verify content matches
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	
	if string(dstContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(dstContent))
	}
}

func TestCopyFile_CreateDirectory(t *testing.T) {
	// Create a temporary source file
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "subdir", "nested", "destination.txt")
	
	// Create source file
	content := "Hello, World!"
	err := os.WriteFile(srcPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Copy to nested directory that doesn't exist
	err = CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Expected no error copying file, got %v", err)
	}
	
	// Verify destination file exists and directory was created
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Destination file was not created")
	}
	
	// Verify parent directories were created
	dstDir := filepath.Dir(dstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Error("Destination directory was not created")
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")
	
	// Try to copy non-existent file
	err := CopyFile(srcPath, dstPath)
	if err == nil {
		t.Error("Expected error when source file doesn't exist")
	}
}