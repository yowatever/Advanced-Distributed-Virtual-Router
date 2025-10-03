package main

import (
    "log"
    "net/http"
)

func main() {
    log.Println("Starting Distributed DVR Control Plane...")
    
    // Basic HTTP server for health checks
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "healthy"}`))
    })
    
    log.Println("Server listening on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}
