# ADR-003 — Cloudflare R2 como object storage inicial

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

El sistema necesita guardar imágenes originales, previews, artefactos serializados del pipeline y posibles exportaciones. AWS S3 no es la mejor opción inicial para MVP por costo y egress.

## Decisión

Se adopta **Cloudflare R2** como object storage inicial, detrás de una interfaz `ObjectStorage` agnóstica del provider.

## Razonamiento

- menor presión de costos para MVP
- egress más amigable
- modelo compatible con object storage tipo S3
- suficiente para uploads, revisiones y assets derivados

## Consecuencias

### Positivas

- costo inicial más bajo
- posibilidad de migrar de provider con menor fricción
- interfaz clara para testing y mocking

### Negativas

- dependencia de una abstracción bien diseñada
- posible necesidad de ajustes si se cambia a otro provider

## Convención de keys

```txt
projects/{projectId}/uploads/{fileId}/original.png
projects/{projectId}/uploads/{fileId}/preview.webp
projects/{projectId}/extractions/{revisionId}/raw.json
projects/{projectId}/circuits/{revisionId}/circuit-spec.json
projects/{projectId}/assemblies/{revisionId}/assembly-plan.json
projects/{projectId}/scenes/{revisionId}/scene-spec.json
projects/{projectId}/exports/{exportId}/snapshot.png
```

## Regla

La aplicación no debe acoplarse a APIs propietarias del storage fuera del adapter.
