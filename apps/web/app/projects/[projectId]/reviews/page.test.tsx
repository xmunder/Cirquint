import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import QueuePage from "./page";

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock("@/lib/review-api", () => ({
  listProjectReviews: vi.fn(),
}));

import { listProjectReviews } from "@/lib/review-api";

const mockedListProjectReviews = vi.mocked(listProjectReviews);

describe("queue page", () => {
  it("renders project review rows with detail links", async () => {
    mockedListProjectReviews.mockResolvedValueOnce([
      {
        job_id: "job-001",
        circuit_revision_id: "cir-001",
        upload_id: "upl-001",
        confidence: 0.42,
        warnings: ["ambiguous-node-label"],
        queued_for_review_at: "2026-05-23T09:00:00Z",
      },
    ]);

    render(await QueuePage({ params: Promise.resolve({ projectId: "prj-001" }) }));

    expect(
      screen.getByRole("heading", { name: "Review queue" }),
    ).toBeInTheDocument();
    expect(screen.getByText("job-001")).toBeInTheDocument();
    expect(screen.getByText("upl-001")).toBeInTheDocument();
    expect(screen.getByText("cir-001")).toBeInTheDocument();
    expect(screen.getByText("0.42")).toBeInTheDocument();
    expect(screen.getByText("ambiguous-node-label")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Open review" }),
    ).toHaveAttribute("href", "/projects/prj-001/reviews/job-001");
    expect(mockedListProjectReviews).toHaveBeenCalledWith("prj-001");
  });

  it("renders an explicit empty state when the queue is empty", async () => {
    mockedListProjectReviews.mockResolvedValueOnce([]);

    render(await QueuePage({ params: Promise.resolve({ projectId: "prj-empty" }) }));

    expect(screen.getByText("No pending reviews for this project.")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
    expect(mockedListProjectReviews).toHaveBeenCalledWith("prj-empty");
  });

  it("renders success and stale banners from queue search params", async () => {
    mockedListProjectReviews.mockResolvedValueOnce([]).mockResolvedValueOnce([]);

    render(
      await QueuePage({
        params: Promise.resolve({ projectId: "prj-001" }),
        searchParams: Promise.resolve({ success: "approve" }),
      }),
    );

    expect(
      screen.getByText("Review approved and removed from the queue."),
    ).toBeInTheDocument();

    render(
      await QueuePage({
        params: Promise.resolve({ projectId: "prj-001" }),
        searchParams: Promise.resolve({ stale: "1" }),
      }),
    );

    expect(
      screen.getByText("That revision is no longer current. Pick the latest queued item."),
    ).toBeInTheDocument();
  });
});
