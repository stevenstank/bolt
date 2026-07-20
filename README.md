# Bolt

Bolt is a Redis-inspired in-memory key-value database written in Go.

The goal of this project is to learn how modern databases are built from first principles. Bolt starts as a simple in-memory key-value store and gradually evolves to include networking, persistence, replication, transactions, and distributed systems concepts.

## Current Status

Bolt has completed **Stage 5 – Distributed Bolt**.

Implemented so far:

- Storage engine
- SET
- GET
- TCP server
- Configurable listen address
- Concurrent client connections
- Connection logging
- Graceful shutdown
- Plain-text command parsing
- Command dispatch
- Storage integration over TCP
- Append Only File (AOF) persistence
- Persistent store recovery from AOF
- Snapshot save/load
- Recovery from incomplete AOF writes
- Configurable AOF and snapshot file paths
- Primary/replica replication
- Replica auto-connect to a primary
- Initial snapshot synchronization
- Streaming live writes to replicas
- Replica read-only mode
- TTL (key expiration)
- Pub/Sub messaging
- Transactions (`MULTI`, `EXEC`, `DISCARD`)
- Runtime server metrics (`INFO`)
- Per-client transaction isolation

## Project Structure

```text
bolt/
├── cmd/
│   └── bolt/
├── docs/
│   ├── PRD.md
│   ├── SSOT.md
│   ├── ARCHITECTURE.md
│   └── adr/
├── internal/
│   ├── command/
│   ├── engine/
│   ├── persistence/
│   ├── protocol/
│   ├── pubsub/
│   ├── replication/
│   ├── server/
│   ├── storage/
│   └── transaction/
├── README.md
└── go.mod
```

## Build

```bash
go build ./...
```

## Run

Start the primary server:

```bash
go run ./cmd/bolt -addr 127.0.0.1:6380
```

Start a replica:

```bash
go run ./cmd/bolt \
  -addr 127.0.0.1:6381 \
  -replicaof 127.0.0.1:6380
```

By default Bolt stores persistence files in the current working directory:

- `bolt.aof`
- `bolt.snapshot`

Override them if needed:

```bash
go run ./cmd/bolt \
  -addr 127.0.0.1:6380 \
  -aof /tmp/bolt.aof \
  -snapshot /tmp/bolt.snapshot
```

## Connect

Open another terminal:

```bash
nc 127.0.0.1 6380
```

Example session:

```text
SET name saksham
OK

GET name
saksham

SET session abc123 EX 60
OK

INFO
node_id: 0a4aa3d53da799c0
role: primary
uptime: 25
connected_clients: 1
replication_status: waiting
connected_replicas: 0
key_count: 2
memory_usage: 22
```

## Supported Commands

### Key-Value

- `SET <key> <value>`
- `SET <key> <value> EX <seconds>`
- `GET <key>`

### Transactions

- `MULTI`
- `EXEC`
- `DISCARD`

### Pub/Sub

- `SUBSCRIBE <channel>`
- `UNSUBSCRIBE <channel>`
- `PUBLISH <channel> <message>`

### Server

- `INFO`

Replica nodes remain read-only. Any write command sent directly to a replica returns an error, while replicated updates from the primary continue to apply normally.

## Test

Run all tests:

```bash
go test ./...
```

Run with the race detector:

```bash
go test -race ./...
```

## Documentation

Project documentation lives in the `docs/` directory.

- `PRD.md` – Product requirements and roadmap
- `SSOT.md` – Architecture and engineering decisions
- `ARCHITECTURE.md` – Package structure and request flow
- `adr/` – Architectural decision records

## Roadmap

- ✅ Stage 1 – Core in-memory database
- ✅ Stage 2 – Networking
- ✅ Stage 3 – Persistence
- ✅ Stage 4 – Replication
- ✅ Stage 5 – Distributed Bolt

## Development

Bolt is built incrementally using Test-Driven Development (TDD). Every feature begins with tests before implementation, with each stage introducing a major database concept while keeping the codebase modular and easy to understand.