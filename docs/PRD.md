# Bolt — Product Requirements Document (PRD)

**Project:** Bolt

**Language:** Go

**Type:** Redis-inspired in-memory key-value database

**Status:** Living Document

---

# Vision

Bolt is a Redis-inspired in-memory key-value database written in Go.

The project starts as a simple in-memory key-value store and gradually evolves into a production-inspired distributed database to explore storage engines, networking, persistence, concurrency, replication, transactions, and distributed systems.

The objective is to understand how modern databases are built from first principles—not to recreate Redis feature-for-feature.

---

# Objectives

## Primary Goals

- Learn database internals
- Learn concurrent programming in Go
- Learn TCP networking
- Learn storage engine design
- Learn persistence
- Learn replication
- Learn distributed systems
- Build production-quality Go code

---

# Functional Requirements

Bolt should eventually support:

### Core Operations

- SET
- GET
- DEL
- EXISTS
- KEYS

### Numeric Operations

- INCR
- DECR

### Expiration

- TTL support

### Persistence

- Append Only File (AOF)
- Snapshotting

### Replication

- Primary → Replica replication
- Initial replica synchronization
- Streaming replication

### Communication

- Pub/Sub

### Transactions

- MULTI
- EXEC
- DISCARD

### Runtime Introspection

- INFO

### Distribution

- Basic clustering
- Multiple nodes

---

# Non-Functional Requirements

Bolt should be:

- Fast
- Concurrent
- Thread-safe
- Deterministic
- Modular
- Well-tested
- Production-inspired

---

# Out of Scope

- SQL
- Authentication
- ACLs
- REST API
- Web Dashboard
- Raft Consensus
- Automatic failover
- Multi-region replication

---

# Technology

## Language

- Go

## Networking

- TCP

## Storage

- Memory
- Append Only File
- Snapshots

## Testing

- Go testing package

---

# Development Roadmap

## Stage 1 — Core Database

Build an in-memory key-value database with thread-safe storage and basic key-value operations.

Completed:

- Thread-safe storage engine
- SET
- GET

---

## Stage 2 — Networking

Turn Bolt into a TCP database server.

Completed:

- TCP server
- Plain-text protocol
- Concurrent client handling
- Configurable listen address
- Connection lifecycle management

---

## Stage 3 — Persistence

Implement durability and crash recovery.

Completed:

- Append Only File (AOF)
- AOF replay during startup
- Recovery from incomplete AOF writes
- Snapshot save/load
- Configurable persistence paths
- Graceful shutdown persistence

---

## Stage 4 — Replication

Implement primary/replica replication.

Completed:

- Primary accepts replica connections
- Replica connects using `-replicaof`
- Initial snapshot synchronization
- Live write replication
- Replica applies replicated writes
- Read-only replica mode

---

## Stage 5 — Distributed Bolt

Implement higher-level database features inspired by production systems.

Completed:

- TTL (key expiration)
- Pub/Sub
- Transactions (`MULTI`, `EXEC`, `DISCARD`)
- Runtime metrics (`INFO`)
- Per-client transaction isolation
- Graceful server shutdown
- Replication runtime metadata

---

# Success Criteria

By the end of the project Bolt should:

- Handle multiple concurrent clients
- Persist data across restarts
- Replicate data between nodes
- Support key expiration
- Support Pub/Sub messaging
- Support transactions
- Expose runtime server metrics
- Demonstrate production-inspired database architecture
- Maintain comprehensive automated test coverage