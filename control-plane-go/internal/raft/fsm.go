package raft

import (
    "encoding/json"
    "fmt"
    "io"
    "sync"
    
    "github.com/hashicorp/raft"
)

type Command struct {
    Op    string `json:"op"`
    Key   string `json:"key"`
    Value Route  `json:"value"`
}

type Route struct {
    Destination string `json:"destination"`
    NextHop     string `json:"next_hop"`
    Metric      int    `json:"metric"`
}

type DVRFSM struct {
    routes map[string]Route
    mu     sync.RWMutex
}

func NewDVRFSM() *DVRFSM {
    return &DVRFSM{
        routes: make(map[string]Route),
    }
}

func (f *DVRFSM) Apply(log *raft.Log) interface{} {
    var cmd Command
    if err := json.Unmarshal(log.Data, &cmd); err != nil {
        return fmt.Errorf("failed to unmarshal command: %v", err)
    }
    
    f.mu.Lock()
    defer f.mu.Unlock()
    
    switch cmd.Op {
    case "ADD_ROUTE":
        f.routes[cmd.Key] = cmd.Value
        log.Printf("FSM: Added route %s -> %s", cmd.Key, cmd.Value.NextHop)
    case "DELETE_ROUTE":
        delete(f.routes, cmd.Key)
        log.Printf("FSM: Deleted route %s", cmd.Key)
    default:
        return fmt.Errorf("unknown operation: %s", cmd.Op)
    }
    
    return nil
}

func (f *DVRFSM) Snapshot() (raft.FSMSnapshot, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    return &DVRSnapshot{routes: f.routes}, nil
}

func (f *DVRFSM) Restore(rc io.ReadCloser) error {
    // Implementation for restoring from snapshot
    return nil
}

type DVRSnapshot struct {
    routes map[string]Route
}

func (s *DVRSnapshot) Persist(sink raft.SnapshotSink) error {
    // Implementation for persisting snapshot
    return nil
}

func (s *DVRSnapshot) Release() {}
