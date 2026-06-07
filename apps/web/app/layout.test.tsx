import { render, screen } from "@testing-library/react";

import { ReviewShell } from "./layout";

describe("ReviewShell", () => {
  it("renders the shared review shell heading", () => {
    render(
      <ReviewShell>
        <div>Queue content</div>
      </ReviewShell>,
    );

    expect(
      screen.getByRole("heading", { name: "Circuit review workbench" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Queue content")).toBeInTheDocument();
  });
});
