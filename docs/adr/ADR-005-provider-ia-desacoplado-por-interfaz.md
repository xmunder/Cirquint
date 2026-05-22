# ADR-005 — Provider IA desacoplado por interfaz

- **Estado:** Aprobado
- **Fecha:** 2026-05-22

## Contexto

La calidad de extracción puede variar entre providers. El MVP probablemente use Gemini, pero el producto necesita comparar alternativas y cambiar de provider sin reescribir el dominio.

## Decisión

La extracción IA se consumirá mediante una interfaz estable, por ejemplo:

```go
type CircuitExtractionProvider interface {
    ExtractCircuit(ctx context.Context, input ExtractionInput) (ExtractionResult, error)
}
```

Implementaciones iniciales o futuras:

- `GeminiProvider`
- `OpenAIProvider`
- `MockProvider`
- `PythonWorkerProvider` si más adelante hace falta

## Razonamiento

- evita acoplar prompts y respuestas crudas al dominio
- facilita testing y mocks
- habilita benchmark entre providers
- permite introducir Python como detalle de infraestructura, no como decisión estructural del core

## Consecuencias

### Positivas

- flexibilidad tecnológica
- mejor testabilidad
- menor lock-in con un vendor IA

### Negativas

- requiere normalización cuidadosa de outputs
- agrega una capa más de adaptación
