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
- [x] 3.1 Add failing API scenarios in `backend/internal/server/server_test.go` for project queue listing, review detail, project scoping, and decision validation/error codes from the specs.
- [x] 3.2 Extend `backend/internal/server/server.go` with `GET /projects/{projectID}/circuit-reviews`, `GET /projects/{projectID}/circuit-reviews/{jobID}`, and `POST /projects/{projectID}/circuit-reviews/{jobID}/decision` using `identity.ActorFromContext`.
- [x] 3.3 Keep detail responses limited to job, extraction, normalized `CircuitSpec`, confidence, warnings, and review state; explicitly exclude planning/viewer payloads in route serialization.
- [x] 4.1 Add repository/integration assertions that review decisions leave extraction attempts immutable and preserve prior audit rows across later retries.
- [x] 4.2 Run `cd backend && go test ./...` after implementation and update this checklist with completed items during `sdd-apply`.
- [x] 4.3 Wire `sql.DB` bootstrap plus Postgres-backed `processing`/`circuit` repositories into `backend/cmd/api/main.go` and `backend/cmd/worker/main.go`, keeping in-memory defaults only for dependencies that still lack SQL implementations.
- [x] 4.4 Add focused runtime wiring tests plus one Docker-Postgres worker integration path, then rerun the relevant backend packages serially against `TEST_DATABASE_URL`.
- [x] 4.5 Add Postgres-backed `workspace`/`uploads` repositories plus a Docker-Postgres API upload -> worker proof that narrows the remaining runtime warning to object storage only.
- [x] 4.6 Replace the memory-only runtime object storage fallback with shared filesystem-backed storage when `OBJECT_STORAGE_PATH` is configured, then re-verify the upload -> worker -> `needs_review` path against Docker Postgres.
- [x] Follow-up: isolate Docker-Postgres integration tests per package schema so the formerly serial verification subset now passes in default parallel mode without `-p 1`.
- [x] Follow-up: add direct spec-proof tests for an empty project review queue and high-confidence `ready` persistence, then refresh the compliance matrix to 11/11.

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `backend/internal/config/config.go` | Modified | Added database/object-storage runtime settings plus `OBJECT_STORAGE_PATH` for shared filesystem-backed storage selection. |
| `backend/internal/config/config_test.go` | Modified | Added RED/GREEN coverage for env loading, including `OBJECT_STORAGE_PATH`, and shared runtime validation. |
| `backend/cmd/api/main.go` | Modified | Opened runtime `sql.DB`, injected Postgres-backed `workspace`/`uploads`/`processing`/`circuit` repositories into API bootstrap, and now resolves shared filesystem object storage when configured. |
| `backend/cmd/api/main_test.go` | Modified | Added runtime wiring assertions for shared filesystem storage and upgraded the Docker-Postgres upload -> worker proof to use separate API/worker storage instances over the same directory. |
| `backend/cmd/worker/main.go` | Modified | Opened runtime `sql.DB`, composed Postgres-backed `uploads`/`processing`/`circuit` services, and now resolves shared filesystem object storage from runtime config when not test-injected. |
| `backend/cmd/worker/main_test.go` | Modified | Added a Docker-Postgres worker runtime path that seeds upload bytes through one filesystem storage instance and proves the worker can read them through another via config wiring. |
| `backend/internal/storage/filesystem.go` | Created | Added shared filesystem-backed object storage that persists keys across independent runtime instances on the same host. |
| `backend/internal/storage/storage_test.go` | Modified | Added coverage proving separate filesystem storage instances can read the same stored object. |
| `backend/migrations/0002_circuit_review_decisions.sql` | Created | Added immutable review audit table with reviewed-revision uniqueness and query indexes. |
| `backend/migrations/migrations_test.go` | Created | Added structural migration tests for the review audit DDL. |
| `backend/internal/circuit/circuit.go` | Modified | Added review queue/detail DTOs, stale-revision checks, review submission flow, and circuit-spec validation. |
| `backend/internal/circuit/circuit_test.go` | Modified | Added review queue/detail/decision coverage, audit persistence assertions, rollback protection tests, and a direct high-confidence `ready` persistence proof. |
| `backend/internal/circuit/memory_repository.go` | Modified | Added latest-reviewable queries, immutable decision persistence, and atomic job-transition coordination for tests. |
| `backend/internal/circuit/postgres_repository.go` | Created | Added SQL-backed review repository behavior and now rejects zero-row `processing_jobs` transitions with rollback semantics. |
| `backend/internal/circuit/postgres_repository_test.go` | Created | Added real Postgres integration coverage for rollback semantics and review-audit preservation across later retries. |
| `backend/internal/processing/processing.go` | Modified | Added review-resolution and project/status query helpers. |
| `backend/internal/processing/memory_repository.go` | Modified | Added in-memory project/status job listing used by review flows. |
| `backend/internal/processing/postgres_repository.go` | Created | Added SQL-backed processing repository scaffolding for status updates and project/status lookup. |
| `backend/internal/uploads/postgres_repository.go` | Created | Added SQL-backed upload persistence for create, status transitions, lookup, and cleanup. |
| `backend/internal/uploads/postgres_repository_test.go` | Created | Added real Postgres lifecycle coverage for upload create/update/get/delete behavior. |
| `backend/internal/workspace/postgres_repository.go` | Created | Added SQL-backed workspace/project persistence and lookup used by API upload routes. |
| `backend/internal/workspace/postgres_repository_test.go` | Created | Added real Postgres coverage for workspace/project round-trips and missing-project mapping. |
| `backend/internal/testpostgres/testpostgres.go` | Created | Added shared Docker-Postgres test helper that assigns each package its own schema via `search_path`, resets only that schema, and reapplies migrations. |
| `backend/internal/testpostgres/testpostgres_test.go` | Created | Added RED/GREEN coverage for schema-name sanitization and isolated connection URL generation. |
| `backend/internal/server/server.go` | Modified | Added review queue/detail/decision routes, project-route dispatch, and HTTP error mapping for review flows. |
| `backend/internal/server/server_test.go` | Modified | Added strict-TDD HTTP coverage for queue/detail contracts, project scoping, invalid payloads, review conflict mapping, and a direct empty-queue scenario proof. |
| `backend/go.mod` | Modified | Added the Postgres SQL driver needed for real repository integration tests. |
| `backend/go.sum` | Modified | Recorded checksums for the Postgres SQL driver. |
| `openspec/changes/circuit-review-workbench-mvp/tasks.md` | Modified | Marked the final Phase 4 runtime-storage follow-up item complete. |
| `openspec/changes/circuit-review-workbench-mvp/apply-progress.md` | Modified | Merged the cumulative strict-TDD progress through the final direct spec-proof follow-up without overwriting prior slice history. |
| `openspec/changes/circuit-review-workbench-mvp/verify-report.md` | Modified | Replaced the last two PARTIAL matrix entries with direct test evidence and raised compliance to 11/11. |

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
| 3.1 | `backend/internal/server/server_test.go` | Integration | ✅ `go test ./internal/server` | ✅ Written | ✅ `go test ./internal/server -run 'TestCircuitReview'` | ✅ Covered queue listing, detail retrieval, project scoping, and decision validation/error mapping | ✅ Reused focused review HTTP fixtures |
| 3.2 | `backend/internal/server/server_test.go` | Integration | ✅ `go test ./internal/server` | ✅ Written | ✅ `go test ./internal/server -run 'TestCircuitReview'` | ✅ Forced route dispatch for GET queue/detail and POST decision, including actor propagation from identity middleware | ✅ Kept `server` thin |
| 3.3 | `backend/internal/server/server_test.go` | Integration | ✅ `go test ./internal/server` | ✅ Written | ✅ `go test ./internal/server -run 'TestCircuitReview'` | ✅ Verified detail responses exclude planning and viewer payloads | ➖ None needed |
| 4.1 | `backend/internal/circuit/postgres_repository_test.go` | Integration | ✅ `go test ./internal/circuit` and pre-existing review-domain tests | ✅ Written first, then failed on missing job-transition rollback in SQL path | ✅ `TEST_DATABASE_URL=... go test ./internal/circuit -run TestPostgresRepository -count=1` | ✅ Covered both rollback-on-zero-row-transition and preserved audit/extraction data across a later retry | ✅ Fixed repo-level transition guard with minimal SQL-result check |
| 4.2 | `backend/internal/circuit/postgres_repository_test.go` plus existing backend test suite | Integration | ✅ `go test ./...` from `backend` | ✅ Used the new Postgres slice as the final tracer bullet before full verification | ✅ `TEST_DATABASE_URL=... go test ./...` | ✅ Combined targeted real-Postgres verification with full backend regression run | ➖ None needed |
| 4.3 | `backend/cmd/api/main_test.go`, `backend/cmd/worker/main_test.go` | Unit | ✅ `go test ./cmd/api`, `go test ./cmd/worker` | ✅ Added bootstrap wiring tests first; existing unit tests failed until DB dialing was isolated behind seams | ✅ `go test ./cmd/api -run TestNewHandlerUsesPostgresRepositoriesWhenDatabaseConfigured -count=1`; `go test ./cmd/worker -run TestBuildRuntimeRunnerProcessesQueuedJobWithPostgresRepositories -count=1` | ✅ Proved both binaries now open runtime `sql.DB` and compose Postgres-backed `processing`/`circuit` services | ✅ Kept memory only for `workspace`, `uploads`, and object storage where SQL/runtime implementations do not yet exist |
| 4.4 | `backend/cmd/worker/main_test.go` plus relevant package suites | Integration | ✅ serial package verification against Docker Postgres | ✅ Worker runtime integration test was added before the final serial package run | ✅ `TEST_DATABASE_URL=... go test -p 1 ./cmd/api ./cmd/worker ./internal/circuit ./internal/config ./internal/processing ./internal/server -count=1` | ✅ Covered a real worker dequeue -> extract -> persist -> needs_review path with Postgres-backed job/circuit repositories | ✅ Switched to serial verification because package-parallel schema resets race on the shared Docker database |
| 4.5 | `backend/internal/workspace/postgres_repository_test.go`, `backend/internal/uploads/postgres_repository_test.go`, `backend/cmd/api/main_test.go` | Integration | ✅ serial Docker-Postgres tests for changed packages | ✅ Added repo/runtime RED tests first; they failed on missing Postgres repositories and runtime wiring | ✅ `TEST_DATABASE_URL=... go test ./cmd/api -run TestUploadRouteAndWorkerUsePostgresWorkspaceAndUploadRepositories -count=1`; `TEST_DATABASE_URL=... go test -p 1 ./cmd/api ./cmd/worker ./internal/uploads ./internal/workspace -count=1` | ✅ Proved API workspace/project/upload creation and worker upload lookup now share real Postgres persistence, with only object storage still in-memory | ✅ Kept the slice surgical by reusing existing routes/services instead of adding new product behavior |
| 4.6 | `backend/internal/storage/storage_test.go`, `backend/cmd/api/main_test.go`, `backend/cmd/worker/main_test.go` | Integration | ✅ `go test ./internal/storage ./internal/config ./cmd/api ./cmd/worker` | ✅ Added filesystem-object-storage RED tests before wiring `OBJECT_STORAGE_PATH` into both binaries | ✅ `go test ./internal/storage -run TestFilesystemObjectStorageSharesObjectsAcrossInstances -count=1`; `TEST_DATABASE_URL=... go test ./cmd/api -run TestUploadRouteAndWorkerUsePostgresWorkspaceAndUploadRepositories -count=1`; `TEST_DATABASE_URL=... go test ./cmd/worker -run TestBuildRuntimeRunnerProcessesQueuedJobWithPostgresRepositories -count=1` | ✅ Proved upload bytes written by the API can be read by the worker through separate filesystem storage instances over the same directory, while job/upload/circuit state stays in Postgres | ✅ Kept the implementation small by extending the existing storage abstraction instead of adding new infrastructure |
| Follow-up | `backend/internal/testpostgres/testpostgres_test.go`, `backend/cmd/api/main_test.go`, `backend/cmd/worker/main_test.go`, `backend/internal/circuit/postgres_repository_test.go`, `backend/internal/uploads/postgres_repository_test.go`, `backend/internal/workspace/postgres_repository_test.go` | Integration | ✅ Parallel Docker-Postgres package subset | ✅ Added helper RED tests first; existing package helpers still shared `public` and could not safely run together | ✅ `go test ./internal/testpostgres -count=1`; `TEST_DATABASE_URL=... go test ./cmd/api ./cmd/worker ./internal/circuit ./internal/uploads ./internal/workspace ./internal/storage ./internal/config -count=1` | ✅ Proved the formerly serial subset now passes in default package-parallel mode by isolating each test package to its own schema, and fixed `cmd/worker` to use the same isolated DSN as its seed data | ✅ Removed duplicated schema-reset/migration helpers from five packages into one focused test helper |
| Follow-up | `backend/internal/server/server_test.go`, `backend/internal/circuit/circuit_test.go` | Integration + Unit | ✅ `go test ./internal/server ./internal/circuit -count=1` | ✅ Added direct scenario tests first to replace the matrix's indirect/partial coverage claims | ✅ `go test ./internal/server -run TestCircuitReviewQueueReturnsEmptyWhenProjectHasNoPendingReviewWork -count=1`; `go test ./internal/circuit -run TestPersistHighConfidenceExtractionBecomesReady -count=1`; `go test ./internal/server ./internal/circuit -count=1` | ✅ Proved `GET /projects/{projectID}/circuit-reviews` returns an empty list when that project has no pending review work, and `Persist` stores a high-confidence extraction as `ready` while the job completes to `ready` | ➖ No refactor needed; behavior already existed |

