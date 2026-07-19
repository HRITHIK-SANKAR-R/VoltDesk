package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetOrCreateCustomer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	queries := NewQueries(db)

	email := "test@example.com"
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO users \(email, role\)`).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "role", "created_at"}).
			AddRow("uuid-123", email, "customer", now))

	user, err := queries.GetOrCreateCustomer(email)
	if err != nil {
		t.Errorf("error was not expected while getting customer: %s", err)
	}

	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	queries := NewQueries(db)

	convID := "conv-123"
	now := time.Now()

	mock.ExpectQuery(`SELECT id, conversation_id, sender_id, content, is_ai_draft, created_at FROM messages`).
		WithArgs(convID, 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "conversation_id", "sender_id", "content", "is_ai_draft", "created_at"}).
			AddRow("msg-1", convID, "sender-1", "Hello", false, now))

	messages, err := queries.GetMessages(convID, 50)
	if err != nil {
		t.Errorf("error was not expected while getting messages: %s", err)
	}

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello" {
		t.Errorf("expected Hello, got %s", messages[0].Content)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
