import { describe, expect, it, vi } from "vitest";

import {
  ReviewApiStaleConflictError,
  listProjectReviews,
  parseCorrectedSpecInput,
  submitReviewDecision,
} from "./review-api";
import type { SubmitReviewDecisionPayload } from "./review-types";

function createJsonResponse(body: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    headers: { "content-type": "application/json" },
    status: init?.status ?? 200,
  });
}

describe("review-api", () => {
  it("lists review queue items with no-store fetches", async () => {
    const fetchFn = vi.fn(async () =>
      createJsonResponse({
        items: [
          {
            job_id: "job-001",
            circuit_revision_id: "cir-001",
            upload_id: "upl-001",
            confidence: 0.42,
            warnings: ["ambiguous-node-label"],
            queued_for_review_at: "2026-05-23T09:00:00Z",
          },
        ],
      }),
    );

    const result = await listProjectReviews("project 1", {
      baseUrl: "http://backend.test",
      fetchFn,
    });

    expect(result).toEqual([
      {
        job_id: "job-001",
        circuit_revision_id: "cir-001",
        upload_id: "upl-001",
        confidence: 0.42,
        warnings: ["ambiguous-node-label"],
        queued_for_review_at: "2026-05-23T09:00:00Z",
      },
    ]);
    expect(fetchFn).toHaveBeenCalledWith(
      "http://backend.test/projects/project%201/circuit-reviews",
      { cache: "no-store" },
    );
  });

  it("maps 409 decision responses to stale conflicts", async () => {
    const payload: SubmitReviewDecisionPayload = {
      circuit_revision_id: "cir-001",
      decision: "reject",
      note: "Already handled",
      reviewed_at: "2026-05-23T09:00:00Z",
    };
    const fetchFn = vi.fn(async () =>
      createJsonResponse({ error: "stale review revision" }, { status: 409 }),
    );

    const promise = submitReviewDecision("prj-001", "job-001", payload, {
      baseUrl: "http://backend.test",
      fetchFn,
    });

    await expect(promise).rejects.toBeInstanceOf(ReviewApiStaleConflictError);
    expect(fetchFn).toHaveBeenCalledWith(
      "http://backend.test/projects/prj-001/circuit-reviews/job-001/decision",
      {
        body: JSON.stringify(payload),
        cache: "no-store",
        headers: {
          "content-type": "application/json",
        },
        method: "POST",
      },
    );
  });

  it("parses corrected circuit specs from JSON input", () => {
    expect(parseCorrectedSpecInput("")).toBeUndefined();
    expect(
      parseCorrectedSpecInput(
        '{"confidence":0.99,"warnings":["human-corrected"],"status":"ready"}',
      ),
    ).toEqual({
      confidence: 0.99,
      warnings: ["human-corrected"],
      status: "ready",
    });
  });

  it("rejects invalid corrected circuit spec JSON", () => {
    expect(() => parseCorrectedSpecInput("not-json")).toThrow(
      "corrected_spec must be valid JSON",
    );
    expect(() =>
      parseCorrectedSpecInput('{"confidence":"high","warnings":[]}'),
    ).toThrow("corrected_spec.confidence must be a number");
  });
});
