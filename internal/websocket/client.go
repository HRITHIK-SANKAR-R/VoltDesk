package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
	"voltdesk/internal/models"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 5120
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In production, check origin
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub
	conn *websocket.Conn
	send chan *models.Message
	
	UserID         string
	Role           string
	ConversationID string
}

// WsEvent represents the typed JSON payload
type WsEvent struct {
	Type    string          `json:"type"`
	Payload *models.Message `json:"payload"`
}

type AcceptDraftPayload struct {
	MessageID string `json:"message_id"`
}

type WsIncomingEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// bufferPool provides a global pool of bytes.Buffer for zero-allocation websocket reads.
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	
	for {
		_, r, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Fetch a buffer from the pool and ensure it's explicitly reset
		buf := bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		
		_, err = buf.ReadFrom(r)
		if err != nil {
			log.Printf("error reading from ws: %v", err)
			bufferPool.Put(buf)
			break
		}
		
		var incoming WsIncomingEvent
		if err := json.Unmarshal(buf.Bytes(), &incoming); err != nil {
			log.Printf("error unmarshalling event: %v", err)
			bufferPool.Put(buf)
			continue
		}

		switch incoming.Type {
		case "chat_message":
			var payload models.Message
			if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
				bufferPool.Put(buf)
				continue
			}
			// Save to DB (Confirm transaction BEFORE broadcast)
			savedMsg, err := c.hub.queries.SaveMessage(payload.ConversationID, c.UserID, payload.Content, false)
			if err != nil {
				log.Printf("error saving message: %v", err)
				bufferPool.Put(buf)
				continue
			}
			
			// Broadcast
			c.hub.broadcast <- savedMsg
			
			// Trigger AI asynchronously if it's a customer
			if c.Role == "customer" {
				go c.hub.GenerateAIDraft(savedMsg.ConversationID)
			}

		case "accept_ai_draft":
			var payload AcceptDraftPayload
			if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
				bufferPool.Put(buf)
				continue
			}
			err := c.hub.queries.AcceptAIDraft(payload.MessageID)
			if err != nil {
				log.Printf("error accepting draft: %v", err)
			}
			// Let agents know it was accepted by refetching or sending a sync event
		}

		// Safely return the buffer to the pool after all processing is complete
		bufferPool.Put(buf)
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Format for client
			eventType := "chat_message"
			if message.IsAIDraft {
				eventType = "ai_smart_draft"
			}
			
			event := WsEvent{
				Type:    eventType,
				Payload: message,
			}

			if err := c.conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID, role, conversationID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan *models.Message, 256),
		UserID:         userID,
		Role:           role,
		ConversationID: conversationID,
	}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
