# Arquitectura Acordada de circuit-storys

Documento consolidado de decisiones arquitectónicas para `circuit-storys`. Resume la forma del sistema, sus límites y las reglas que NO se deben romper.

## Decisión Principal

`circuit-storys` se construye como un **monolito modular** con:

| Área | Tecnología inicial |
|-------|---------------------|
| Frontend | Next.js |
| Backend API | Go |
| Workers | Go |
| Base de datos | PostgreSQL |
| Cola, cache y locks | Redis |
| Object storage | Cloudflare R2 |
| 3D | React Three Fiber + Three.js |
| IA | Proveedores detrás de una interfaz adaptadora |

Proveedor inicial probable de IA: **Gemini**.

## Regla de Oro del Dominio

El flujo canónico del producto es:

`image -> extraction result -> CircuitSpec -> AssemblyPlan -> SceneSpec`

Reglas:

- Nunca saltar directamente de `image` a animación 3D.
- Cada artefacto intermedio debe ser persistible, inspeccionable y testeable.
- Cada transición debe poder re-ejecutarse sin reprocesar todo el pipeline.
- El frontend consume resultados del dominio; no inventa estructura eléctrica ni geométrica.

## Qué Hace el Producto

La aplicación web recibe imágenes de esquemas electrónicos, usa IA para inferir un circuito y luego genera una animación 3D paso a paso para montar el circuito en breadboard.

## Estilo Arquitectónico

### Monolito modular

Se adopta **monolito modular, no microservicios**.

Por qué:

- El dominio todavía está explorándose.
- El pipeline principal necesita iteración rápida entre extracción, validación, ensamblado y escena.
- Los límites del negocio están claros, pero el costo operacional de microservicios no se justifica al inicio.
- Go permite separar módulos con buena disciplina sin fragmentar despliegue, observabilidad y debugging.

Qué implica:

- Un despliegue principal para la app backend.
- Módulos internos con contratos explícitos.
- Workers separados por proceso, pero dentro de la misma arquitectura y base de código en Go.
- Redis y PostgreSQL compartidos como infraestructura común del sistema.

## Contextos Delimitados

Los contextos iniciales acordados son:

| Contexto | Responsabilidad |
|----------|------------------|
| `identity` | autenticación, autorización, actores y sesiones |
| `workspace` | espacios de trabajo, proyectos, pertenencia y colaboración |
| `uploads` | recepción, validación y trazabilidad de imágenes subidas |
| `processing` | orquestación del pipeline, jobs, estados y reintentos |
| `circuit` | modelo canónico del circuito inferido (`CircuitSpec`) |
| `assembly` | pasos de montaje sobre breadboard (`AssemblyPlan`) |
| `scene` | representación lista para renderizar y animar (`SceneSpec`) |
| `provider.ai` | adapters para LLM/OCR/CV y políticas de fallback |
| `storage` | acceso a R2 y manejo de objetos/binarios |

Regla de diseño:

- Cada contexto expone contratos claros.
- Ningún contexto debe leer o mutar internals de otro sin pasar por su API o servicio de aplicación.
- `processing` orquesta; no absorbe la lógica de `circuit`, `assembly` ni `scene`.

## Pipeline Canónico

### 1. Upload

- El usuario sube una imagen de esquema.
- `uploads` valida tipo, tamaño y metadatos mínimos.
- El archivo original se guarda en R2.
- Se registra una referencia persistente en PostgreSQL.

### 2. Extraction Result

- `processing` dispara un job.
- `provider.ai` invoca el proveedor de IA activo.
- El resultado bruto de extracción se guarda como `extraction result` versionado.
- Este artefacto conserva trazabilidad hacia imagen, proveedor, prompt/configuración y fecha.

### 3. CircuitSpec

- `circuit` transforma el resultado de extracción en un modelo canónico del circuito.
- `CircuitSpec` debe ser independiente del proveedor.
- Debe permitir validaciones estructurales y eléctricas básicas.

### 4. AssemblyPlan

- `assembly` deriva pasos de montaje desde `CircuitSpec`.
- `AssemblyPlan` define orden, componentes, posiciones esperadas y restricciones.
- Esta capa resuelve la lógica pedagógica y física del armado.

### 5. SceneSpec

- `scene` transforma `AssemblyPlan` en una especificación visual renderizable.
- `SceneSpec` describe entidades, transforms, highlights, cámara y secuencia temporal.
- El frontend 3D renderiza `SceneSpec`; no calcula el plan de ensamblado.

