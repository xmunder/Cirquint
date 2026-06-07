import { render, screen } from "@testing-library/react";

import QueueLoading from "./loading";

describe("queue loading", () => {
  it("renders loading feedback while the queue is pending", () => {
    render(<QueueLoading />);

    expect(
      screen.getByRole("heading", { name: "Loading review queue" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fetching the latest pending review work from the backend.")).toBeInTheDocument();
    expect(screen.getByRole("status", { busy: true })).toBeInTheDocument();
  });
});
