import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SubmitButton } from "./submit-button";

const { mockedUseFormStatus } = vi.hoisted(() => ({
  mockedUseFormStatus: vi.fn(),
}));

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");

  return {
    ...actual,
    useFormStatus: mockedUseFormStatus,
  };
});

describe("SubmitButton", () => {
  beforeEach(() => {
    mockedUseFormStatus.mockReset();
  });

  it("shows the non-idle submit state while the form is pending", () => {
    mockedUseFormStatus.mockReturnValue({ pending: true });

    render(<SubmitButton />);

    expect(screen.getByRole("button", { name: "Submitting..." })).toBeDisabled();
  });

  it("shows the idle submit state when the form is not pending", () => {
    mockedUseFormStatus.mockReturnValue({ pending: false });

    render(<SubmitButton />);

    expect(screen.getByRole("button", { name: "Submit decision" })).toBeEnabled();
  });
});
