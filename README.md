# Distributed Distance Vector Router (DVR)

A **production-grade, fault-tolerant distributed routing system** built with Go + Raft consensus and high-performance C++ data plane. This system provides distributed route management with strong consistency guarantees.

## 🏗️ Architecture

```
┌─────────────────┐   Raft Consensus   ┌─────────────────┐
│  Control Plane  │◄──────────────────►│  Control Plane  │
│    (Go/Raft)    │                    │    (Go/Raft)    │
│   Node 1        │                    │   Node 2        │
└─────────┬───────┘                    └─────────┬───────┘
          │                                      │
          ▼                                      ▼
┌─────────┴───────┐                    ┌─────────┴───────┐
│   Data Plane    │                    │   Data Plane    │
│     (C++)       │                    │     (C++)       │
└─────────────────┘                    └─────────────────┘
```

##  Features

- **Distributed Consensus**: Raft protocol for fault tolerance and strong consistency
- **High Availability**: Automatic leader election and failover
- **Horizontal Scaling**: Add nodes dynamically to the cluster
- **Persistent State**: Snapshot-based recovery and log replication
- **RESTful API**: Simple HTTP interface for route management
- **gRPC Integration**: High-performance communication with data plane

## Quick Start

### Prerequisites

- Go 1.21+
- GCC/Clang (for C++ data plane)
- (Optional) Docker for containerized deployment

### Single Node Setup

```bash
# Clone and build
git clone <repository>
cd distributed-dvr/control-plane-go

# Build the control plane
go build -o dvr-control-plane ./cmd

# Run single node
./dvr-control-plane
```

### Test the API

```bash
# Check health
curl http://localhost:9090/health

# Add a route
curl -X POST http://localhost:9090/routes \
  -H "Content-Type: application/json" \
  -d '{"destination": "10.1.0.0/16", "next_hop": "192.168.1.1", "metric": 100}'

# Get all routes
curl http://localhost:9090/routes
```

### Multi-Node Cluster

```bash
# Start 3-node cluster using the provided script
chmod +x scripts/start-cluster.sh
./scripts/start-cluster.sh
```

Test the cluster:
```bash
# Check all nodes
curl http://localhost:9091/health
curl http://localhost:9092/health  
curl http://localhost:9093/health

# Add route (automatically routed to leader)
curl -X POST http://localhost:9091/routes \
  -H "Content-Type: application/json" \
  -d '{"destination": "192.168.0.0/24", "next_hop": "10.0.0.1", "metric": 50}'

# Verify consistency - all nodes show same routes
curl http://localhost:9091/routes
curl http://localhost:9092/routes
curl http://localhost:9093/routes
```

## 📁 Project Structure

```
distributed-dvr/
├── control-plane-go/          # Go control plane with Raft
│   ├── cmd/main.go           # Main application entry point
│   ├── internal/
│   │   ├── api/              # HTTP REST API handlers
│   │   └── raft/             # Raft FSM and setup
│   └── go.mod
├── data-plane-cpp/           # C++ data plane (in progress)
│   ├── src/                  # Source files
│   ├── include/              # Header files
│   └── proto/                # gRPC protocol definitions
├── deploy/
│   └── docker/               # Docker configurations
├── scripts/
│   └── start-cluster.sh      # Multi-node startup script
└── README.md
```

## 🔌 API Reference

### Health Check
```http
GET /health
```
Response:
```json
{
  "status": "healthy",
  "raft_state": "Leader"
}
```

### Add Route
```http
POST /routes
Content-Type: application/json

{
  "destination": "10.1.0.0/16",
  "next_hop": "192.168.1.1", 
  "metric": 100
}
```

### Get Routes
```http
GET /routes
```
Response:
```json
{
  "routes": {
    "10.1.0.0/16": {
      "destination": "10.1.0.0/16",
      "next_hop": "192.168.1.1",
      "metric": 100
    }
  }
}
```

### Cluster Status
```http
GET /cluster/status
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ID` | `node-1` | Unique node identifier |
| `RAFT_ADDR` | `127.0.0.1:8300` | Raft consensus address |
| `API_ADDR` | `:9090` | HTTP API address |
| `DATA_DIR` | `./data` | Data directory for snapshots |

### Multi-Node Configuration

For a 3-node cluster:

**Node 1:**
```bash
export NODE_ID=node-1
export RAFT_ADDR=127.0.0.1:8301
export API_ADDR=:9091
export DATA_DIR=./data/node1
```

**Node 2:**
```bash
export NODE_ID=node-2  
export RAFT_ADDR=127.0.0.1:8302
export API_ADDR=:9092
export DATA_DIR=./data/node2
```

**Node 3:**
```bash
export NODE_ID=node-3
export RAFT_ADDR=127.0.0.1:8303
export API_ADDR=:9093
export DATA_DIR=./data/node3
```

## 🐳 Docker Deployment

```bash
cd deploy/docker
docker-compose up -d
```

This starts a 3-node cluster with ports:
- Node 1: API 9091, Raft 8301
- Node 2: API 9092, Raft 8302  
- Node 3: API 9093, Raft 8303

## 🔧 Development

### Building from Source

```bash
# Control Plane (Go)
cd control-plane-go
go build -o dvr-control-plane ./cmd

# Data Plane (C++) - Coming Soon
cd ../data-plane-cpp
mkdir build && cd build
cmake ..
make
```

### Running Tests

```bash
# Run unit tests
cd control-plane-go
go test ./...

# Test API endpoints
./scripts/test-api.sh
```

## Monitoring

### Logs
The system provides detailed Raft consensus logs:
- Leader election process
- Log replication
- Snapshot creation
- FSM state changes

### Metrics
- Raft term and commit index
- Node health status
- Route table size
- API request metrics

## Fault Tolerance

### Leader Failure
- Automatic leader election within seconds
- Zero data loss due to log replication
- Clients automatically redirected to new leader

### Network Partitions
- Raft maintains consistency during partitions
- Automatic recovery when partition heals
- No split-brain scenarios

### Node Recovery
- Recovering nodes automatically sync missing logs
- Snapshots enable fast recovery
- Consistent state maintained across cluster



## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

