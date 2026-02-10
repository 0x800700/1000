package server

import (
	"log"
	"net/http"
)

// WSHandler handles WebSocket connections for the single-session MVP.
func WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	session := GetSession()
	session.HandleConnection(conn)
}
