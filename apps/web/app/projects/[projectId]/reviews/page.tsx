import Link from "next/link";

import { listProjectReviews } from "@/lib/review-api";

type QueuePageProps = {
  params: Promise<{
    projectId: string;
  }>;
  searchParams?: Promise<{
    success?: string;
    stale?: string;
  }>;
};

export default async function QueuePage({ params, searchParams }: QueuePageProps) {
  const { projectId } = await params;
  const query = (await searchParams) ?? {};
  const items = await listProjectReviews(projectId);

  return (
    <section>
      <header style={{ marginBottom: "24px" }}>
        <h2 style={{ margin: 0, fontSize: "1.5rem" }}>Review queue</h2>
        <p style={{ margin: "8px 0 0", color: "#475569" }}>
          Pending review work for project <strong>{projectId}</strong>.
        </p>
      </header>

      {renderQueueBanner(query)}

      {items.length === 0 ? (
        <p
          style={{
            margin: 0,
            padding: "18px 20px",
            border: "1px solid #dbe2f0",
            borderRadius: "14px",
            background: "#f8fafc",
          }}
        >
          No pending reviews for this project.
        </p>
      ) : (
        <table
          style={{
            width: "100%",
            borderCollapse: "collapse",
          }}
        >
          <thead>
            <tr>
              {[
                "Job ID",
                "Upload ID",
                "Revision ID",
                "Confidence",
                "Warnings",
                "Queued",
                "Action",
              ].map((label) => (
                <th
                  key={label}
                  style={{
                    padding: "12px 10px",
                    textAlign: "left",
                    fontSize: "0.85rem",
                    color: "#475569",
                    borderBottom: "1px solid #dbe2f0",
                  }}
                >
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.job_id}>
                <td style={cellStyle}>{item.job_id}</td>
                <td style={cellStyle}>{item.upload_id}</td>
                <td style={cellStyle}>{item.circuit_revision_id}</td>
                <td style={cellStyle}>{item.confidence.toFixed(2)}</td>
                <td style={cellStyle}>{formatWarnings(item.warnings)}</td>
                <td style={cellStyle}>{formatTimestamp(item.queued_for_review_at)}</td>
                <td style={cellStyle}>
                  <Link href={`/projects/${projectId}/reviews/${item.job_id}`}>
                    Open review
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

const cellStyle = {
  padding: "14px 10px",
  borderBottom: "1px solid #e2e8f0",
  verticalAlign: "top" as const,
};

function formatWarnings(warnings: string[]): string {
  return warnings.length === 0 ? "None" : warnings.join(", ");
}

function formatTimestamp(value: string): string {
  return value.replace("T", " ").replace("Z", " UTC");
}

function renderQueueBanner(query: { success?: string; stale?: string }) {
  if (query.stale === "1") {
    return <Banner tone="warning">That revision is no longer current. Pick the latest queued item.</Banner>;
  }

  if (query.success === "approve") {
    return <Banner tone="success">Review approved and removed from the queue.</Banner>;
  }

  if (query.success === "reject") {
    return <Banner tone="success">Review rejected and removed from the queue.</Banner>;
  }

  return null;
}

function Banner({
  children,
  tone,
}: Readonly<{
  children: React.ReactNode;
  tone: "success" | "warning";
}>) {
  return (
    <p
      role="status"
      style={{
        margin: "0 0 18px",
        padding: "14px 16px",
        borderRadius: "14px",
        border: `1px solid ${tone === "success" ? "#86efac" : "#fcd34d"}`,
        background: tone === "success" ? "#f0fdf4" : "#fffbeb",
        color: tone === "success" ? "#166534" : "#92400e",
      }}
    >
      {children}
    </p>
  );
}
