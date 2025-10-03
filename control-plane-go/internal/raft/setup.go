package raft

import (
    "net"
    "os"
    "path/filepath"
    "time"
    
    "github.com/hashicorp/raft"
    raftboltdb "github.com/hashicorp/raft-boltdb"
)

type Route struct {
    Destination string `json:"destination"`
    NextHop     string `json:"next_hop"`
    Metric      int    `json:"metric"`
}

type Command struct {
    Op    string `json:"op"`
    Key   string `json:"key"`
    Value Route  `json:"value"`
}

func SetupRaft(nodeID, raftAddr, dataDir string) (*raft.Raft, error) {
    config := raft.DefaultConfig()
    config.LocalID = raft.ServerID(nodeID)
    config.SnapshotInterval = 30 * time.Second
    config.SnapshotThreshold = 2
    
    // Create transport
    addr, err := net.ResolveTCPAddr("tcp", raftAddr)
    if err != nil {
        return nil, err
    }
    
    transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
    if err != nil {
        return nil, err
    }
    
    // Create snapshot store
    snapshots, err := raft.NewFileSnapshotStore(dataDir, 2, os.Stderr)
    if err != nil {
        return nil, err
    }
    
    // Create log store
    logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
    if err != nil {
        return nil, err
    }
    
    // Create FSM
    fsm := NewDVRFSM()
    
    // Create Raft instance
    raftNode, err := raft.NewRaft(config, fsm, logStore, logStore, snapshots, transport)
    if err != nil {
        return nil, err
    }
    
    return raftNode, nil
}
