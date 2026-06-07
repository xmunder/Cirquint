## Implementation Progress

**Change**: `circuit-review-workbench-frontend-mvp`
**Mode**: Strict TDD

### Completed Tasks
- [x] 1.1 Create `apps/web/package.json`, `tsconfig.json`, and `next.config.ts` with `next`, `react`, `typescript`, `vitest`, RTL, and Playwright scripts.
- [x] 1.2 Create `apps/web/app/layout.tsx` and `app/globals.css` for one minimal review shell with no broader product navigation.
- [x] 1.3 Create `apps/web/lib/review-types.ts` from `backend/internal/circuit/circuit.go` queue/detail/decision DTOs.
- [x] 1.4 Create `apps/web/lib/review-api.ts` with `listProjectReviews`, `getProjectReview`, and `submitReviewDecision`, `cache: "no-store"`, and typed `409` stale mapping.
- [x] 2.1 Create `apps/web/app/projects/[projectId]/reviews/page.tsx` to render the project queue, row links, and explicit empty state.
- [x] 2.2 Create `apps/web/app/projects/[projectId]/reviews/loading.tsx` for queue loading feedback.
- [x] 2.3 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/page.tsx` to render job metadata, extraction payload, normalized `CircuitSpec`, warnings, and review state.
- [x] 2.4 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/error.tsx` for non-stale detail load failures.
- [x] 3.1 Create `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.ts` server action that generates RFC3339 `reviewed_at`, posts the decision, redirects on success, and redirects stale `409` to queue with `?stale=1`.
- [x] 3.2 Create `apps/web/components/review-decision-form.tsx` with approve/reject choice, required note, optional `corrected_spec` JSON only for approve, and inline submit errors.
- [x] 3.3 Create `apps/web/components/submit-button.tsx` using `useFormStatus` for non-idle submit state.
- [x] 3.4 Update queue and detail routes to surface success/stale query banners and bind the form to `review_state.circuit_revision_id`.
- [x] 4.1 Add `apps/web/lib/review-api.test.ts` for non-2xx parsing, stale conflict mapping, and corrected JSON parsing.
- [x] 4.2 Add route/component integration tests covering queue populated, queue empty, detail render, required note, and corrected-spec visibility rules. (Queue populated/empty and detail render complete in slice 2; decision-form cases remain for slice 3.)
- [x] 4.3 Add Playwright flow for queue -> detail -> approve/reject redirect, backend error retry, and stale `409` return-to-queue behavior.
- [x] 5.1 Document the chosen backend base URL env in `apps/web/README.md` or equivalent app-local setup note once implementation confirms the convention.
- [x] 5.2 Re-check tasks against spec scenarios and mark PR boundaries before `sdd-apply` starts.

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `apps/web/.gitignore` | Created | Scoped generated frontend artifacts out of git so salvage cleanup can remove local installs and transient typecheck output safely. |
| `apps/web/package.json` | Created | Added the standalone Next.js app manifest plus `vitest`, RTL, and Playwright scripts/dependencies. |
| `apps/web/package-lock.json` | Created | Recorded the resolved dependency tree for the new frontend scaffold. |
| `apps/web/tsconfig.json` | Created | Added strict TypeScript config, Next plugin wiring, and Vitest globals for app/test typing. |
| `apps/web/next.config.ts` | Created | Added the minimal Next config for the app router scaffold. |
| `apps/web/next-env.d.ts` | Created | Added the standard Next TypeScript ambient references. |
| `apps/web/vitest.config.ts` | Created | Added the JS DOM test runner config, `@/*` alias resolution, scoped include globs for app tests, and an explicit `e2e/**/*.spec.ts` exclusion so `pnpm test` only runs the intended Vitest suite. |
| `apps/web/test/setup.ts` | Created | Registered Testing Library DOM matchers for Vitest. |
| `apps/web/app/layout.tsx` | Created | Added the shared review shell and root layout without broader product navigation. |
| `apps/web/app/globals.css` | Created | Added minimal global resets and base theme styles. |
| `apps/web/app/layout.test.tsx` | Created | Added a shell rendering test for the shared layout wrapper. |
| `apps/web/app/projects/[projectId]/reviews/page.tsx` | Created | Added the server-rendered queue page with row links, warning formatting, and an explicit empty state. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.ts` | Created | Added the server action that stamps `reviewed_at`, submits decisions, redirects on success, and sends stale `409` cases back to the queue banner. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.test.ts` | Created | Added strict-TDD coverage for success redirect, stale redirect, and inline action error returns. |
| `apps/web/app/projects/[projectId]/reviews/loading.tsx` | Created | Added the queue loading state with accessible busy/status semantics. |
| `apps/web/app/projects/[projectId]/reviews/loading.test.tsx` | Created | Added integration coverage for the queue loading state. |
| `apps/web/app/projects/[projectId]/reviews/page.test.tsx` | Created | Added integration coverage for populated and empty queue rendering. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/page.tsx` | Created | Added the server-rendered detail page with job metadata, extraction payload, review state, and normalized circuit output. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/error.tsx` | Created | Added the client error fallback for non-stale detail load failures. |
| `apps/web/app/projects/[projectId]/reviews/[jobId]/page.test.tsx` | Created | Added integration coverage for detail rendering and the retryable error fallback. |
| `apps/web/components/review-decision-form.tsx` | Created | Added the client decision form with approve/reject selection, required note, optional corrected JSON, and inline error rendering. |
| `apps/web/components/review-decision-form.test.tsx` | Created | Added integration coverage for note requirements, hidden revision binding, and corrected-spec visibility rules. |
| `apps/web/components/submit-button.tsx` | Created | Added the `useFormStatus` submit button for non-idle loading feedback. |
| `apps/web/components/submit-button.test.tsx` | Created | Added direct unit coverage for the non-idle submit UX (`Submitting...` plus disabled button while pending). |
| `apps/web/lib/review-types.ts` | Created | Mirrored Go review queue/detail/decision DTOs as TypeScript contracts. |
| `apps/web/lib/review-api.ts` | Created | Added typed backend fetch helpers, `BACKEND_BASE_URL` resolution, corrected-spec parsing, and typed stale-conflict mapping. |
| `apps/web/lib/review-api.test.ts` | Created | Added unit coverage for queue fetches, `409` stale mapping, and corrected-spec parsing failures/successes. |
| `apps/web/.env.example` | Created | Captured the chosen backend base URL convention for local setup. |
| `apps/web/README.md` | Created | Documented the backend env contract and local verification commands. |
| `apps/web/pnpm-lock.yaml` | Created | Recorded the pnpm dependency tree required by the frontend slice. |
| `apps/web/playwright.config.ts` | Created | Added the Playwright harness with paired mock-backend and Next dev web servers for the final review-flow slice. |
| `apps/web/e2e/mock-review-backend.mjs` | Created | Added a controllable HTTP mock backend so browser flows can drive queue, detail, retry, and stale-conflict scenarios through the real server-rendered app. |
| `apps/web/e2e/review-workbench.spec.ts` | Created | Added end-to-end coverage for approve redirect, reject redirect, backend retry on the same route, and stale `409` return-to-queue behavior. |
| `openspec/changes/circuit-review-workbench-frontend-mvp/tasks.md` | Modified | Marked slice-1 and slice-2 checklist progress while keeping the chained delivery notes intact. |
| `openspec/changes/circuit-review-workbench-frontend-mvp/apply-progress.md` | Modified | Merged the final Playwright/bookkeeping slice into the cumulative progress record without dropping earlier slice history. |