### Test Summary
- **Targeted new tests written in this follow-up slice**: 1 direct empty-review-queue HTTP test and 1 direct high-confidence `ready` persistence test, on top of the prior runtime wiring coverage
- **Verification commands run**:
- `go test ./cmd/api -run TestNewHandlerUsesPostgresRepositoriesWhenDatabaseConfigured -count=1`
- `go test ./internal/storage -run TestFilesystemObjectStorageSharesObjectsAcrossInstances -count=1`
- `TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:55432/cirquint_test?sslmode=disable go test ./cmd/api -run TestUploadRouteAndWorkerUsePostgresWorkspaceAndUploadRepositories -count=1`
- `TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:55432/cirquint_test?sslmode=disable go test ./cmd/worker -run TestBuildRuntimeRunnerProcessesQueuedJobWithPostgresRepositories -count=1`
- `go test ./internal/server -run TestCircuitReviewQueueReturnsEmptyWhenProjectHasNoPendingReviewWork -count=1`
- `go test ./internal/circuit -run TestPersistHighConfidenceExtractionBecomesReady -count=1`
- `go test ./internal/server ./internal/circuit -count=1`
- `go test ./internal/config ./internal/storage -count=1`
- `go test ./internal/testpostgres -count=1`
- `TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:55432/cirquint_test?sslmode=disable go test ./cmd/api ./cmd/worker ./internal/circuit ./internal/uploads ./internal/workspace ./internal/storage ./internal/config -count=1`
- **Observed package coverage**: Prior `backend/internal/circuit` 73.9% statement coverage remains informational from the earlier repository slice; this follow-up focused on runtime wiring.
- **Layers used across the change**: Unit, Integration, real Docker-Postgres runtime verification

