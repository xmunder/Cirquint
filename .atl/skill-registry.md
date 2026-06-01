# Skill Registry

**Delegator use only.** This registry captures project conventions and SDD-adjacent docs for the cirquint backend scaffold.

## User Skills

None added yet. No project-specific sub-agent skills were detected in this workspace.

## Compact Rules

None.

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| ADR index | `docs/adr/README.md` | Lists ADR-001 through ADR-006 and links each decision file |
| ADR-001 | `docs/adr/ADR-001-monolito-modular-y-pipeline-canonico.md` | Canonical monolith/pipeline decision |
| ADR-002 | `docs/adr/ADR-002-separacion-circuit-spec-assembly-plan-scene-spec.md` | Domain artifact separation |
| ADR-003 | `docs/adr/ADR-003-cloudflare-r2-como-storage-inicial.md` | R2 storage decision |
| ADR-004 | `docs/adr/ADR-004-pipeline-asincrono-con-redis-y-workers-go.md` | Async pipeline with Redis + Go workers |
| ADR-005 | `docs/adr/ADR-005-provider-ia-desacoplado-por-interfaz.md` | AI provider adapter boundary |
| ADR-006 | `docs/adr/ADR-006-revision-manual-ante-baja-confianza.md` | Manual review on low-confidence extraction |
| Architecture summary | `docs/architecture/arquitectura-decisiones.md` | Primary architecture summary and constraints |
| Architecture summary | `docs/architecture/decisiones-de-arquitectura.md` | Alternate architecture summary with same core stack |
| OpenSpec config | `openspec/config.yaml` | SDD bootstrap config; contains Go test command discovery and strict TDD flag |
| SDD exploration | `docs/sdd/circuit-ingestion-pipeline-mvp/exploration.md` | Early exploration of the backend-first slice and pipeline risks |

## Notes

- No root-level `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.cursorrules`, or `copilot-instructions.md` was found.
- The repo now has a Go backend scaffold under `backend/` with API, worker, and internal packages.
- Testing style is Go-native: unit tests, `net/http/httptest` contract tests, golden files, and in-memory fakes.
