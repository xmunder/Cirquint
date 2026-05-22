# Apply Progress — circuit-ingestion-pipeline-mvp

- **Proyecto canónico:** `cirquint`
- **Cambio:** `circuit-ingestion-pipeline-mvp`
- **Estrategia de entrega:** `ask-on-risk`
- **Chain strategy:** `feature-branch-chain`
- **Estado actual:** `slice-1-completed`

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

### Slice 2 / PR2

- adapter real de `ObjectStorage`
- generación de keys para R2
- persistencia de metadata de uploads
- creación de `processing_jobs`
- estados `created -> uploaded -> queued`
- endpoint `GET /jobs/{jobID}`

## Notas

- `AssemblyPlan`, `SceneSpec`, viewer 3D y ejecución real de extracción IA siguen fuera de scope en este punto.
- Este archivo complementa el artifact en Engram y mantiene trazabilidad híbrida real.
