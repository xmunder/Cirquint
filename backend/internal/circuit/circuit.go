package circuit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
)

const (
	StatusReady       = "ready"
	StatusNeedsReview = "needs_review"
)

type ExtractionResult struct {
	Provider   string   `json:"provider"`
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
}

type CircuitSpec struct {
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
	Status     string   `json:"status"`
}

type ExtractionRevision struct {
	ID          string           `json:"id"`
	WorkspaceID string           `json:"workspace_id"`
	ProjectID   string           `json:"project_id"`
	UploadID    string           `json:"upload_id"`
	JobID       string           `json:"job_id"`
	Version     int              `json:"version"`
	ObjectKey   string           `json:"object_key"`
	Provider    string           `json:"provider"`
	Confidence  float64          `json:"confidence"`
	Warnings    []string         `json:"warnings"`
	Result      ExtractionResult `json:"result"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type CircuitRevision struct {
	ID                     string      `json:"id"`
	WorkspaceID            string      `json:"workspace_id"`
	ProjectID              string      `json:"project_id"`
	UploadID               string      `json:"upload_id"`
	JobID                  string      `json:"job_id"`
	Version                int         `json:"version"`
	ObjectKey              string      `json:"object_key"`
	Confidence             float64     `json:"confidence"`
	Warnings               []string    `json:"warnings"`
	Status                 string      `json:"status"`
	Spec                   CircuitSpec `json:"spec"`
	AssemblyPlanObjectKey  string      `json:"assembly_plan_object_key,omitempty"`
	SceneSpecObjectKey     string      `json:"scene_spec_object_key,omitempty"`
	ViewerPayloadObjectKey string      `json:"viewer_payload_object_key,omitempty"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
}

type RevisionPair struct {
	Extraction ExtractionRevision `json:"extraction"`
	Circuit    CircuitRevision    `json:"circuit"`
}

type Repository interface {
	GetByJobID(ctx context.Context, jobID string) (RevisionPair, bool, error)
	Save(ctx context.Context, pair RevisionPair) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type PersistInput struct {
	WorkspaceID string
	ProjectID   string
	UploadID    string
	Job         processing.Job
	Result      ExtractionResult
}

type PersistOutput struct {
	Extraction ExtractionRevision
	Circuit    CircuitRevision
	Status     processing.JobStatus
}

type Service struct {
	repo                Repository
	storage             storage.ObjectStorage
	clock               Clock
	idGen               IDGenerator
	reviewMinConfidence float64
}

func NewService(repo Repository, objectStorage storage.ObjectStorage, clock Clock, idGen IDGenerator, reviewMinConfidence float64) *Service {
	return &Service{repo: repo, storage: objectStorage, clock: clock, idGen: idGen, reviewMinConfidence: reviewMinConfidence}
}

func (s *Service) Persist(ctx context.Context, input PersistInput) (PersistOutput, error) {
	if pair, ok, err := s.repo.GetByJobID(ctx, input.Job.ID); err != nil {
		return PersistOutput{}, err
	} else if ok {
		return PersistOutput{Extraction: pair.Extraction, Circuit: pair.Circuit, Status: statusFromCircuit(pair.Circuit.Status)}, nil
	}

	now := s.clock.Now()
	extractionID := s.idGen.NewID()
	circuitID := s.idGen.NewID()
	status := DecideStatus(input.Result.Confidence, input.Result.Warnings, s.reviewMinConfidence)
	spec := Normalize(input.Result, status)

	extraction := ExtractionRevision{
		ID:          extractionID,
		WorkspaceID: input.WorkspaceID,
		ProjectID:   input.ProjectID,
		UploadID:    input.UploadID,
		JobID:       input.Job.ID,
		Version:     1,
		ObjectKey:   storageExtractionObjectKey(input.WorkspaceID, extractionID),
		Provider:    input.Result.Provider,
		Confidence:  input.Result.Confidence,
		Warnings:    append([]string(nil), input.Result.Warnings...),
		Result:      input.Result,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	circuitRevision := CircuitRevision{
		ID:          circuitID,
		WorkspaceID: input.WorkspaceID,
		ProjectID:   input.ProjectID,
		UploadID:    input.UploadID,
		JobID:       input.Job.ID,
		Version:     1,
		ObjectKey:   storageCircuitObjectKey(input.WorkspaceID, circuitID),
		Confidence:  spec.Confidence,
		Warnings:    append([]string(nil), spec.Warnings...),
		Status:      spec.Status,
		Spec:        spec,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.putJSON(ctx, extraction.ObjectKey, extraction); err != nil {
		return PersistOutput{}, err
	}
	if err := s.putJSON(ctx, circuitRevision.ObjectKey, circuitRevision.Spec); err != nil {
		return PersistOutput{}, err
	}

	pair := RevisionPair{Extraction: extraction, Circuit: circuitRevision}
	if err := s.repo.Save(ctx, pair); err != nil {
		return PersistOutput{}, err
	}

	return PersistOutput{Extraction: extraction, Circuit: circuitRevision, Status: statusFromCircuit(status)}, nil
}

func Normalize(result ExtractionResult, status string) CircuitSpec {
	return CircuitSpec{
		Confidence: result.Confidence,
		Warnings:   append([]string(nil), result.Warnings...),
		Status:     status,
	}
}

func DecideStatus(confidence float64, warnings []string, threshold float64) string {
	if confidence < threshold {
		return StatusNeedsReview
	}
	for _, warning := range warnings {
		if isReviewBlockingWarning(warning) {
			return StatusNeedsReview
		}
	}
	return StatusReady
}

func isReviewBlockingWarning(warning string) bool {
	value := strings.ToLower(strings.TrimSpace(warning))
	return strings.Contains(value, "ambiguous") || strings.Contains(value, "incomplete-circuit") || strings.Contains(value, "incomplete circuit")
}

func statusFromCircuit(status string) processing.JobStatus {
	if status == StatusNeedsReview {
		return processing.JobStatusNeedsReview
	}
	return processing.JobStatusReady
}

func storageExtractionObjectKey(workspaceID, revisionID string) string {
	return fmt.Sprintf("workspaces/%s/extractions/%s/v1/result.json", workspaceID, revisionID)
}

func storageCircuitObjectKey(workspaceID, revisionID string) string {
	return fmt.Sprintf("workspaces/%s/circuits/%s/v1/circuit-spec.json", workspaceID, revisionID)
}

func (s *Service) putJSON(ctx context.Context, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.storage.Put(ctx, key, strings.NewReader(string(payload)), storage.ObjectMeta{ContentType: "application/json", SizeBytes: int64(len(payload))})
	return err
}
