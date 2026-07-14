# Bolt — Product Requirements Document (PRD)

**Project:** Bolt

**Language:** Go

**Type:** Redis-inspired in-memory key-value database

**Status:** Living Document

---

# Vision

Bolt is a Redis-inspired in-memory key-value database written in Go.

The project starts as a simple in-memory key-value store and gradually evolves into a production-inspired distributed database to explore storage engines, networking, persistence, concurrency, replication, and distributed systems.

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

- EXPIRE
- TTL
- PERSIST

### Persistence

- Append Only File (AOF)
- Snapshotting

### Replication

- Primary → Replica replication
- Replica synchronization

### Communication

- Pub/Sub

### Transactions

- MULTI
- EXEC

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

# Out of Scope (Initially)

- SQL
- Authentication
- ACLs
- REST API
- Web Dashboard
- Raft Consensus
- Multi-region replication

---

# Technology

Language

- Go

Networking

- TCP

Storage

- Memory
- Append Only File
- Snapshots

Testing

- Go testing package

---

# Development Roadmap

## Phase 1 — Core Database

Build an in-memory key-value database with thread-safe storage and core commands.

---

## Phase 2 — Networking

Turn Bolt into a TCP server capable of serving multiple concurrent clients.

---

## Phase 3 — Durability

Implement expiration, append-only persistence, snapshots, and crash recovery.

Completed Stage 3 scope:

- Append Only File (AOF) records for SET operations
- AOF replay into the in-memory store on startup
- AOF crash recovery for incomplete trailing records
- Deterministic snapshot save/load primitives
- Server configuration for AOF and snapshot file paths
- Plain-text TCP command parsing for SET and GET
- Command dispatch through the engine and storage layers
- Manual TCP usage through Netcat

Deferred beyond Stage 3:

- Expiration support

---

## Phase 4 — Replication

Implement primary/replica replication with synchronization and heartbeats.

---

## Phase 5 — Distributed Bolt

Implement pub/sub, transactions, clustering basics, and production-inspired distributed behavior.

---

# Success Criteria

By the end of the project Bolt should:

- Handle multiple concurrent clients
- Persist data across restarts
- Replicate data between nodes
- Support pub/sub
- Support transactions
- Demonstrate production-inspired database architecture
