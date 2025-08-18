package common

import (
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