### TDD Cycle Evidence
| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `apps/web/lib/review-api.test.ts` | Unit | N/A (new) | ✅ Wrote the client contract test before `review-api.ts` existed; package scaffold was added so the runner could execute | ✅ `npm test` | ✅ Covered list fetch, `409` conflict mapping, and corrected-spec cases to force real request/error logic | ✅ Extracted shared request/error helpers |
| 1.2 | `apps/web/app/layout.test.tsx` | Integration | N/A (new) | ✅ Wrote the shell render test before `app/layout.tsx` existed | ✅ `npm test` | ➖ Single shell-render scenario | ✅ Extracted `ReviewShell` so the shared UI is testable without DOM nesting hacks |
| 1.3 | `apps/web/lib/review-api.test.ts`, `npx tsc --noEmit` | Unit + Typecheck | N/A (new) | ✅ Tests/types referenced DTO contracts before `review-types.ts` existed | ✅ `npm test` and `npx tsc --noEmit` | ✅ Used both queue-item and decision payload shapes plus optional corrected-spec status | ➖ DTO mirror kept flat; no extra refactor needed |
| 1.4 | `apps/web/lib/review-api.test.ts` | Unit | N/A (new) | ✅ Wrote failing imports/assertions before `review-api.ts` existed | ✅ `npm test` | ✅ Forced GET and POST paths, non-empty payloads, and `409` error mapping | ✅ Kept the API layer thin and server-oriented |
| 2.1 | `apps/web/app/projects/[projectId]/reviews/page.test.tsx` | Integration | N/A (new) | ✅ Wrote queue tests before the route existed | ✅ `pnpm test "app/projects/[projectId]/reviews/page.test.tsx"` | ✅ Covered both populated and empty queue scenarios from the spec | ✅ Kept the route server-rendered with tiny formatting helpers only |
| 2.2 | `apps/web/app/projects/[projectId]/reviews/loading.test.tsx` | Integration | N/A (new) | ✅ Wrote the loading-state test before the component existed | ✅ `pnpm test "app/projects/[projectId]/reviews/loading.test.tsx"` | ➖ Structural loading copy plus accessible busy/status semantics | ✅ Added semantic status markup so the state is testable and accessible |
| 2.3 | `apps/web/app/projects/[projectId]/reviews/[jobId]/page.test.tsx` | Integration | N/A (new) | ✅ Wrote detail-route assertions before the page existed | ✅ `pnpm test "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"` | ✅ Covered metadata, extraction payload, warnings, review state, and normalized `CircuitSpec` output | ✅ Reused small detail-card/list helpers to keep the route readable |
| 2.4 | `apps/web/app/projects/[projectId]/reviews/[jobId]/page.test.tsx` | Integration | N/A (new) | ✅ Wrote the error fallback assertion before the component existed | ✅ `pnpm test "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"` | ➖ Single non-stale failure path required for this slice | ➖ None needed |
| 3.1 | `apps/web/app/projects/[projectId]/reviews/[jobId]/actions.test.ts` | Unit | N/A (new) | ✅ Wrote action redirect/error assertions before `actions.ts` existed | ✅ `pnpm test "app/projects/[projectId]/reviews/[jobId]/actions.test.ts"` | ✅ Covered approve success, stale `409`, corrected JSON failure, and backend retryable error paths | ✅ Kept the action thin by extracting small form-data readers and stale detection |
| 3.2 | `apps/web/components/review-decision-form.test.tsx` | Integration | N/A (new) | ✅ Wrote form assertions before the component existed | ✅ `pnpm test "components/review-decision-form.test.tsx"` | ✅ Covered approve/reject visibility changes plus inline error rendering and required note binding | ✅ Used `useActionState` with minimal local selection state only |
| 3.3 | `apps/web/components/submit-button.test.tsx`, `apps/web/components/review-decision-form.test.tsx` | Unit + Integration | N/A (new) | ✅ Added a direct pending-state test for `SubmitButton` to prove the non-idle UX required by the spec | ✅ `pnpm test "components/submit-button.test.tsx"` and `pnpm test` | ✅ Covers both pending (`Submitting...` + disabled) and idle button states without coupling to internals beyond the public rendered output | ➖ Button stayed intentionally tiny; no refactor needed |
| 3.4 | `apps/web/app/projects/[projectId]/reviews/page.test.tsx`, `apps/web/app/projects/[projectId]/reviews/[jobId]/page.test.tsx` | Integration | ✅ 4/4 passing baseline on touched route tests | ✅ Wrote banner/form-binding assertions before wiring the routes | ✅ `pnpm test "app/projects/[projectId]/reviews/page.test.tsx" "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"` | ✅ Covered success banner, stale banner, decision card render, hidden revision binding, and required note presence | ✅ Kept route updates server-rendered; the only client boundary remains the form |
| 4.1 | `apps/web/lib/review-api.test.ts` | Unit | N/A (new) | ✅ Written | ✅ `npm test` | ✅ Added success, stale conflict, valid JSON, and invalid JSON scenarios | ➖ None needed |
| 4.2 | `apps/web/components/review-decision-form.test.tsx`, `apps/web/app/projects/[projectId]/reviews/[jobId]/page.test.tsx`, `apps/web/app/projects/[projectId]/reviews/page.test.tsx` | Integration | ✅ 4/4 passing baseline on touched route tests | ✅ Added the missing decision-form coverage after slice-2 route tests already existed | ✅ `pnpm test "components/review-decision-form.test.tsx" "app/projects/[projectId]/reviews/page.test.tsx" "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"` | ✅ Triangulated queue banners, detail binding, required note, and corrected-spec visibility rules | ➖ Existing slice-2 tests already covered populated/empty queue and detail payload, so this pass filled only the remaining gaps |
| 4.3 | `apps/web/e2e/review-workbench.spec.ts` | E2E | ✅ 8/8 passing baseline on queue/action/form safety-net tests | ✅ Wrote the browser-flow spec before the Playwright config and backend harness existed | ✅ `pnpm test:e2e -- e2e/review-workbench.spec.ts` | ✅ Covered queue -> detail -> approve redirect, reject redirect, retry-after-error on the same route, and stale `409` return-to-queue behavior | ✅ Extracted scenario/state control into the mock backend so each browser case stays isolated while driving the real app |
| 5.1 | `apps/web/README.md`, `apps/web/.env.example` | Documentation | N/A (new) | ✅ Documented the env contract once the helper existed | ✅ Verified against `getBackendBaseUrl` and backend route paths | ➖ Single env convention | ➖ None needed |
| 5.2 | `openspec/changes/circuit-review-workbench-frontend-mvp/tasks.md`, `openspec/changes/circuit-review-workbench-frontend-mvp/apply-progress.md` | Artifact audit | N/A (artifact update) | ✅ Re-read the spec/task deltas before closing the final slice | ✅ Cross-checked the queue, detail, decision, retry, and stale scenarios against the cumulative Vitest + Playwright coverage before marking the checklist done | ✅ Compared spec scenarios, task rows, and final-slice PR boundary so no acceptance case stayed untracked | ✅ Merged the final slice into the cumulative progress record instead of overwriting earlier slices |

