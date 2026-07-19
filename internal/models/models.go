package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Conversation struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
	Status         string    `json:"status"`
	LastActivityAt time.Time `json:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	IsAIDraft      bool      `json:"is_ai_draft"`
	CreatedAt      time.Time `json:"created_at"`
}

type Queries struct {
	db *sql.DB
}

func NewQueries(db *sql.DB) *Queries {
	return &Queries{db: db}
}

// GetOrCreateCustomer handles customer initialization
func (q *Queries) GetOrCreateCustomer(email string) (*User, error) {
	var user User
	err := q.db.QueryRow(`
		INSERT INTO users (email, role)
		VALUES ($1, 'customer')
		ON CONFLICT (email) DO UPDATE SET role = 'customer'
		RETURNING id, email, role, created_at
	`, email).Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetOrCreateOpenConversation handles getting an open conversation for a customer
func (q *Queries) GetOrCreateOpenConversation(customerID string) (*Conversation, error) {
	var conv Conversation
	err := q.db.QueryRow(`
		SELECT id, customer_id, status, last_activity_at, created_at
		FROM conversations
		WHERE customer_id = $1 AND status = 'open'
		ORDER BY last_activity_at DESC LIMIT 1
	`, customerID).Scan(&conv.ID, &conv.CustomerID, &conv.Status, &conv.LastActivityAt, &conv.CreatedAt)
	
	if err == sql.ErrNoRows {
		err = q.db.QueryRow(`
			INSERT INTO conversations (customer_id, status)
			VALUES ($1, 'open')
			RETURNING id, customer_id, status, last_activity_at, created_at
		`, customerID).Scan(&conv.ID, &conv.CustomerID, &conv.Status, &conv.LastActivityAt, &conv.CreatedAt)
	}
	
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// SaveMessage saves a message and updates conversation activity
func (q *Queries) SaveMessage(conversationID, senderID, content string, isAIDraft bool) (*Message, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var msg Message
	err = tx.QueryRow(`
		INSERT INTO messages (conversation_id, sender_id, content, is_ai_draft)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, sender_id, content, is_ai_draft, created_at
	`, conversationID, senderID, content, isAIDraft).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.IsAIDraft, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		UPDATE conversations
		SET last_activity_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, conversationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &msg, nil
}

// AcceptAIDraft changes a draft to a sent message
func (q *Queries) AcceptAIDraft(messageID string) error {
	_, err := q.db.Exec(`
		UPDATE messages
		SET is_ai_draft = FALSE, created_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, messageID)
	return err
}

// GetMessages retrieves recent messages for a conversation
func (q *Queries) GetMessages(conversationID string, limit int) ([]Message, error) {
	rows, err := q.db.Query(`
		SELECT id, conversation_id, sender_id, content, is_ai_draft, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.IsAIDraft, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// GetOpenConversations gets open conversations for the agent view
func (q *Queries) GetOpenConversations() ([]Conversation, error) {
	rows, err := q.db.Query(`
		SELECT id, customer_id, status, last_activity_at, created_at
		FROM conversations
		WHERE status = 'open'
		ORDER BY last_activity_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(&conv.ID, &conv.CustomerID, &conv.Status, &conv.LastActivityAt, &conv.CreatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, nil
}
