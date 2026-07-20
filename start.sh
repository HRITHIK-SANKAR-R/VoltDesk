#!/bin/bash

# start.sh - Script to start the entire VoltDesk stack simultaneously

echo "🚀 Starting the VoltDesk stack..."

# 1. Start the PostgreSQL Database (Detached)
echo "📦 Starting PostgreSQL database via Docker..."
make docker-up

# Wait a few seconds for the database to accept connections
echo "⏳ Waiting for database to initialize..."
sleep 3

# 2. Start the Backend and Frontend in the background
echo "⚙️ Starting Go Backend..."
make run-backend &
BACKEND_PID=$!

echo "🌐 Starting Vite Frontend..."
make run-frontend &
FRONTEND_PID=$!

# Handle shutdown gracefully on Ctrl+C
trap 'echo "🛑 Shutting down VoltDesk stack..."; kill $BACKEND_PID; kill $FRONTEND_PID; make docker-down; exit 0' SIGINT SIGTERM

echo "✅ VoltDesk is running!"
echo "   - Frontend: http://localhost:5173"
echo "   - Backend: http://localhost:8081"
echo "Press Ctrl+C to stop the entire stack."

# Wait for background processes to keep the script running
wait $BACKEND_PID $FRONTEND_PID
