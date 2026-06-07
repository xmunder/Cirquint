"use client";

import { useFormStatus } from "react-dom";

export function SubmitButton() {
  const { pending } = useFormStatus();

  return (
    <button
      type="submit"
      disabled={pending}
      style={{
        border: 0,
        borderRadius: "999px",
        padding: "12px 18px",
        fontWeight: 700,
        background: pending ? "#94a3b8" : "#0f172a",
        color: "#f8fafc",
        cursor: pending ? "progress" : "pointer",
      }}
    >
      {pending ? "Submitting..." : "Submit decision"}
    </button>
  );
}
