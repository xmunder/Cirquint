# Design: Circuit Ingestion Pipeline MVP

## Technical Approach

Build the first backend-first vertical slice in the Go modular monolith: create a minimal workspace/project, accept a schematic image upload, store the binary in R2 through `ObjectStorage`, persist metadata and a processing job in PostgreSQL, enqueue extraction in Redis, let a Go worker call `CircuitExtractionProvider`, then persist `ExtractionResult` and a first `CircuitSpec` revision. `AssemblyPlan`, `SceneSpec`, viewer, and planning stages stay explicitly out of scope.

## Architecture Decisions

| Decision | Choice | Alternatives considered | Rationale |
|---|---|---|---|
| Module boundaries | `workspace`, `uploads`, `processing`, `provider`, `storage`, `circuit` under `backend/internal/` | One large pipeline package | Matches ADR-001 and keeps orchestration, storage, AI, and domain normalization separate. |
| Async execution | API writes DB rows and enqueues Redis job; worker owns extraction | Synchronous upload+extract | ADR-004 requires visible state, retries, and non-blocking requests. |
| Artifact persistence | PostgreSQL metadata + R2 JSON payloads for raw extraction/spec | Store all JSON only in DB | ADR-003 expects versioned object keys; DB remains source of truth for references/status. |
| Review gate | `needs_review` persisted on job and circuit revision | Let frontend infer review state | ADR-006 makes review a backend quality decision, not UI policy. |

## Data Flow

```txt
API POST /projects ──→ workspace.projects
API POST /projects/{id}/uploads ──→ uploads metadata ──→ R2 original
                                      │
                                      └──→ processing_jobs queued ──→ Redis
                                                                    │
Worker extract ──→ provider.CircuitExtractionProvider ──→ ExtractionResult
        │                                                    │
        └──→ circuit.Normalize(...) ──→ CircuitSpec revision ──→ ready|needs_review|failed
```

State path for this MVP: `created -> uploaded -> queued -> extracting -> ready|needs_review|failed`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/cmd/api/main.go` | Create | HTTP API entrypoint. |
| `backend/cmd/worker/main.go` | Create | Redis worker entrypoint. |
| `backend/internal/workspace/` | Create | Minimal workspace/project repository and service. |
| `backend/internal/uploads/` | Create | Upload validation, metadata, storage coordination. |
| `backend/internal/processing/` | Create | Job state machine, enqueue, idempotency locks. |
| `backend/internal/provider/` | Create | AI provider interface plus mock/initial adapter. |
| `backend/internal/storage/` | Create | `ObjectStorage` interface and R2 adapter. |
| `backend/internal/circuit/` | Create | `ExtractionResult`, `CircuitSpec`, normalization and review policy. |
| `backend/migrations/` | Create | PostgreSQL tables for MVP entities. |

## Interfaces / Contracts

```go
type ObjectStorage interface {
    Put(ctx context.Context, key string, body io.Reader, meta ObjectMeta) (StoredObject, error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
}

type CircuitExtractionProvider interface {
    ExtractCircuit(ctx context.Context, input ExtractionInput) (ExtractionResult, error)
}
```

Minimal entities: `Workspace`, `Project`, `Upload`, `ProcessingJob`, `ExtractionRevision`, `CircuitRevision`. Revisions store `project_id`, `upload_id`, `job_id`, `version`, `object_key`, `confidence`, `warnings`, `status`, timestamps, and provider metadata.

API surface: `POST /workspaces`, `POST /workspaces/{workspaceID}/projects`, `POST /projects/{projectID}/uploads`, `GET /jobs/{jobID}`, `GET /projects/{projectID}/revisions/latest`.

R2 keys should use the workspace-scoped convention from architecture docs: `workspaces/{workspaceId}/uploads/{uploadId}/v1/source.{ext}`, `workspaces/{workspaceId}/extractions/{revisionId}/v1/result.json`, `workspaces/{workspaceId}/circuits/{revisionId}/v1/circuit-spec.json`.

## Worker Orchestration and Idempotency

`processing` owns state transitions with compare-and-swap updates: a worker may claim only `queued` jobs and set `extracting` atomically. Job payload contains `job_id`, `project_id`, `upload_id`, and expected `upload_version`. Retries reuse the same job and must not create duplicate revisions: revision creation is guarded by `(job_id, artifact_type)` uniqueness and deterministic object keys.

## Review Decision

`circuit` decides `needs_review` after normalization. Persist it when `confidence < configured_threshold` or `warnings` contains blocking ambiguity/incomplete-circuit categories. The job status and latest circuit revision status both store `needs_review`; raw warnings and confidence remain inspectable.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit | State transitions, review policy, key generation, normalization | Go table tests with fixtures; no real R2/Redis/provider. |
| Integration | Upload creates DB rows, enqueues job, worker persists revisions idempotently | Testcontainers or local fakes once scaffold exists. |
| Contract | API status payloads and provider/storage interfaces | Golden JSON fixtures from docs-first contracts. |

## Migration / Rollout

No data migration required because the repo is docs-only today. First implementation creates the initial backend scaffold and migrations.

## Open Questions

- [ ] Exact confidence threshold value for MVP configuration.
