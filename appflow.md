# VoltDesk: Application Flow Specification

## 1. Global Architectural Lifecycle Overview

VoltDesk leverages a highly decoupled, real-time message loop driving two distinct user workflows concurrently. Communication is sustained via raw, persistent WebSockets for message delivery, supplemented by stateless JSON REST endpoints for bootstrap synchronization.

```text
[Customer UI] <==== WebSocket Connection ====> [Go WS Hub Engine] <==== WebSocket Connection ====> [Agent UI]
     │                                                 │                                                 │
(REST Sync)                                      (DB Reads/Writes)                                  (REST Sync)
     │                                                 │                                                 │
     ▼                                                 ▼                                                 ▼
[HTTP GET API] ───────────────────────────> [PostgreSQL Database] <─────────────────────────── [HTTP GET API]
                                                       ▲
                                                       │ (Polls & Dispatches Alerts)
                                                       │
                                            [Background Idle Worker] ───(HTTP POST)───> [Resend Email API]
```

---

## 2. Bootstrapping & Handshake Flows

### 2.1. Customer Widget Initialization & Connection Setup
When the floating customer widget is initialized on a host webpage, it executes an initialization handshake to guarantee session persistence.

```text
Customer Client                                  Go Server (REST)                             Go Server (WS)
      │                                                 │                                            │
      │─── 1. POST /api/auth/customer (Email/ID) ──────>│                                            │
      │    (Verify or provision new user record)        │                                            │
      │                                                 │                                            │
      │<── 2. Return User UUID & Active Conv UUID ──────│                                            │
      │                                                 │                                            │
      │─── 3. GET /api/conversations/{id}/messages ────>│                                            │
      │    (Fetch past historical chat markers)         │                                            │
      │                                                 │                                            │
      │<── 4. Return JSON array of last 50 messages ────│                                            │
      │                                                 │                                            │
      │─── 5. Initiate WebSocket Connection ────────────────────────────────────────────────────────>│
      │    URL: ws://localhost:8080/ws?token={jwt/uuid}                                              │
      │                                                                                              │
      │<── 6. Protocol Upgrade Approved (101 Switching Protocols) ───────────────────────────────────│
```

### 2.2. Agent Dashboard Handshake & Active Session Mapping
When an authenticated support agent launches the dashboard client:
1.  **State Sync:** The client fires an `HTTP GET /api/conversations?status=open` to pull the active, unresolved chat queue. The application populates the left-hand sidebar list state.
2.  **WS Upgrades:** The client initializes a connection to `ws://localhost:8080/ws?token={agent_uuid}`.
3.  **Hub Memory Binding:** The Go server upgrades the protocol using `gorilla/websocket.Upgrader`. The connection is encapsulated into a `Client` struct instance and pushed into the central `Hub.register` channel.
4.  **Channel Activation:** The server binds the client's network buffer descriptor to two long-running sub-routines: `go client.readPump()` and `go client.writePump()`.

---

## 3. Real-Time Message Propagation & Mutation Lifecycle

This diagram walks through the complete round-trip flow of a customer sending a message, the server handling broadcasting and database persistence, and the asynchronous triggers matching the Sticker Mule core optimization architecture.

```text
Customer Widget                Go WS readPump()               Go DB/Hub Loop               Agent Dashboard
       │                              │                              │                            │
       │─── 1. WS: chat_message ─────>│                              │                            │
       │    (JSON Payload)            │                              │                            │
       │                              │─── 2. Push to Write Channel ─>│                            │
       │                              │                               │                            │
       │                              │                               ├── 3. Async DB Insert ──┐   │
       │                              │                               │    (messages table)    │   │
       │                              │                               │<───────────────────────┘   │
       │                              │                               │                            │
       │                              │                               ├── 4. Broadcast to Agent ──>│
       │                              │                               │    (WS payload dispatch)   │
       │                              │                               │                            │
       │                              │                               ├── 5. Fire AI Goroutine ┐   │
       │                              │                               │    (Async non-blocking)│   │
       │                              │                               │<───────────────────────┘   │
       │                              │                               │                            │
       │                              │                               │── 6. Gemini API Fetch ─┐   │
       │                              │                               │    (Context + Prompt)  │   │
       │                              │                               │<───────────────────────┘   │
       │                              │                               │                            │
       │                              │                               ├── 7. Save AI Suggestion ──┐
       │                              │                               │    (is_ai_draft = true)   │
       │                              │                               │<──────────────────────────┘
       │                              │                               │                            │
       │                              │                               └── 8. WS: ai_smart_draft ──>
       │                              │                                    (Slides into composer)
```

### Step-by-Step Execution Mechanics:
1. **Serialization:** The Customer UI serializes a `WsEvent` containing a `chat_message` payload type and sends it down the TCP socket frame line.
2. **Parsing:** The Go client thread’s `readPump()` catches the frame, parses the schema format, handles escaping routines, stamps it with a server-side RFC3339 timestamp, and drops the generated struct pointer onto the internal central broadcast pipeline channel.
3. **Database Write:** The central `Hub` runtime consumes the pointer, executing an immediate non-blocking database insert:
   ```sql
   INSERT INTO messages (conversation_id, sender_id, content, is_ai_draft) 
   VALUES ($1, $2, $3, FALSE);
   ```
   Simultaneously, it updates `conversations.last_activity_at = CURRENT_TIMESTAMP` to refresh its standing order position within the agent's queue.