## Frontend

Frontend en **Next.js**.

Responsabilidades:

- autenticación y experiencia de usuario
- carga de imágenes
- seguimiento de estados de procesamiento
- visualización del circuito inferido
- reproducción de la animación 3D paso a paso

Stack 3D:

- `React Three Fiber`
- `Three.js`

Regla:

- La capa 3D es un renderer del dominio visual (`SceneSpec`), no el lugar donde se decide la lógica del circuito.

## Backend y Workers

Backend principal en **Go**.

Responsabilidades del backend:

- API para frontend
- coordinación de casos de uso
- persistencia en PostgreSQL
- publicación de trabajos a Redis
- lectura de estado de procesamiento

Workers en **Go**.

Responsabilidades de workers:

- ejecutar extracción con IA
- transformar artefactos intermedios
- regenerar etapas sin bloquear requests HTTP
- manejar reintentos y errores recuperables

Decisión adicional:

- **No** introducir worker en Python de entrada.
- Python queda como opción futura **solo** si OCR, CV o experimentación de IA realmente lo exigen.

## Persistencia

### PostgreSQL

Usar PostgreSQL para:

- usuarios y workspaces
- metadatos de uploads
- jobs y estados de procesamiento
- referencias a objetos en R2
- versiones de `extraction result`, `CircuitSpec`, `AssemblyPlan` y `SceneSpec`

### Redis

Usar Redis para:

- cola de jobs
- cache transitoria
- locks distribuidos
- coordinación de reintentos y deduplicación operativa

Redis NO es fuente de verdad del dominio.

### Cloudflare R2

Usar Cloudflare R2 como object storage inicial para:

- imágenes originales
- derivados pesados
- snapshots o artefactos grandes del pipeline cuando convenga

### Estrategia de keys en R2

Regla: keys predecibles, versionables y segmentadas por workspace y recurso.

Formato base sugerido:

`workspaces/{workspaceId}/{resource}/{entityId}/v{version}/{filename}`

Ejemplos:

- `workspaces/ws_123/uploads/up_456/v1/source.png`
- `workspaces/ws_123/extractions/ex_789/v1/result.json`
- `workspaces/ws_123/scenes/sc_321/v2/preview.glb`

Decisiones asociadas:

- Incluir `workspaceId` para aislamiento lógico temprano.
- Incluir `resource` para navegación y políticas de lifecycle.
- Incluir `entityId` para trazabilidad estable.
- Incluir `v{version}` para evitar overwrite implícito y soportar regeneración.
- Mantener `filename` expresivo para debugging y operaciones.

Reglas adicionales:

- Nunca usar nombres basados solo en timestamp.
- Nunca sobreescribir el original subido por el usuario.
- Los paths de R2 se guardan en PostgreSQL junto con tipo de artefacto, versión y checksum cuando aplique.

## Integración con IA

Los proveedores de IA deben vivir detrás de una **interfaz adaptadora**.

Objetivo:

- desacoplar prompts, SDKs y formatos propietarios del resto del dominio
- permitir cambio de proveedor sin contaminar `circuit`, `assembly` o `scene`
- habilitar pruebas con fakes y fixtures

Decisión inicial:

- proveedor probable inicial: `Gemini`
- no acoplar estructuras del dominio a respuestas nativas del proveedor

## Calidad y Proceso

Mentalidad obligatoria:

- **SDD estricto**
- **TDD estricto**

Implicancias:

- especificar antes de expandir comportamiento
- modelar artefactos del pipeline con contratos explícitos
- escribir tests por contexto y por transformación
- usar fixtures para imágenes, resultados de extracción y specs intermedios

## Límites que No Se Deben Romper

- No convertir el renderer 3D en motor de negocio.
- No dejar que el proveedor de IA defina el modelo del dominio.
- No saltar de imagen a animación sin artefactos intermedios.
- No usar microservicios prematuramente.
- No usar Redis como persistencia de negocio.
- No meter Python salvo necesidad real demostrable.

## Próximas Decisiones Recomendadas

- contrato JSON de `CircuitSpec`
- contrato JSON de `AssemblyPlan`
- contrato JSON de `SceneSpec`
- estrategia de jobs, reintentos e idempotencia en `processing`
- modelo de versionado de artefactos del pipeline
- autenticación y multi-tenant por workspace
