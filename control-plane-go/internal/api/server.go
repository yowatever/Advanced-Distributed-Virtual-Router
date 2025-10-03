package api

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
    
        "github.com/hashicorp/raft"
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


type Server struct {
    raft *raft.Raft
    fsm  *raft.DVRFSM
}

func NewServer(raftNode *raft.Raft, fsm *raft.DVRFSM) *Server {
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
    var route raft.Route
    if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Create Raft command
    cmd := raft.Command{
        Op:  "ADD_ROUTE",
        Key: route.Destination,
        Value: route,
    }
    
    data, err := json.Marshal(cmd)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Apply to Raft cluster
    future := s.raft.Apply(data, 5*time.Second)
    if err := future.Error(); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "route added",
        "destination": route.Destination,
    })
}

func (s *Server) getRoutes(w http.ResponseWriter, r *http.Request) {
    // Get routes from FSM (read from local state)
    // In production, you might want to handle this differently
    // to ensure read-after-write consistency
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "routes": "TODO: implement route retrieval",
    })
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    status := "healthy"
    if s.raft.State() != raft.Leader {
        status = "follower"
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": status,
        "node_id": s.raft.Stats()["node_id"],
        "raft_state": s.raft.State().String(),
    })
}

func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(s.raft.Stats())
}
