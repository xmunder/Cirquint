# Delta for Circuit Extraction Review

## MODIFIED Requirements

### Requirement: Reviewable Extraction Persistence

The system MUST persist the ExtractionResult and a normalized CircuitSpec revision for each completed extraction attempt. For review-gated results, the system SHALL keep the job and current revision in `needs_review` until a manual decision is persisted, SHALL store review audit metadata separately from extraction-attempt identity, and SHALL transition the job and current revision to `ready` on approved review or `failed` on rejected review.
(Previously: persistence ended once a low-confidence extraction was stored and marked `needs_review`.)

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
