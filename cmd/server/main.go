package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"thousand/internal/server"
)

func main() {
	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", server.WSHandler)

	// Serve frontend build
	webDist := filepath.Join("web", "dist")
	fs := http.FileServer(http.Dir(webDist))
	mux.Handle("/", fs)

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
