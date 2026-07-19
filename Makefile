.PHONY: all build run test clean docker-up docker-down

all: build

build:
	go build -o bin/server cmd/server/main.go
	cd web && bun run build

run-backend:
	go run cmd/server/main.go

run-frontend:
	cd web && bun run dev

test:
	go test ./... -v

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

clean:
	rm -rf bin/
