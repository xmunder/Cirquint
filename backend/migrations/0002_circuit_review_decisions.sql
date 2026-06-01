CREATE TABLE circuit_review_decisions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    extraction_revision_id TEXT NOT NULL REFERENCES extraction_revisions(id) ON DELETE RESTRICT,
    reviewed_circuit_revision_id TEXT NOT NULL REFERENCES circuit_revisions(id) ON DELETE RESTRICT,
    resolved_circuit_revision_id TEXT REFERENCES circuit_revisions(id) ON DELETE RESTRICT,
    reviewer_id TEXT NOT NULL,
    decision TEXT NOT NULL,
    note TEXT NOT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (reviewed_circuit_revision_id)
);

CREATE INDEX circuit_review_decisions_project_created_at_idx
    ON circuit_review_decisions(project_id, created_at);

CREATE INDEX circuit_review_decisions_job_id_idx
    ON circuit_review_decisions(job_id);
