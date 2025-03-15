# Makefile

.PHONY: build run clean vet

# Variables
CLI_BINARY = textindex
WEB_BINARY = tiweb

# Build the CLI app
build:
	go build -o $(CLI_BINARY) ./cmd/cli

buildw:
	go build -o $(WEB_BINARY) ./cmd/web/

web: buildw
	./tiweb

# Build and run the CLI app
run: build
	./$(CLI_BINARY) -c index -i resources/t.txt -s 4096 -o index.idx

run-custom: build
	./$(CLI_BINARY) $(ARGS)

test:
	@echo "Running tests..."
	go test -v ./...

# Clean up the binary
clean:
	rm -f $(CLI_BINARY) *.idx $(WEB_BINARY)
	clear

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

vet:
	go vet ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build files"
	@echo "  test         - Run tests"
	@echo "  run          - Build and run the application"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  help         - Show this help message"