import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ReviewDecisionForm } from "./review-decision-form";

describe("ReviewDecisionForm", () => {
  it("shows corrected_spec only while approve is selected", () => {
    render(
      <ReviewDecisionForm
        action={vi.fn()}
        circuitRevisionId="cir-001"
      />,
    );

    expect(screen.getByLabelText("Corrected spec JSON")).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText("Reject"));

    expect(screen.queryByLabelText("Corrected spec JSON")).not.toBeInTheDocument();

    fireEvent.click(screen.getByLabelText("Approve"));

    expect(screen.getByLabelText("Corrected spec JSON")).toBeInTheDocument();
  });

  it("requires a review note, binds the revision id, and renders inline errors", () => {
    render(
      <ReviewDecisionForm
        action={vi.fn()}
        circuitRevisionId="cir-001"
        initialState={{ error: "backend unavailable" }}
      />,
    );

    expect(screen.getByLabelText("Review note")).toBeRequired();
    expect(screen.getByDisplayValue("cir-001")).toHaveAttribute("type", "hidden");
    expect(screen.getByText("backend unavailable")).toBeInTheDocument();
  });
});
