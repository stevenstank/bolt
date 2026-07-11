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
- Request handling
- Response writing

Must never manipulate storage directly.

---

## Protocol

Responsible for:

- Parsing requests
- Encoding responses
- Protocol validation

Initially:

Plain text protocol

Later:

RESP

---

## Command Dispatcher

Responsible for:

- Command validation
- Command routing
- Calling the storage engine

---

## Storage Engine

Responsible for:

- Reads
- Writes
- Deletes
- Locking
- Expiration

Storage must never know about networking.

---

## Persistence

Responsible for:

- Append Only File
- Snapshot creation
- Snapshot loading
- Crash recovery

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

Storage Engine

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