# Bolt Architecture

Bolt is organized as small internal packages with one-directional responsibilities.

## Current Request Flow

```text
Client
  |
  v
TCP Server
  |
  v
Protocol Parser
  |
  v
Command Dispatcher
  |
  v
Engine
  |
  v
Storage
  |
  v
Persistence
```

The server accepts TCP connections, logs connection lifecycle events, reads newline-delimited command lines, and writes newline-delimited responses. It delegates command processing through an interface and does not manipulate storage directly.

## Package Responsibilities

### `internal/server`

Owns TCP listener setup, accepted client connections, connection lifecycle logging, and graceful shutdown.

The server must not manipulate storage directly.

### `internal/protocol`

Owns plain-text command parsing.

### `internal/command`

Owns command validation, command dispatch, and plain-text responses.

### `internal/engine`

Owns database operations and coordinates access to storage.

### `internal/storage`

Owns the in-memory key-value map and synchronization.

`NewStore` creates an in-memory-only store. `NewPersistentStore` creates a store backed by an Append Only File and replays that file on startup. `NewDurableStore` loads a snapshot first, then replays the AOF.

### `internal/persistence`

Owns durable file formats, AOF append/replay, snapshot save/load, and crash recovery behavior.

Persistence must not depend on networking.
