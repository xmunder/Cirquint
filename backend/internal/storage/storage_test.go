package storage_test

import (
	"testing"

	"github.com/msi/circuit-storys/backend/internal/storage"
)

func TestUploadObjectKey(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "png", filename: "diagram.png", want: "workspaces/ws-001/uploads/up-001/v1/source.png"},
		{name: "jpeg normalized", filename: "diagram.jpeg", want: "workspaces/ws-001/uploads/up-001/v1/source.jpg"},
		{name: "missing ext falls back", filename: "diagram", want: "workspaces/ws-001/uploads/up-001/v1/source.bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.UploadObjectKey("ws-001", "up-001", tt.filename)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
