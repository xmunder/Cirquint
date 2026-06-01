# Circuit Review Workbench Specification

## Purpose

Define the backend MVP contract for manually resolving `needs_review` extraction cases.

## Requirements

### Requirement: Project Review Queue Listing

The system MUST list pending review items for a project and SHALL include the job ID, revision ID, image/upload reference, confidence, warnings summary, and queued-for-review timestamp for each `needs_review` item.

#### Scenario: Project has pending review work

- GIVEN a project with one or more jobs in `needs_review`
- WHEN a client requests the review queue for that project
- THEN the system returns only that project's pending review items
- AND each item includes the review summary fields needed to select work

#### Scenario: Project has no pending review work

- GIVEN a project with no jobs in `needs_review`
- WHEN a client requests the review queue for that project
- THEN the system returns an empty collection

### Requirement: Review Detail Retrieval

The system MUST return review detail for a specific reviewable job/revision pair and SHALL expose only reviewable circuit artifacts: job metadata, extraction metadata, normalized `CircuitSpec`, confidence, warnings, and current review state. It MUST NOT require or expose planning or viewer artifacts.

#### Scenario: Reviewer opens a reviewable revision

- GIVEN a job and revision in `needs_review` for the requested project
- WHEN a client requests the review detail
- THEN the system returns the persisted extraction payload and normalized `CircuitSpec`
- AND the response excludes `AssemblyPlan`, `SceneSpec`, and viewer-specific payloads

#### Scenario: Requested revision is not reviewable for that project

- GIVEN a job or revision that is missing, belongs to another project, or is not in `needs_review`
- WHEN a client requests the review detail
- THEN the system rejects the request as not reviewable

### Requirement: Manual Review Decision Submission

The system MUST accept exactly one manual decision command per pending review state with `approve` or `reject`, reviewer identity, note, decision timestamp, and an optional corrected `CircuitSpec`. An approved review SHALL move the job and current revision to `ready`; a rejected review SHALL move them to `failed`.

#### Scenario: Reviewer approves with corrections

- GIVEN a job and revision in `needs_review`
- WHEN the reviewer submits `approve` with audit fields and a corrected `CircuitSpec`
- THEN the system persists the decision and corrected circuit data atomically
- AND the job and current revision transition to `ready`

#### Scenario: Reviewer rejects without corrections

- GIVEN a job and revision in `needs_review`
- WHEN the reviewer submits `reject` with audit fields
- THEN the system persists the decision without requiring corrected circuit data
- AND the job and current revision transition to `failed`

### Requirement: Review Audit Boundary

The system MUST persist review actions as immutable audit records separate from extraction attempts and SHALL preserve reviewer identity, decision, note, timestamp, and reviewed revision reference without reusing extraction retry identity.

#### Scenario: Review history remains auditable

- GIVEN a review decision was submitted for a revision
- WHEN the system stores the review result
- THEN the audit record remains linked to the reviewed revision and reviewer metadata
- AND the original extraction attempt data remains unchanged

#### Scenario: Later extraction retry does not overwrite review history

- GIVEN a reviewed job is later re-extracted through a new attempt
- WHEN the new extraction revision is persisted
- THEN the prior review audit record remains intact for the earlier revision
- AND the new extraction attempt is tracked separately from the manual review action