### Test Summary
- **Total tests written**: 22
- **Total tests passing**: 22
- **Layers used**: Unit (9), Integration (9), E2E (4), Typecheck (1 command)
- **Approval tests**: None - all files in this slice were new.
- **Pure functions created**: 10 (`getBackendBaseUrl`, `parseCorrectedSpecInput`, error mapping helpers, queue/detail formatting helpers, route banner mapping, and action form-data readers)
- **Verification commands run**:
- `cd apps/web && npm test`
- `cd apps/web && npx tsc --noEmit`
- `cd apps/web && pnpm test "app/projects/[projectId]/reviews/page.test.tsx"`
- `cd apps/web && pnpm test "app/projects/[projectId]/reviews/loading.test.tsx"`
- `cd apps/web && pnpm test "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"`
- `cd apps/web && pnpm exec tsc --noEmit`
- `cd apps/web && pnpm test "app/projects/[projectId]/reviews/[jobId]/actions.test.ts" "components/review-decision-form.test.tsx" "app/projects/[projectId]/reviews/page.test.tsx" "app/projects/[projectId]/reviews/[jobId]/page.test.tsx"`
- `cd apps/web && pnpm test "components/submit-button.test.tsx"`
- `cd apps/web && pnpm test`
- `cd apps/web && pnpm test:e2e -- e2e/review-workbench.spec.ts`
- `cd apps/web && pnpm exec tsc --noEmit`

