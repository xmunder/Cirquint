import type { ReactNode } from "react";

import Link from "next/link";

import { submitReviewDecisionAction } from "./actions";

import { ReviewDecisionForm } from "@/components/review-decision-form";
import { getProjectReview } from "@/lib/review-api";

type ReviewDetailPageProps = {
  params: Promise<{
    projectId: string;
    jobId: string;
  }>;
};

export default async function ReviewDetailPage({ params }: ReviewDetailPageProps) {
  const { projectId, jobId } = await params;
  const detail = await getProjectReview(projectId, jobId);
  const submitAction = submitReviewDecisionAction.bind(null, { projectId, jobId });

  return (
    <section style={{ display: "grid", gap: "24px" }}>
      <header>
        <p style={{ margin: 0 }}>
          <Link href={`/projects/${projectId}/reviews`}>Back to queue</Link>
        </p>
        <h2 style={{ margin: "12px 0 0", fontSize: "1.5rem" }}>Review detail</h2>
        <p style={{ margin: "8px 0 0", color: "#475569" }}>
          Inspect the backend review payload before taking action.
        </p>
      </header>

      <DetailCard title="Job metadata">
        <DetailList
          rows={[
            ["Job ID", detail.job.id],
            ["Project ID", detail.job.project_id],
            ["Upload ID", detail.job.upload_id],
          ]}
        />
      </DetailCard>

      <DetailCard title="Review state">
        <DetailList
          rows={[
            ["Job status", detail.review_state.job_status],
            ["Circuit revision", detail.review_state.circuit_revision_id],
            ["Circuit status", detail.review_state.circuit_status],
            ["Confidence", detail.confidence.toFixed(2)],
            ["Warnings", formatWarnings(detail.warnings)],
          ]}
        />
      </DetailCard>

      <DetailCard title="Extraction payload">
        <DetailList
          rows={[
            ["Extraction ID", detail.extraction.id],
            ["Provider", detail.extraction.provider],
            ["Extraction confidence", detail.extraction.confidence.toFixed(2)],
            ["Version", String(detail.extraction.version)],
            ["Object key", detail.extraction.object_key],
            ["Created at", formatTimestamp(detail.extraction.created_at)],
            ["Updated at", formatTimestamp(detail.extraction.updated_at)],
            ["Extraction warnings", formatWarnings(detail.extraction.warnings)],
          ]}
        />
        <h4 style={{ margin: "16px 0 12px", fontSize: "0.95rem" }}>Raw extraction result</h4>
        <pre
          style={{
            margin: 0,
            padding: "16px",
            overflowX: "auto",
            borderRadius: "14px",
            background: "#0f172a",
            color: "#e2e8f0",
          }}
        >
          {JSON.stringify(detail.extraction.result, null, 2)}
        </pre>
      </DetailCard>

      <DetailCard title="Normalized CircuitSpec">
        <DetailList
          rows={[
            ["Circuit confidence", detail.circuit.confidence.toFixed(2)],
            ["Circuit warnings", formatWarnings(detail.circuit.warnings)],
            ["Circuit status", detail.circuit.status],
          ]}
        />
        <pre
          style={{
            margin: "16px 0 0",
            padding: "16px",
            overflowX: "auto",
            borderRadius: "14px",
            background: "#0f172a",
            color: "#e2e8f0",
          }}
        >
          {JSON.stringify(detail.circuit, null, 2)}
        </pre>
      </DetailCard>

      <DetailCard title="Decision">
        <p style={{ margin: "0 0 16px", color: "#475569" }}>
          Submit an approval or rejection for the current reviewable revision.
        </p>
        <ReviewDecisionForm
          action={submitAction}
          circuitRevisionId={detail.review_state.circuit_revision_id}
        />
      </DetailCard>
    </section>
  );
}

function DetailCard({
  title,
  children,
}: Readonly<{
  title: string;
  children: ReactNode;
}>) {
  return (
    <section
      style={{
        padding: "20px",
        border: "1px solid #dbe2f0",
        borderRadius: "18px",
        background: "#f8fafc",
      }}
    >
      <h3 style={{ margin: "0 0 14px", fontSize: "1.05rem" }}>{title}</h3>
      {children}
    </section>
  );
}

function DetailList({ rows }: Readonly<{ rows: Array<[string, string]> }>) {
  return (
    <dl
      style={{
        margin: 0,
        display: "grid",
        gridTemplateColumns: "minmax(180px, 220px) 1fr",
        gap: "12px 16px",
      }}
    >
      {rows.map(([label, value]) => (
        <FragmentRow key={label} label={label} value={value} />
      ))}
    </dl>
  );
}

function FragmentRow({ label, value }: Readonly<{ label: string; value: string }>) {
  return (
    <>
      <dt style={{ fontWeight: 700, color: "#0f172a" }}>{label}</dt>
      <dd style={{ margin: 0, color: "#334155" }}>{value}</dd>
    </>
  );
}

function formatWarnings(warnings: string[]): string {
  return warnings.length === 0 ? "None" : warnings.join(", ");
}

function formatTimestamp(value: string): string {
  return value.replace("T", " ").replace("Z", " UTC");
}
