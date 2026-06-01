# Tasks: Circuit Ingestion Pipeline MVP

## Review Workload Forecast

| Campo | Valor |
|---|---|
| Estimated changed lines | 900-1400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1 base + PR2 ingestion + PR3 worker/review |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|---|---|---|---|
| 1 | Scaffold backend MVP y proyecto mínimo | PR 1 | Base para API, migraciones y contratos |
| 2 | Upload a R2 + job `queued` + polling | PR 2 | Depende de PR1; cierra ingestion end-to-end |
| 3 | Worker, provider mock, revisión y idempotencia | PR 3 | Depende de PR2; completa el gate `needs_review` |

## Phase 1: Base MVP

- [x] 1.1 Crear `backend/cmd/api/main.go` y `backend/internal/workspace/` con `POST /workspaces` y `POST /workspaces/{workspaceID}/projects` para cumplir `workspace-projects/spec.md`.
- [x] 1.2 Crear `backend/migrations/` para `workspaces`, `projects`, `uploads`, `processing_jobs`, `extraction_revisions` y `circuit_revisions` con claves de ownership y unicidad por job/artifact.
- [x] 1.3 Agregar tests de contrato/golden para creación mínima de proyecto y rechazo de upload sin `project_id` válido.

## Phase 2: Ingestion Vertical Slice

- [x] 2.1 Crear `backend/internal/storage/` con `ObjectStorage` y adapter R2 que emita keys `workspaces/{workspaceId}/...` según `design.md`.
- [x] 2.2 Crear `backend/internal/uploads/` para validar imagen, persistir metadata, subir binario y crear job con estados `created -> uploaded -> queued`.
- [x] 2.3 Exponer `POST /projects/{projectID}/uploads` y `GET /jobs/{jobID}` en `backend/cmd/api/main.go` con payloads alineados a `circuit-image-ingestion/spec.md`.
- [x] 2.4 Escribir tests unitarios para generación de keys/estados y tests de integración con fakes para asegurar que falla de DB o storage no deja jobs `queued` huérfanos.

## Phase 3: Extraction and Review Slice

- [x] 3.1 Crear `backend/cmd/worker/main.go`, `backend/internal/processing/` y cola Redis para reclamar solo jobs `queued` y pasar a `extracting` con compare-and-swap.
- [x] 3.2 Crear `backend/internal/provider/` con `CircuitExtractionProvider` mockeable y una implementación inicial detrás de la interfaz.
- [x] 3.3 Crear `backend/internal/circuit/` para persistir `ExtractionResult`, normalizar `CircuitSpec` y decidir `ready` o `needs_review` por `confidence` y warnings.
- [x] 3.4 Agregar tests de tabla para review policy e idempotencia, más un test de integración worker->persistencia que cubra `ready`, `needs_review` y `failed`.

## Phase 4: Verification and Scope Guard

- [x] 4.1 Verificar que ningún flujo genere `AssemblyPlan`, `SceneSpec` ni payloads de viewer; dejar assertion explícita en tests de `circuit-extraction-review/spec.md`.
- [x] 4.2 Documentar configuración mínima y threshold pendiente en `backend/README.md` o equivalente, dejando el valor exacto como decisión abierta del MVP.
