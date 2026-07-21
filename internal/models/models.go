package models

import (
	"database/sql"
	"time"
)

type User struct {
	CreatedAt time.Time `json:"created_at" msgpack:"created_at"`
	ID        string    `json:"id" msgpack:"id"`
	Email     string    `json:"email" msgpack:"email"`
	Role      string    `json:"role" msgpack:"role"`
	Name      *string   `json:"name" msgpack:"name"`
	AvatarURL *string   `json:"avatar_url" msgpack:"avatar_url"`
	GoogleID  *string   `json:"google_id" msgpack:"google_id"`
}

type Conversation struct {
	LastActivityAt time.Time `json:"last_activity_at" msgpack:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at" msgpack:"created_at"`
	ID             string    `json:"id" msgpack:"id"`
	CustomerID     string    `json:"customer_id" msgpack:"customer_id"`
	Status         string    `json:"status" msgpack:"status"`
}

type Message struct {
	CreatedAt      time.Time `json:"created_at" msgpack:"created_at"`
	ID             string    `json:"id" msgpack:"id"`
	ConversationID string    `json:"conversation_id" msgpack:"conversation_id"`
	SenderID       string    `json:"sender_id" msgpack:"sender_id"`
	Content        string    `json:"content" msgpack:"content"`
	IsAIDraft      bool      `json:"is_ai_draft" msgpack:"is_ai_draft"`
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

// ResolveConversation marks a conversation as resolved
func (q *Queries) ResolveConversation(conversationID string) error {
	_, err := q.db.Exec(`
		UPDATE conversations
		SET status = 'resolved', last_activity_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, conversationID)
	return err
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
func (q *Queries) AcceptAIDraft(messageID string) (*Message, error) {
	var msg Message
	err := q.db.QueryRow(`
		UPDATE messages
		SET is_ai_draft = FALSE, created_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, conversation_id, sender_id, content, is_ai_draft, created_at
	`, messageID).Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.IsAIDraft, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	
	// Also update conversation last_activity_at
	_, err = q.db.Exec(`
		UPDATE conversations
		SET last_activity_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, msg.ConversationID)
	
	return &msg, err
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

// IdleConversation represents the data needed for the notification worker
type IdleConversation struct {
	ConversationID string
	CustomerEmail  string
}

// GetIdleConversations finds open conversations where the last activity is > 2 mins old
// and the last message sender was a customer
func (q *Queries) GetIdleConversations() ([]IdleConversation, error) {
	rows, err := q.db.Query(`
		SELECT c.id, u.email 
		FROM conversations c
		JOIN users u ON c.customer_id = u.id
		WHERE c.status = 'open' 
		  AND c.last_activity_at < NOW() - INTERVAL '2 minutes'
		  AND (
		      SELECT sender_id FROM messages 
		      WHERE conversation_id = c.id 
		      ORDER BY created_at DESC LIMIT 1
		  ) = c.customer_id;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idleConvs []IdleConversation
	for rows.Next() {
		var ic IdleConversation
		if err := rows.Scan(&ic.ConversationID, &ic.CustomerEmail); err != nil {
			return nil, err
		}
		idleConvs = append(idleConvs, ic)
	}
	return idleConvs, nil
}

func (q *Queries) GetOrCreateUserByGoogleID(email, googleID, name, avatarURL string) (*User, error) {
	role := "customer"
	if email == "hrithiksrr@gmail.com" {
		role = "agent"
	}
	var user User
	err := q.db.QueryRow(`
		INSERT INTO users (email, role, google_id, name, avatar_url)
		VALUES ($1, $5, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET google_id = $2, name = $3, avatar_url = $4, role = $5
		RETURNING id, email, role, name, avatar_url, google_id, created_at
	`, email, googleID, name, avatarURL, role).Scan(&user.ID, &user.Email, &user.Role, &user.Name, &user.AvatarURL, &user.GoogleID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetOldResolvedConversations fetches conversations older than 30 days that are resolved
func (q *Queries) GetOldResolvedConversations() ([]string, error) {
	rows, err := q.db.Query(`
		SELECT id FROM conversations 
		WHERE status = 'resolved' AND last_activity_at < NOW() - INTERVAL '30 days'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetMessagesForArchiving gets all messages for a specific conversation to archive
func (q *Queries) GetMessagesForArchiving(conversationID string) ([]Message, error) {
	rows, err := q.db.Query(`
		SELECT id, conversation_id, sender_id, content, is_ai_draft, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
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

// DeleteConversationAndMessages deletes the conversation (which cascades to messages if FK is set, but we do it manually to be safe)
func (q *Queries) DeleteConversationAndMessages(conversationID string) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM messages WHERE conversation_id = $1`, conversationID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM conversations WHERE id = $1`, conversationID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
