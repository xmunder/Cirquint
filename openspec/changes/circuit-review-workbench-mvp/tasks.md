# Tasks: Circuit Review Workbench MVP

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 900-1300 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 persistence bootstrap -> PR 2 review domain -> PR 3 HTTP/tests |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No (resolved in apply: feature-branch-chain)
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Shared persistence foundation | PR 1 | Migration, config, API/worker bootstrap; base sets chain strategy boundary |
| 2 | Review domain + state transitions | PR 2 | Depends on PR 1; circuit/processing repos and decision flow |
| 3 | HTTP contracts + integration coverage | PR 3 | Depends on PR 2; routes, error mapping, end-to-end API tests |

## Phase 1: Persistence Foundation

- [x] 1.1 Add failing config/bootstrap tests in `backend/internal/config/config_test.go`, `backend/cmd/api/main_test.go`, and `backend/cmd/worker/main_test.go` for shared DB/object-storage wiring.
- [x] 1.2 Create `backend/migrations/0002_circuit_review_decisions.sql` with immutable audit table, unique reviewed-revision constraint, and project/job indexes.
- [x] 1.3 Extend `backend/internal/config/config.go` with runtime DB/object-storage settings required by both binaries.
- [x] 1.4 Wire shared persistence and object storage into `backend/cmd/api/main.go` and `backend/cmd/worker/main.go`, keeping memory defaults test-only.

## Phase 2: Review Domain And Repositories

- [x] 2.1 Write RED cases in `backend/internal/circuit/circuit_test.go` for queue/detail DTOs, stale revision rejection, approve-with-correction, reject-without-correction, and actor audit persistence.
- [x] 2.2 Expand `backend/internal/circuit/circuit.go` with review queries, `SubmitReviewDecision`, successor revision creation, and `CircuitSpec` validation.
- [x] 2.3 Update `backend/internal/circuit/memory_repository.go` and add `backend/internal/circuit/postgres_repository.go` plus repo tests for latest-reviewable lookup, one-decision-per-revision, and rollback semantics.
- [x] 2.4 Add RED/GREEN coverage in `backend/internal/processing/processing_test.go`, then extend `backend/internal/processing/processing.go` and create `backend/internal/processing/postgres_repository.go` for `needs_review -> ready|failed` transitions and project/status lookup.

## Phase 3: HTTP Review Workbench Slice

- [ ] 3.1 Add failing API scenarios in `backend/internal/server/server_test.go` for project queue listing, review detail, project scoping, and decision validation/error codes from the specs.
- [ ] 3.2 Extend `backend/internal/server/server.go` with `GET /projects/{projectID}/circuit-reviews`, `GET /projects/{projectID}/circuit-reviews/{jobID}`, and `POST /projects/{projectID}/circuit-reviews/{jobID}/decision` using `identity.ActorFromContext`.
- [ ] 3.3 Keep detail responses limited to job, extraction, normalized `CircuitSpec`, confidence, warnings, and review state; explicitly exclude planning/viewer payloads in route serialization.

## Phase 4: Verification And Polish

- [ ] 4.1 Add repository/integration assertions that review decisions leave extraction attempts immutable and preserve prior audit rows across later retries.
- [ ] 4.2 Run `cd backend && go test ./...` after implementation and update this checklist with completed items during `sdd-apply`.
