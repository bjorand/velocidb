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

Here is a list of Redis "compatible" commands:
- `PING`
- `GET <key>`
- `SET <key> <value>`

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
