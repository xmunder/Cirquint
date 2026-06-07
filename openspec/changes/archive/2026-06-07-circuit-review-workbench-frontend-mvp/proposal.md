# Proposal: Circuit Review Workbench Frontend MVP

## Intent

Ship the smallest usable review UI so an operator can process `needs_review` work through the already-live backend contract from a new Next.js app in `apps/web`.

## Scope

### In Scope
- Bootstrap a minimal Next.js app under `apps/web` with one shared layout.
- Project review queue page for pending review items.
- Review detail page showing job metadata, extraction payload, normalized `CircuitSpec`, confidence, warnings, and review state.
- Manual decision submit flow for `approve` or `reject` with required note and optional JSON `corrected_spec` on approve.
- Basic empty, error, and stale-review (`409`) handling with redirect back to the queue.

### Out of Scope
- Circuit viewer, planning, editor-rich correction UX, uploads, project creation, assignment, notifications, bulk actions, retries, and live updates.
- Real auth/session UX or reviewer attribution beyond the backend's current `anonymous` stub.

## Capabilities

### New Capabilities
- `circuit-review-workbench-web`: Project-scoped queue, review detail, and manual decision screens for the backend review workbench.

### Modified Capabilities
- None.

## Approach

Use the existing project-scoped review endpoints as the source of truth. Render queue and detail on the server for initial load, keep state local to each route, and submit decisions through a plain form action that sends RFC3339 `reviewed_at`, then redirects on success. Keep the shell and component set minimal so the MVP establishes frontend conventions without inventing broader product structure.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `apps/web/*` | New | Next.js scaffold, app layout, review routes, and UI components |
| `openspec/changes/circuit-review-workbench-frontend-mvp/*` | New | Proposal, specs, design, and tasks for this slice |
| `openspec/specs/circuit-review-workbench-web/spec.md` | New | Frontend behavior contract for the MVP |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Scaffold creep beyond MVP | Med | Lock routes to queue/detail/decision only |
| Misleading reviewer UX | Med | Do not imply real identity while backend uses `anonymous` |
| Incorrect audit timestamps | Med | Generate one server-side/client action timestamp in RFC3339 |

## Rollback Plan

Remove `apps/web` and the new frontend spec artifacts; backend review APIs remain untouched and continue to define the review contract.

## Dependencies

- `openspec/specs/circuit-review-workbench/spec.md`
- ADR-001 and ADR-006
- Existing backend review routes in `backend/internal/server/server.go`

## Success Criteria

- [ ] A reviewer can open `/projects/{projectId}/reviews` and see pending items.
- [ ] A reviewer can open one item, inspect the review payload, and submit `approve` or `reject` with the required note.
- [ ] The MVP defers viewer/planning/editor-rich work and does not require auth beyond the current backend stub.
