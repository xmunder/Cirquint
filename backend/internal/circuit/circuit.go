package circuit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/msi/circuit-storys/backend/internal/processing"
	"github.com/msi/circuit-storys/backend/internal/storage"
)

const (
	StatusReady       = "ready"
	StatusNeedsReview = "needs_review"
	StatusFailed      = "failed"
)

type ReviewDecision string

const (
	ReviewDecisionApprove ReviewDecision = "approve"
	ReviewDecisionReject  ReviewDecision = "reject"
)

var (
	ErrReviewNotFound          = errors.New("review not found")
	ErrStaleReviewRevision     = errors.New("stale review revision")
	ErrRevisionAlreadyReviewed = errors.New("revision already reviewed")
	ErrInvalidReviewDecision   = errors.New("invalid review decision")
	ErrInvalidReviewNote       = errors.New("invalid review note")
	ErrInvalidReviewerID       = errors.New("invalid reviewer id")
	ErrCorrectedSpecNotAllowed = errors.New("corrected circuit spec not allowed")
	ErrInvalidCircuitSpec      = errors.New("invalid circuit spec")
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

type ReviewQueueItem struct {
	JobID             string    `json:"job_id"`
	CircuitRevisionID string    `json:"circuit_revision_id"`
	UploadID          string    `json:"upload_id"`
	Confidence        float64   `json:"confidence"`
	Warnings          []string  `json:"warnings"`
	QueuedForReviewAt time.Time `json:"queued_for_review_at"`
}

type ReviewJob struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	UploadID  string `json:"upload_id"`
}

type ReviewState struct {
	JobStatus         processing.JobStatus `json:"job_status"`
	CircuitRevisionID string               `json:"circuit_revision_id"`
	CircuitStatus     string               `json:"circuit_status"`
}

type ReviewDetail struct {
	Job         ReviewJob          `json:"job"`
	Extraction  ExtractionRevision `json:"extraction"`
	Circuit     CircuitSpec        `json:"circuit"`
	Confidence  float64            `json:"confidence"`
	Warnings    []string           `json:"warnings"`
	ReviewState ReviewState        `json:"review_state"`
}

type ReviewDecisionRecord struct {
	ID                        string         `json:"id"`
	WorkspaceID               string         `json:"workspace_id"`
	ProjectID                 string         `json:"project_id"`
	JobID                     string         `json:"job_id"`
	ExtractionRevisionID      string         `json:"extraction_revision_id"`
	ReviewedCircuitRevisionID string         `json:"reviewed_circuit_revision_id"`
	ResolvedCircuitRevisionID string         `json:"resolved_circuit_revision_id"`
	ReviewerID                string         `json:"reviewer_id"`
	Decision                  ReviewDecision `json:"decision"`
	Note                      string         `json:"note"`
	ReviewedAt                time.Time      `json:"reviewed_at"`
	CreatedAt                 time.Time      `json:"created_at"`
}

type SubmitReviewDecisionInput struct {
	ProjectID            string
	JobID                string
	CircuitRevisionID    string
	Decision             ReviewDecision
	Note                 string
	ReviewerID           string
	ReviewedAt           time.Time
	CorrectedCircuitSpec *CircuitSpec
}

type SubmitReviewDecisionOutput struct {
	Decision         ReviewDecisionRecord `json:"decision"`
	ReviewedRevision CircuitRevision      `json:"reviewed_revision"`
	ResolvedRevision CircuitRevision      `json:"resolved_revision"`
	JobStatus        processing.JobStatus `json:"job_status"`
}

type Repository interface {
	GetByJobID(ctx context.Context, jobID string) (RevisionPair, bool, error)
	GetLatestReviewable(ctx context.Context, projectID, jobID string) (RevisionPair, bool, error)
	ListLatestReviewable(ctx context.Context, projectID string) ([]RevisionPair, error)
	Save(ctx context.Context, pair RevisionPair) error
	SaveReviewDecision(ctx context.Context, decision ReviewDecisionRecord, resolved CircuitRevision) error
	GetReviewDecision(ctx context.Context, reviewedCircuitRevisionID string) (ReviewDecisionRecord, bool, error)
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

func (s *Service) ListReviewQueue(ctx context.Context, projectID string) ([]ReviewQueueItem, error) {
	pairs, err := s.repo.ListLatestReviewable(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]ReviewQueueItem, 0, len(pairs))
	for _, pair := range pairs {
		items = append(items, ReviewQueueItem{
			JobID:             pair.Circuit.JobID,
			CircuitRevisionID: pair.Circuit.ID,
			UploadID:          pair.Circuit.UploadID,
			Confidence:        pair.Circuit.Confidence,
			Warnings:          append([]string(nil), pair.Circuit.Warnings...),
			QueuedForReviewAt: pair.Circuit.CreatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].QueuedForReviewAt.Equal(items[j].QueuedForReviewAt) {
			return items[i].JobID < items[j].JobID
		}
		return items[i].QueuedForReviewAt.Before(items[j].QueuedForReviewAt)
	})
	return items, nil
}

func (s *Service) GetReviewDetail(ctx context.Context, projectID, jobID string) (ReviewDetail, error) {
	pair, ok, err := s.repo.GetLatestReviewable(ctx, projectID, jobID)
	if err != nil {
		return ReviewDetail{}, err
	}
	if !ok {
		return ReviewDetail{}, ErrReviewNotFound
	}
	return ReviewDetail{
		Job:        ReviewJob{ID: pair.Circuit.JobID, ProjectID: pair.Circuit.ProjectID, UploadID: pair.Circuit.UploadID},
		Extraction: cloneExtractionRevision(pair.Extraction),
		Circuit:    cloneCircuitSpec(pair.Circuit.Spec),
		Confidence: pair.Circuit.Confidence,
		Warnings:   append([]string(nil), pair.Circuit.Warnings...),
		ReviewState: ReviewState{
			JobStatus:         processing.JobStatusNeedsReview,
			CircuitRevisionID: pair.Circuit.ID,
			CircuitStatus:     pair.Circuit.Status,
		},
	}, nil
}

