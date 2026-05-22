# ADR-001 — Monolito modular con pipeline canónico

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

`circuit-storys` es una aplicación web que recibe imágenes de esquemas electrónicos, usa IA para inferir un circuito y luego genera una animación 3D paso a paso para montar el circuito en breadboard.

El sistema necesita:

- iterar rápido sobre un dominio todavía en consolidación
- mantener trazabilidad entre imagen, inferencia, circuito, plan de armado y escena 3D
- desacoplar providers IA del modelo del dominio
- soportar procesamiento asíncrono y reintentos
- evitar complejidad operacional temprana sin perder límites de dominio claros

## Decisión

Se adopta una arquitectura de **monolito modular** con:

- **Next.js** para frontend y experiencia de producto
- **Go** para backend principal
- **Go** para workers iniciales
- **PostgreSQL** como fuente de verdad de negocio
- **Redis** para colas, locks, cache y retries
- **Cloudflare R2** como object storage inicial
- **React Three Fiber + Three.js** para render 3D

El flujo obligatorio del dominio será:

`image -> ExtractionResult -> CircuitSpec -> AssemblyPlan -> SceneSpec`

Queda explícitamente prohibido saltar directo de la imagen a la animación final.

Los providers IA quedan detrás de una interfaz adaptadora. El provider inicial probable es `Gemini`.

No se introduce Python al inicio. Solo podrá agregarse un worker Python si experimentación real de OCR/CV/IA demuestra una necesidad concreta.

## Razonamiento

### Por qué monolito modular

Porque el MVP necesita:

- velocidad de iteración
- menor complejidad operativa
- límites de dominio claros
- una base compatible con TDD y refactor seguro

No se eligen microservicios en esta etapa porque agregan complejidad de despliegue, observabilidad y coordinación sin aportar valor proporcional al riesgo actual.

### Por qué pipeline canónico

Porque el riesgo principal del proyecto no es el throughput distribuido sino modelar bien la transformación desde un esquema visual hacia instrucciones de armado reproducibles.

Materializar artefactos intermedios permite:

- trazabilidad
- debugging
- persistencia de revisiones
- reproceso por etapa
- testing sobre contratos estables

### Por qué Go

Porque el núcleo del producto tiene una parte determinística fuerte:

- validación
- normalización
- planificación
- generación de escena
- state machine del pipeline

Go ofrece una base sólida para concurrencia, tipado fuerte, testing y operación simple.

## Consecuencias

### Positivas

- menor costo operativo que microservicios
- mejor velocidad de iteración sobre el pipeline principal
- artefactos intermedios testeables, persistibles y auditables
- menor acoplamiento entre IA, dominio y render 3D
- posibilidad de regenerar etapas sin rehacer todo el proceso

### Negativas

- exige disciplina fuerte de modularidad dentro del monolito
- puede crecer el acoplamiento si los contextos no respetan sus contratos
- requiere definir temprano contratos estables para `CircuitSpec`, `AssemblyPlan` y `SceneSpec`
- la extracción IA sigue siendo el principal riesgo técnico

## Alternativas consideradas

### Microservicios desde el inicio

Rechazada.

Motivo:

- agrega complejidad operativa demasiado pronto
- dificulta debugging transversal del pipeline
- no aporta suficiente valor en esta etapa

### Pipeline opaco `image -> animation`

Rechazada.

Motivo:

- reduce trazabilidad
- dificulta validación y testing
- acopla demasiado la IA con el resultado visual

### Backend principal en Python

Rechazada por ahora.

Motivo:

- aumenta superficie operativa y cognitiva
- Go calza mejor con el core determinístico y concurrente del sistema

## Notas de implementación

- Los contextos base son: `identity`, `workspace`, `uploads`, `processing`, `circuit`, `assembly`, `scene`, `provider.ai`, `storage`.
- R2 debe usar keys versionadas y segmentadas por proyecto y revisión.
- SDD y TDD son parte obligatoria del proceso de diseño e implementación.
