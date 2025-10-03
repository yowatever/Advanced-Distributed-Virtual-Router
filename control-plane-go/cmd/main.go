package main

import (
    "log"
    "os"
    "path/filepath"
    
    "github.com/yowatever/dvr-control-plane/internal/api"
    "github.com/yowatever/dvr-control-plane/internal/raft"
)

func main() {
    // Get configuration from environment
    nodeID := os.Getenv("NODE_ID")
    if nodeID == "" {
        nodeID = "node-1"
    }
    
    raftAddr := os.Getenv("RAFT_ADDR")
    if raftAddr == "" {
        raftAddr = "127.0.0.1:8300"
    }
    
    apiAddr := os.Getenv("API_ADDR")
    if apiAddr == "" {
        apiAddr = ":9090"
    }
    
    dataDir := os.Getenv("DATA_DIR")
    if dataDir == "" {
        dataDir = "./data"
    }
    
    // Ensure data directory exists
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Starting node %s with Raft address %s", nodeID, raftAddr)
    
    // Setup Raft
    raftNode, err := raft.SetupRaft(nodeID, raftAddr, dataDir)
    if err != nil {
        log.Fatal(err)
    }
    
    // Bootstrap single-node cluster if needed
    if raftNode.State() == raft.Follower {
        configuration := raft.Configuration{
            Servers: []raft.Server{
                {
                    ID:      raft.ServerID(nodeID),
                    Address: raft.ServerAddress(raftAddr),
                },
            },
        }
        raftNode.BootstrapCluster(configuration)
    }
    
    // Create FSM instance for API access
    fsm := raft.NewDVRFSM()
    
    // Start API server
    server := api.NewServer(raftNode, fsm)
    log.Fatal(server.Start(apiAddr))
}
