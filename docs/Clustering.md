# Velocidb clustering

## Definitions

## Peer

A peer is member of a Velocidb cluster.
Its main role is to maintain a channel of communications with other peers.

## Mesh

Mesh designates a network of interconnected peers.

# Protocol specification

## General

Peer 1 opens a TCP connection to peer 2.
P1 requests mesh info of P2 while P2 requests mesh topology of P1. Each nodes merge remote mesh info with its local content.
Mesh info list known peers and their information like:

- network parameters to connect to peer
- network connection state

## Message format

### Simple strings

First byte is `+` followed by a one line string terminated by CRLF.

```
+OK\r\n
```

### Errors

First byte is `-` followed by a one line string terminated by CRLF.

### Bulk strings

To represent binary safe message and handle large data size efficiently, bulk strings are used.

Format is:

- first byte is `*` followed by the number of bytes of the data to read ended by CRLF

```
$6\r\nfoobar\r\n
```

## Peering protocol

### Mesh commands

- `+PING` check if other end is alive and return latency in nanosecond
- `+MESH PEER REGISTER <data>` announce a new peer in the mesh
- bulk string `WALWRITE <id> <data>`
