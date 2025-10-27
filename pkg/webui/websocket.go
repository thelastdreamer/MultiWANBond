package webui

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	conn *websocket.Conn
	send chan *Event
	mu   sync.Mutex
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WSClient{
		conn: conn,
		send: make(chan *Event, 256),
	}

	// Register client
	s.wsMu.Lock()
	s.wsClients[client] = true
	s.wsMu.Unlock()

	// Start read and write pumps
	go client.writePump()
	go client.readPump(s)
}

// writePump sends messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send event as JSON
			msg := WebSocketMessage{
				Type:      string(event.Type),
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump(s *Server) {
	defer func() {
		// Unregister client
		s.wsMu.Lock()
		delete(s.wsClients, c)
		s.wsMu.Unlock()
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle incoming messages
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Process message based on type
		c.handleMessage(&msg, s)
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WSClient) handleMessage(msg *WebSocketMessage, s *Server) {
	switch msg.Type {
	case "subscribe":
		// Handle subscription requests
	case "unsubscribe":
		// Handle unsubscription requests
	case "ping":
		// Respond with pong
		response := &WebSocketMessage{
			Type:      "pong",
			Timestamp: time.Now(),
		}
		c.send <- &Event{
			Type:      EventType(response.Type),
			Timestamp: response.Timestamp,
		}
	}
}
