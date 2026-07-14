# Bolt

Bolt is a Redis-inspired in-memory key-value database written in Go.

The goal of this project is to learn how modern databases are built from first principles. Bolt starts as a simple in-memory key-value store and will gradually evolve to include networking, persistence, replication, and distributed systems concepts.

## Current Status

Bolt has completed **Stage 3 – Persistence**.

Implemented so far:

- Storage engine
- SET
- GET
- TCP server
- Configurable listen address
- Concurrent client connections
- Connection logging
- Clean shutdown
- Plain-text command parsing
- Command dispatch
- Storage integration over TCP
- Append Only File (AOF) persistence primitive
- Persistent store recovery from AOF
- Snapshot save/load primitives
- AOF recovery from incomplete trailing writes
- Configurable AOF and snapshot file paths

Coming next:

- Expiration support
- Replication

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
│   ├── server/
│   └── storage/
├── README.md
└── go.mod
```

## Build

```bash
go build ./...
```

## Run

Start the Bolt server:

```bash
go run ./cmd/bolt -addr 127.0.0.1:6380
```

You should see:

```text
server listening on 127.0.0.1:6380
```

By default Bolt writes persistence files in the current working directory:

- `bolt.aof`
- `bolt.snapshot`

You can override them:

```bash
go run ./cmd/bolt \
  -addr 127.0.0.1:6380 \
  -aof /tmp/bolt.aof \
  -snapshot /tmp/bolt.snapshot
```

### Connect to the server

Open a new terminal and connect using `netcat`:

```bash
nc 127.0.0.1 6380
```

If the connection succeeds, the terminal will wait for input and the server will log the new client connection.

Run commands:

```text
SET name saksham
OK
GET name
saksham
GET missing
(nil)
```

Supported commands:

- `SET <key> <value>`
- `GET <key>`

Invalid commands return an `ERR ...` response and the connection remains open.

To disconnect, press **Ctrl+C** or **Ctrl+D** in the client terminal.

> **Note:** Bolt defaults to `127.0.0.1:6379`, which is also Redis's default port. If Redis is running on your machine, use another port such as `6380`.


## Test

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
- `ARCHITECTURE.md` – Current package boundaries and request flow
- `adr/` – Architectural decision records

## Roadmap

### Stage 1
Core in-memory database

### Stage 2
Networking

### Stage 3
Persistence

### Stage 4
Replication

### Stage 5
Distributed Bolt

## Development

Bolt is being built incrementally using Test-Driven Development (TDD). Every feature starts with tests before implementation.

Each phase focuses on one major concept so the codebase stays small, readable, and easy to understand.
