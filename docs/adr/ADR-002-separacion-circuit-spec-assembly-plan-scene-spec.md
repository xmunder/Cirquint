# ADR-002 — Separación entre CircuitSpec, AssemblyPlan y SceneSpec

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

El producto transforma una imagen ambigua en una animación 3D reproducible. Si el sistema mezcla extracción IA, reglas eléctricas, armado físico y render visual en un solo modelo, la evolución del producto se vuelve frágil.

## Decisión

Se definen tres contratos de dominio separados:

- `CircuitSpec`: verdad eléctrica del circuito
- `AssemblyPlan`: verdad física del montaje
- `SceneSpec`: verdad visual consumida por el viewer

Cada contrato tendrá responsabilidades explícitas y no compartirá lógica que pertenezca a otra capa.

## Razonamiento

- `CircuitSpec` debe ser validable sin depender del viewer
- `AssemblyPlan` debe poder recalcularse desde un circuito corregido
- `SceneSpec` debe poder regenerarse sin tocar la lógica eléctrica

Esta separación permite testing más fuerte, debugging por etapas y cambio de reglas sin acoplar todo el sistema.

## Consecuencias

### Positivas

- contratos más claros
- refactor seguro
- mejor trazabilidad
- mejor soporte para corrección manual y reproceso

### Negativas

- más artefactos y serialización
- exige disciplina para no filtrar detalles entre capas

## Reglas

- `CircuitSpec` no conoce Three.js
- `AssemblyPlan` no depende del provider IA
- `SceneSpec` no define verdad eléctrica
- el reproceso puede arrancar desde cualquiera de estos artefactos según el caso
