package services

import (
	"testing"

	"kleinpdf/internal/config"
)

func TestNewPDFService(t *testing.T) {
	cfg := &config.Config{
		GhostscriptPath: "/usr/local/bin/gs",
	}
	
	service := NewPDFService(cfg)
	
	if service == nil {
		t.Fatal("Expected PDFService instance, got nil")
	}
	
	if service.config != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestDefaultCompressionOptions(t *testing.T) {
	opts := DefaultCompressionOptions()
	
	expectedDPI := 150
	if opts.ImageDPI != expectedDPI {
		t.Errorf("Expected ImageDPI to be %d, got %d", expectedDPI, opts.ImageDPI)
	}
	
	expectedQuality := 85
	if opts.ImageQuality != expectedQuality {
		t.Errorf("Expected ImageQuality to be %d, got %d", expectedQuality, opts.ImageQuality)
	}
	
	expectedVersion := "1.4"
	if opts.PDFVersion != expectedVersion {
		t.Errorf("Expected PDFVersion to be %s, got %s", expectedVersion, opts.PDFVersion)
	}
	
	if !opts.EmbedFonts {
		t.Error("Expected EmbedFonts to be true")
	}
	
	if opts.RemoveMetadata {
		t.Error("Expected RemoveMetadata to be false")
	}
}

func TestIsGhostscriptAvailable(t *testing.T) {
	tests := []struct {
		name           string
		ghostscriptPath string
		expected       bool
	}{
		{
			name:           "empty path",
			ghostscriptPath: "",
			expected:       false,
		},
		{
			name:           "non-empty path",
			ghostscriptPath: "/usr/local/bin/gs",
			expected:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				GhostscriptPath: tt.ghostscriptPath,
			}
			service := NewPDFService(cfg)
			
			result := service.IsGhostscriptAvailable()
			if result != tt.expected {
				t.Errorf("Expected IsGhostscriptAvailable() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetGhostscriptPath(t *testing.T) {
	expectedPath := "/usr/local/bin/gs"
	cfg := &config.Config{
		GhostscriptPath: expectedPath,
	}
	service := NewPDFService(cfg)
	
	result := service.GetGhostscriptPath()
	if result != expectedPath {
		t.Errorf("Expected GetGhostscriptPath() to return %s, got %s", expectedPath, result)
	}
}

func TestCompressPDF_NoGhostscript(t *testing.T) {
	cfg := &config.Config{
		GhostscriptPath: "", // No ghostscript
	}
	service := NewPDFService(cfg)
	
	err := service.CompressPDF("input.pdf", "output.pdf", "good_enough", nil)
	
	if err == nil {
		t.Error("Expected error when ghostscript is not available")
	}
	
	expectedErrorMsg := "ghostscript not found"
	if !contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestCompressPDF_ValidatesOptions(t *testing.T) {
	cfg := &config.Config{
		GhostscriptPath: "/usr/local/bin/gs", // Mock path
	}
	service := NewPDFService(cfg)
	
	// Test with nil options - should use defaults
	options := &CompressionOptions{
		PDFVersion:   "", // Empty
		ImageDPI:     0,  // Invalid
		ImageQuality: 0,  // Invalid
	}
	
	// This would fail due to ghostscript execution, but we can test option validation
	// by checking that the method doesn't panic and processes the options
	err := service.CompressPDF("nonexistent.pdf", "output.pdf", "good_enough", options)
	
	// We expect an error because the file doesn't exist or ghostscript fails,
	// but it shouldn't be a panic due to invalid options
	if err == nil {
		t.Error("Expected error for nonexistent input file")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || len(substr) == 0 || 
		   (len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}