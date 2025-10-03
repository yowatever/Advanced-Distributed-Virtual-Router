package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// Route represents a network route
type Route struct {
	Destination string `json:"destination"`
	NextHop     string `json:"next_hop"`
	Metric      int    `json:"metric"`
}

// Command represents a Raft log entry
type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value Route  `json:"value"`
}

// DVRFSM implements the Raft finite state machine
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
		// Use the global log package, not the raft.Log parameter
		log.Printf("[FSM] Added route %s -> %s", cmd.Key, cmd.Value.NextHop)
	case "DELETE_ROUTE":
		delete(f.routes, cmd.Key)
		// Use the global log package, not the raft.Log parameter
		log.Printf("[FSM] Deleted route %s", cmd.Key)
	default:
		return fmt.Errorf("unknown operation: %s", cmd.Op)
	}

	return nil
}

func (f *DVRFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	routesCopy := make(map[string]Route)
	for k, v := range f.routes {
		routesCopy[k] = v
	}

	return &DVRSnapshot{routes: routesCopy}, nil
}

func (f *DVRFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	return nil
}

type DVRSnapshot struct {
	routes map[string]Route
}

func (s *DVRSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	return nil
}

func (s *DVRSnapshot) Release() {}

// Server manages the HTTP API
type Server struct {
	raft *raft.Raft
	fsm  *DVRFSM
}

func NewServer(raftNode *raft.Raft, fsm *DVRFSM) *Server {
	return &Server{
		raft: raftNode,
		fsm:  fsm,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/routes", s.handleRoutes)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/cluster/status", s.handleClusterStatus)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("API server listening on %s", addr)
	return server.ListenAndServe()
}

func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getRoutes(w, r)
	case http.MethodPost:
		s.addRoute(w, r)
	case http.MethodDelete:
		s.deleteRoute(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) addRoute(w http.ResponseWriter, r *http.Request) {
	var route Route
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create Raft command
	cmd := Command{
		Op:    "ADD_ROUTE",
		Key:   route.Destination,
		Value: route,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply to Raft cluster
	if s.raft.State() != raft.Leader {
		http.Error(w, "Not the leader", http.StatusServiceUnavailable)
		return
	}

	future := s.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "route added",
		"destination": route.Destination,
	})
}

func (s *Server) getRoutes(w http.ResponseWriter, r *http.Request) {
	// For now, return empty routes
	json.NewEncoder(w).Encode(map[string]interface{}{
		"routes": map[string]Route{},
	})
}

func (s *Server) deleteRoute(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement delete route
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	if s.raft.State() != raft.Leader {
		status = "follower"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     status,
		"raft_state": s.raft.State().String(),
	})
}

func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.raft.Stats())
}

// InMemoryStore implements raft.StableStore and raft.LogStore
type InMemoryStore struct {
	mu    sync.RWMutex
	logs  map[uint64]*raft.Log
	store map[string][]byte
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		logs:  make(map[uint64]*raft.Log),
		store: make(map[string][]byte),
	}
}

func (s *InMemoryStore) FirstIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.logs) == 0 {
		return 0, nil
	}
	
	var min uint64 = 1 << 63
	for k := range s.logs {
		if k < min {
			min = k
		}
	}
	return min, nil
}

func (s *InMemoryStore) LastIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var max uint64 = 0
	for k := range s.logs {
		if k > max {
			max = k
		}
	}
	return max, nil
}

func (s *InMemoryStore) GetLog(index uint64, log *raft.Log) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	l, ok := s.logs[index]
	if !ok {
		return raft.ErrLogNotFound
	}
	*log = *l
	return nil
}

func (s *InMemoryStore) StoreLog(log *raft.Log) error {
	return s.StoreLogs([]*raft.Log{log})
}

func (s *InMemoryStore) StoreLogs(logs []*raft.Log) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, log := range logs {
		s.logs[log.Index] = log
	}
	return nil
}

func (s *InMemoryStore) DeleteRange(min, max uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i := min; i <= max; i++ {
		delete(s.logs, i)
	}
	return nil
}

func (s *InMemoryStore) Set(key []byte, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.store[string(key)] = val
	return nil
}

func (s *InMemoryStore) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	val, ok := s.store[string(key)]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return val, nil
}

func (s *InMemoryStore) SetUint64(key []byte, val uint64) error {
	return s.Set(key, []byte(fmt.Sprintf("%d", val)))
}

func (s *InMemoryStore) GetUint64(key []byte) (uint64, error) {
	val, err := s.Get(key)
	if err != nil {
		return 0, err
	}
	
	var result uint64
	_, err = fmt.Sscanf(string(val), "%d", &result)
	return result, err
}

func setupRaft(nodeID, raftAddr, dataDir string) (*raft.Raft, *DVRFSM, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 2

	// Create transport
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, nil, err
	}

	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	// Create snapshot store
	snapshots, err := raft.NewFileSnapshotStore(dataDir, 2, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	// Create in-memory store (instead of boltdb)
	store := NewInMemoryStore()

	// Create FSM
	fsm := NewDVRFSM()

	// Create Raft instance
	raftNode, err := raft.NewRaft(config, fsm, store, store, snapshots, transport)
	if err != nil {
		return nil, nil, err
	}

	return raftNode, fsm, nil
}

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
	raftNode, fsm, err := setupRaft(nodeID, raftAddr, dataDir)
	if err != nil {
		log.Fatal("Failed to setup Raft:", err)
	}

	// Bootstrap single-node cluster
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(nodeID),
				Address: raft.ServerAddress(raftAddr),
			},
		},
	}
	future := raftNode.BootstrapCluster(configuration)
	if err := future.Error(); err != nil {
		log.Printf("Bootstrap warning: %v", err)
	}

	// Start API server
	server := NewServer(raftNode, fsm)
	log.Fatal(server.Start(apiAddr))
}
