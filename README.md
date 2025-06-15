# Distributed Key-Value Store with Raft Consensus

A distributed key-value store written in Go, providing strong consistency, leader election, and fault tolerance through the [Raft consensus algorithm](https://raft.github.io/raft.pdf). It allows replicated state machines, dynamic cluster membership, snapshotting, and log replication, while exposing an HTTP interface for client interaction.

---

## Features

* **Raft Consensus**: Ensures strong consistency across all nodes using Hashicorp's Raft implementation.
* **Fault Tolerance**: Survives failure of up to (N-1)/2 nodes in an N-node cluster.
* **Leader Election**: Automatically elects a new leader when the current one fails.
* **Log Replication**: All state changes are proposed by the leader and replicated to followers.
* **Snapshotting**: Periodically captures the FSM state and compacts the Raft log for recovery efficiency.
* **HTTP API**: Exposes REST endpoints (`/set`, `/get`, `/delete`, `/join`) for client interaction.
* **Leader Forwarding**: Follower nodes redirect write requests to the cluster leader automatically.
* **Dynamic Cluster Membership**: New nodes can join an existing cluster via the `/join` endpoint.
* **In-Memory State Machine**: Backed by a thread-safe key-value store using `sync.RWMutex`.
* **Persistent Log Storage**: Uses BoltDB for durable log and state storage.

---

## Architecture

Each node runs:

* A Raft server (leader or follower)
* An HTTP server for client interaction

Write operations (`/set`, `/delete`) are always handled by the cluster leader. Follower nodes automatically proxy write requests to the current leader. Read operations (`/get`) are served from any node’s local state machine.

The Raft log and stable state are persisted using BoltDB. FSM snapshots are written to disk to allow fast recovery and limit log growth.

---

## Technology Stack

* Language: Go (Golang)
* Consensus: [HashiCorp Raft](https://github.com/hashicorp/raft)
* Persistence: [BoltDB](https://github.com/boltdb/bolt)
* Networking: TCP (Raft) and HTTP (API)
* Concurrency: Goroutines, channels, RWMutex synchronization

---

## Setup

### Clone and prepare the project

```bash
git clone https://github.com/PranavPipariya/ConsensusKV.git
cd distributed-kv-store
go mod tidy
go build -o kvnode main.go
```

---

## Running a 3-Node Cluster

### Start Node 1 (Bootstrap)

```bash
./kvnode -raft-address=127.0.0.1:5000 -api-address=127.0.0.1:8000 -data-dir=./data1 -bootstrap=true
```

### Start Node 2

```bash
./kvnode -raft-address=127.0.0.1:5001 -api-address=127.0.0.1:8001 -data-dir=./data2
```

Join Node 2 to the cluster:

```bash
curl http://127.0.0.1:8000/join?peerAddress=127.0.0.1:5001
```

### Start Node 3

```bash
./kvnode -raft-address=127.0.0.1:5002 -api-address=127.0.0.1:8002 -data-dir=./data3
```

Join Node 3 to the cluster:

```bash
curl http://127.0.0.1:8000/join?peerAddress=127.0.0.1:5002
```

Leader election occurs automatically.

---

## API Usage

### Set Key

```bash
curl -X POST http://127.0.0.1:8000/set \
-H "Content-Type: application/json" \
-d '{"key":"foo","value":"bar"}'
```

### Get Key

```bash
curl http://127.0.0.1:8000/get?key=foo
```

### Delete Key

```bash
curl -X POST http://127.0.0.1:8000/delete \
-H "Content-Type: application/json" \
-d '{"key":"foo"}'
```

### Cluster Join (for new nodes)

```bash
curl http://127.0.0.1:8000/join?peerAddress=<new-node-raft-address>
```

### Cluster Status

```bash
curl http://127.0.0.1:8000/status
```

---

## References

* [Raft Consensus Algorithm (Ongaro & Ousterhout, 2014)](https://raft.github.io/raft.pdf)
* [HashiCorp Raft Library](https://github.com/hashicorp/raft)
* [BoltDB Storage Engine](https://github.com/boltdb/bolt)

---
