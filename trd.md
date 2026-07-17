# VoltDesk: Technical Requirements Document (TRD)

## 1. Environment & Toolchain
To maintain a high-performance, reproducible environment, the project will rely on standard Unix-philosophy tooling and containerization.

*   **Backend Language:** Go 1.21+
*   **Frontend Runtime:** Node.js 20+ / Bun (for ultra-fast Vite builds)
*   **Database:** PostgreSQL 15+ 
*   **Containerization:** Docker & Docker Compose (for local database and Redis/cache if ever needed, though we rely entirely on Postgres for now to keep it lean).
*   **Task Runner:** GNU Make (`Makefile` to combine Go and Vite build commands).

## 2. Directory Structure (Monorepo)
VoltDesk will be organized as a monorepo to ensure frontend and backend remain perfectly synced. The Go backend follows the standard idiomatic project layout.

```text
voltdesk/
├── .env.example
├── Makefile
├── docker-compose.yml          # Provisions local Postgres
├── go.mod
├── go.sum
├── cmd/
│   └── server/
│       └── main.go             # Application entrypoint, sets up routes & workers
├── internal/
│   ├── database/               # Postgres connection pool and raw SQL queries
│   ├── models/                 # Go structs representing DB tables
│   ├── websocket/              # Gorilla Hub, Client, and Message payloads
│   ├── ai/                     # Gemini SDK integration and system prompts
│   └── worker/                 # The time.Ticker background notification engine
├── migrations/                 # .sql files for schema creation
└── web/                        # Vite + React + TypeScript frontend
    ├── package.json
    ├── tailwind.config.js
    ├── src/
    │   ├── components/         # ChatBubbles, AgentSidebar, SmartReplyBtn
    │   ├── hooks/              # useWebSocket.ts custom hook
    │   └── types/              # strict TypeScript interfaces (syncs with Go models)
```

## 3. Database Schema Design (PostgreSQL)
We bypass ORMs entirely. The database will be accessed via the standard `database/sql` package and `lib/pq` (or `pgx`). 

### 3.1. UUID & Extensions
The database requires the `pgcrypto` extension for v4 UUID generation.
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

### 3.2. Core Tables
**Table: `users`**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('customer', 'agent')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**Table: `conversations`**
```sql
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_conversations_status_time ON conversations(status, last_activity_at);
```
*Note: The composite index on `status` and `last_activity_at` is critical. It allows the background worker to execute the 60-second idle check with zero table scans.*

**Table: `messages`**
```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    is_ai_draft BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_messages_conversation ON messages(conversation_id);
```

## 4. Go Backend Architecture

### 4.1. WebSocket Concurrency Model
Managing thousands of persistent connections requires strict thread safety to avoid race conditions.

*   **The Hub (`internal/websocket/hub.go`):** A centralized struct running in its own Goroutine. It maintains a map `clients map[*Client]bool`. It listens on three unbuffered channels: `register`, `unregister`, and `broadcast`.
*   **The Client (`internal/websocket/client.go`):** Represents a single WebSocket connection. Each client spawns two Goroutines upon connection:
    *   `readPump()`: Listens for incoming JSON from the frontend, unmarshals it, saves it to Postgres, and pushes it to the `broadcast` channel.
    *   `writePump()`: Listens on a dedicated `send chan []byte` for outgoing messages and writes them to the WebSocket network connection.
*   **Mutexes:** While channels handle the primary flow of data, any direct modifications to the active sessions map outside the main Hub select block will be protected by `sync.RWMutex`.

### 4.2. Asynchronous AI Integration
When `readPump()` processes a message from a `customer`, it triggers `go ai.GenerateDraft(conversationID)`. 
*   This detached Goroutine fetches the last 5 messages of the conversation.
*   It issues an HTTP request to the Gemini API.
*   Upon receiving the response, it writes the draft to the `messages` table with `is_ai_draft = true`.
*   It then constructs a WebSocket payload and injects it directly into the Hub's `broadcast` channel, targeting only the agent connected to that conversation.

### 4.3. The Idle Notification Worker
In `cmd/server/main.go`, a background Goroutine is initialized:
```go
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()

go func() {
    for range ticker.C {
        worker.CheckIdleConversations(dbPool)
    }
}()
```
The query looks for `last_activity_at < NOW() - INTERVAL '2 minutes'`. For hits, the worker batches HTTP calls to the Resend API to avoid exhausting external rate limits.

## 5. React / TypeScript Frontend Architecture

### 5.1. The `useWebSocket` Hook
The frontend lifecycle must gracefully handle component unmounts and network drops. The custom hook will encapsulate:
*   `WebSocket.onopen`, `onmessage`, `onerror`, `onclose`.
*   Automatic exponential backoff for reconnections.
*   A `sendMessage` function that marshals TypeScript objects into JSON strings.

### 5.2. Type Definitions (`web/src/types/index.ts`)
The TypeScript interfaces must exactly mirror the Go JSON structs.

```typescript
export interface MessagePayload {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  is_ai_draft: boolean;
  created_at: string;
}

export interface WsEvent {
  type: 'chat_message' | 'system_alert' | 'agent_typing';
  payload: MessagePayload;
}
```

## 6. Environment Variables (`.env`)
The system requires the following environment variables to run locally:

```ini
# Server
PORT=8080
GO_ENV=development

# Postgres (Local Docker DB)
DATABASE_URL=postgres://postgres:postgres@localhost:5432/voltdesk?sslmode=disable

# External APIs
GEMINI_API_KEY=your_google_ai_studio_key
RESEND_API_KEY=your_resend_api_key
```
