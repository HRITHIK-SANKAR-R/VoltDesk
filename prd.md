# VoltDesk: Product Requirements Document (PRD)

## 1. Executive Summary
VoltDesk is a lightweight, real-time customer support platform designed to handle live chat, automated AI-drafted responses, and background idle notifications.

The strategic objective of this codebase is to serve as a high-signal portfolio artifact. For a pre-final year AI&DS student aiming to bypass traditional resume filters for a remote senior-level role at Sticker Mule, standard applications are insufficient. VoltDesk is architected to prove immediate, production-ready competence. It mirrors Sticker Mule’s engineering ethos: lean dependencies, aggressive utilization of AI to optimize workflows, autonomous product ownership, and an absolute mastery of Go and type-safe frontend architecture.

## 2. Core Philosophy & Constraints
To maintain a high-performance, enterprise-grade standard, the development of VoltDesk is strictly bound by the following technical constraints:
* **Zero-Bloat Backend:** The Go backend will rely heavily on the standard `net/http` library. No heavy frameworks (like Fiber or Gin) will be used. `gorilla/websocket` is the only permitted exception for upgrading connections.
* **Strict Type Safety:** The frontend is exclusively React + TypeScript built via Vite. Every prop, state, and WebSocket payload must have an explicit interface.
* **Asynchronous AI:** AI API calls must never block the main WebSocket broadcasting thread. Goroutines will handle LLM integrations concurrently.
* **Relational Integrity:** PostgreSQL is the source of truth. All tables must utilize UUIDs for primary keys to prevent enumeration attacks, and strict foreign keys must enforce data integrity.

## 3. System Architecture
VoltDesk operates on a decoupled client-server model communicating primarily over persistent WebSockets, with a secondary background worker running on the server.

| Component | Technology | Primary Responsibility |
| :--- | :--- | :--- |
| **Frontend SPA** | Vite, React, TypeScript, Tailwind CSS | Manages UI state, WebSocket lifecycle (`useWebSocket`), and renders the Customer Widget and Agent Dashboard. |
| **Backend API / WS Hub** | Go (1.21+), `gorilla/websocket` | Upgrades HTTP to WebSockets, manages concurrent client maps via a `Hub` struct, and broadcasts messages safely using Channels. |
| **Database** | PostgreSQL | Stores persistent state (Users, Conversations, Messages) with strict relational modeling. |
| **AI Auto-Responder** | Google Gemini API (Go SDK) | Asynchronously reads incoming customer messages and generates support-tailored draft responses for the agent. |
| **Idle Worker** | Go `time.Ticker`, Resend API | Scans the database every 60 seconds for unresolved messages older than 2 minutes and dispatches alert emails. |

## 4. Feature Specifications: The 4 Pillars

### 4.1. Real-Time Chat Infrastructure
**Objective:** Enable instant, bi-directional communication between a customer and an agent.
* **Connection Upgrade:** The Go backend exposes a `/ws` endpoint that upgrades standard HTTP requests to WebSockets.
* **Concurrency Handling:** A central `Hub` struct manages a map of active connections (`map[*Client]bool`). A dedicated Goroutine listens for incoming messages, writes them to the database, and pushes them to a broadcast channel.
* **State Recovery:** On initial load, the frontend fetches the last 50 messages of the conversation via a standard `GET /api/messages` REST endpoint before establishing the WebSocket connection to ensure no context is lost.

### 4.2. Relational Database Schema
**Objective:** Store data securely and efficiently, supporting fast lookups for the chat UI and background workers.
* **Primary Keys:** All IDs are generated using `gen_random_uuid()` (v4 UUIDs).
* **Users Table:** Distinguishes between `role: 'customer'` and `role: 'agent'`.
* **Conversations Table:** Links a customer to an agent, tracks `status` (open, resolved), and maintains a `last_activity_at` timestamp.
* **Messages Table:** Contains the payload, `sender_id`, `conversation_id`, and `is_ai_draft` boolean flags.

### 4.3. "Aggressive" AI Support Engine
**Objective:** Radically decrease agent response time by pre-computing replies.
* **Trigger:** When a `message` is received from a `customer` over the WebSocket, the Go backend fires a non-blocking Goroutine.
* **Context Gathering:** The Goroutine retrieves the last 5 messages of the specific conversation to provide context to the LLM.
* **LLM Prompting:** A strict System Prompt instructs Gemini to act as a concise, helpful Sticker Mule support agent.
* **Delivery:** The generated draft is saved to the DB with `is_ai_draft = true` and pushed to the agent's WebSocket feed, rendering as a "Smart Reply" button in the UI.

### 4.4. Automated Notification Engine
**Objective:** Ensure no customer is left waiting indefinitely if an agent switches tabs or loses focus.
* **Background Process:** When the Go server boots, it initializes a `time.NewTicker(1 * time.Minute)`.
* **Query:** The worker executes a SQL query identifying all conversations where `status = 'open'` AND `last_activity_at < NOW() - INTERVAL '2 minutes'` AND the last message sender was a customer.
* **Action:** For each identified conversation, the Go backend calls the Resend API to dispatch an email to the agent's address with a direct link to the chat session.

## 5. Interface & Payload Contracts

### 5.1. WebSocket JSON Payload Schema
All messages sent over the WebSocket must adhere to this strict structure to allow the TypeScript frontend to discriminate union types.

```json
{
  "type": "chat_message",
  "payload": {
    "id": "uuid-v4-string",
    "conversation_id": "uuid-v4-string",
    "sender_id": "uuid-v4-string",
    "content": "I need help with my die-cut sticker order.",
    "is_ai_draft": false,
    "created_at": "2026-07-17T10:28:19Z"
  }
}
```

### 5.2. Frontend UI Layouts
* **Customer Widget:** A fixed, floating window positioned at `bottom-4 right-4`. Consists of a header, a scrollable message list (flex column, reverse), and a sticky input area.
* **Agent Dashboard:** A full-screen CSS Grid layout.
  * **Left Sidebar (25%):** Active chat queue, sorted by `last_activity_at` descending.
  * **Main Window (75%):** The active conversation history.
  * **Action Bar (Bottom):** Text input field, flanked by the prominent AI "Smart Draft" suggestion box that populates asynchronously.

## 6. Execution Roadmap
This PRD dictates the sequence of development. We will build from the data layer up to the presentation layer.

1. **Phase 1:** Database Architecture (Migrations, Schemas, Go `database/sql` setup)
2. **Phase 2:** The Real-Time Backend (Go WebSockets, Hub struct, Goroutines)
3. **Phase 3:** AI & Notification Engines (Gemini API integration, Ticker, Resend API)
4. **Phase 4:** The Type-Safe Interface (Vite + React UI, WebSocket hooks, Tailwind styling)
