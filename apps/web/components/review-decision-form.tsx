"use client";

import { useActionState, useState } from "react";

import type { ReviewDecisionFormState } from "@/app/projects/[projectId]/reviews/[jobId]/actions";

import { SubmitButton } from "./submit-button";

type ReviewDecisionFormProps = {
  action: (state: ReviewDecisionFormState, formData: FormData) => Promise<ReviewDecisionFormState>;
  circuitRevisionId: string;
  initialState?: ReviewDecisionFormState;
};

export function ReviewDecisionForm({
  action,
  circuitRevisionId,
  initialState = {},
}: Readonly<ReviewDecisionFormProps>) {
  const [decision, setDecision] = useState<"approve" | "reject">("approve");
  const [state, formAction] = useActionState(action, initialState);

  return (
    <form action={formAction} style={{ display: "grid", gap: "16px" }}>
      <input type="hidden" name="circuit_revision_id" value={circuitRevisionId} />

      <fieldset style={fieldSetStyle}>
        <legend style={legendStyle}>Decision</legend>
        <label style={choiceStyle}>
          <input
            type="radio"
            name="decision"
            value="approve"
            checked={decision === "approve"}
            onChange={() => setDecision("approve")}
          />
          Approve
        </label>
        <label style={choiceStyle}>
          <input
            type="radio"
            name="decision"
            value="reject"
            checked={decision === "reject"}
            onChange={() => setDecision("reject")}
          />
          Reject
        </label>
      </fieldset>

      <label style={fieldStyle}>
        <span style={labelStyle}>Review note</span>
        <textarea
          name="note"
          required
          rows={4}
          style={textAreaStyle}
          placeholder="Explain the decision for the audit trail."
        />
      </label>

      {decision === "approve" ? (
        <label style={fieldStyle}>
          <span style={labelStyle}>Corrected spec JSON</span>
          <textarea
            name="corrected_spec"
            rows={8}
            style={textAreaStyle}
            placeholder='{"confidence":0.99,"warnings":["human-corrected"]}'
          />
        </label>
      ) : null}

      {state.error ? (
        <p
          role="alert"
          style={{
            margin: 0,
            padding: "12px 14px",
            borderRadius: "12px",
            background: "#fee2e2",
            color: "#991b1b",
          }}
        >
          {state.error}
        </p>
      ) : null}

      <div>
        <SubmitButton />
      </div>
    </form>
  );
}

const fieldSetStyle = {
  margin: 0,
  padding: 0,
  border: 0,
  display: "flex",
  gap: "16px",
  flexWrap: "wrap" as const,
};

const legendStyle = {
  marginBottom: "8px",
  fontWeight: 700,
};

const choiceStyle = {
  display: "inline-flex",
  alignItems: "center",
  gap: "8px",
};

const fieldStyle = {
  display: "grid",
  gap: "8px",
};

const labelStyle = {
  fontWeight: 700,
  color: "#0f172a",
};

const textAreaStyle = {
  width: "100%",
  border: "1px solid #cbd5e1",
  borderRadius: "14px",
  padding: "12px 14px",
  font: "inherit",
  resize: "vertical" as const,
  background: "#ffffff",
  color: "#0f172a",
};
