# Exploration: circuit-review-workbench-mvp

### Current State
- Jobs already model the review gate in `backend/internal/processing/processing.go`: `created -> uploaded -> queued -> extracting -> needs_review | ready | failed`.
- Revisions already persist the reviewable payload in `backend/internal/circuit/circuit.go`: `ExtractionRevision` + `CircuitRevision`, with `confidence`, `warnings`, `status`, provider metadata, timestamps, and deterministic object keys.
- The worker already writes `needs_review` when `circuit.DecideStatus(...)` says so, and the job status mirrors that outcome.
- Exposed APIs are still narrow: `POST /workspaces`, `POST /workspaces/{workspaceID}/projects`, `GET /projects/{projectID}`, `POST /projects/{projectID}/uploads`, and `GET /jobs/{jobID}`.
- There is no API to list `needs_review` work, no API to fetch a revision by project/job for review, and no API to submit a review decision.
- Persistence also lacks review metadata: no reviewer, decision, note, or review timestamp fields exist in `processing_jobs` or `circuit_revisions`.

### Affected Areas
- `backend/internal/processing/*` — job listing/filtering and review-state transitions.
- `backend/internal/circuit/*` — revision lookup and review metadata persistence.
- `backend/internal/server/server.go` — new review queue/detail/decision endpoints.
- `backend/migrations/0001_initial_schema.sql` — schema gap for review decisions/notes/timestamps.
- `openspec/specs/circuit-extraction-review/spec.md` — current spec stops at `needs_review`; review workbench adds a new boundary.
- `docs/adr/ADR-006-revision-manual-ante-baja-confianza.md` — establishes the review gate, but not the workbench contract.

### Approaches
1. **Backend-first review workbench API** — add list/detail/decision endpoints and persistence, defer UI.
   - Pros: fits current repo shape; keeps frontend from guessing contracts; smallest safe slice.
   - Cons: no visible UI yet; reviewer flow must be exercised via API/tests first.
   - Effort: Medium

2. **Backend + minimal UI contract together** — define and implement the frontend workbench immediately.
   - Pros: end-to-end demoability; forces response shapes early.
   - Cons: bigger slice; higher risk of coupling UI to unstable review semantics; more review surface.
   - Effort: High

### Recommendation
Go backend-first, but lock the minimal API contract now. For MVP, the workbench should be able to: list `needs_review` items, inspect the revision payload, submit one review decision with notes, and transition the job/revision out of `needs_review`. Frontend stays deferred; the API shape becomes the contract.

### Risks
- We currently have no query path for “all needs_review work”, so list endpoints and repository methods are mandatory.
- Review semantics are still ambiguous: approval vs rejection vs “needs changes” must be settled before implementation.
- If review metadata is stored only on the job row, audit/history will be weak; if stored in a separate table, the slice gets a bit larger but safer.
- Existing revision idempotency is by `job_id`; review edits must not accidentally look like re-extraction retries.

### Ready for Proposal
Yes — next step should be a proposal for the backend review workbench slice, with explicit API and persistence contracts and frontend deferred.
