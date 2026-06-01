# Backend MVP Configuration

This document defines the **minimum runtime configuration** for the backend MVP slice of `circuit-ingestion-pipeline-mvp`.

## Minimum environment variables

| Variable | Default | Required for MVP | Purpose |
|---|---|---|---|
| `HTTP_ADDR` | `:8080` | No | HTTP listen address for `cmd/api`. |
| `REDIS_ADDR` | `127.0.0.1:6379` | Yes (runtime) | Redis endpoint used by the processing queue. |
| `REDIS_QUEUE_KEY` | `processing:jobs` | No | Redis list key for queued processing jobs. |
| `REVIEW_MIN_CONFIDENCE` | `0.8` (current code default) | No | Confidence threshold used by review gating (`ready` vs `needs_review`). |

## MVP decision note: review threshold is still open

`REVIEW_MIN_CONFIDENCE` exists and is configurable now, but the **exact threshold value for MVP is intentionally still an open product/quality decision** (see change design open questions).

This means:

- you MAY set `REVIEW_MIN_CONFIDENCE` per environment,
- the default `0.8` is an implementation fallback,
- and it MUST NOT be treated as a finalized business threshold yet.
