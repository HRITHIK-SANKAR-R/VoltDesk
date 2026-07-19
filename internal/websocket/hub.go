package websocket

import (
	"log"
	"sync"
	"voltdesk/internal/models"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients map protected by mutex
	clients map[*Client]bool
	mu      sync.RWMutex

	// Inbound messages from the clients.
	broadcast chan *models.Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
	
	// Database queries instance
	queries *models.Queries
}

func NewHub(q *models.Queries) *Hub {
	return &Hub{
		broadcast:  make(chan *models.Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		queries:    q,
	}
}

func (h *Hub) Run() {
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
			
		case message := <-h.broadcast:
			// Broadcast message to relevant clients
			// In a real app, you'd only broadcast to clients in the same conversation
			h.mu.RLock()
			for client := range h.clients {
				// To keep it simple, we check if the client is associated with this conversation
				// Or if the client is an agent
				if client.Role == "agent" || client.ConversationID == message.ConversationID {
					select {
					case client.send <- message:
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

// BroadcastToAgent sends a specific message directly to agents
func (h *Hub) BroadcastToAgent(message *models.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.Role == "agent" {
			select {
			case client.send <- message:
			default:
				// ignore
			}
		}
	}
}

func (h *Hub) BroadcastToConversation(message *models.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.ConversationID == message.ConversationID || client.Role == "agent" {
			select {
			case client.send <- message:
			default:
				// ignore
			}
		}
	}
}

// GenerateAIDraft triggers the AI generation and broadcasts the result
func (h *Hub) GenerateAIDraft(conversationID string) {
	// Call AI logic and get the draft
	// We need to avoid import cycle if hub imports ai and ai imports hub
	// Since AI just returns a message, hub can broadcast it
	// Actually we should just call ai.GenerateDraft
	// Let's assume we do that from client.go or hub.go
}