### Deviations from Design
None - implementation matches the MVP design, including the final browser-flow verification slice.

### Issues Found
- TypeScript 6 now errors on `baseUrl` without an explicit deprecation acknowledgement, so the scaffold needed `"ignoreDeprecations": "6.0"` to keep the standard Next path-alias config typecheck-clean.
- `pnpm` 11 aborted the first install because `sharp` build scripts were blocked by the package manager's `ignored build scripts` policy; `pnpm install --ignore-scripts` completed successfully and was sufficient for this read-only slice's tests/typecheck.
- Next injects a route announcer with `role="alert"`, so Playwright assertions for retry failures need to target the visible form error copy instead of a bare role selector.
- After a backend submit error, the route stays retryable but the required note field must be filled again before the browser will emit a second submit; the final E2E flow now covers that real retry behavior explicitly.
- Vitest 4 needed both a scoped `include` and an `e2e/**/*.spec.ts` exclusion in `apps/web/vitest.config.ts`; excluding Playwright alone still let `pnpm test` discover dependency tests under `node_modules`.

### Salvage Verification
- Re-ran the minimal slice-1 safety net after the timeout left partial work behind: `npm test -- app/layout.test.tsx`, `npm test -- lib/review-api.test.ts`, and `npx tsc --noEmit` all passed from `apps/web`.
- Added `apps/web/.gitignore` and removed local `node_modules/` plus `tsconfig.tsbuildinfo` so generated artifacts do not remain in the repo worktree for this slice.

### Remaining Tasks
- [x] None.

### Workload / PR Boundary
- Mode: chained PR slice
- Current work unit: final slice - Playwright harness and final checklist closure
- Boundary: Playwright config, controllable mock backend, browser-flow coverage, and cumulative spec/task bookkeeping only
- Estimated review budget impact: bounded, one test harness plus four end-to-end journeys and no product-surface expansion

### Status
17/17 tasks complete. Final frontend MVP slice now verifies cleanly for `sdd-verify`.
