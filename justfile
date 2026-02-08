# LlamaOfFate Justfile

# Go parameters
gocmd := "go"
gofmt := "gofmt"
golint := "golangci-lint"

# Build parameters
binary_name := "llamaoffate"
binary_path := "./bin/" + binary_name
main_path := "./cmd/cli"

# Default recipe (shown when running `just`)
default: clean deps vet fmt build

# Run all validation checks
validate: vet fmtcheck lint test
    @echo "All validations passed!"

# Build the application
build:
    @echo "Building {{binary_name}}..."
    @mkdir -p bin
    {{gocmd}} build -o {{binary_path}} {{main_path}}
    @echo "Build complete: {{binary_path}}"

# Clean build artifacts
clean:
    @echo "Cleaning..."
    {{gocmd}} clean
    @rm -rf bin/
    @echo "Clean complete"

# Run tests
test:
    @echo "Running tests..."
    {{gocmd}} test -v ./...

# Run LLM evaluation tests (requires AZURE_API_ENDPOINT and AZURE_API_KEY)
test-llm:
    @echo "Running LLM evaluation tests..."
    @echo "Requires AZURE_API_ENDPOINT and AZURE_API_KEY environment variables"
    {{gocmd}} test -v -tags=llmeval ./test/llmeval/...

# Run go vet
vet:
    @echo "Running go vet..."
    {{gocmd}} vet ./...

# Format code
fmt:
    @echo "Formatting code..."
    {{gofmt}} -s -w .

# Check code formatting (fails if unformatted)
fmtcheck:
    @echo "Checking formatting..."
    @test -z "$({{gofmt}} -s -l .)" || (echo "Files not formatted:" && {{gofmt}} -s -l . && exit 1)

# Run golangci-lint
lint:
    @echo "Running golangci-lint..."
    {{golint}} run ./...

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    {{gocmd}} mod download
    {{gocmd}} mod tidy

# Build and run the application
run: build
    @echo "Running {{binary_name}}..."
    {{binary_path}}

# Build scenario generation eval tool
scenario-generator:
    @echo "Building scenario-generator..."
    @mkdir -p bin
    {{gocmd}} build -o ./bin/scenario-generator ./examples/scenario-generator
    @echo "Build complete: ./bin/scenario-generator"

# Build scene generation eval tool
scene-generator:
    @echo "Building scene-generator..."
    @mkdir -p bin
    {{gocmd}} build -o ./bin/scene-generator ./examples/scene-generator
    @echo "Build complete: ./bin/scene-generator"

# Build scenario walkthrough eval tool
scenario-walkthrough:
    @echo "Building scenario-walkthrough..."
    @mkdir -p bin
    {{gocmd}} build -o ./bin/scenario-walkthrough ./examples/scenario-walkthrough
    @echo "Build complete: ./bin/scenario-walkthrough"
