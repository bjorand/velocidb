# Velocidb

## Introduction

Velocidb is a fast persistent key-value store with the following goals:
  - optional consistency at object level
  - highly available
  - optional tolerance to network splits

More details about Velocidb engine:
 - written in Golang
 - fast TCP wire protocol
 - peer with up to 1000 Velocidb nodes for data replication, clustering and requests distribution.
 - plugin system for data query and storage, networking or administration
 - hashicorp/consul auto-clustering support


## Status

### VQL (Velocidb Query Language)

Velocidb is in early development. We support a small subset of the Redis protocol to validate Velocidb distributed models.

Here is a list of Redis "compatible" commands (commands are case-insensitive):
- `INFO [category]`
- `PING [value]`
- `GET <key>`
- `SET <key> <value>`
- `INCR <key>`
- `DECR <key>`
- `DEL <key>`
- `KEYS <glob>`
- `SCAN <cursor> [COUNT count] [MATCH glob] [TYPE type]`
- `TTL <key`
- `TYPE <key>`
- `SELECT <db>`
- `TIME`
- `FLUSHDB`
- `CLIENT LIST`
- `CLIENT SETNAME <value>`
- `CLIENT GETNAME`
- `CLIENT KILL <addr:port>`
- `QUIT`

Interactive session with `redis-cli`:

```
redis-cli -p 4300
127.0.0.1:4300> keys *
(empty list or set)
127.0.0.1:4300> incr a
(integer) 1
127.0.0.1:4300> keys *
1) "a"
127.0.0.1:4300> incr z
(integer) 1
127.0.0.1:4300> keys *
1) "a"
2) "z"
127.0.0.1:4300> del a z
(integer) 2
127.0.0.1:4300> keys *
(empty list or set)
127.0.0.1:4300> get z
(nil)
127.0.0.1:4300> incr z
(integer) 1
127.0.0.1:4300> get z
"1"
127.0.0.1:4300> info keyspace
# Keyspace
db0:keys=1
```

### Data storage

Two parallel projects are in development:
- memory storage: in-memory storage (can be rebuild on boot with WAL or peers)
- WAL (aka write ahead logging): securely log all write modification on disk

All operations are consistent. It means all peers see the same data.

### Clustering (aka Peering)

Peering is in a very early stage of development. Side work integrates Hashicorp Consul for peers discovery.

## Build

Dependencies are handled by `dep` tool:

```
# install dep if necessary
go get -u github.com/golang/dep/cmd/dep
# install dependencies
dep ensure
```

Build binary:

```
go build
./velocidb --help
```

### CPU Profiling

```
go build && ./velocidb -cpuprofile out.prof # Ctrl-c to generate report
go tool pprof -http=":8081" velocidb out.prof
```
