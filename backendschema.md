# VoltDesk: Backend Schema & Data Layer Specification (Textual Blueprint)

## 1. Data Architecture Rationales
VoltDesk purposefully avoids heavy abstraction layers, such as Object-Relational Mappers (ORMs), to maximize query visibility and prevent hidden processing traps like N+1 queries. Relational state mapping is managed through precise, lower-level database paradigms.

### 1.1 The Security Model: Universally Unique Identifiers (UUIDv4)
VoltDesk rejects auto-incrementing integers (`1, 2, 3...`) across primary keys to defend against Insecure Direct Object Reference (IDOR) vectors. By generating non-sequential, random 128-bit identifiers via the database's `pgcrypto` engine, the data layer hides system metrics and blocks enumeration attempts. URLs referencing specific items remain unpredictable.

### 1.2 Relational Guardrails
Data dependencies are locked directly into the storage hardware via referential constraints:
*   **Cascade Mechanisms:** Deleting structural entities triggers an automated purge down the relational tree. For instance, removing a conversation systematically wipes all downstream text exchanges without requiring additional query statements from the Go server.
*   **Domain Controls:** Inputs are bound to rigorous validation lists. The data pool rejects arbitrary category strings, preventing data pollution across functional states.

---

## 2. Relational Entity Structures

### 2.1 The Users Entity
This catalog logs every distinct actor authorized to open communication or manage threads within VoltDesk.
*   **Identifier:** A unique 128-bit token generated on entry.
*   **Electronic Mail:** A character string that uniquely maps to one account. Blank values or duplicate configurations are rejected.
*   **System Privilege:** A categorized state tracking role permissions. Enforced by check constraints, it allows only two configurations:
    *   `customer`: An individual initiating help sessions.
    *   `agent`: A corporate responder managing incoming feeds.
*   **Creation Timestamp:** The exact calendar point, tracking timezone data, indicating when the identity record was established.

### 2.2 The Conversations Entity
This tracks the core operational units of VoltDesk, mapping unique customer requests to active sessions.
*   **Identifier:** A unique token acting as the permanent session address.
*   **Customer Linkage:** A relational bond explicitly matching a user entry marked as a `customer`. If the underlying identity is expunged, the entire session tree drops automatically.
*   **Workflow Status:** A state marker tracking the lifecycle of the request. Enforced by strict rules, it oscillates between two values:
    *   `open`: Currently awaiting or undergoing evaluation.
    *   `resolved`: Concluded and cleared from active operational views.
*   **Activity Timestamp:** A timezone-aware marker indicating when the record last shifted or received input. It plays a primary role in sorting queues.
*   **Creation Timestamp:** A locked chronological record tracking when the help ticket was generated.

### 2.3 The Messages Entity
This table maps individual message exchanges inside a live session.
*   **Identifier:** A unique tracking token assigned to every chunk of text sent.
*   **Conversation Linkage:** A referential constraint tying the exchange to a parent session. If the session drops, all dependent message nodes clear instantly.
*   **Sender Linkage:** A referential constraint mapping the string payload to the individual user who created it.
*   **Content Payload:** A clean, un-truncated block storing raw text inputs.
*   **Automation Flag:** A binary variable distinguishing origin states:
    *   `true`: The text is an un-submitted, machine-generated draft from the Gemini AI layer.
    *   `false`: The text is a finalized message typed by a human actor.
*   **Creation Timestamp:** A timezone-aware timestamp documenting the exact arrival order of the text payload.

---

## 3. Database Indexes & Indexing Strategies

### 3.1 Background Scanning Index (`idx_conversations_status_time`)
A multi-column composite index designed specifically for the 60-second background worker loop.
*   **Target:** `status` combined with `last_activity_at` within the Conversations collection.
*   **Impact:** Instead of sorting through thousands of dead rows, the worker pinpoint-queries the table. It immediately drops entries marked `resolved` or updated within the last 2 minutes, ensuring sub-millisecond lookups.

### 3.2 Historical Sync Index (`idx_messages_conversation`)
A composite structure supporting fast state initialization.
*   **Target:** `conversation_id` combined with `created_at` in descending order within the Messages collection.
*   **Impact:** When a user reboots a client or bridges a dropped socket, this index allows the system to pull the last 50 entries instantly, removing the need for sorting steps.

---

## 4. Operational Lifecycle Operations

### 4.1 Message Commit & Session Tracking
To achieve consistent writes across threads without racing, the system wraps data collection inside a multi-step query flow:
1.  **Payload Writing:** The message payload, origin details, and AI status flags populate the message table.
2.  **Parent Synchronization:** The system updates the conversation table's activity timestamp to match the current time, shifting its location in active visual queues.

### 4.2 Background Escalation Querying
The automated notification routine applies a layered sub-query design to protect network resources:
1.  **Status Check:** Isolates session clusters configured to the `open` state.
2.  **Chronological Filter:** Excludes records whose activity timestamp sits inside the 2-minute margin.
3.  **Sender Verification:** Executes a nested lookup isolating the absolute latest message in that specific session. If the sender's identity matches a role assigned to an `agent`, the escalation step aborts, preventing errors when a customer goes offline mid-conversation.
