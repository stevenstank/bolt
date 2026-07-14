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
      ┌────────┼─────────┐
      ▼        ▼         ▼
   Memory  Persistence Replication
```

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
    server/
    storage/

docs/
    PRD.md
    SSOT.md
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
    expire/
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
- Client connections
- One goroutine per accepted client connection
- Connection lifecycle logging
- Graceful shutdown

Must never manipulate storage directly.

Current Stage 3 status:

- The server accepts TCP connections.
- The listen address is configurable.
- Active client connections are closed during shutdown.
- Each client connection is handled by one goroutine.
- The server continuously reads newline-delimited command lines.
- The server delegates command processing through a `Processor` interface.
- The server writes one newline-terminated response per command.
- The server never manipulates storage directly.

---

## Protocol

Responsible for:

- Parsing requests
- Encoding responses
- Protocol validation

Current:

- Plain text protocol.
- One command per line.
- Whitespace separates command name and arguments.
- Command names are case-insensitive.
- Supported commands are `SET <key> <value>` and `GET <key>`.

Later:

RESP

---

## Command Dispatcher

Responsible for:

- Command validation
- Command routing
- Calling the engine
- Plain-text response selection

Current:

- `SET` returns `OK`.
- `GET` returns the stored value.
- Missing `GET` returns `(nil)`.
- Invalid commands return `ERR ...`.

## Engine

Responsible for:

- Coordinating protocol parsing and command dispatch
- Providing the server-facing command processor interface
- Converting malformed input into error responses

---

## Storage Engine

Responsible for:

- Reads
- Writes
- Deletes
- Locking
- Expiration

Storage must never know about networking.

Current Stage 3 status:

- `NewStore` creates an in-memory-only store.
- `NewPersistentStore` creates a store backed by an Append Only File.
- `NewDurableStore` creates a store backed by AOF and snapshot files.
- Durable startup loads the snapshot first and replays the AOF second.
- `SaveSnapshot` writes a point-in-time copy of current data.
- Persistent writes are appended before the in-memory map is updated.
- Startup replay returns an error for complete corrupt AOF records.
- Startup replay recovers from incomplete trailing AOF records by truncating the partial tail.

---

## Persistence

Responsible for:

- Append Only File
- Snapshot creation
- Snapshot loading
- Crash recovery

Current Stage 3 status:

- `internal/persistence` owns all on-disk record formatting and parsing.
- The AOF records SET operations in a deterministic length-prefixed text format.
- Snapshot files use the same SET record representation and write keys in sorted order.
- Persistence has no dependency on networking or server lifecycle code.
- Default files are `bolt.aof` and `bolt.snapshot`.

---

## Replication

Responsible for:

- Replica synchronization
- Streaming updates
- Heartbeats

---

## Pub/Sub

Responsible for:

- Channels
- Subscribers
- Message broadcasting

---

## Transactions

Responsible for:

- Queuing commands
- Atomic execution

---

## Cluster

Responsible for:

- Node metadata
- Request routing
- Cluster communication

---

# Request Flow

```
Client

↓

TCP Server

↓

Parser

↓

Dispatcher

↓

Engine

↓

Storage Engine

↓

Persistence

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

Networking never modifies storage directly.

All writes pass through the storage engine.

---

# Concurrency Rules

- One goroutine per client.
- Shared state must be synchronized.
- Avoid unnecessary global state.
- Storage must be thread-safe.
- Favor simple locking strategies.

---

# Error Handling

- Never panic because of client input.
- Return meaningful errors.
- Log internal failures.
- Invalid commands must never crash the server.

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
- No flaky tests

---

# Development Rules

- Build one phase at a time.
- Finish tests before moving forward.
- Update the SSOT whenever architecture changes.
- Avoid premature optimization.
- Keep the codebase modular.

---

# Development Phases

| Phase | Goal |
|--------|------|
| 1 | Core in-memory database |
| 2 | TCP server & networking |
| 3 | Persistence & expiration |
| 4 | Replication |
| 5 | Distributed Bolt |

---

# Long-Term Vision

Bolt should evolve from a simple in-memory key-value store into a production-inspired distributed database while remaining educational, maintainable, and easy to understand.
