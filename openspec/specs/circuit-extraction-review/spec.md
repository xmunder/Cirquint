# Circuit Extraction Review Specification

## Purpose

Define the provider boundary, extraction persistence, review gating, and MVP output limits.

## Requirements

### Requirement: Provider Invocation Boundary

The system MUST invoke extraction through a provider adapter boundary and SHALL persist job progress independently of any concrete provider.

#### Scenario: Worker enters extraction through the adapter

- GIVEN a queued processing job
- WHEN asynchronous processing starts
- THEN the system transitions the job to `extracting`
- AND invokes circuit extraction only through the provider adapter contract

#### Scenario: Provider failure is isolated to job state

- GIVEN a processing job in `extracting`
- WHEN the provider returns an unrecoverable error
- THEN the system marks the job as `failed`
- AND no `ready` revision is published

### Requirement: Reviewable Extraction Persistence

The system MUST persist the ExtractionResult and a normalized CircuitSpec revision for each completed extraction attempt. For review-gated results, the system SHALL keep the job and current revision in `needs_review` until a manual decision is persisted, SHALL store review audit metadata separately from extraction-attempt identity, and SHALL transition the job and current revision to `ready` on approved review or `failed` on rejected review.

#### Scenario: High-confidence extraction becomes ready

- GIVEN an extraction result with acceptable confidence and no review-blocking warnings
- WHEN persistence succeeds
- THEN the system stores the ExtractionResult and a normalized CircuitSpec revision
- AND the job transitions to `ready`

#### Scenario: Low-confidence extraction requires review

- GIVEN an extraction result with confidence below threshold or relevant warnings
- WHEN persistence succeeds
- THEN the system stores the ExtractionResult and CircuitSpec revision
- AND the job transitions to `needs_review`

#### Scenario: Manual review resolves a pending revision

- GIVEN a persisted job and current revision in `needs_review`
- WHEN a manual review decision is persisted for that revision
- THEN the review metadata is stored separately from the extraction attempt record
- AND the job and current revision leave `needs_review` according to the persisted decision

### Requirement: MVP Output Scope Guard

The system MUST persist only ExtractionResult and CircuitSpec review artifacts, and MUST NOT generate AssemblyPlan, SceneSpec, or 3D viewer payloads.

#### Scenario: Downstream planning artifacts remain out of scope

- GIVEN a completed extraction job
- WHEN the MVP pipeline finishes
- THEN no AssemblyPlan, SceneSpec, or viewer-specific artifact is created
- AND the result remains limited to reviewable circuit data
