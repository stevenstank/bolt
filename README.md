# Bolt

Bolt is a Redis-inspired in-memory key-value database written in Go.

The goal of this project is to learn how modern databases are built from first principles. Bolt starts as a simple in-memory key-value store and will gradually evolve to include networking, persistence, replication, and distributed systems concepts.

## Current Status

Bolt is currently in **Phase 1 – Core In-Memory Database**.

Implemented so far:

- Storage engine
- SET
- GET

Coming next:

- DEL
- EXISTS
- KEYS
- Concurrency tests

## Project Structure

```text
bolt/
├── docs/
│   ├── PRD.md
│   └── SSOT.md
├── internal/
│   └── storage/
├── README.md
└── go.mod
```

## Build

```bash
go build ./...
```

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