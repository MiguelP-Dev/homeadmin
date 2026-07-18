.PHONY: test run migrate build clean

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
