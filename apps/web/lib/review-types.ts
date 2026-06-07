export type ReviewDecision = "approve" | "reject";

export type ExtractionResult = {
  provider: string;
  confidence: number;
  warnings: string[];
};

export type CircuitSpec = {
  confidence: number;
  warnings: string[];
  status: string;
};

export type CorrectedCircuitSpec = {
  confidence: number;
  warnings: string[];
  status?: string;
};

export type ExtractionRevision = {
  id: string;
  workspace_id: string;
  project_id: string;
  upload_id: string;
  job_id: string;
  version: number;
  object_key: string;
  provider: string;
  confidence: number;
  warnings: string[];
  result: ExtractionResult;
  created_at: string;
  updated_at: string;
};

export type CircuitRevision = {
  id: string;
  workspace_id: string;
  project_id: string;
  upload_id: string;
  job_id: string;
  version: number;
  object_key: string;
  confidence: number;
  warnings: string[];
  status: string;
  spec: CircuitSpec;
  assembly_plan_object_key?: string;
  scene_spec_object_key?: string;
  viewer_payload_object_key?: string;
  created_at: string;
  updated_at: string;
};

export type ReviewQueueItem = {
  job_id: string;
  circuit_revision_id: string;
  upload_id: string;
  confidence: number;
  warnings: string[];
  queued_for_review_at: string;
};

export type ReviewJob = {
  id: string;
  project_id: string;
  upload_id: string;
};

export type ReviewState = {
  job_status: string;
  circuit_revision_id: string;
  circuit_status: string;
};

export type ReviewDetail = {
  job: ReviewJob;
  extraction: ExtractionRevision;
  circuit: CircuitSpec;
  confidence: number;
  warnings: string[];
  review_state: ReviewState;
};

export type ReviewDecisionRecord = {
  id: string;
  workspace_id: string;
  project_id: string;
  job_id: string;
  extraction_revision_id: string;
  reviewed_circuit_revision_id: string;
  resolved_circuit_revision_id: string;
  reviewer_id: string;
  decision: ReviewDecision;
  note: string;
  reviewed_at: string;
  created_at: string;
};

export type SubmitReviewDecisionPayload = {
  circuit_revision_id: string;
  decision: ReviewDecision;
  note: string;
  reviewed_at: string;
  corrected_spec?: CorrectedCircuitSpec;
};

export type SubmitReviewDecisionResponse = {
  decision: ReviewDecisionRecord;
  reviewed_revision: CircuitRevision;
  resolved_revision: CircuitRevision;
  job_status: string;
};
