package provider

import (
	"context"

	"github.com/msi/circuit-storys/backend/internal/circuit"
)

type ExtractionInput struct {
	ProjectID string
	UploadID  string
	ObjectKey string
}

type CircuitExtractionProvider interface {
	ExtractCircuit(ctx context.Context, input ExtractionInput) (circuit.ExtractionResult, error)
}
