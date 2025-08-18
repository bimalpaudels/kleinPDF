package transport

import (
	"context"
	"testing"
)

func TestNewDialogsHandler(t *testing.T) {
	ctx := context.Background()
	handler := NewDialogsHandler(ctx)
	
	if handler == nil {
		t.Fatal("Expected DialogHandler instance, got nil")
	}
	
	// Verify it implements the interface
	var _ DialogHandler = handler
}