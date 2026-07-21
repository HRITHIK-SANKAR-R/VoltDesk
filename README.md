# ⚡ VoltDesk - Real-Time AI Support Engine

VoltDesk is a high-performance, production-grade Customer Support platform. It combines a real-time WebSocket chat interface with an autonomous AI Agent that instantly resolves customer queries, while giving human administrators complete oversight and the ability to seamlessly step in.

## 🏗️ Architecture

VoltDesk uses a modern, decoupled microservice architecture:
- **Frontend**: React (Vite), Bun, Tailwind CSS
- **Backend**: Go (Golang)
- **Database**: PostgreSQL 17
- **Message Broker**: Redis 7
- **AI Brain**: Google Gemini AI
- **Proxy/Web Server**: Nginx

---

## ⚙️ How it Works

VoltDesk operates on a highly concurrent, event-driven architecture. Here is the lifecycle of a support ticket:

1. **Authentication (OAuth 2.0)**:
   Users log in via Google OAuth. The Go backend securely encrypts their session using AES-GCM and stores it in an HTTP-only cookie. Administrative users (agents) are dynamically assigned elevated roles based on their email.
2. **WebSocket Connection**:
   When a user opens the chat widget, a persistent WebSocket connection is established with the Go backend.
3. **The Pub/Sub Engine**:
   Every chat message sent through the WebSocket is intercepted by the Go server, permanently stored in PostgreSQL, and instantly published to a Redis channel specific to that conversation using `msgpack` serialization.
4. **Autonomous AI Interception**:
   If a customer sends a message, a background Goroutine intercepts the payload and queries the Gemini AI. The AI's response is formatted, saved to the database under the AI's permanent UUID, and broadcast back to the WebSocket channel seamlessly.
5. **Human Override**:
   Agents monitor all active conversations on the Agent Dashboard. Because Redis acts as a distributed message broker, agents see the AI and customer chatting in real-time. The agent can type a message at any time, instantly broadcasting it to the customer.

---

## 🧠 Why it Works (Technical Decisions)

- **Go (Golang)**: Support chats require thousands of persistent WebSocket connections. Go's lightweight Goroutines and channel-based concurrency make it the perfect tool to handle massive connection pools without memory bloat.
- **Redis Pub/Sub**: By decoupling the WebSocket routing logic into Redis, the Go backend becomes horizontally scalable. You can spin up 10 API containers behind a load balancer, and Redis will ensure a message sent on Container A reaches the customer connected to Container B.
- **MsgPack**: Instead of standard JSON, internal backend-to-Redis communication is heavily serialized using MsgPack, reducing memory footprint and maximizing parse speeds.
- **Nginx Reverse Proxy**: To bypass complex CORS configurations and preflight `OPTIONS` requests, Nginx runs in front of both the React frontend and Go backend. It serves the static UI on port `80` while invisibly routing `/api` and `/ws` requests to the internal Go cluster.

---

## 🚀 Deployment Guide (Production)

The entire VoltDesk platform is heavily containerized and designed for a 1-click deployment using Docker. 

### Prerequisites
- A Linux VPS (DigitalOcean, AWS EC2, Render, etc.)
- `docker` and `docker-compose` installed.
- A Google Cloud Console project (for OAuth credentials).
- A Google Gemini API Key.

### 1. Environment Setup
Clone the repository and create a `.env` file in the root directory:

```env
# Database configurations
DATABASE_URL=postgres://postgres:password@postgres:5432/voltdesk?sslmode=disable
REDIS_URL=redis:6379

# Server configuration
PORT=8081
SESSION_SECRET=your-highly-secure-32-byte-aes-key

# Google OAuth Credentials
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_CALLBACK_URL=http://your-production-domain.com/api/auth/google/callback

# AI
GEMINI_API_KEY=your-gemini-key
```

### 2. Google OAuth Configuration
In your Google Cloud Console, ensure you have added your production domain to the OAuth consent screen:
- **Authorized JavaScript Origins**: `http://your-production-domain.com`
- **Authorized Redirect URIs**: `http://your-production-domain.com/api/auth/google/callback`

### 3. Launching the Stack
Run the following command to pull the images, bundle the React frontend, compile the Go binary, and launch the Nginx proxy:

```bash
docker-compose up -d --build
```

### 4. Admin Initialization
Upon first boot, the system dynamically registers the Autonomous AI identity. To access the Agent Dashboard, log into the web interface using the administrative email assigned in the `internal/models/models.go` role escalation block.

---

## 📁 Directory Structure
```
VoltDesk/
├── cmd/
│   └── server/          # Go application entrypoint
├── internal/
│   ├── ai/              # Gemini AI integration
│   ├── auth/            # Google OAuth & AES Session Management
│   ├── models/          # Postgres queries and schema models
│   └── websocket/       # Real-time connection hub and client routing
├── migrations/          # SQL database initialization scripts
├── web/                 # React frontend (Vite + Tailwind)
│   ├── src/             # Frontend components and WebSocket hooks
│   ├── Dockerfile       # Frontend build and Nginx serve instructions
│   └── nginx.conf       # Reverse proxy routing rules
├── docker-compose.yml   # Multi-container orchestration
└── Dockerfile           # Backend Go build instructions
```
