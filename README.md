# Bolt

Bolt is a Redis-inspired in-memory key-value database written in Go.

The goal of this project is to learn how modern databases are built from first principles. Bolt starts as a simple in-memory key-value store and will gradually evolve to include networking, persistence, replication, and distributed systems concepts.

## Current Status

Bolt is currently in **Phase 2 – Networking**.

Implemented so far:

- Storage engine
- SET
- GET
- TCP server
- Configurable listen address
- Concurrent client connections
- Connection logging
- Clean shutdown

Coming next:

- Protocol parsing
- Command dispatch
- Storage integration over TCP

## Project Structure

```text
bolt/
├── cmd/
│   └── bolt/
├── docs/
│   ├── PRD.md
│   └── SSOT.md
├── internal/
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

### Connect to the server

Open a new terminal and connect using `netcat`:

```bash
nc 127.0.0.1 6380
```

If the connection succeeds, the terminal will wait for input and the server will log the new client connection.

At the current stage, Bolt accepts TCP connections but does not yet process commands, so typing text into the client will not produce a response.

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

## Roadmap

### Phase 1
Core in-memory database

### Phase 2
Networking

### Phase 3
Persistence

### Phase 4
Replication

### Phase 5
Distributed Bolt

## Development

Bolt is being built incrementally using Test-Driven Development (TDD). Every feature starts with tests before implementation.

Each phase focuses on one major concept so the codebase stays small, readable, and easy to understand.
