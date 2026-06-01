## Implementation Progress

**Change**: `circuit-review-workbench-mvp`
**Mode**: Strict TDD

### Completed Tasks
- [x] 1.1 Add failing config/bootstrap tests in `backend/internal/config/config_test.go`, `backend/cmd/api/main_test.go`, and `backend/cmd/worker/main_test.go` for shared DB/object-storage wiring.
- [x] 1.2 Create `backend/migrations/0002_circuit_review_decisions.sql` with immutable audit table, unique reviewed-revision constraint, and project/job indexes.
- [x] 1.3 Extend `backend/internal/config/config.go` with runtime DB/object-storage settings required by both binaries.
- [x] 1.4 Wire shared persistence and object storage into `backend/cmd/api/main.go` and `backend/cmd/worker/main.go`, keeping memory defaults test-only.
- [x] 2.1 Write RED cases in `backend/internal/circuit/circuit_test.go` for queue/detail DTOs, stale revision rejection, approve-with-correction, reject-without-correction, and actor audit persistence.
- [x] 2.2 Expand `backend/internal/circuit/circuit.go` with review queries, `SubmitReviewDecision`, successor revision creation, and `CircuitSpec` validation.
- [x] 2.3 Update `backend/internal/circuit/memory_repository.go` and add `backend/internal/circuit/postgres_repository.go` plus repo tests for latest-reviewable lookup, one-decision-per-revision, and rollback semantics.
- [x] 2.4 Add RED/GREEN coverage in `backend/internal/processing/processing_test.go`, then extend `backend/internal/processing/processing.go` and create `backend/internal/processing/postgres_repository.go` for `needs_review -> ready|failed` transitions and project/status lookup.

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `backend/internal/config/config.go` | Modified | Added database/object-storage runtime settings and shared runtime validation. |
| `backend/internal/config/config_test.go` | Modified | Added RED/GREEN coverage for env loading and shared runtime validation. |
| `backend/cmd/api/main.go` | Modified | Made API handler bootstrap validate shared runtime config before serving. |
| `backend/cmd/api/main_test.go` | Modified | Added wiring tests for runtime validation and config propagation in API bootstrap. |
| `backend/cmd/worker/main.go` | Modified | Made worker runner bootstrap validate shared runtime config before execution. |
| `backend/cmd/worker/main_test.go` | Modified | Added wiring tests for runtime validation and config propagation in worker bootstrap. |
| `backend/migrations/0002_circuit_review_decisions.sql` | Created | Added immutable review audit table with reviewed-revision uniqueness and query indexes. |
| `backend/migrations/migrations_test.go` | Created | Added structural migration tests for the review audit DDL. |
| `backend/internal/circuit/circuit.go` | Modified | Added review queue/detail DTOs, stale-revision checks, review submission flow, and circuit-spec validation. |
| `backend/internal/circuit/circuit_test.go` | Modified | Added review queue/detail/decision coverage, audit persistence assertions, and rollback protection tests. |
| `backend/internal/circuit/memory_repository.go` | Modified | Added latest-reviewable queries, immutable decision persistence, and atomic job-transition coordination for tests. |
| `backend/internal/circuit/postgres_repository.go` | Created | Added SQL-backed review repository scaffolding for latest-reviewable lookup and transactional decision persistence. |
| `backend/internal/processing/processing.go` | Modified | Added review-resolution and project/status query helpers. |
| `backend/internal/processing/processing_test.go` | Modified | Added RED/GREEN coverage for `needs_review -> ready|failed` resolution and project/status filtering. |
| `backend/internal/processing/memory_repository.go` | Modified | Added in-memory project/status job listing used by review flows. |
| `backend/internal/processing/postgres_repository.go` | Created | Added SQL-backed processing repository scaffolding for status updates and project/status lookup. |
| `openspec/changes/circuit-review-workbench-mvp/tasks.md` | Modified | Marked Phase 1 foundation tasks complete and recorded resolved chain strategy. |
| `openspec/changes/circuit-review-workbench-mvp/tasks.md` | Modified | Marked Phase 2 review-domain and repository tasks complete. |
| `openspec/changes/circuit-review-workbench-mvp/apply-progress.md` | Modified | Merged PR 2 progress into the cumulative strict-TDD apply artifact. |

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `backend/internal/config/config_test.go`, `backend/cmd/api/main_test.go`, `backend/cmd/worker/main_test.go` | Unit | ✅ `go test ./internal/config`, `go test ./cmd/api`, `go test ./cmd/worker` | ✅ Written | ✅ Same package tests passing | ✅ Added env propagation and missing-config cases across both binaries | ✅ Extracted shared validation seam |
| 1.2 | `backend/migrations/migrations_test.go` | Unit | N/A (new) | ✅ Written | ✅ `go test ./migrations` | ✅ Verified both audit schema and query indexes | ➖ None needed |
| 1.3 | `backend/internal/config/config_test.go` | Unit | ✅ `go test ./internal/config` | ✅ Written | ✅ `go test ./internal/config` | ✅ Covered defaults, env values, and five invalid shared-runtime cases | ✅ Added `ObjectStorageConfig` + `ValidateSharedRuntime` |
| 1.4 | `backend/cmd/api/main_test.go`, `backend/cmd/worker/main_test.go` | Unit | ✅ `go test ./cmd/api`, `go test ./cmd/worker` | ✅ Written | ✅ `go test ./cmd/api`, `go test ./cmd/worker` | ✅ Covered validation failure plus loaded-config propagation in both entrypoints | ➖ None needed |
| 2.1 | `backend/internal/circuit/circuit_test.go` | Unit | ✅ `go test ./internal/circuit` | ✅ Written | ✅ `go test ./internal/circuit` | ✅ Covered queue/detail DTOs, stale revision rejection, corrected approve, reject, and actor audit persistence | ✅ Extracted review DTO and clone helpers |
| 2.2 | `backend/internal/circuit/circuit_test.go` | Unit | ✅ `go test ./internal/circuit` | ✅ Written | ✅ `go test ./internal/circuit` | ✅ Covered successor revision creation, corrected-spec validation seam, and terminal review states | ✅ Kept decision flow inside `circuit.Service` with minimal new types |
| 2.3 | `backend/internal/circuit/circuit_test.go` | Unit | ✅ `go test ./internal/circuit` | ✅ Written | ✅ `go test ./internal/circuit` | ✅ Covered latest-reviewable lookup, one-decision-per-revision, and rollback when job transition fails | ✅ Added atomic memory-repo decision persistence plus SQL scaffold |
| 2.4 | `backend/internal/processing/processing_test.go` | Unit | ✅ `go test ./internal/processing` | ✅ Written | ✅ `go test ./internal/processing` | ✅ Covered `needs_review -> ready`, `needs_review -> failed`, invalid targets, and project/status filtering | ➖ None needed |

