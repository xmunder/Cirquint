package worker

import (
	"context"
	"errors"
	"io"

	"github.com/msi/circuit-storys/backend/internal/circuit"
	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/provider"
	"github.com/msi/circuit-storys/backend/internal/storage"
	"github.com/msi/circuit-storys/backend/internal/uploads"
)

type Runner struct {
	jobs     *processing.Service
	uploads  uploads.Repository
	storage  storage.ObjectStorage
	provider provider.CircuitExtractionProvider
	circuits *circuit.Service
}

func NewRunner(jobs *processing.Service, uploadRepo uploads.Repository, objectStorage storage.ObjectStorage, extractionProvider provider.CircuitExtractionProvider, circuitService *circuit.Service) *Runner {
	return &Runner{jobs: jobs, uploads: uploadRepo, storage: objectStorage, provider: extractionProvider, circuits: circuitService}
}

func (r *Runner) ProcessNext(ctx context.Context) error {
	payload, err := r.jobs.Dequeue(ctx)
	if err != nil {
		return err
	}

	job, err := r.jobs.ClaimQueued(ctx, payload)
	if err != nil {
		return err
	}

	upload, err := r.uploads.GetUpload(ctx, job.UploadID)
	if err != nil {
		_, _ = r.jobs.MarkFailed(ctx, job.ID)
		return err
	}

	body, err := r.storage.Get(ctx, upload.StorageKey)
	if err != nil {
		_, _ = r.jobs.MarkFailed(ctx, job.ID)
		return err
	}
	defer body.Close()

	payloadBytes, err := io.ReadAll(body)
	if err != nil {
		_, _ = r.jobs.MarkFailed(ctx, job.ID)
		return err
	}

	result, err := r.provider.ExtractCircuit(ctx, provider.ExtractionInput{
		ProjectID:   job.ProjectID,
		UploadID:    job.UploadID,
		ObjectKey:   upload.StorageKey,
		ContentType: upload.ContentType,
		Data:        payloadBytes,
	})
	if err != nil {
		_, _ = r.jobs.MarkFailed(ctx, job.ID)
		return err
	}

	persisted, err := r.circuits.Persist(ctx, circuit.PersistInput{
		WorkspaceID: job.WorkspaceID,
		ProjectID:   job.ProjectID,
		UploadID:    job.UploadID,
		Job:         job,
		Result:      result,
	})
	if err != nil {
		_, _ = r.jobs.MarkFailed(ctx, job.ID)
		return err
	}

	_, err = r.jobs.MarkCompleted(ctx, job.ID, persisted.Status)
	return err
}

func (r *Runner) ProcessUntilEmpty(ctx context.Context) error {
	for {
		err := r.ProcessNext(ctx)
		if errors.Is(err, processing.ErrQueueEmpty) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