### Deviations from Design
The design goal is now met for the strongest feasible local-runtime path: both binaries bootstrap Postgres-backed `workspace`, `uploads`, `processing`, and `circuit` repositories where those implementations exist, and they switch to shared filesystem-backed object storage when `OBJECT_STORAGE_PATH` is configured. A cloud-backed runtime implementation still does not exist, but the former memory-only runtime warning is closed for the proven shared-host path.

### Issues Found
- The SQL review-decision path previously committed audit/circuit changes even when `processing_jobs` did not transition from `needs_review`; Phase 4 fixed this by treating zero-row updates as `processing.ErrInvalidStatusTransition` or `processing.ErrJobNotFound` and rolling back.
- The serial verification warning came from five packages independently resetting the shared `public` schema on the same Docker database. This slice fixed it by isolating each package to its own schema through a shared test helper and matching runtime DSNs where needed.

### Remaining Tasks
- [x] None for this runtime-wiring follow-up slice.

### Workload / PR Boundary
- Mode: chained PR slice
- Current work unit: PR 4b - runtime wiring follow-up
- Boundary: runtime SQL bootstrap, explicit memory-only fallbacks, and targeted Docker-Postgres verification
- Estimated review budget impact: small entrypoint wiring diff plus focused runtime/integration tests

### Status
17/17 tasks complete. Phase 4 now also includes package-isolated Docker-Postgres test schemas, the previously serial verification subset passes in default parallel mode without `-p 1`, and the spec compliance matrix is closed at 11/11 with no PARTIAL rows.
