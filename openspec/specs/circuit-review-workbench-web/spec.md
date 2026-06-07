# circuit-review-workbench-web Specification

## Purpose

Define the MVP frontend in `apps/web` for reviewing `needs_review` jobs against the existing backend contract.

## Requirements

### Requirement: Minimal Review Workbench Scaffold

The system MUST provide a minimal Next.js app in `apps/web` with one shared layout and only the routes required for queue and detail review flow. The MVP MUST treat the backend review API as the source of truth and MUST NOT add uploads, project management, auth UX, viewer, planning, or rich correction editors.

#### Scenario: Reviewer enters the workbench

- GIVEN the frontend app is deployed
- WHEN a reviewer opens a review route
- THEN the app renders within one shared shell from `apps/web`
- AND only queue and review detail routes are required

### Requirement: Project Review Queue Page

The system MUST provide a project queue page at `/projects/{projectId}/reviews` that loads pending items from the backend queue contract and SHALL show job ID, upload ID, reviewable revision ID, confidence, warnings, and queued-for-review timestamp.

#### Scenario: Queue contains pending review items

- GIVEN the backend returns one or more queue items for the project
- WHEN the reviewer opens the queue page
- THEN the page lists only that project's pending items
- AND each item exposes navigation to its review detail page

#### Scenario: Queue is empty

- GIVEN the backend returns an empty queue for the project
- WHEN the reviewer opens the queue page
- THEN the page shows an explicit empty-state message
- AND the page does not imply hidden or failed-to-load work

### Requirement: Review Detail Page

The system MUST provide a review detail page at `/projects/{projectId}/reviews/{jobId}` that renders the backend review detail payload. The page SHALL show job metadata, extraction payload, normalized `CircuitSpec`, confidence, warnings, and current review state, and MUST NOT show planning or viewer artifacts.

#### Scenario: Reviewer opens a reviewable job

- GIVEN the backend returns review detail for the requested project and job
- WHEN the reviewer opens the detail page
- THEN the page renders the reviewable payload and current state for that job
- AND the decision form is bound to the returned `circuit_revision_id`

### Requirement: Manual Review Decision Submission

The system MUST allow `approve` or `reject` from the detail page through the backend decision contract. A note MUST be required for both decisions, `corrected_spec` MAY be submitted only for `approve`, and the UI MUST expose loading, success, and error states.

#### Scenario: Reviewer submits an approval successfully

- GIVEN the reviewer is on a pending review detail page
- WHEN the reviewer submits `approve` with a note and optional `corrected_spec`
- THEN the UI shows a non-idle submitting state until the backend responds
- AND on success redirects the reviewer back to the project queue

#### Scenario: Reviewer submits a rejection successfully

- GIVEN the reviewer is on a pending review detail page
- WHEN the reviewer submits `reject` with a note
- THEN the UI sends the reviewed revision ID and an RFC3339 `reviewed_at`
- AND on success redirects the reviewer back to the project queue

#### Scenario: Backend rejects the submission

- GIVEN the reviewer submits a decision for a pending revision
- WHEN the backend returns a non-conflict error
- THEN the UI shows a visible error state on the detail page
- AND the reviewer can retry from the same route

### Requirement: Stale Review Conflict Handling

The system MUST treat backend `409 Conflict` responses as stale-or-resolved review outcomes, SHALL show a stale revision message, and SHALL return the reviewer to the queue instead of presenting success.

#### Scenario: Revision becomes stale before submit completes

- GIVEN another actor or process resolves the same review before submission completes
- WHEN the backend returns `409 Conflict`
- THEN the UI explains that the displayed revision is no longer current
- AND the reviewer is redirected or guided back to the project queue
