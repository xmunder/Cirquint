import { beforeEach, describe, expect, it, vi } from "vitest";

const { redirectMock } = vi.hoisted(() => ({
  redirectMock: vi.fn((url: string) => {
    throw new Error(`redirect:${url}`);
  }),
}));

vi.mock("next/navigation", () => ({
  redirect: redirectMock,
}));

vi.mock("@/lib/review-api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/review-api")>();
  return {
    ...actual,
    submitReviewDecision: vi.fn(),
  };
});

import { submitReviewDecision } from "@/lib/review-api";

import { submitReviewDecisionAction } from "./actions";

const mockedSubmitReviewDecision = vi.mocked(submitReviewDecision);

describe("submitReviewDecisionAction", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-07T10:00:00Z"));
    redirectMock.mockClear();
    mockedSubmitReviewDecision.mockReset();
  });

  it("submits approvals with reviewed_at and redirects to the queue success banner", async () => {
    mockedSubmitReviewDecision.mockResolvedValueOnce({
      decision: {} as never,
      reviewed_revision: {} as never,
      resolved_revision: {} as never,
      job_status: "completed",
    });

    await expect(
      submitReviewDecisionAction(
        { projectId: "prj-001", jobId: "job-001" },
        { error: "old error" },
        createFormData({
          circuit_revision_id: "cir-001",
          decision: "approve",
          note: "human verified labels",
          corrected_spec: '{"confidence":0.99,"warnings":["human-corrected"]}',
        }),
      ),
    ).rejects.toThrow("redirect:/projects/prj-001/reviews?success=approve");

    expect(mockedSubmitReviewDecision).toHaveBeenCalledWith(
      "prj-001",
      "job-001",
      {
        circuit_revision_id: "cir-001",
        decision: "approve",
        note: "human verified labels",
        reviewed_at: "2026-06-07T10:00:00.000Z",
        corrected_spec: {
          confidence: 0.99,
          warnings: ["human-corrected"],
        },
      },
    );
  });

  it("redirects stale conflicts back to the queue stale banner", async () => {
    mockedSubmitReviewDecision.mockRejectedValueOnce({
      name: "ReviewApiStaleConflictError",
      message: "stale review revision",
      status: 409,
    });

    await expect(
      submitReviewDecisionAction(
        { projectId: "prj-001", jobId: "job-001" },
        { error: "old error" },
        createFormData({
          circuit_revision_id: "cir-001",
          decision: "reject",
          note: "superseded elsewhere",
        }),
      ),
    ).rejects.toThrow("redirect:/projects/prj-001/reviews?stale=1");
  });

  it("returns inline form errors for backend and corrected-spec failures", async () => {
    mockedSubmitReviewDecision.mockRejectedValueOnce(new Error("backend unavailable"));

    await expect(
      submitReviewDecisionAction(
        { projectId: "prj-001", jobId: "job-001" },
        { error: undefined },
        createFormData({
          circuit_revision_id: "cir-001",
          decision: "approve",
          note: "retry later",
          corrected_spec: "{bad-json",
        }),
      ),
    ).resolves.toEqual({ error: "corrected_spec must be valid JSON" });

    await expect(
      submitReviewDecisionAction(
        { projectId: "prj-001", jobId: "job-001" },
        { error: undefined },
        createFormData({
          circuit_revision_id: "cir-001",
          decision: "reject",
          note: "retry later",
        }),
      ),
    ).resolves.toEqual({ error: "backend unavailable" });
  });
});

function createFormData(fields: Record<string, string>): FormData {
  const formData = new FormData();
  for (const [key, value] of Object.entries(fields)) {
    formData.set(key, value);
  }
  return formData;
}
