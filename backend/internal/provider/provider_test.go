package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/msi/circuit-storys/backend/internal/circuit"
)

func TestStaticProviderReturnsResult(t *testing.T) {
	want := circuit.ExtractionResult{Provider: "mock", Confidence: 0.9}
	got, err := (StaticProvider{Result: want}).ExtractCircuit(context.Background(), ExtractionInput{})
	if err != nil {
		t.Fatalf("ExtractCircuit error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func TestStaticProviderReturnsError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := (StaticProvider{Err: wantErr}).ExtractCircuit(context.Background(), ExtractionInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestUnconfiguredProviderReturnsExpectedError(t *testing.T) {
	_, err := (UnconfiguredProvider{}).ExtractCircuit(context.Background(), ExtractionInput{})
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("error = %v, want %v", err, ErrProviderNotConfigured)
	}
}
