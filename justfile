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
default: clean go-deps go-vet go-fmt build

# ─── Unified targets ────────────────────────────────────────────────

# Run all validation checks (Go + Web)
validate: go-validate web-validate
    @echo "All validations passed!"

# ─── Go targets ─────────────────────────────────────────────────────

# Run all Go validation checks
go-validate: go-vet go-fmtcheck go-lint go-test go-build-llmeval build-server build-mcpserver
    @echo "Go validations passed!"

# Build the CLI application
build:
    @echo "Building {{binary_name}}..."
    @mkdir -p bin
    {{gocmd}} build -o {{binary_path}} {{main_path}}
    @echo "Build complete: {{binary_path}}"

# Build the web server (includes embedded frontend from web/dist)
build-server: web-build
    @echo "Building web server..."
    @mkdir -p bin
    {{gocmd}} build -o ./bin/server ./cmd/server
    @echo "Build complete: ./bin/server"

# Build the MCP server
build-mcpserver:
    @echo "Building MCP server..."
    @mkdir -p bin
    {{gocmd}} build -o ./bin/mcpserver ./cmd/mcpserver
    @echo "Build complete: ./bin/mcpserver"

# Build and run the web server
serve: build-server
    @echo "Starting web server on :8080..."
    ./bin/server

# Run Go tests
go-test:
    @echo "Running Go tests..."
    {{gocmd}} test -v ./...

# Compile LLM evaluation tests without running them
go-build-llmeval:
    @echo "Checking llmeval tests compile..."
    {{gocmd}} test -tags=llmeval -count=0 ./test/llmeval/...

# Run LLM evaluation tests (requires AZURE_API_ENDPOINT and AZURE_API_KEY, or LLM_PROVIDER=ollama)
test-llm:
    @echo "Running LLM evaluation tests..."
    @echo "Requires AZURE_API_ENDPOINT and AZURE_API_KEY, or LLM_PROVIDER=ollama"
    {{gocmd}} test -v -tags=llmeval ./test/llmeval/...

# Run LLM eval tests and record results for flakiness tracking
test-llm-track:
    @echo "Running LLM eval tests with tracking..."
    {{gocmd}} test -v -json -tags=llmeval -timeout 10m ./test/llmeval/... \
      | {{gocmd}} run ./cmd/llmeval-tracker record

# Run a specific LLM eval test N times and record each run
test-llm-track-n test count="5":
    #!/usr/bin/env bash
    for i in $(seq 1 {{count}}); do
      echo "=== Run $i/{{count}} ==="
      {{gocmd}} test -v -json -tags=llmeval -timeout 10m \
        -run {{test}} ./test/llmeval/... \
        | {{gocmd}} run ./cmd/llmeval-tracker record
    done

# Show LLM eval stability report
test-llm-report:
    @{{gocmd}} run ./cmd/llmeval-tracker report

# Show only flaky LLM eval tests
test-llm-flaky:
    @{{gocmd}} run ./cmd/llmeval-tracker report --flaky

# Fetch llmeval results from CI and show a combined report
test-llm-fetch *args:
    ./scripts/llmeval-fetch-results.sh {{args}}

# Run go vet
go-vet:
    @echo "Running go vet..."
    {{gocmd}} vet ./...

# Format Go code
go-fmt:
    @echo "Formatting Go code..."
    {{gofmt}} -s -w .

# Check Go code formatting (fails if unformatted)
go-fmtcheck:
    @echo "Checking Go formatting..."
    @test -z "$({{gofmt}} -s -l .)" || (echo "Files not formatted:" && {{gofmt}} -s -l . && exit 1)

# Run golangci-lint
go-lint:
    @echo "Running golangci-lint..."
    {{golint}} run ./...

# Download Go dependencies
go-deps:
    @echo "Downloading Go dependencies..."
    {{gocmd}} mod download
    {{gocmd}} mod tidy

# Build and run the CLI application
run: build
    @echo "Running {{binary_name}}..."
    {{binary_path}}

# ─── Web targets ────────────────────────────────────────────────────

# Run all web validation checks
web-validate: web-lint web-test web-build
    @echo "Web validations passed!"

# Install web UI dependencies
web-install:
    @echo "Installing web dependencies..."
    cd web && npm install

# Start web UI dev server (Vite)
web-dev:
    @echo "Starting web dev server..."
    cd web && npm run dev

# Build web UI for production (set VITE_ENABLE_DEMOS=true to include demo pages)
web-build:
    @echo "Building web UI..."
    cd web && npm run build

# Run web UI tests
web-test:
    @echo "Running web tests..."
    cd web && npm test

# Run web UI linter
web-lint:
    @echo "Linting web UI..."
    cd web && npm run lint

# Start Storybook dev server
web-storybook:
    @echo "Starting Storybook..."
    cd web && npm run storybook

# Build Storybook static site
web-storybook-build:
    @echo "Building Storybook..."
    cd web && npm run build-storybook

# ─── Utilities ──────────────────────────────────────────────────────

# Clean build artifacts
clean:
    @echo "Cleaning..."
    {{gocmd}} clean
    @rm -rf bin/
    @rm -rf web/dist/
    @echo "Clean complete"

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

# Clean up old session logs, keeping the last n (default 5)
clean-sessions n="5":
    #!/usr/bin/env bash
    if [ ! -d "sessions" ]; then
        echo "No sessions directory found"
        exit 0
    fi
    total=$(find sessions -maxdepth 1 -type f -name "*.yaml" | wc -l)
    if [ "$total" -le "{{n}}" ]; then
        echo "Found $total session files (keeping all, threshold is {{n}})"
        exit 0
    fi
    to_delete=$((total - {{n}}))
    echo "Found $total session files, removing $to_delete oldest files (keeping {{n}})"
    find sessions -maxdepth 1 -type f -name "*.yaml" -printf '%T+ %p\0' | sort -z | head -z -n "$to_delete" | cut -z -d' ' -f2- | xargs -0 rm -v
    echo "Cleanup complete"
