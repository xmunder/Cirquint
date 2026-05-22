# ADR-004 — Pipeline asíncrono con Redis y workers Go

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

La extracción IA y la generación derivada pueden tardar más que un request HTTP normal. Además, el sistema necesita retries, control de estados y capacidad de reproceso por etapa.

## Decisión

Se adopta un pipeline asíncrono con:

- API en Go para orquestación
- `Redis` para colas, locks, deduplicación y retries
- workers en `Go` para ejecutar etapas del pipeline

## Razonamiento

- evita bloquear requests de usuario
- permite reintentos controlados
- hace visible el estado del procesamiento
- facilita ejecutar etapas independientes

## Consecuencias

### Positivas

- mejor UX
- resiliencia frente a fallos temporales
- soporte natural para estados como `queued`, `extracting`, `needs_review`, `ready`

### Negativas

- más complejidad que un request síncrono simple
- exige idempotencia y observabilidad mínimas

## Regla de diseño

Cada job debe:

- leer entradas explícitas
- producir salidas persistibles
- ser reintentable sin corrupción
- actualizar estado de forma atómica
