# Circuit Image Ingestion Specification

## Purpose

Define backend-owned upload, storage, job kickoff, and status polling.

## Requirements

### Requirement: Schematic Upload and Job Creation

The system MUST accept a schematic image for an existing project, persist upload metadata, store the binary through the storage adapter in R2, and create one persistent processing job.

#### Scenario: Upload creates a queued job

- GIVEN an existing project and a supported schematic image
- WHEN the upload completes successfully
- THEN the binary is stored in R2 through the storage adapter and metadata is persisted
- AND a processing job is persisted with status reaching `queued`

#### Scenario: Upload failure does not queue processing

- GIVEN an existing project and an upload attempt
- WHEN metadata persistence or object storage fails
- THEN the system reports the upload as failed
- AND no queued processing job is left behind

### Requirement: Current Processing Status Retrieval

The system MUST expose current processing status so the UI can poll the latest persisted state.

#### Scenario: Poll current job status

- GIVEN a project upload with an existing processing job
- WHEN a client requests current processing status
- THEN the system returns the latest persisted job state
- AND the response reflects one of `created`, `uploaded`, `queued`, `extracting`, `needs_review`, `ready`, or `failed`
