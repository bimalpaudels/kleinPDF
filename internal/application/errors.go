package application

import (
	"errors"
	"fmt"
)

// Application error types
var (
	ErrNoFilesProvided         = errors.New("no files provided for compression")
	ErrInvalidCompressionLevel = errors.New("invalid compression level")
	ErrPreferencesLoad         = errors.New("failed to load preferences")
	ErrFileProcessing          = errors.New("file processing failed")
	ErrFileNotFound            = errors.New("file not found")
	ErrInvalidFileType         = errors.New("invalid file type")
)

// CompressionError represents compression-specific errors
type CompressionError struct {
	Operation string
	FilePath  string
	Err       error
}

func (e *CompressionError) Error() string {
	if e.FilePath != "" {
		return fmt.Sprintf("compression %s failed for file %s: %v", e.Operation, e.FilePath, e.Err)
	}
	return fmt.Sprintf("compression %s failed: %v", e.Operation, e.Err)
}

func (e *CompressionError) Unwrap() error {
	return e.Err
}

// NewCompressionError creates a new compression error
func NewCompressionError(operation, filePath string, err error) *CompressionError {
	return &CompressionError{
		Operation: operation,
		FilePath:  filePath,
		Err:       err,
	}
}

// PreferencesError represents preferences-related errors
type PreferencesError struct {
	Operation string
	Err       error
}

func (e *PreferencesError) Error() string {
	return fmt.Sprintf("preferences %s failed: %v", e.Operation, e.Err)
}

func (e *PreferencesError) Unwrap() error {
	return e.Err
}

// NewPreferencesError creates a new preferences error
func NewPreferencesError(operation string, err error) *PreferencesError {
	return &PreferencesError{
		Operation: operation,
		Err:       err,
	}
}