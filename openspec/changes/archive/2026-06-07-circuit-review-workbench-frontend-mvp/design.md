# Design: Circuit Review Workbench Frontend MVP

## Technical Approach

Create a new `Next.js` app in `apps/web` using App Router. Queue and detail pages render on the server from the existing Go review endpoints; only the decision form uses a small client component for submit-state UX. The frontend stays thin: no global store, no viewer/planning/editor features, and no auth assumptions beyond the backend's current `anonymous` reviewer.

## Architecture Decisions

| Decision | Choice | Alternatives considered | Rationale |
|---|---|---|---|
| App shape | `apps/web` standalone App Router app | Broader monorepo shell or dashboard | Repo has no frontend scaffold, so the smallest valid product shell is a single app with one layout. |
| Rendering | Server components for pages; client component only for form UX | Fully client-rendered pages | Queue/detail are backend-driven reads and do not need client state hydration. |
| Mutation path | Next server action posts decision to backend | Client-side fetch mutation | Server action keeps backend URL/private env server-only and lets `reviewed_at` be generated once on the server. |
| Conflict handling | Treat `409` as stale workflow, redirect to queue with banner | Keep user on detail and retry same revision | Backend marks the revision as no longer reviewable; retrying the same payload is wrong. |

## Data Flow

`/projects/[projectId]/reviews` page -> server fetch `GET /projects/{projectId}/circuit-reviews` -> render table or empty/error state.

`/projects/[projectId]/reviews/[jobId]` page -> server fetch `GET /projects/{projectId}/circuit-reviews/{jobId}` -> render metadata, extraction payload, normalized `CircuitSpec`, warnings, and form.

Form submit -> Next server action -> generate `reviewed_at = new Date().toISOString()` on the server -> `POST /projects/{projectId}/circuit-reviews/{jobId}/decision` -> on success `redirect()` to queue -> on `409` redirect to queue with `?stale=1` -> on other errors return form error state on same page.

## File Changes

| File | Action | Description |
|---|---|---|
| `apps/web/package.json` | Create | Next.js app manifest and scripts. |
| `apps/web/tsconfig.json` | Create | TypeScript config for the app. |
| `apps/web/next.config.ts` | Create | Minimal Next config. |
| `apps/web/app/layout.tsx` | Create | Shared shell for review routes only. |
| `apps/web/app/globals.css` | Create | Minimal base styles. |
| `apps/web/app/projects/[projectId]/reviews/page.tsx` | Create | Server-rendered queue page. |
| `apps/web/app/projects/[projectId]/reviews/loading.tsx` | Create | Queue loading skeleton. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/page.tsx` | Create | Server-rendered detail page. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/error.tsx` | Create | Detail load failure fallback. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.ts` | Create | Server action for approve/reject submit. |
| `apps/web/components/review-decision-form.tsx` | Create | Client form with note, decision toggle, optional corrected JSON textarea, and submit state. |
| `apps/web/components/submit-button.tsx` | Create | `useFormStatus` loading button. |
| `apps/web/lib/review-api.ts` | Create | Typed backend client and error mapping. |
| `apps/web/lib/review-types.ts` | Create | TS types mirroring backend review DTOs. |

## Interfaces / Contracts

```ts
type ReviewDecision = "approve" | "reject";

type SubmitDecisionPayload = {
  circuit_revision_id: string;
  decision: ReviewDecision;
  note: string;
  reviewed_at: string; // RFC3339 from server action
  corrected_spec?: { confidence: number; warnings: string[]; status?: string };
};
```

`review-api.ts` should expose `listProjectReviews`, `getProjectReview`, and `submitReviewDecision`. All requests use `cache: "no-store"`. Non-2xx responses parse `{ error }`; `409` becomes a typed stale-conflict error.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | API client error mapping and corrected JSON parsing | `vitest` on `lib/*` helpers. |
| Integration | Queue/detail page render states and form validation UX | React Testing Library with mocked `review-api` / server action boundaries. |
| E2E | Queue -> detail -> approve/reject redirect, empty queue, `409 stale` banner | Playwright against the scaffold once the app exists. |

## Migration / Rollout

No migration required. Rollout is additive: new `apps/web` app only.

## Open Questions

- [ ] Confirm the frontend backend base URL convention (`BACKEND_BASE_URL` or equivalent) when implementation starts.
- [ ] Confirm whether `corrected_spec.status` should be omitted by the UI and left for the backend to normalize on approve.
