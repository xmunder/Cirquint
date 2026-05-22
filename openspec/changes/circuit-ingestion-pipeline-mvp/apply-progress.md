# Apply Progress — circuit-ingestion-pipeline-mvp

- **Proyecto canónico:** `cirquint`
- **Cambio:** `circuit-ingestion-pipeline-mvp`
- **Estrategia de entrega:** `ask-on-risk`
- **Chain strategy:** `feature-branch-chain`
- **Estado actual:** `slice-2-completed`

## Slice 1 implementado

**Boundary:** base backend scaffold + minimal project/workspace API foundation.

### Implementado

- scaffold inicial en `backend/` para monolito modular en Go
- bootstrap HTTP mínimo
- boundaries explícitos para `identity`, `workspace`, `uploads`, `processing`, `circuit`, `provider`, `storage`, `platform`
- flujo mínimo de `workspace/project`
- migración inicial SQL
- stub de uploads fuera de scope funcional
- tests/golden tests iniciales

### Archivos principales

- `backend/go.mod`
- `backend/cmd/api/main.go`
- `backend/internal/config/config.go`
- `backend/internal/platform/runtime.go`
- `backend/internal/platform/httpjson/httpjson.go`
- `backend/internal/identity/*`
- `backend/internal/workspace/*`
- `backend/internal/uploads/uploads.go`
- `backend/internal/processing/processing.go`
- `backend/internal/circuit/circuit.go`
- `backend/internal/provider/provider.go`
- `backend/internal/storage/storage.go`
- `backend/internal/server/*`
- `backend/migrations/0001_initial_schema.sql`

### Verificación ejecutada

- `gofmt -w` sobre archivos Go del backend
- `cd backend && go test ./...`

### Hallazgos

- se corrigió un bug real de routing en `/projects/{id}/uploads`
- la persistencia sigue siendo in-memory en este slice para mantener review scope acotado

## Próximo slice recomendado

## Slice 2 implementado

**Boundary:** upload vertical slice con storage adapter, metadata persistence, job polling y compensación de fallos sin worker.

### Implementado

- adapter concreto `R2Storage` sobre bucket interface y generación de keys `workspaces/{workspaceId}/uploads/{uploadId}/v1/source.{ext}`
- repos in-memory para `uploads` y `processing_jobs` respetando boundaries del monolito modular
- servicio `uploads` que valida imagen, persiste metadata, sube binario y mueve job por `created -> uploaded -> queued`
- endpoint `POST /projects/{projectID}/uploads` con multipart `file`
- endpoint `GET /jobs/{jobID}` para polling de estado persistido
- tests unitarios para key generation y transitions
- tests con fakes para fallos de storage/DB garantizando que no quedan jobs `queued` huérfanos

### Archivos principales

- `backend/internal/storage/storage.go`
- `backend/internal/storage/storage_test.go`
- `backend/internal/uploads/uploads.go`
- `backend/internal/uploads/memory_repository.go`
- `backend/internal/uploads/uploads_test.go`
- `backend/internal/processing/processing.go`
- `backend/internal/processing/memory_repository.go`
- `backend/internal/processing/processing_test.go`
- `backend/internal/server/server.go`
- `backend/internal/server/memory_bucket.go`
- `backend/internal/server/server_test.go`
- `backend/internal/server/testdata/job_status_queued.golden`

### Verificación ejecutada

- `gofmt -w ./cmd ./internal`
- `cd backend && go test ./...`

### Hallazgos

- el contrato HTTP de upload necesitaba `multipart/form-data`; los tests fallaban si el part no seteaba `Content-Type`
- para mantener el slice reviewable no se introdujo DB real ni cola real; la persistencia sigue detrás de repos/adapters listos para reemplazo

## Próximo slice recomendado

### Slice 3 / PR3

- worker que reclama solo jobs `queued`
- boundary del provider de extracción
- persistencia de `ExtractionResult` y `CircuitSpec`
- resolución `ready | needs_review | failed`

## Notas

- `AssemblyPlan`, `SceneSpec`, viewer 3D y ejecución real de extracción IA siguen fuera de scope en este punto.
- Este archivo complementa el artifact en Engram y mantiene trazabilidad híbrida real.
