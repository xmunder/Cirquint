## Exploration: circuit-ingestion-pipeline-mvp

### Current State
Repo is docs-only today: ADRs 001-006 and two architecture summaries define the target shape, but there is no executable scaffold yet. The canonical flow is fixed as `image -> ExtractionResult -> CircuitSpec -> AssemblyPlan -> SceneSpec`, with async Go workers, R2 storage, AI provider adapters, and manual review on low confidence.

### Affected Areas
- `docs/architecture/arquitectura-decisiones.md` — canonical context and MVP slicing guidance.
- `docs/architecture/decisiones-de-arquitectura.md` — explicit MVP starters, state machine, and scaffold sketch.
- `docs/adr/ADR-001-*.md` through `ADR-006-*.md` — non-negotiable constraints for monolith, artifact separation, R2, async pipeline, AI boundary, and review flow.
- `apps/web/*` — future Next.js UI for project creation, upload, job status, and review UX.
- `backend/cmd/api/*`, `backend/cmd/worker/*`, `backend/internal/{uploads,processing,provider,storage,workspace,circuit}/*` — future Go API, worker, and bounded-context implementation.

### Approaches
1. **Backend-first vertical slice** — build the smallest end-to-end pipeline first: project create -> upload -> enqueue -> extract -> persist revision -> maybe review-needed.
   - Pros: validates the hard parts early; keeps contracts and state machine honest; matches ADRs tightly.
   - Cons: UI stays thin at first; requires disciplined API design before polished UX.
   - Effort: Medium

2. **UI-first mocked flow** — ship the Next.js experience with mocked/temporary pipeline responses, then fill backend later.
   - Pros: fast visible progress; easier to demo screens early.
   - Cons: risks inventing UI around unstable contracts; can drift from the real async/revision model.
   - Effort: Medium

### Recommendation
Choose the backend-first vertical slice. The real risk is not rendering screens; it is modeling the pipeline boundary correctly: persistent uploads, versioned extraction artifacts, provider abstraction, and the `needs_review` branch. Once those contracts exist, the frontend can be thin but real.

### Risks
- Review-needed UX may be underspecified until confidence thresholds and warning rules are concretized.
- Revision versioning must be precise or reprocessing will become inconsistent.
- With no scaffold yet, the first implementation phase must establish package/module boundaries carefully or the monolith will become muddy fast.

### Ready for Proposal
Yes — the next step should be a proposal for the first vertical slice covering project creation, schematic upload, async kickoff, AI extraction boundary, revision persistence, and low-confidence review handling.
