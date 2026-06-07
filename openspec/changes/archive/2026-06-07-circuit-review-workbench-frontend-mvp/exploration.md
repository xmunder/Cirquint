# Exploration: circuit-review-workbench-frontend-mvp

### Current State
- The repo has no frontend app scaffold yet: there is no `package.json`, no `apps/web`, and no existing route/layout/component tree to extend.
- Architecture docs and ADR-001 define the intended frontend stack as Next.js, with the frontend consuming domain results rather than inventing circuit structure.
- The backend review workbench contract is already live in `backend/internal/server/server.go` with three project-scoped endpoints: queue list, review detail, and decision submission.
- The review detail payload is intentionally narrow: job metadata, extraction payload, normalized `CircuitSpec`, confidence, warnings, and review state only. Planning/viewer artifacts are explicitly excluded by spec and tests.
- Identity is still a stub: `identity.Middleware` injects `reviewer_id = anonymous`, so the frontend cannot rely on real auth/session behavior yet.

### Affected Areas
- `openspec/changes/archive/2026-06-01-circuit-review-workbench-mvp/*` — backend artifacts that define the review contract the frontend must consume.
- `openspec/specs/circuit-review-workbench/spec.md` — canonical queue/detail/decision requirements and response boundaries.
- `backend/internal/server/server.go` — actual HTTP routes, request bodies, and error mapping.
- `backend/internal/circuit/circuit.go` — DTOs for queue, detail, and decision result, plus review transition semantics.
- `docs/adr/ADR-001-monolito-modular-y-pipeline-canonico.md` — Next.js is the intended frontend platform.
- `docs/adr/ADR-006-revision-manual-ante-baja-confianza.md` — manual review is mandatory before downstream planning/viewer work.

### Approaches
1. **Thin Next.js review workbench** — create the smallest frontend slice around the existing backend contract.
   - Pros: matches ADR direction; unlocks real manual review; keeps scope focused on queue -> inspect -> decide.
   - Cons: requires creating base frontend structure from zero; route/layout conventions must be established as part of MVP.
   - Effort: Medium

2. **Broader operator console** — include uploads, job polling, and richer circuit editing in the same frontend slice.
   - Pros: more demoable end to end.
   - Cons: expands beyond the review contract, mixes unrelated flows, and risks inventing UI before auth/editor contracts exist.
   - Effort: High

### Recommendation
Build a thin Next.js MVP focused on one usable operator flow:

1. Open a project-scoped review queue page.
2. Select one `needs_review` item.
3. Open a review detail page showing job ID, upload ID, confidence, warnings, extraction payload, and normalized `CircuitSpec`.
4. Submit either `approve` or `reject` with a required note.
5. Allow optional JSON-based `corrected_spec` only for `approve`, because the backend already supports it and ADR-006 requires manual correction persistence before downstream regeneration.

Recommended route/layout baseline for the new frontend:
- `apps/web/app/projects/[projectId]/reviews/page.tsx` — queue list.
- `apps/web/app/projects/[projectId]/reviews/[jobId]/page.tsx` — review detail + decision form.
- `apps/web/app/layout.tsx` — minimal shell only; no global dashboard work in this slice.

Recommended data strategy for MVP:
- Server-render queue and detail pages from the backend endpoints for the initial load.
- Use a plain client form submit for the decision action, then redirect back to the queue on success.
- Avoid introducing global state management; per-route fetch state is enough for this slice.

Recommended in-scope vs out-of-scope split:
- In scope: queue list, detail view, approve/reject form, optimistic-free success/error handling, empty queue state, stale revision error handling.
- Out of scope: visual circuit editor, viewer/3D work, uploads/project creation flows, assignment, notifications, live collaboration, bulk actions, retries/regeneration UI.

### Risks
- There is no existing frontend scaffold, so even the MVP must establish app structure, API client conventions, and deployment/runtime assumptions.
- Auth is not real yet; every decision will currently persist `reviewer_id = anonymous`, so any reviewer attribution UI would be misleading.
- `corrected_spec` shape is extremely minimal today (`confidence`, `warnings`, `status`), which means a JSON editor is possible but a rich form would be speculative.
- The backend requires `reviewed_at` from the client; if the frontend generates local timestamps inconsistently, audit records become noisy.
- Stale review conflicts (`409`) are part of the normal concurrency model and need an explicit UX path back to the queue.

### Ready for Proposal
Yes — the next step should propose a new Next.js frontend slice that creates the minimal app scaffold and implements only the project review queue, review detail screen, and manual decision form against the existing backend endpoints.
