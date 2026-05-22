# Proposal: Circuit Ingestion Pipeline MVP

## Intent

Establecer el primer slice end-to-end real: proyecto mínimo, upload de imagen, job async, extracción desacoplada, persistencia de revisión y gate de `needs_review`.

## Scope

### In Scope
- Crear un concepto mínimo de `workspace/project` para ownership de uploads, jobs y revisiones.
- Subir imagen del esquema, persistir metadata en PostgreSQL y binario en R2 vía adapter backend-owned.
- Crear job async con estados visibles (`created`, `uploaded`, `queued`, `extracting`, `needs_review`, `ready`, `failed`).
- Definir `CircuitExtractionProvider` mockeable y ejecutar una implementación inicial detrás de esa interfaz.
- Persistir `ExtractionResult` y una primera revisión normalizada de `CircuitSpec`.
- Marcar `needs_review` cuando `confidence` quede bajo umbral o existan warnings relevantes.

### Out of Scope
- `AssemblyPlan`, `SceneSpec`, viewer 3D y reproducción paso a paso.
- Edición manual completa del circuito; solo queda preparado el estado y datos para revisión.
- Multi-workspace avanzado, auth completa, soporte multi-provider real y Python workers.

## Capabilities

### New Capabilities
- `workspace-projects`: ownership mínimo de proyectos y revisiones para iniciar el pipeline canónico.
- `circuit-image-ingestion`: upload a R2, metadata persistida, job async y tracking de estado.
- `circuit-extraction-review`: extracción vía interfaz, persistencia de `ExtractionResult`/`CircuitSpec` y gate de `needs_review`.

### Modified Capabilities
- None.

## Approach

Slice backend-first alineado a ADR-001..006: monolito modular Go, `image -> ExtractionResult -> CircuitSpec`, R2 detrás de `ObjectStorage`, Redis + worker Go para extracción, y revisión manual como guardrail de calidad. El frontend posterior solo consume APIs y estado; no define contratos.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `openspec/specs/workspace-projects/` | New | Spec del modelo mínimo de workspace/project |
| `openspec/specs/circuit-image-ingestion/` | New | Spec de upload, storage y job tracking |
| `openspec/specs/circuit-extraction-review/` | New | Spec de extracción, revisión y persistencia de revisión |
| `backend/internal/{workspace,uploads,processing,circuit,provider,storage}/` | New | Límites modulares del slice MVP |
| `backend/cmd/{api,worker}/` | New | Entrypoints de API y worker |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Modelo de revisión insuficiente | Med | Persistir warnings/confidence desde el día 1 |
| Reintentos corrupten revisiones | Med | Jobs idempotentes y updates atómicos |
| Scope creep hacia viewer/planning | High | Non-goals explícitos en spec y design |

## Rollback Plan

Apagar el endpoint/worker del pipeline, conservar uploads en R2 como artefactos pasivos y revertir tablas/módulos nuevos sin tocar ADRs ni docs base.

## Dependencies

- ADR-001 a ADR-006 aprobados
- PostgreSQL, Redis y Cloudflare R2 disponibles para el diseño posterior

## Success Criteria

- [ ] Existe un proyecto mínimo que puede recibir un upload de esquemático.
- [ ] Cada upload crea un job async con estado consultable.
- [ ] El worker persiste `ExtractionResult` y `CircuitSpec` revisionados.
- [ ] Baja confianza o warnings fuerzan `needs_review` en lugar de continuar silenciosamente.
