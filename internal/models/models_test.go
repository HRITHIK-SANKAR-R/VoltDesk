package models

import (
	"testing"
	"time"
	"unsafe"

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

func TestStructMemoryPacking(t *testing.T) {
	// Assert theoretical sizes without compiler padding.
	// time.Time is 24 bytes (on 64-bit systems)
	// string is 16 bytes (ptr + length)
	// *string is 8 bytes
	// bool is 1 byte

	userSize := unsafe.Sizeof(User{})
	expectedUserSize := uintptr(24 + 16 + 16 + 16 + 8 + 8 + 8) // 96 bytes
	if userSize != expectedUserSize {
		t.Errorf("User struct memory packing failed. Expected %d bytes, got %d", expectedUserSize, userSize)
	}

	convSize := unsafe.Sizeof(Conversation{})
	expectedConvSize := uintptr(24 + 24 + 16 + 16 + 16) // 96 bytes
	if convSize != expectedConvSize {
		t.Errorf("Conversation struct memory packing failed. Expected %d bytes, got %d", expectedConvSize, convSize)
	}

	msgSize := unsafe.Sizeof(Message{})
	// time.Time(24) + string(16)*4 + bool(1) = 89 bytes
	// Due to 64-bit word alignment of the struct itself, it rounds up to a multiple of 8, which is 96 bytes.
	expectedMsgSize := uintptr(96)
	if msgSize != expectedMsgSize {
		t.Errorf("Message struct memory packing failed. Expected %d bytes, got %d", expectedMsgSize, msgSize)
	}
}
