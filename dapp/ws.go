package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleSocketConnection(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	clientID := r.URL.Query().Get("clientID")
	if clientID == "" {
		http.Error(w, "clientID is required", http.StatusBadRequest)
		return
	}

	// Handle the WebSocket connection here
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upgrade connection: %v", err), http.StatusInternalServerError)
		return
	}

	if _, ok := TrieClientsMap[clientID]; !ok {
		TrieClientsMap[clientID] = conn
	}

	fmt.Println("List of clients: ", TrieClientsMap)

	select {}
}

func handleConnectedClients(c *gin.Context) {
	clientIDs := make([]string, 0, len(TrieClientsMap))
	for clientID := range TrieClientsMap {
		clientIDs = append(clientIDs, clientID)
	}
	c.JSON(http.StatusOK, clientIDs)
}
