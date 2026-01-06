.PHONY: build run test lint clean docker-build fmt vet test-e2e test-e2e-setup

# Build variables
BINARY_NAME=server
BUILD_DIR=bin
CMD_DIR=cmd/server

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Run the server
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run tests with race detection and coverage
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests and generate HTML coverage report
coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Build Docker image
docker-build:
	docker build -t openai-claude-proxy .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/ coverage.out coverage.html

# Setup e2e test dependencies (using uv)
test-e2e-setup:
	cd tests/e2e && uv venv && uv pip install -r requirements.txt

# Run e2e tests (requires server to be running)
test-e2e: build
	@echo "Starting server in background..."
	@./$(BUILD_DIR)/$(BINARY_NAME) & echo $$! > .server.pid
	@sleep 2
	@echo "Running e2e tests..."
	@cd tests/e2e && uv run pytest -v; TEST_EXIT=$$?; \
		kill `cat ../../.server.pid` 2>/dev/null; \
		rm -f ../../.server.pid; \
		exit $$TEST_EXIT

# Run all checks before commit
check: fmt vet lint test build
