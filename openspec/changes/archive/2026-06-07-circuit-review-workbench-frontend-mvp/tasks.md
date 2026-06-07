# Tasks: Circuit Review Workbench Frontend MVP

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 700-1100 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 scaffold+API -> PR 2 queue/detail -> PR 3 decisions+e2e |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Resolved by orchestrator
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Bootstrap `apps/web` and typed backend client | PR 1 | Standalone scaffold plus `vitest` |
| 2 | Ship queue/detail read flow | PR 2 | Depends on PR 1; add RTL coverage |
| 3 | Ship decision submit and stale handling | PR 3 | Depends on PR 2; add Playwright flow |

## Phase 1: Foundation

- [x] 1.1 Create `apps/web/package.json`, `tsconfig.json`, and `next.config.ts` with `next`, `react`, `typescript`, `vitest`, RTL, and Playwright scripts.
- [x] 1.2 Create `apps/web/app/layout.tsx` and `app/globals.css` for one minimal review shell with no broader product navigation.
- [x] 1.3 Create `apps/web/lib/review-types.ts` from `backend/internal/circuit/circuit.go` queue/detail/decision DTOs.
- [x] 1.4 Create `apps/web/lib/review-api.ts` with `listProjectReviews`, `getProjectReview`, and `submitReviewDecision`, `cache: "no-store"`, and typed `409` stale mapping.

## Phase 2: Read Flow

- [x] 2.1 Create `apps/web/app/projects/[projectId]/reviews/page.tsx` to render the project queue, row links, and explicit empty state.
- [x] 2.2 Create `apps/web/app/projects/[projectId]/reviews/loading.tsx` for queue loading feedback.
- [x] 2.3 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/page.tsx` to render job metadata, extraction payload, normalized `CircuitSpec`, warnings, and review state.
- [x] 2.4 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/error.tsx` for non-stale detail load failures.

## Phase 3: Decision Flow

- [x] 3.1 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.ts` server action that generates RFC3339 `reviewed_at`, posts the decision, redirects on success, and redirects stale `409` to queue with `?stale=1`.
- [x] 3.2 Create `apps/web/components/review-decision-form.tsx` with approve/reject choice, required note, optional `corrected_spec` JSON only for approve, and inline submit errors.
- [x] 3.3 Create `apps/web/components/submit-button.tsx` using `useFormStatus` for non-idle submit state.
- [x] 3.4 Update queue and detail routes to surface success/stale query banners and bind the form to `review_state.circuit_revision_id`.

## Phase 4: Verification

- [x] 4.1 Add `apps/web/lib/review-api.test.ts` for non-2xx parsing, stale conflict mapping, and corrected JSON parsing.
- [x] 4.2 Add route/component integration tests covering queue populated, queue empty, detail render, required note, and corrected-spec visibility rules. (Queue populated/empty and detail render complete in slice 2; decision-form cases remain for slice 3.)
- [x] 4.3 Add Playwright flow for queue -> detail -> approve/reject redirect, backend error retry, and stale `409` return-to-queue behavior.

## Phase 5: Cleanup

- [x] 5.1 Document the chosen backend base URL env in `apps/web/README.md` or equivalent app-local setup note once implementation confirms the convention.
- [x] 5.2 Re-check tasks against spec scenarios and mark PR boundaries before `sdd-apply` starts.
