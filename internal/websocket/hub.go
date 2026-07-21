package websocket

import (
	"context"
	"log"
	"sync"

	"voltdesk/internal/ai"
	"voltdesk/internal/models"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

// Hub maintains the set of active clients and subscribes to Redis Pub/Sub.
type Hub struct {
	// Registered clients map protected by mutex
	clients map[*Client]bool
	mu      sync.RWMutex

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
	
	// Database queries instance
	queries *models.Queries
	
	// Redis client
	rdb *redis.Client
}

func NewHub(q *models.Queries, rdb *redis.Client) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		queries:    q,
		rdb:        rdb,
	}
}

func (h *Hub) Run() {
	ctx := context.Background()
	pubsub := h.rdb.PSubscribe(ctx, "room:*")
	defer pubsub.Close()
	
	ch := pubsub.Channel()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))
			
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))
			
		case msg := <-ch:
			// Handle incoming Redis Pub/Sub payload
			var payload map[string]interface{}
			if err := msgpack.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				log.Printf("Failed to unmarshal msgpack: %v", err)
				continue
			}
			
			// Find conversation ID
			var convID string
			if cid, ok := payload["conversation_id"].(string); ok {
				convID = cid
			} else {
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				// Route only to matching client or any agent
				if client.ConversationID == convID || client.Role == "agent" {
					// We push the raw payload map to client.send (it's chan any)
					select {
					case client.send <- payload:
					default:
						// If send buffer is full or blocked, drop the connection
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastControl publishes a control message to Redis
func (h *Hub) BroadcastControl(conversationID string, controlType string) {
	controlMsg := map[string]interface{}{
		"type":            controlType,
		"conversation_id": conversationID,
	}
	
	b, err := msgpack.Marshal(controlMsg)
	if err != nil {
		log.Printf("Failed to marshal control msg: %v", err)
		return
	}
	
	ctx := context.Background()
	h.rdb.Publish(ctx, "room:"+conversationID, b)
}

// GenerateAIDraft triggers the AI generation and publishes the result
func (h *Hub) GenerateAIDraft(conversationID string) {
	draft, err := ai.GenerateDraft(h.queries, conversationID)
	if err != nil {
		log.Printf("AI Draft Generation Error: %v", err)
		return
	}
	if draft != nil {
		// Bypass draft phase and save immediately
		savedMsg, err := h.queries.SaveMessage(conversationID, "ai-bot", draft.Content, false)
		if err != nil {
			log.Printf("error saving AI message: %v", err)
			return
		}

		// Publish to Redis as a real chat_message
		msgData := map[string]interface{}{
			"type":            "chat_message",
			"payload":         savedMsg,
			"conversation_id": conversationID,
		}
		b, err := msgpack.Marshal(msgData)
		if err == nil {
			ctx := context.Background()
			h.rdb.Publish(ctx, "room:"+conversationID, b)
		}
	}
}
