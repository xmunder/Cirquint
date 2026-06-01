# Proposal: Circuit Review Workbench MVP

## Intent

Add a backend-first manual review workbench so `needs_review` extractions can be queued, inspected, corrected, and resolved without inventing frontend behavior before the contract is stable.

## Scope

### In Scope
- Project-scoped API to list pending `needs_review` jobs.
- Review detail API exposing job, extraction revision, circuit revision, confidence, warnings, and current review state.
- Review decision API that persists reviewer identity, note, timestamp, decision, and optional corrected `CircuitSpec` payload.
- State transitions that move reviewed work out of `needs_review` and keep an auditable review record.

### Out of Scope
- Full frontend workbench implementation.
- Downstream `AssemblyPlan` or `SceneSpec` regeneration.
- Multi-step review workflows, assignment, or notifications.

## Capabilities

### New Capabilities
- `circuit-review-workbench`: Manual review queue, review detail retrieval, and review decision submission for `needs_review` revisions.

### Modified Capabilities
- `circuit-extraction-review`: Extend review-gated persistence so reviewed revisions can store correction metadata and exit `needs_review` only after a persisted decision.

## Approach

Add thin HTTP contracts in the Go monolith: queue, detail, and submit-review. Persist review audit data in a dedicated review table linked to `circuit_revisions`, and update the current revision/job atomically when a decision is submitted. Accept corrected circuit content in the review command so ADR-006's manual correction rule is satisfied without shipping UI first.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/internal/server/server.go` | Modified | Review queue/detail/decision routes |
| `backend/internal/processing/*` | Modified | List/filter `needs_review` jobs and reviewed transitions |
| `backend/internal/circuit/*` | Modified | Revision lookup, corrected spec persistence, review audit model |
| `backend/migrations/*` | New/Modified | Review table and indexes |
| `openspec/specs/circuit-extraction-review/spec.md` | Modified | Post-review behavior |
| `openspec/specs/circuit-review-workbench/spec.md` | New | Review workbench contract |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Review semantics drift | Med | Freeze MVP to `approve`/`reject` + optional corrected spec |
| Weak audit trail | Low | Use dedicated review table, not only job fields |
| Retry/review confusion | Med | Keep review records separate from extraction attempt identity |

## Rollback Plan

Remove review routes, ignore the new table, and keep the pipeline stopping at `needs_review`; existing extraction persistence remains valid.

## Dependencies

- ADR-006 review gate remains authoritative.
- PostgreSQL migration for review persistence.

## Success Criteria

- [ ] API clients can list pending `needs_review` work by project.
- [ ] A reviewer can fetch one review payload and submit `approve` or `reject` with note and optional corrected spec.
- [ ] Review submission persists audit metadata and transitions the job/revision out of `needs_review` deterministically.
