# ADR 0001: Plain-Text Replication Protocol

## Status

Accepted

## Context

Bolt Stage 4 needs primary-to-replica replication that is easy to understand, easy to test, and consistent with the project’s educational goals.

The replication path must preserve package boundaries:

- The TCP server accepts connections.
- The engine handles writes.
- Storage owns the in-memory data.
- Replication observes successful writes and streams them to replicas.

The protocol must also preserve values with spaces without introducing a binary framing format or RESP.

## Decision

Bolt uses a plain-text replication stream with these records:

- `SNAPSHOT BEGIN`
- length-prefixed `SET` records
- `SNAPSHOT END`
- `PING`
- `PONG`

`SET` records use the same length-safe text shape as persistence records so keys and values can contain spaces without ambiguity.

Primary nodes send a snapshot first, then stream live writes after the replica connection is established.

Replicas reconnect automatically when the primary connection drops.

## Consequences

- The protocol stays readable and simple.
- Replication can reuse the same data modeling approach as persistence.
- Replicas can bootstrap from a consistent snapshot before applying live writes.
- The implementation is not RESP-compatible, which is intentional for Stage 4.

## Related

- `README.md`
- `docs/PRD.md`
- `docs/SSOT.md`
- `docs/ARCHITECTURE.md`