### Test Summary
- **Total tests written**: 14
- **Total tests passing**: 35
- **Layers used**: Unit (35), Integration (0), E2E (0)
- **Approval tests** (refactoring): None - no refactoring tasks
- **Pure functions created**: 3

### Deviations from Design
None - implementation stays inside the PR 2 boundary and keeps HTTP/viewer work out of scope.

### Issues Found
Runtime entrypoints still instantiate memory repositories today; the new Postgres repositories are implemented for this slice but are not wired until the later HTTP/runtime integration slice.

### Remaining Tasks
- [ ] 3.1 Add failing API scenarios in `backend/internal/server/server_test.go` for project queue listing, review detail, project scoping, and decision validation/error codes from the specs.
- [ ] 3.2 Extend `backend/internal/server/server.go` with `GET /projects/{projectID}/circuit-reviews`, `GET /projects/{projectID}/circuit-reviews/{jobID}`, and `POST /projects/{projectID}/circuit-reviews/{jobID}/decision` using `identity.ActorFromContext`.
- [ ] 3.3 Keep detail responses limited to job, extraction, normalized `CircuitSpec`, confidence, warnings, and review state; explicitly exclude planning/viewer payloads in route serialization.
- [ ] 4.1 Add repository/integration assertions that review decisions leave extraction attempts immutable and preserve prior audit rows across later retries.
- [ ] 4.2 Run `cd backend && go test ./...` after implementation and update this checklist with completed items during `sdd-apply`.

### Workload / PR Boundary
- Mode: chained PR slice
- Current work unit: PR 2 - review domain + repositories only
- Boundary: `circuit` and `processing` domain/repository behavior only; no HTTP handlers, server serialization, or viewer payload work
- Estimated review budget impact: bounded backend-domain diff with focused review behavior and repository scaffolding

### Status
8/13 tasks complete. Ready for PR 3 HTTP slice.
