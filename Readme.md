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