4. **Targeted Broadcast:** The `Hub` loops through its thread-safe active agent client connections map, matches the target `conversation_id`, and pushes the processed payload onto that specific Agent's outbound `send` channel, where `writePump()` flushes it immediately down the socket wire.
5. **AI Triggering:** If the sender role is confirmed as `customer`, a background AI worker is spawned asynchronously without impeding the loop.

---

## 4. Asynchronous AI Smart Reply Execution Flow

1. **Context Extraction:** The spawned AI worker Goroutine retrieves the historical conversation context using a targeted query:
   ```sql
   SELECT sender_id, content FROM messages 
   WHERE conversation_id = $1 AND is_ai_draft = FALSE 
   ORDER BY created_at DESC LIMIT 5;
   ```
2. **API Execution:** The worker passes this data vector to the Google Gemini SDK using `google-genai` wrappers, matching the System Prompt parameters to enforce rapid support responses.
3. **Draft Staging:** Once the response stream resolves, the text is persisted with `is_ai_draft = TRUE` in the database to prevent losing it on sudden client drops.
4. **UI Event:** A `WsEvent` with the type `ai_smart_draft` is sent to the target agent. The Agent UI intercepts this specific event, keeping the regular chat space intact while sliding a staging element right above the chat textbox.

### Agent Conversion Actions:
* **Scenario A (Acceptance):** The agent reviews the text and clicks "Send" or hits `Cmd+Enter`. The UI fires a `WsEvent` back to the server containing `type: "accept_ai_draft", payload: { message_id: "draft_uuid" }`. The server updates the database record state (`is_ai_draft = FALSE`) and broadcasts the final text immediately into the live channel for both user views.
* **Scenario B (Modification):** The agent clicks inside the staging block, edits the phrasing directly, and hits send. The UI converts this action into a completely new, standard `chat_message` event payload, sending it over the line while silently issuing a minor cleanup script to wipe the old, unused draft entry from memory.

---

## 5. The Automated Idle Notification Engine Flow

To safeguard response SLAs without relying on manual client oversight, VoltDesk implements a headless time-slice scanning workflow.

```text
[ Go Server Lifecycle ]
          │
          ▼
┌───────────────────────────┐
│ time.NewTicker(60 * Sec)  │ <────────────────────────────────────────┐
└─────────────┬─────────────┘                                          │
              │ (Fires every minute)                                   │
              ▼                                                        │
┌───────────────────────────┐                                          │
│  Execute Scan Query       │                                          │
│  (Find open, unattended   │                                          │
│   customer chats > 2 min) │                                          │
└─────────────┬─────────────┘                                          │
              │                                                        │
              ├─────── [ No Results Found ] ───────────────────────────┤
              │                                                        │
              └─────── [ Active Matches Identified ]                   │
                             │                                         │
                             ▼                                         │
               ┌───────────────────────────┐                           │
               │ Loop Collection & Batch   │                           │
               │ (Throttle to rate limits) │                           │
               └─────────────┬─────────────┘                           │
                             │                                         │
                             ▼                                         │
               ┌───────────────────────────┐                           │
               │ HTTP POST to Resend API   │                           │
               │ (Dispatch agent alert)    │                           │
               └─────────────┬─────────────┘                           │
                             │                                         │
                             └─────────────────────────────────────────┘
```

### The Scanning Database Query:
```sql
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
```
*Note: This specific check ensures that notifications are sent only if the customer was the last individual to type, preventing the system from erroneously alerting agents when a customer simply drops out mid-conversation.*

---

## 6. Disconnection, Fault Tolerance, & Reconnection Scenarios

### 6.1. Sudden Client Disconnection Flow
1. **Network Failure:** The internet connection drops on either client device, causing the socket link to drop frames.
2. **Server Cleanup:** The server-side `readPump()` catches the resulting network read error. It immediately breaks its active loop execution, safely terminates the connection pointer, and forwards the current socket state identifier down to `Hub.unregister`.
3. **State Eviction:** The `Hub` receives the disconnect notice, removes the item from the live memory tracking maps, and closes the client's internal `send` channel to release local resources and avoid leaks.

### 6.2. Client-Side Reconnection Flow
1. **Reconnection Loop:** The custom `useWebSocket` hook catches the network failure event via `onclose`. It switches the interface into a grayed-out connection warning state and initiates an active exponential backoff reconnection loop (2s, 4s, 8s, ..., max 30s).
2. **Re-handshake:** Once a network path opens up again, the hook establishes a fresh WebSocket request using the exact same session configuration parameters.
3. **UI Sync Reset:** Before turning the real-time websocket components back on, the UI re-runs its initial `GET /api/conversations/{id}/messages` state query. This pulls down any updates or system modifications that occurred while the client was offline, patching the local component arrays before restoring the socket to active duty.
