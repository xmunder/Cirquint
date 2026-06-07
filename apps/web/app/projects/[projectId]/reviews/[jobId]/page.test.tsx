import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ReviewDetailError from "./error";
import ReviewDetailPage from "./page";

vi.mock("@/lib/review-api", () => ({
  getProjectReview: vi.fn(),
}));

import { getProjectReview } from "@/lib/review-api";

const mockedGetProjectReview = vi.mocked(getProjectReview);

describe("review detail page", () => {
  it("renders the review detail payload", async () => {
    mockedGetProjectReview.mockResolvedValueOnce({
      job: {
        id: "job-001",
        project_id: "prj-001",
        upload_id: "upl-001",
      },
      extraction: {
        id: "ext-001",
        workspace_id: "ws-001",
        project_id: "prj-001",
        upload_id: "upl-001",
        job_id: "job-001",
        version: 3,
        object_key: "extract.json",
        provider: "mock-provider",
        confidence: 0.42,
        warnings: ["ambiguous-node-label"],
        result: {
          provider: "mock-provider",
          confidence: 0.42,
          warnings: ["ambiguous-node-label"],
        },
        created_at: "2026-05-23T08:45:00Z",
        updated_at: "2026-05-23T09:00:00Z",
      },
      circuit: {
        confidence: 0.4,
        warnings: ["missing-label"],
        status: "needs_review",
      },
      confidence: 0.42,
      warnings: ["ambiguous-node-label"],
      review_state: {
        job_status: "needs_review",
        circuit_revision_id: "cir-001",
        circuit_status: "needs_review",
      },
    });

    render(
      await ReviewDetailPage({
        params: Promise.resolve({ projectId: "prj-001", jobId: "job-001" }),
      }),
    );

    expect(
      screen.getByRole("heading", { name: "Review detail" }),
    ).toBeInTheDocument();
    expect(screen.getByText("job-001")).toBeInTheDocument();
    expect(screen.getByText("upl-001")).toBeInTheDocument();
    expect(screen.getByText("mock-provider")).toBeInTheDocument();
    expect(screen.getByText("Raw extraction result")).toBeInTheDocument();
    expect(screen.getAllByText("ambiguous-node-label")).toHaveLength(2);
    expect(screen.getByText("missing-label")).toBeInTheDocument();
    expect(screen.getAllByText("needs_review")).toHaveLength(3);
    expect(screen.getByText((content) => content.includes('"status": "needs_review"'))).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Decision" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("cir-001")).toHaveAttribute("type", "hidden");
    expect(screen.getByLabelText("Review note")).toBeRequired();
    expect(mockedGetProjectReview).toHaveBeenCalledWith("prj-001", "job-001");
  });

  it("renders the detail error fallback", () => {
    const reset = vi.fn();

    render(<ReviewDetailError error={new Error("backend unavailable")} reset={reset} />);

    expect(screen.getByRole("heading", { name: "Unable to load this review" })).toBeInTheDocument();
    expect(screen.getByText("backend unavailable")).toBeInTheDocument();
    screen.getByRole("button", { name: "Try again" }).click();
    expect(reset).toHaveBeenCalledTimes(1);
  });
});
