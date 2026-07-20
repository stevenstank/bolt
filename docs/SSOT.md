# Bolt — Single Source of Truth (SSOT)

**Project:** Bolt

**Status:** Living Document

---

# Purpose

This document defines Bolt's architecture, conventions, terminology, and engineering decisions.

Whenever a major architectural decision changes, this document should be updated before implementation.

---

# Philosophy

- Learn by building.
- Simplicity before optimization.
- Incremental development.
- Test every feature.
- Keep responsibilities separated.
- Build production-inspired software.

---

# High-Level Architecture

```
            Client
               │
               ▼
          TCP Server
               │
               ▼
       Protocol Parser
               │
               ▼
     Command Dispatcher
               │
               ▼
        Storage Engine
      ┌──────┼─────────┬────────────┐
      ▼      ▼         ▼            ▼
   Memory Persistence Replication Transactions
               │
               ▼
            Pub/Sub
```

Replication observes successful writes from the engine and streams them to replicas without bypassing the server, command, or engine layers.

Replica clients are read-only. The engine rejects client writes on replicas while still accepting trusted replicated writes from the primary.

---

# Project Structure

Current implemented structure:

```
bolt/

cmd/
    bolt/

internal/
    command/
    engine/
    persistence/
    protocol/
    pubsub/
    replication/
    server/
    storage/
    transaction/

docs/
    PRD.md
    SSOT.md
    ARCHITECTURE.md
    adr/
```

Expected long-term structure:

```
bolt/

cmd/
    bolt/
    bolt-cli/

internal/
    server/
    protocol/
    command/
    engine/
    storage/
    persistence/
    replication/
    pubsub/
    transaction/
    cluster/

pkg/

test/

docs/
    PRD.md
    SSOT.md
    ARCHITECTURE.md
    adr/
```

---

# Responsibilities

## Server

Responsible for:

- TCP listener
- Client connection lifecycle
- One goroutine per client
- Graceful shutdown
- Processor creation
- Replica connection handoff

Must never manipulate storage directly.

Current Stage 5 status:

- Accepts TCP connections.
- Supports configurable listen addresses.
- Creates an isolated processor per client.
- Delivers Pub/Sub messages asynchronously.
- Hands replica connections to the replication subsystem.
- Closes active client connections during shutdown.
- Never accesses storage directly.

---

## Protocol

Responsible for:

- Parsing requests
- Encoding responses
- Protocol validation

Current:

- Plain-text protocol
- One command per line
- Whitespace-delimited arguments
- Case-insensitive commands

Supported commands:

- SET
- GET
- INFO
- MULTI
- EXEC
- DISCARD
- SUBSCRIBE
- UNSUBSCRIBE
- PUBLISH

Replication uses a plain-text stream with:

- SNAPSHOT BEGIN
- SNAPSHOT END
- Length-prefixed SET records

Future:

- RESP protocol

---

## Command Dispatcher

Responsible for:

- Command validation
- Command routing
- Transaction management
- Pub/Sub routing
- Runtime information
- Response generation

Current Stage 5 status:

- Handles key-value commands.
- Handles transactions.
- Handles Pub/Sub commands.
- Produces INFO responses.
- Delegates storage operations to the engine.

---

## Engine

Responsible for:

- Coordinating reads and writes
- Managing TTL-aware operations
- Rejecting client writes on replicas
- Accepting trusted replicated writes
- Notifying replication observers
- Providing runtime statistics

The engine owns the application's business logic and is the only component allowed to manipulate storage.

---

## Storage Engine

Responsible for:

- Reads
- Writes
- Deletes
- Locking
- Expiration metadata

Storage never knows about networking.

Current Stage 5 status:

- Thread-safe in-memory key-value storage.
- Optional expiration timestamps.
- Snapshot support.
- AOF-backed durability.
- Snapshot export for replication.
- Key count reporting.
- Memory usage estimation.

---

## Persistence

Responsible for:

- Append Only File
- Snapshot creation
- Snapshot loading
- Crash recovery

Current Stage 5 status:

- Owns all on-disk serialization.
- Uses deterministic length-prefixed records.
- Persists expiration timestamps.
- Supports snapshot loading.
- Replays AOF during startup.
- Recovers from incomplete trailing AOF writes.
- Default files are `bolt.aof` and `bolt.snapshot`.

---

## Replication

Responsible for:

- Replica synchronization
- Initial snapshot transfer
- Streaming primary writes
- Replica read-only mode

Current Stage 5 status:

- Primary accepts replica connections.
- Replica synchronizes from a snapshot.
- Successful writes are streamed to replicas.
- Replicated writes preserve expiration metadata.
- Replication integrates through the engine.
- Replica clients are read-only.

---

## Pub/Sub

Responsible for:

- Channel management
- Subscriber management
- Message broadcasting
- Subscriber cleanup

Current Stage 5 status:

- Multiple subscribers per channel.
- Broadcasts messages to all subscribers.
- Subscribers are isolated per client.
- Resources are cleaned up during disconnect and shutdown.

---

## Transactions

Responsible for:

- Per-client transaction state
- Command queuing
- Atomic execution

Current Stage 5 status:

- Supports MULTI.
- Queues commands.
- Executes queued commands using EXEC.
- Clears queued commands using DISCARD.
- Transaction state is isolated per client.

---

## Cluster

Responsible for:

- Node metadata
- Request routing
- Cluster communication

Status:

Not yet implemented.

---

# Request Flow

```
Client

↓

TCP Server

↓

Protocol Parser

↓

Command Dispatcher

↓

Engine

├── Storage

├── Persistence

├── Replication

├── Transactions

└── Pub/Sub

↓

Response

↓

Client
```

---

# Storage Model

Primary storage is an in-memory hash map.

Persistence is layered on top through:

- Append Only File
- Snapshots

All writes pass through the engine.

Networking never manipulates storage directly.

Replication never bypasses the engine.

---

# Concurrency Rules

- One goroutine per client connection.
- One processor instance per client.
- Transaction state is isolated per client.
- Shared state must be synchronized.
- Storage must be thread-safe.
- Pub/Sub delivery occurs asynchronously.
- Favor simple locking strategies.

---

# Error Handling

- Never panic because of client input.
- Return meaningful errors.
- Log internal failures.
- Invalid commands must never crash the server.
- Replicas reject client write commands.

---

# Logging

Development:

- Human-readable logs
- Server startup
- Client connect/disconnect events

Future:

- Structured logging

---

# Testing Strategy

Every package owns its own tests.

Requirements:

- Table-driven tests
- Unit tests first
- Deterministic behavior
- Race-free concurrency
- No flaky tests

---

# Development Rules

- Build one stage at a time.
- Finish tests before moving forward.
- Update the SSOT whenever architecture changes.
- Avoid premature optimization.
- Keep the codebase modular.

---

# Development Stages

| Stage | Goal | Status |
|--------|------|--------|
| 1 | Core in-memory database | ✅ |
| 2 | TCP networking | ✅ |
| 3 | Persistence | ✅ |
| 4 | Replication | ✅ |
| 5 | Distributed Bolt | ✅ |

---

# Long-Term Vision

Bolt should evolve from a simple in-memory key-value store into a production-inspired distributed database while remaining educational, maintainable, and easy to understand.

Future work may include:

- Additional data structures
- More Redis-compatible commands
- RESP protocol
- Clustering
- Automatic failover
- Leader election
- Sharding
- Improved observability