import type {
  CorrectedCircuitSpec,
  ReviewDetail,
  ReviewQueueItem,
  SubmitReviewDecisionPayload,
  SubmitReviewDecisionResponse,
} from "./review-types";

type ReviewFetch = typeof fetch;

type ReviewApiOptions = {
  baseUrl?: string;
  fetchFn?: ReviewFetch;
};

type ErrorBody = {
  error?: string;
};

export class ReviewApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "ReviewApiError";
  }
}

export class ReviewApiStaleConflictError extends ReviewApiError {
  constructor(message: string) {
    super(message, 409);
    this.name = "ReviewApiStaleConflictError";
  }
}

export function getBackendBaseUrl(env: NodeJS.ProcessEnv = process.env): string {
  const raw = env.BACKEND_BASE_URL?.trim();
  if (!raw) {
    throw new Error("BACKEND_BASE_URL is required");
  }

  return raw.endsWith("/") ? raw.slice(0, -1) : raw;
}

export function parseCorrectedSpecInput(raw: string): CorrectedCircuitSpec | undefined {
  const trimmed = raw.trim();
  if (trimmed === "") {
    return undefined;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("corrected_spec must be valid JSON");
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("corrected_spec must be a JSON object");
  }

  const candidate = parsed as Record<string, unknown>;
  if (typeof candidate.confidence !== "number") {
    throw new Error("corrected_spec.confidence must be a number");
  }
  if (!Array.isArray(candidate.warnings) || candidate.warnings.some((item) => typeof item !== "string")) {
    throw new Error("corrected_spec.warnings must be an array of strings");
  }
  if (candidate.status !== undefined && typeof candidate.status !== "string") {
    throw new Error("corrected_spec.status must be a string when provided");
  }

  return {
    confidence: candidate.confidence,
    warnings: [...candidate.warnings],
    ...(candidate.status === undefined ? {} : { status: candidate.status }),
  };
}

export async function listProjectReviews(
  projectId: string,
  options?: ReviewApiOptions,
): Promise<ReviewQueueItem[]> {
  const response = await requestJSON<{ items: ReviewQueueItem[] }>(
    `/projects/${encodeURIComponent(projectId)}/circuit-reviews`,
    {
      cache: "no-store",
    },
    options,
  );

  return response.items;
}

export async function getProjectReview(
  projectId: string,
  jobId: string,
  options?: ReviewApiOptions,
): Promise<ReviewDetail> {
  return requestJSON<ReviewDetail>(
    `/projects/${encodeURIComponent(projectId)}/circuit-reviews/${encodeURIComponent(jobId)}`,
    {
      cache: "no-store",
    },
    options,
  );
}

export async function submitReviewDecision(
  projectId: string,
  jobId: string,
  payload: SubmitReviewDecisionPayload,
  options?: ReviewApiOptions,
): Promise<SubmitReviewDecisionResponse> {
  return requestJSON<SubmitReviewDecisionResponse>(
    `/projects/${encodeURIComponent(projectId)}/circuit-reviews/${encodeURIComponent(jobId)}/decision`,
    {
      method: "POST",
      cache: "no-store",
      headers: {
        "content-type": "application/json",
      },
      body: JSON.stringify(payload),
    },
    options,
  );
}

async function requestJSON<T>(
  path: string,
  init: RequestInit,
  options?: ReviewApiOptions,
): Promise<T> {
  const fetchFn = options?.fetchFn ?? fetch;
  const baseUrl = options?.baseUrl ?? getBackendBaseUrl();
  const response = await fetchFn(`${baseUrl}${path}`, init);

  if (!response.ok) {
    throw await toReviewApiError(response);
  }

  return (await response.json()) as T;
}

async function toReviewApiError(response: Response): Promise<ReviewApiError> {
  const body = await readErrorBody(response);
  const message = body.error?.trim() || `Request failed with status ${response.status}`;

  if (response.status === 409) {
    return new ReviewApiStaleConflictError(message);
  }

  return new ReviewApiError(message, response.status);
}

async function readErrorBody(response: Response): Promise<ErrorBody> {
  try {
    return (await response.json()) as ErrorBody;
  } catch {
    return {};
  }
}
