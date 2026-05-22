# ADR-006 — Revisión manual ante baja confianza de extracción

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

La IA puede devolver componentes ambiguos, conexiones dudosas o circuitos incompletos. El producto no puede asumir extracción perfecta si quiere producir montajes confiables.

## Decisión

Cuando la extracción no alcance el umbral de confianza o incluya warnings relevantes, el pipeline debe pasar a un estado `needs_review` y exigir corrección manual antes de continuar.

## Razonamiento

- protege la calidad del `CircuitSpec`
- evita generar montajes físicamente erróneos
- convierte a la IA en acelerador, no en fuente absoluta de verdad

## Consecuencias

### Positivas

- reduce errores silenciosos
- mejora trazabilidad
- permite reprocesar desde un artefacto corregido

### Negativas

- suma fricción al happy path
- exige UI de revisión y edición

## Reglas mínimas

- registrar `confidence`
- registrar `warnings`
- permitir edición de componentes y conexiones
- persistir la revisión antes de regenerar `AssemblyPlan` o `SceneSpec`
