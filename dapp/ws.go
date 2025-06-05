package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

	fmt.Println("List of clients: ", TrieClientsMap)

	if tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(3 * time.Second)
	}

	conn.SetCloseHandler(func(code int, text string) error {
		defer conn.Close()
		fmt.Printf("Client disconnected (CLOSING): %v, errCode: %v, txt: %v", clientID, code, text)
		delete(TrieClientsMap, clientID)
		return nil
	})

	_, msgOpen, err := conn.ReadMessage()
	if err != nil {
		fmt.Println("Error reading message when at OPEN phase :", err)
		delete(TrieClientsMap, clientID)
	}

	fmt.Println("Message received when at OPEN phase: ", string(msgOpen))
	// go func() {
	// 	ticker := time.NewTicker(4 * time.Second)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			conn.SetWriteDeadline(time.Now().Add(4 * time.Second))
	// 			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
	// 				log.Println("ping error:", err)
	// 				return
	// 			} else {
	// 				fmt.Println("Ping sent successfully")
	// 			}
	// 		}
	// 	}
	// }()

	conn.SetPingHandler(func(appData string) error {
		fmt.Println(fmt.Sprintf("Ping received: %v\n", appData))
		return nil
	})
	// conn.SetPongHandler(func(appData string) error {
	// 	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	// 	fmt.Println("Pong received: ", appData)
	// 	return nil
	// })

	TrieClientsMap[clientID] = conn

	select {}
}

func handleConnectedClients(c *gin.Context) {
	clientIDs := make([]string, 0, len(TrieClientsMap))
	for clientID := range TrieClientsMap {
		clientIDs = append(clientIDs, clientID)
	}
	c.JSON(http.StatusOK, clientIDs)
}

func handlePingClient(c *gin.Context) {
	clientID := c.Query("clientID")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, "clientID is required")
		return
	}

	conn, ok := TrieClientsMap[clientID]
	if !ok {
		c.JSON(http.StatusNotFound, "Client not found")
		return
	}

	err := conn.WriteMessage(websocket.PingMessage, []byte("ping"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to send ping: %v", err))
		return
	}
	c.JSON(http.StatusOK, "Ping sent successfully")
}
