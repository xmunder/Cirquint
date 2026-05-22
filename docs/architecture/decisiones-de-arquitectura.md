# Decisiones de arquitectura

## Resumen ejecutivo

La arquitectura base de **Circuit Storys** será un **monolito modular** con:

- **Frontend:** Next.js
- **Backend principal:** Go
- **Workers:** Go
- **Storage de objetos:** Cloudflare R2
- **Base de datos:** PostgreSQL
- **Colas, locks y cache:** Redis
- **3D:** React Three Fiber + Three.js
- **IA:** providers desacoplados por adapter
- **Python worker:** opcional, solo si OCR/CV/experimentación IA lo exige

## Decisiones principales

| Área | Decisión |
|---|---|
| Estilo arquitectónico | Monolito modular |
| Flujo central | `imagen -> ExtractionResult -> CircuitSpec -> AssemblyPlan -> SceneSpec` |
| Lenguaje backend | Go |
| Lenguaje worker inicial | Go |
| Storage | Cloudflare R2 detrás de una interfaz de object storage |
| Persistencia | PostgreSQL para datos de negocio y revisiones |
| Procesamiento async | Redis + workers |
| Framework frontend | Next.js |
| Motor 3D | React Three Fiber sobre Three.js |
| Estrategia IA | Provider agnóstico; Gemini como candidato inicial |
| Estrategia de calidad | SDD + TDD estricto |

## Regla de arquitectura más importante

El sistema **no** debe saltar de una imagen directamente a una animación 3D.

La IA participa en la **extracción estructurada** del circuito. Luego el backend transforma ese resultado en modelos de dominio y finalmente en una escena visual.

## Modelos de dominio

### 1. `ExtractionResult`
Salida cruda del provider IA.

Contiene, por ejemplo:
- componentes detectados
- conexiones inferidas
- warnings
- confidence
- provider origen

### 2. `CircuitSpec`
Verdad eléctrica del circuito.

Contiene:
- componentes normalizados
- nets
- valores
- warnings
- confidence consolidada

### 3. `AssemblyPlan`
Verdad física del montaje.

Contiene:
- breadboard objetivo
- placements
- wires
- pasos de montaje

### 4. `SceneSpec`
Verdad visual consumida por el viewer.

Contiene:
- assets
- timeline
- posiciones
- highlights
- anotaciones

## Regla de separación

- `CircuitSpec` no conoce Three.js
- `AssemblyPlan` no depende del provider IA
- `SceneSpec` no contiene lógica eléctrica

## Bounded contexts

### `identity`
- usuarios
- autenticación
- ownership

### `workspace`
- workspaces
- proyectos
- revisiones

### `uploads`
- recepción de archivos
- validación
- metadata

### `processing`
- jobs
- reintentos
- estados
- orquestación del pipeline

### `circuit`
- normalización
- validación
- reglas del dominio eléctrico

### `assembly`
- generación del plan de montaje
- placement rules
- secuencia pedagógica

### `scene`
- generación de escena y timeline

### `provider.ai`
- adapters hacia Gemini/OpenAI/u otros

### `storage`
- adapter de object storage
- implementación inicial: R2

## Flujo principal

1. El usuario crea un proyecto.
2. Sube una imagen del esquemático.
3. El backend guarda metadata y archivo en R2.
4. Se crea un job de extracción.
5. El worker llama al provider IA.
6. Se genera un `ExtractionResult`.
7. El módulo `circuit` normaliza a `CircuitSpec`.
8. Si el circuito es válido, `assembly` genera `AssemblyPlan`.
9. `scene` genera `SceneSpec`.
10. El frontend reproduce el paso a paso en 3D.

## Manejo de ambigüedad

La IA puede fallar. Por eso el pipeline debe soportar:

- `confidence`
- `warnings`
- estado `needs_review`
- corrección manual
- reproceso desde `CircuitSpec`

## Estados sugeridos del pipeline

- `created`
- `uploaded`
- `queued`
- `extracting`
- `extracted`
- `needs_review`
- `validated`
- `planning`
- `scene_generating`
- `ready`
- `failed`

## Persistencia

### PostgreSQL
Guardar:
- users
- workspaces
- projects
- uploads
- processing_jobs
- circuit_revisions
- assembly_revisions
- scene_revisions

### Redis
Usar para:
- colas
- retries
- locks
- deduplicación
- estado transitorio

### Cloudflare R2
Guardar:
- imágenes originales
- previews
- outputs intermedios serializados
- escenas/exportaciones si aplica

## Convención de keys en R2

```txt
projects/{projectId}/uploads/{fileId}/original.png
projects/{projectId}/uploads/{fileId}/preview.webp
projects/{projectId}/extractions/{revisionId}/raw.json
projects/{projectId}/circuits/{revisionId}/circuit-spec.json
projects/{projectId}/assemblies/{revisionId}/assembly-plan.json
projects/{projectId}/scenes/{revisionId}/scene-spec.json
projects/{projectId}/exports/{exportId}/snapshot.png
```

## Estructura base sugerida

```txt
apps/
  web/

backend/
  cmd/
    api/
    worker/
  internal/
    identity/
    workspace/
    upload/
    processing/
    circuit/
    assembly/
    scene/
    provider/
    platform/
  migrations/

packages/
  contracts/
```

## Criterio de adopción de Python

Python **no** entra al stack inicial.

Solo se habilita si aparece una necesidad real de:
- OCR avanzado
- computer vision
- tooling multimodal mejor resuelto en Python
- experimentación IA que justifique el costo de un segundo runtime

## Estrategia de calidad

### SDD
Antes de implementar:
- proposal
- spec
- design
- tasks

### TDD
Aplicar TDD fuerte sobre:
- normalización del circuito
- validación
- planner
- scene generator
- state machine del pipeline

## Primer recorte de MVP

Soportar primero:
- 555
- resistencias
- capacitores
- switches
- LED
- jumpers
- breadboard estándar

## Próximos ADRs recomendados

- ADR-002 Separación entre `CircuitSpec`, `AssemblyPlan` y `SceneSpec`
- ADR-003 Cloudflare R2 como object storage inicial
- ADR-004 Pipeline asíncrono con Redis + workers
- ADR-005 Provider de IA desacoplado por interfaz
- ADR-006 Revisión manual ante baja confianza
