.PHONY: test run migrate build clean lint coverage templ docker-build docker-up docker-stop

# Run all tests
test:
	go test ./... -v

# Run the application
run:
	go run cmd/server/main.go

# Run database migrations
migrate:
	go run cmd/server/main.go -migrate

# Build the binary
build:
	go build -o bin/homeadmin cmd/server/main.go

# Clean build artifacts
clean:
	rm -rf bin/

# Static analysis
lint:
	go vet ./...

# Coverage report
coverage:
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

# Compile templ templates
templ:
	templ generate ./...

# Build Docker image
docker-build:
	docker build -t homeadmin .

# Start via docker-compose
docker-up:
	docker compose up -d

# Stop containers
docker-stop:
	docker compose down
