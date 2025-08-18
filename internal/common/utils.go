package common

import (
	"github.com/google/uuid"
)

const (
	// Compression constants
	DefaultCompressionLevel = "good_enough"
	MaxConcurrencyLimit     = 8

	// File operation constants
	DefaultFilePermissions = 0755
)

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}