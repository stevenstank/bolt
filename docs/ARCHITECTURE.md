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

Replication is a side channel attached to successful primary writes:

```text
Primary Client
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
  +----> Replication Observer ----> Replica TCP Session ----> Replica Store
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

The engine notifies replication observers after successful writes.

Replica mode sets the engine to read-only for client traffic while leaving a trusted apply path available for replicated updates.

### `internal/storage`

Owns the in-memory key-value map and synchronization.

`NewStore` creates an in-memory-only store. `NewPersistentStore` creates a store backed by an Append Only File and replays that file on startup. `NewDurableStore` loads a snapshot first, then replays the AOF.

`Snapshot` returns a copy of current data for replica bootstrapping.

### `internal/persistence`

Owns durable file formats, AOF append/replay, snapshot save/load, and crash recovery behavior.

Persistence must not depend on networking.

### `internal/replication`

Owns primary/replica synchronization, live write streaming, keepalive messages, and reconnect behavior.

The replication package must not bypass the server or mutate storage directly.

Replication uses a simple plain-text protocol with snapshot markers, heartbeat messages, and length-prefixed SET records.

Replica command rejection is enforced by the engine, not by storage or the TCP server.
