# ADR 0002: Engine-Enforced Read-Only Replicas

## Status

Accepted

## Context

Stage 4 replicas must reject client writes while still accepting updates streamed from the primary.

The restriction belongs in the engine layer so it stays independent of TCP handling and storage internals.

## Decision

Bolt marks replica engines as read-only for normal client writes.

The engine exposes a separate trusted apply path for replication traffic so primary writes can still be applied on replicas.

The TCP server and storage layer do not enforce the read-only rule.

## Consequences

- Client `SET` commands sent to replicas return a read-only error.
- Reads continue to work on replicas.
- Replication can still apply updates through the engine's trusted path.
- The architecture stays consistent with the existing client → server → command → engine → storage flow.

## Related

- `internal/engine/engine.go`
- `internal/replication/replication.go`
- `README.md`
- `docs/PRD.md`
- `docs/SSOT.md`
- `docs/ARCHITECTURE.md`