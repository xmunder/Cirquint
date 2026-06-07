"use server";

import { redirect } from "next/navigation";

import {
  parseCorrectedSpecInput,
  ReviewApiError,
  ReviewApiStaleConflictError,
  submitReviewDecision,
} from "@/lib/review-api";
import type { ReviewDecision } from "@/lib/review-types";

export type ReviewDecisionFormState = {
  error?: string;
};

type ReviewDecisionActionContext = {
  projectId: string;
  jobId: string;
};

export async function submitReviewDecisionAction(
  context: ReviewDecisionActionContext,
  _previousState: ReviewDecisionFormState,
  formData: FormData,
): Promise<ReviewDecisionFormState> {
  const circuitRevisionId = readRequiredField(formData, "circuit_revision_id");
  const decision = readDecision(formData);
  const note = readRequiredField(formData, "note");

  try {
    await submitReviewDecision(context.projectId, context.jobId, {
      circuit_revision_id: circuitRevisionId,
      decision,
      note,
      reviewed_at: new Date().toISOString(),
      ...(decision === "approve"
        ? {
            corrected_spec: parseCorrectedSpecInput(readOptionalField(formData, "corrected_spec")),
          }
        : {}),
    });
  } catch (error) {
    if (isStaleConflict(error)) {
      redirect(`/projects/${context.projectId}/reviews?stale=1`);
    }

    if (error instanceof ReviewApiError || error instanceof Error) {
      return { error: error.message };
    }

    return { error: "Unable to submit this review decision." };
  }

  redirect(`/projects/${context.projectId}/reviews?success=${decision}`);
}

function readDecision(formData: FormData): ReviewDecision {
  const value = formData.get("decision");
  return value === "reject" ? "reject" : "approve";
}

function readRequiredField(formData: FormData, key: string): string {
  return String(formData.get(key) ?? "").trim();
}

function readOptionalField(formData: FormData, key: string): string {
  return String(formData.get(key) ?? "");
}

function isStaleConflict(error: unknown): boolean {
  return (
    error instanceof ReviewApiStaleConflictError ||
    (typeof error === "object" &&
      error !== null &&
      "status" in error &&
      error.status === 409)
  );
}
