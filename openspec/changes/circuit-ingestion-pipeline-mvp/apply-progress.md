# Apply Progress — circuit-ingestion-pipeline-mvp

- **Proyecto canónico:** `cirquint`
- **Cambio:** `circuit-ingestion-pipeline-mvp`
- **Estrategia de entrega:** `ask-on-risk`
- **Chain strategy:** `feature-branch-chain`
- **Estado actual:** `slice-3-formal-gaps-closed`

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

## Slice 3 implementado

**Boundary:** worker/review slice con claim flow, adapter de provider, persistencia de revisiones e idempotencia del procesamiento.

### Implementado

- `processing` ahora modela queue + `ClaimQueued` con compare-and-swap `queued -> extracting`
- `uploads` encola el payload del job al entrar en `queued`
- `worker` consume payloads, carga el source desde object storage, invoca solo el boundary `CircuitExtractionProvider` y resuelve `ready | needs_review | failed`
- `circuit` persiste `ExtractionResult` y `CircuitSpec` como revisiones JSON con keys determinísticas por revision ID
- la review policy usa threshold configurable `REVIEW_MIN_CONFIDENCE` con default `0.8` y warnings con categorías `ambiguous` / `incomplete-circuit`
- tests cubren policy, CAS del claim, idempotencia por `job_id` y flujo `worker -> persistencia` para `ready`, `needs_review` y `failed`

### Archivos principales

- `backend/cmd/worker/main.go`
- `backend/internal/worker/worker.go`
- `backend/internal/worker/worker_test.go`
- `backend/internal/processing/processing.go`
- `backend/internal/processing/memory_queue.go`
- `backend/internal/processing/processing_test.go`
- `backend/internal/provider/provider.go`
- `backend/internal/circuit/circuit.go`
- `backend/internal/circuit/memory_repository.go`
- `backend/internal/circuit/circuit_test.go`
- `backend/internal/uploads/uploads.go`
- `backend/internal/config/config.go`
- `backend/internal/server/server.go`
- `backend/internal/server/memory_bucket.go`

### Verificación ejecutada

- `gofmt -w ./cmd ./internal`
- `cd backend && go test ./...`

### Hallazgos

- la cola del worker quedó alineada con el diseño mediante `RedisQueue`, manteniendo `processing.Queue` como boundary testeable y reemplazando el uso por defecto de `MemoryQueue` en wiring de API/worker
- la persistencia idempotente se resolvió por `job_id` a nivel de revisiones, evitando duplicados aunque un job se reprocese
- los tests ahora leen los JSON persistidos de `ExtractionResult` y `CircuitSpec`, y verifican explícitamente que `AssemblyPlan`, `SceneSpec` y viewer payloads siguen fuera de scope

## Slice 3 cierre formal (Phase 4)

**Boundary:** cerrar gaps formales restantes sin expandir scope ni introducir nueva lógica de negocio.

### Implementado

- se verificó que los tests del worker ya contienen assertions explícitas de scope guard para que `AssemblyPlan`, `SceneSpec` y payloads de viewer permanezcan vacíos
- se completó documentación mínima de configuración backend en `backend/README.md`
- se documentó `REVIEW_MIN_CONFIDENCE` y se explicitó que el valor exacto del threshold sigue abierto como decisión MVP
- se marcaron como completadas las tareas `4.1` y `4.2` en `tasks.md`

### Archivos principales

- `backend/README.md`
- `openspec/changes/circuit-ingestion-pipeline-mvp/tasks.md`
- `openspec/changes/circuit-ingestion-pipeline-mvp/apply-progress.md`

### Verificación ejecutada

- `cd backend && go test ./internal/worker -run 'TestWorker(PersistsExtractionArtifactsAndKeepsPlanningOutputsOutOfScope|IsIdempotentAcrossDuplicateQueueAttempts)$'`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 4.1 | `backend/internal/worker/worker_test.go` | Unit | ✅ Existing assertions re-run | ➖ Existing coverage verification task | ✅ Targeted tests pass | ➖ Existing multi-scenario tests (`ready`, `needs_review`, retry) | ➖ No production refactor in closure |
| 4.2 | N/A (documentation-only) | N/A | N/A | ➖ Structural documentation task | ✅ Documentation produced | ➖ Single-output structural task | ➖ No production refactor |

### Hallazgos

- los scope guards del spec ya estaban cubiertos en `backend/internal/worker/worker_test.go` (paths happy-path y retry/idempotencia)
- el valor por defecto actual de `REVIEW_MIN_CONFIDENCE` es `0.8`, pero se mantiene como fallback técnico hasta definir threshold final del MVP

## Notas

- `AssemblyPlan`, `SceneSpec`, viewer 3D y ejecución real de extracción IA siguen fuera de scope en este punto.
- Este archivo complementa el artifact en Engram y mantiene trazabilidad híbrida real.

## Warning cleanup batch (assertion quality)

**Boundary:** cerrar warnings de calidad de assertions del verify report sin expandir scope funcional.

### Implementado

- `backend/cmd/api/main_test.go`: se reemplazaron assertions de `handler != nil` por assertions de comportamiento HTTP (`404` + mensaje `not found`) para handlers creados por `run` y `execute`.
- `backend/cmd/worker/main_test.go`: se reemplazó el smoke test de `buildRunner` por una verificación conductual de wiring (`run(buildRunner(cfg))` devuelve error de dequeue que refleja la dirección Redis configurada).
- `backend/internal/uploads/uploads_test.go`: se eliminó la assertion sobre un stub nuevo/vacío y se validó el comportamiento real de cleanup sobre el repo usado por `uploads.Service` (llamada a `DeleteUpload` + `ErrUploadNotFound` posterior).

### Archivos principales

- `backend/cmd/api/main_test.go`
- `backend/cmd/worker/main_test.go`
- `backend/internal/uploads/uploads_test.go`

### Verificación ejecutada

- Safety net previo: `cd backend && go test ./cmd/api ./cmd/worker ./internal/uploads`
- GREEN final: `cd backend && go test ./cmd/api ./cmd/worker ./internal/uploads`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| AQ-1 (API handler assertions) | `backend/cmd/api/main_test.go` | Integration (`httptest`) | ✅ package baseline pass | ✅ Behavioral assertions written first (`404` + error body) | ✅ `go test ./cmd/api` | ✅ Two paths covered (`run` and `execute`) | ✅ Shared assertion helper extracted |
| AQ-2 (worker runner assertion) | `backend/cmd/worker/main_test.go` | Unit | ✅ package baseline pass | ✅ Behavioral assertion written first (configured Redis address must surface in dequeue error) | ✅ `go test ./cmd/worker` | ✅ Distinct path from existing run/error tests | ➖ No production refactor needed |
| AQ-3 (upload cleanup assertion) | `backend/internal/uploads/uploads_test.go` | Unit | ✅ package baseline pass | ✅ Replaced fresh-stub assertion with repository-behavior assertions | ✅ `go test ./internal/uploads` | ✅ Cleanup verified by delete call + post-delete lookup | ✅ Test double extended only to observe product behavior |

### Test Summary

- **Total tests written/updated**: 3
- **Total relevant tests passing**: 3 paquetes (`cmd/api`, `cmd/worker`, `internal/uploads`)
- **Layers used**: Unit + Integration (`httptest`)
- **Approval tests**: None — no production refactor task
- **Pure functions created**: 0

### Hallazgos

- los warnings provenían de assertions demasiado débiles y no de fallas funcionales de producto.
- para wiring tests de `cmd/*`, validar output observable del handler/runner elimina falsos positivos de smoke tests.
