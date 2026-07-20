# Bolt Manual Test Plan

This document lists the manual integration tests for Bolt. These tests
complement the automated test suite (`go test ./...` and
`go test -race ./...`).

## 1. Build

``` bash
go build ./...
```

Expected: - Builds successfully. - No compile errors.

------------------------------------------------------------------------

## 2. Start Primary

``` bash
go run ./cmd/bolt -addr 127.0.0.1:6380
```

Expected:

``` text
server listening on 127.0.0.1:6380
```

------------------------------------------------------------------------

## 3. Basic Client

``` bash
nc 127.0.0.1 6380
```

Run:

``` text
SET name saksham
GET name
GET missing
```

Expected:

``` text
OK
saksham
(nil)
```

------------------------------------------------------------------------

## 4. Invalid Command

``` text
HELLO
```

Expected:

``` text
ERR unknown command
```

Connection remains open.

------------------------------------------------------------------------

## 5. TTL

``` text
SET token abc EX 5
GET token
```

Expected:

``` text
abc
```

Wait 5--6 seconds.

``` text
GET token
```

Expected:

``` text
(nil)
```

------------------------------------------------------------------------

## 6. Transaction Commit

``` text
MULTI
SET a 1
SET b 2
EXEC
GET a
GET b
```

Expected:

``` text
OK
1
2
```

------------------------------------------------------------------------

## 7. Transaction Discard

``` text
MULTI
SET c 3
DISCARD
GET c
```

Expected:

``` text
(nil)
```

------------------------------------------------------------------------

## 8. Nested MULTI

``` text
MULTI
MULTI
```

Expected:

``` text
ERR ...
```

------------------------------------------------------------------------

## 9. EXEC Without MULTI

``` text
EXEC
```

Expected:

``` text
ERR ...
```

------------------------------------------------------------------------

## 10. DISCARD Without MULTI

``` text
DISCARD
```

Expected:

``` text
ERR ...
```

------------------------------------------------------------------------

## 11. Pub/Sub

Terminal 1:

``` bash
nc 127.0.0.1 6380
```

``` text
SUBSCRIBE chat
```

Terminal 2:

``` bash
nc 127.0.0.1 6380
```

``` text
SUBSCRIBE chat
```

Terminal 3:

``` bash
nc 127.0.0.1 6380
```

``` text
PUBLISH chat hello
```

Expected: - Both subscribers receive the published message.

------------------------------------------------------------------------

## 12. Unsubscribe

``` text
UNSUBSCRIBE chat
```

Publish again.

Expected: - Unsubscribed client receives nothing.

------------------------------------------------------------------------

## 13. INFO

``` text
INFO
```

Verify:

-   node_id
-   role
-   uptime
-   connected_clients
-   replication_status
-   connected_replicas
-   key_count
-   memory_usage

------------------------------------------------------------------------

## 14. AOF Persistence

``` text
SET language go
SET project bolt
```

Restart Bolt.

Verify:

``` text
GET language
GET project
```

Expected:

``` text
go
bolt
```

------------------------------------------------------------------------

## 15. Snapshot Recovery

Insert several keys.

Shutdown gracefully.

Restart.

Verify all keys remain.

------------------------------------------------------------------------

## 16. Crash Recovery

Insert data.

Kill Bolt:

``` bash
kill -9 <pid>
```

Restart.

Verify committed keys still exist.

------------------------------------------------------------------------

## 17. Replica Startup

Primary:

``` bash
go run ./cmd/bolt -addr 127.0.0.1:6380
```

Replica:

``` bash
go run ./cmd/bolt -addr 127.0.0.1:6381 -replicaof 127.0.0.1:6380
```

Expected: - Replica connects automatically.

------------------------------------------------------------------------

## 18. Initial Synchronization

Before starting replica:

``` text
SET one 1
SET two 2
SET three 3
```

After replica starts:

``` text
GET one
GET two
GET three
```

Expected: - All values exist.

------------------------------------------------------------------------

## 19. Live Replication

Primary:

``` text
SET city delhi
```

Replica:

``` text
GET city
```

Expected:

``` text
delhi
```

------------------------------------------------------------------------

## 20. Replica Read-Only

Replica:

``` text
SET hello world
```

Expected:

``` text
ERR replica is read-only
```

------------------------------------------------------------------------

## 21. Replicated TTL

Primary:

``` text
SET temp value EX 5
```

Replica:

``` text
GET temp
```

Expected:

``` text
value
```

Wait 5--6 seconds.

``` text
GET temp
```

Expected:

``` text
(nil)
```

------------------------------------------------------------------------

## 22. Replication INFO

Primary:

``` text
INFO
```

Verify:

``` text
role: primary
connected_replicas: 1
```

Replica:

``` text
INFO
```

Verify:

``` text
role: replica
connected_replicas: 0
```

------------------------------------------------------------------------

## 23. Concurrent Clients

Open multiple client terminals.

Execute SET, GET, MULTI, EXEC and PUBLISH simultaneously.

Expected: - No crashes. - No deadlocks. - Correct data.

------------------------------------------------------------------------

## 24. Graceful Shutdown

Press `Ctrl+C`.

Expected:

-   Persistence saved.
-   Client connections closed.
-   Clean exit.

------------------------------------------------------------------------

## 25. Port Already In Use

Start another Bolt instance on the same port.

Expected:

``` text
bind: address already in use
```

------------------------------------------------------------------------

## 26. Custom Persistence Paths

``` bash
go run ./cmd/bolt \
  -aof /tmp/custom.aof \
  -snapshot /tmp/custom.snapshot
```

Expected: - Custom files are created and used.

------------------------------------------------------------------------

## 27. Multiple Subscribers

Subscribe 3--4 clients to the same channel.

Publish one message.

Expected: - Every subscriber receives exactly one copy.

------------------------------------------------------------------------

## 28. Multiple Channels

``` text
SUBSCRIBE sports
SUBSCRIBE music
```

Publish independently.

Expected: - Only subscribers of the matching channel receive each
message.

------------------------------------------------------------------------

## Completion Criteria

All manual tests pass successfully in addition to:

``` bash
go test ./...
go test -race ./...
```
