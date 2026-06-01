# Design: Circuit Review Workbench MVP

## Technical Approach

Add a backend-first review slice inside the Go modular monolith, but make it real by introducing shared persistence wiring for API and worker instead of per-process memory state. `processing` keeps ownership of job transitions, `circuit` owns review queue/detail/decision behavior, and object storage keeps serialized artifacts. This stays aligned with ADR-001/004's explicit pipeline and ADR-002/006's rule that manual correction happens at `CircuitSpec` before any downstream artifact exists.

## Architecture Decisions

| Decision | Choice | Alternatives considered | Rationale |
|---|---|---|---|
| Runtime persistence | Add shared SQL-backed repositories plus shared object storage wiring in `cmd/api` and `cmd/worker`; keep memory repos for tests | Keep separate in-memory repos in each process | Current runtime bootstrap does not share uploads/jobs/revisions across processes, so review and even queued extraction are not durable system behavior. |
| Review ownership | Put review commands, queries, and validation in `backend/internal/circuit`; keep `server` thin | Put review logic in `server` or `processing` | The reviewable artifact is `CircuitSpec`; `processing` should only enforce state transitions. |
| Audit model | Create immutable `circuit_review_decisions` rows linked to reviewed and resolved revisions | Add mutable review columns on `processing_jobs` or `circuit_revisions` | Separate audit rows preserve traceability across retries and corrections, which ADR-006 requires. |
| Correction persistence | `approve` with `corrected_spec` creates a successor `circuit_revision` version and marks the job `ready`; `reject` leaves the reviewed revision as terminal `failed` | Overwrite the pending revision object | The schema already has versioned revisions; creating a successor keeps the low-confidence artifact inspectable. |

## Data Flow

```text
API upload -> shared repos/storage -> queued job -> worker extract
         -> circuit.Persist(...) -> latest circuit revision = needs_review

GET /projects/{projectID}/circuit-reviews
GET /projects/{projectID}/circuit-reviews/{jobID}
POST /projects/{projectID}/circuit-reviews/{jobID}/decision
  -> circuit.SubmitReviewDecision
  -> validate actor/project/current revision
  -> optional corrected circuit object + audit row + revision/job transition
```

State path for the review slice: `needs_review -> ready | failed`. Only the latest `circuit_revision` for a job is reviewable, and the decision command must carry `circuit_revision_id` for stale-write protection.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/cmd/api/main.go` | Modify | Build shared repository/storage dependencies instead of process-local memory defaults. |
| `backend/cmd/worker/main.go` | Modify | Reuse the same persistence bootstrap so worker sees API-created uploads/jobs/revisions. |
| `backend/internal/config/config.go` | Modify | Add database/object-storage runtime config needed by shared persistence. |
| `backend/internal/circuit/circuit.go` | Modify | Add review queue/detail DTOs, decision command, successor revision creation, and validation. |
| `backend/internal/circuit/memory_repository.go` | Modify | Mirror queue/detail/decision semantics for tests. |
| `backend/internal/circuit/postgres_repository.go` | Create | Query latest reviewable revisions and persist audit/revision changes transactionally. |
| `backend/internal/processing/processing.go` | Modify | Add review resolution transition from `needs_review` to `ready|failed`. |
| `backend/internal/processing/postgres_repository.go` | Create | Support project/status lookup and transactional status changes. |
| `backend/internal/server/server.go` | Modify | Expose queue/detail/decision routes and map review errors to HTTP status codes. |
| `backend/migrations/0002_circuit_review_decisions.sql` | Create | Add review audit table, indexes, and any constraint needed for one decision per reviewed revision. |

## Interfaces / Contracts

- `GET /projects/{projectID}/circuit-reviews` -> `{ "items": [{"job_id","circuit_revision_id","upload_id","confidence","warnings","queued_for_review_at"}] }`
- `GET /projects/{projectID}/circuit-reviews/{jobID}` -> `{ "job", "extraction", "circuit", "review_state" }`
- `POST /projects/{projectID}/circuit-reviews/{jobID}/decision`

```json
{
  "circuit_revision_id": "cir_rev_123",
  "decision": "approve|reject",
  "note": "manual review note",
  "reviewed_at": "2026-05-26T18:00:00Z",
  "corrected_spec": {}
}
```

`reviewer_id` is taken from `identity.ActorFromContext`; MVP may still resolve to `anonymous`, but the value must be persisted. Every query is project-scoped first, then job/revision-scoped. Cross-project access returns not found, and detail never exposes `AssemblyPlan`, `SceneSpec`, or viewer payloads.

## Data Model Changes

Add `circuit_review_decisions(id, workspace_id, project_id, job_id, extraction_revision_id, reviewed_circuit_revision_id, resolved_circuit_revision_id, reviewer_id, decision, note, reviewed_at, created_at)` with unique `(reviewed_circuit_revision_id)` and indexes on `(project_id, created_at)` and `(job_id)`. Keep `extraction_revisions` immutable. Keep `circuit_revisions` versioned; when a correction is supplied, insert a new `circuit_revision` row with the next `version`, a new object key, and status `ready`.

## Validation And Failure Modes

- Reject invalid JSON, bad `decision`, missing `note`, or malformed `reviewed_at` with `400`.
- Reject `approve` without a valid reviewable revision, or with a `corrected_spec` that fails `CircuitSpec` validation, with `400`/`409`.
- Reject missing project/job/revision, cross-project access, or already-resolved revisions with `404`/`409`.
- If object storage write, audit insert, successor revision insert, or job transition fails, the SQL path must roll back the whole decision transaction.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | decision validation, stale revision rejection, successor-version creation | Go table tests in `circuit`. |
| Integration | queue/detail/decision HTTP contracts, actor persistence, project scoping | `httptest` with fixed IDs/clock and golden JSON. |
| Persistence | one-decision-per-revision, transactional rollback, latest-reviewable query | repository tests against memory repo now and SQL repo as part of this change. |

## Migration / Rollout

Ship the additive migration first, then deploy shared persistence bootstrap plus review routes, and exercise the flow with API-level tests before any UI work. Rollback is additive: disable routes and worker wiring, stop creating review rows, and leave existing `needs_review` revisions untouched; the new table can remain unused without corrupting prior artifacts.

## Open Questions

- [ ] `REVIEW_MIN_CONFIDENCE` remains configurable, but the business threshold is still an explicit product decision outside this MVP.
