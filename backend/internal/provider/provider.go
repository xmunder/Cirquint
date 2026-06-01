package provider

import (
	"context"
	"errors"

	"github.com/msi/circuit-storys/backend/internal/circuit"
)

type ExtractionInput struct {
	ProjectID   string
	UploadID    string
	ObjectKey   string
	ContentType string
	Data        []byte
}

type CircuitExtractionProvider interface {
	ExtractCircuit(ctx context.Context, input ExtractionInput) (circuit.ExtractionResult, error)
}

var ErrProviderNotConfigured = errors.New("circuit extraction provider not configured")

type StaticProvider struct {
	Result circuit.ExtractionResult
	Err    error
}

func (p StaticProvider) ExtractCircuit(_ context.Context, _ ExtractionInput) (circuit.ExtractionResult, error) {
	if p.Err != nil {
		return circuit.ExtractionResult{}, p.Err
	}
	return p.Result, nil
}

type UnconfiguredProvider struct{}

func (UnconfiguredProvider) ExtractCircuit(_ context.Context, _ ExtractionInput) (circuit.ExtractionResult, error) {
	return circuit.ExtractionResult{}, ErrProviderNotConfigured
}
