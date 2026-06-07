import http from "node:http";

const host = process.env.MOCK_REVIEW_BACKEND_HOST ?? "127.0.0.1";
const port = Number.parseInt(process.env.MOCK_REVIEW_BACKEND_PORT ?? "4100", 10);

const state = {
  scenario: "approve-success",
  decisionCalls: 0,
  afterDecision: false,
};

const projectId = "prj-001";
const jobId = "job-001";
const uploadId = "upl-001";
const initialRevisionId = "cir-001";
const staleRevisionId = "cir-002";

const server = http.createServer(async (request, response) => {
  try {
    const url = new URL(request.url ?? "/", `http://${request.headers.host ?? `${host}:${port}`}`);

    if (request.method === "GET" && url.pathname === "/__playwright__/health") {
      return writeJson(response, 200, { ok: true });
    }

    if (request.method === "POST" && url.pathname === "/__playwright__/scenario") {
      const body = await readJsonBody(request);
      state.scenario = typeof body.scenario === "string" ? body.scenario : "approve-success";
      state.decisionCalls = 0;
      state.afterDecision = false;
      return writeJson(response, 200, { ok: true, scenario: state.scenario });
    }

    if (request.method === "GET" && url.pathname === "/__playwright__/state") {
      return writeJson(response, 200, state);
    }

    if (request.method === "GET" && url.pathname === `/projects/${projectId}/circuit-reviews`) {
      return writeJson(response, 200, { items: getQueueItems() });
    }

    if (request.method === "GET" && url.pathname === `/projects/${projectId}/circuit-reviews/${jobId}`) {
      return writeJson(response, 200, createReviewDetail(initialRevisionId));
    }

    if (request.method === "POST" && url.pathname === `/projects/${projectId}/circuit-reviews/${jobId}/decision`) {
      const payload = await readJsonBody(request);
      state.decisionCalls += 1;

      if (state.scenario === "retry-error" && state.decisionCalls === 1) {
        return writeJson(response, 503, { error: "backend unavailable" });
      }

      if (state.scenario === "stale-conflict") {
        state.afterDecision = true;
        return writeJson(response, 409, { error: "stale review revision" });
      }

      state.afterDecision = true;
      return writeJson(response, 200, createDecisionResponse(payload));
    }

    return writeJson(response, 404, { error: `Unhandled route: ${request.method} ${url.pathname}` });
  } catch (error) {
    const message = error instanceof Error ? error.message : "unexpected mock backend failure";
    return writeJson(response, 500, { error: message });
  }
});

server.listen(port, host, () => {
  process.stdout.write(`Mock review backend listening on http://${host}:${port}\n`);
});

process.on("SIGTERM", () => {
  server.close(() => process.exit(0));
});

process.on("SIGINT", () => {
  server.close(() => process.exit(0));
});

function getQueueItems() {
  if (!state.afterDecision) {
    return [createQueueItem(jobId, initialRevisionId)];
  }

  if (state.scenario === "stale-conflict") {
    return [createQueueItem("job-002", staleRevisionId)];
  }

  return [];
}

function createQueueItem(queueJobId, revisionId) {
  return {
    job_id: queueJobId,
    circuit_revision_id: revisionId,
    upload_id: uploadId,
    confidence: 0.42,
    warnings: ["ambiguous-node-label"],
    queued_for_review_at: "2026-05-23T09:00:00Z",
  };
}

function createReviewDetail(revisionId) {
  return {
    job: {
      id: jobId,
      project_id: projectId,
      upload_id: uploadId,
    },
    extraction: {
      id: "ext-001",
      workspace_id: "ws-001",
      project_id: projectId,
      upload_id: uploadId,
      job_id: jobId,
      version: 3,
      object_key: "extract.json",
      provider: "mock-provider",
      confidence: 0.42,
      warnings: ["ambiguous-node-label"],
      result: {
        provider: "mock-provider",
        confidence: 0.42,
        warnings: ["ambiguous-node-label"],
      },
      created_at: "2026-05-23T08:45:00Z",
      updated_at: "2026-05-23T09:00:00Z",
    },
    circuit: {
      confidence: 0.4,
      warnings: ["missing-label"],
      status: "needs_review",
    },
    confidence: 0.42,
    warnings: ["ambiguous-node-label"],
    review_state: {
      job_status: "needs_review",
      circuit_revision_id: revisionId,
      circuit_status: "needs_review",
    },
  };
}

function createDecisionResponse(payload) {
  return {
    decision: {
      id: "dec-001",
      workspace_id: "ws-001",
      project_id: projectId,
      job_id: jobId,
      extraction_revision_id: "ext-001",
      reviewed_circuit_revision_id: payload.circuit_revision_id,
      resolved_circuit_revision_id: payload.circuit_revision_id,
      reviewer_id: "anonymous",
      decision: payload.decision,
      note: payload.note,
      reviewed_at: payload.reviewed_at,
      created_at: payload.reviewed_at,
    },
    reviewed_revision: {
      id: payload.circuit_revision_id,
      workspace_id: "ws-001",
      project_id: projectId,
      upload_id: uploadId,
      job_id: jobId,
      version: 3,
      object_key: "circuit.json",
      confidence: 0.4,
      warnings: ["missing-label"],
      status: "needs_review",
      spec: {
        confidence: 0.4,
        warnings: ["missing-label"],
        status: "needs_review",
      },
      created_at: payload.reviewed_at,
      updated_at: payload.reviewed_at,
    },
    resolved_revision: {
      id: payload.circuit_revision_id,
      workspace_id: "ws-001",
      project_id: projectId,
      upload_id: uploadId,
      job_id: jobId,
      version: 4,
      object_key: "circuit-resolved.json",
      confidence: payload.corrected_spec?.confidence ?? 0.4,
      warnings: payload.corrected_spec?.warnings ?? ["missing-label"],
      status: payload.corrected_spec?.status ?? "approved",
      spec: {
        confidence: payload.corrected_spec?.confidence ?? 0.4,
        warnings: payload.corrected_spec?.warnings ?? ["missing-label"],
        status: payload.corrected_spec?.status ?? "approved",
      },
      created_at: payload.reviewed_at,
      updated_at: payload.reviewed_at,
    },
    job_status: "completed",
  };
}

async function readJsonBody(request) {
  const chunks = [];
  for await (const chunk of request) {
    chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
  }

  const raw = Buffer.concat(chunks).toString("utf8").trim();
  if (raw === "") {
    return {};
  }

  return JSON.parse(raw);
}

function writeJson(response, status, body) {
  response.writeHead(status, {
    "content-type": "application/json",
    "cache-control": "no-store",
  });
  response.end(JSON.stringify(body));
}