func (s *Service) SubmitReviewDecision(ctx context.Context, input SubmitReviewDecisionInput) (SubmitReviewDecisionOutput, error) {
	if input.Decision != ReviewDecisionApprove && input.Decision != ReviewDecisionReject {
		return SubmitReviewDecisionOutput{}, ErrInvalidReviewDecision
	}
	if strings.TrimSpace(input.Note) == "" {
		return SubmitReviewDecisionOutput{}, ErrInvalidReviewNote
	}
	if strings.TrimSpace(input.ReviewerID) == "" {
		return SubmitReviewDecisionOutput{}, ErrInvalidReviewerID
	}
	if input.Decision == ReviewDecisionReject && input.CorrectedCircuitSpec != nil {
		return SubmitReviewDecisionOutput{}, ErrCorrectedSpecNotAllowed
	}
	if _, ok, err := s.repo.GetReviewDecision(ctx, input.CircuitRevisionID); err != nil {
		return SubmitReviewDecisionOutput{}, err
	} else if ok {
		return SubmitReviewDecisionOutput{}, ErrRevisionAlreadyReviewed
	}

	pair, ok, err := s.repo.GetLatestReviewable(ctx, input.ProjectID, input.JobID)
	if err != nil {
		return SubmitReviewDecisionOutput{}, err
	}
	if !ok {
		return SubmitReviewDecisionOutput{}, ErrReviewNotFound
	}
	if pair.Circuit.ID != input.CircuitRevisionID {
		return SubmitReviewDecisionOutput{}, ErrStaleReviewRevision
	}

	now := s.clock.Now()
	resolved := cloneCircuitRevision(pair.Circuit)
	resolved.UpdatedAt = now

	if input.Decision == ReviewDecisionApprove {
		if input.CorrectedCircuitSpec != nil {
			spec, validateErr := validateCircuitSpec(*input.CorrectedCircuitSpec, StatusReady)
			if validateErr != nil {
				return SubmitReviewDecisionOutput{}, validateErr
			}
			successorID := s.idGen.NewID()
			resolved = CircuitRevision{
				ID:          successorID,
				WorkspaceID: pair.Circuit.WorkspaceID,
				ProjectID:   pair.Circuit.ProjectID,
				UploadID:    pair.Circuit.UploadID,
				JobID:       pair.Circuit.JobID,
				Version:     pair.Circuit.Version + 1,
				ObjectKey:   storageCircuitObjectKey(pair.Circuit.WorkspaceID, successorID),
				Confidence:  spec.Confidence,
				Warnings:    append([]string(nil), spec.Warnings...),
				Status:      StatusReady,
				Spec:        spec,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.putJSON(ctx, resolved.ObjectKey, resolved.Spec); err != nil {
				return SubmitReviewDecisionOutput{}, err
			}
		} else {
			resolved.Status = StatusReady
			resolved.Spec.Status = StatusReady
		}
	} else {
		resolved.Status = StatusFailed
		resolved.Spec.Status = StatusFailed
	}

	decision := ReviewDecisionRecord{
		ID:                        s.idGen.NewID(),
		WorkspaceID:               pair.Circuit.WorkspaceID,
		ProjectID:                 pair.Circuit.ProjectID,
		JobID:                     pair.Circuit.JobID,
		ExtractionRevisionID:      pair.Extraction.ID,
		ReviewedCircuitRevisionID: pair.Circuit.ID,
		ResolvedCircuitRevisionID: resolved.ID,
		ReviewerID:                strings.TrimSpace(input.ReviewerID),
		Decision:                  input.Decision,
		Note:                      strings.TrimSpace(input.Note),
		ReviewedAt:                input.ReviewedAt,
		CreatedAt:                 now,
	}

	if err := s.repo.SaveReviewDecision(ctx, decision, resolved); err != nil {
		if input.CorrectedCircuitSpec != nil {
			_ = s.storage.Delete(ctx, resolved.ObjectKey)
		}
		return SubmitReviewDecisionOutput{}, err
	}

	return SubmitReviewDecisionOutput{
		Decision:         decision,
		ReviewedRevision: cloneCircuitRevision(pair.Circuit),
		ResolvedRevision: cloneCircuitRevision(resolved),
		JobStatus:        statusFromCircuit(resolved.Status),
	}, nil
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
	if status == StatusFailed {
		return processing.JobStatusFailed
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

func validateCircuitSpec(spec CircuitSpec, status string) (CircuitSpec, error) {
	if spec.Confidence < 0 || spec.Confidence > 1 {
		return CircuitSpec{}, ErrInvalidCircuitSpec
	}
	return CircuitSpec{
		Confidence: spec.Confidence,
		Warnings:   append([]string(nil), spec.Warnings...),
		Status:     status,
	}, nil
}

func cloneExtractionRevision(revision ExtractionRevision) ExtractionRevision {
	revision.Warnings = append([]string(nil), revision.Warnings...)
	revision.Result.Warnings = append([]string(nil), revision.Result.Warnings...)
	return revision
}

func cloneCircuitRevision(revision CircuitRevision) CircuitRevision {
	revision.Warnings = append([]string(nil), revision.Warnings...)
	revision.Spec = cloneCircuitSpec(revision.Spec)
	return revision
}

func cloneCircuitSpec(spec CircuitSpec) CircuitSpec {
	spec.Warnings = append([]string(nil), spec.Warnings...)
	return spec
}
