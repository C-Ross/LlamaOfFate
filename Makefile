# LlamaOfFate Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=llamaoffate
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/cli

.PHONY: all build clean test test-llm vet fmt lint deps validate help

# Default target
all: clean deps vet fmt build

# Run all validation checks
validate: vet fmt lint test
	@echo "All validations passed!"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf bin/
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run LLM evaluation tests (requires AZURE_API_ENDPOINT and AZURE_API_KEY)
test-llm:
	@echo "Running LLM evaluation tests..."
	@echo "Requires AZURE_API_ENDPOINT and AZURE_API_KEY environment variables"
	$(GOTEST) -v -tags=llmeval ./test/llmeval/...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	$(GOLINT) run ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH)

# Show help
help:
	@echo "Available targets:"
	@echo "  all      - Clean, get deps, vet, format, and build"
	@echo "  build    - Build the application"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  test-llm - Run LLM evaluation tests (requires Azure credentials)"
	@echo "  vet      - Run go vet"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run golangci-lint"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  validate - Run all validation checks (vet, fmt, lint, test)"
	@echo "  run      - Build and run the application"
	@echo "  help     - Show this help message"
