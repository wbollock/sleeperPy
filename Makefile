.PHONY: help install build run dev test clean fmt vet lint all

# Variables
BINARY_NAME=sleeperpy
GO=go
PORT?=8080
LOG_LEVEL?=debug
ADMIN_KEY?=changeme

# Default target
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

install: ## Install dependencies
	$(GO) mod download
	$(GO) mod verify

build: ## Build the application
	$(GO) build -o $(BINARY_NAME) .

run: build ## Build and run the application
	PORT=$(PORT) ./$(BINARY_NAME) -log=info

dev: build ## Run in development mode with debug logging
	PORT=$(PORT) ./$(BINARY_NAME) -log=$(LOG_LEVEL)

debug: build ## Run with debug logging (alias for dev)
	@echo "Admin dashboard: http://localhost:$(PORT)/admin?secret=$(ADMIN_KEY)"
	@echo "Admin API:       http://localhost:$(PORT)/admin/api?secret=$(ADMIN_KEY)"
	ADMIN_KEY=$(ADMIN_KEY) ADMIN_ALLOW_INSECURE=1 ADMIN_ALLOW_QUERY=1 PORT=$(PORT) ./$(BINARY_NAME) -log=debug

dev-watch: ## Run with hot reload (requires air: go install github.com/air-verse/air@latest)
	air

test-mode: build ## Run in test mode with mock data (use username: testuser)
	@echo "========================================="
	@echo "🧪 Starting server in TEST MODE"
	@echo "========================================="
	@echo "Open: http://localhost:$(PORT)"
	@echo "Username: testuser"
	@echo "========================================="
	@echo ""
	PORT=$(PORT) ./$(BINARY_NAME) -test -log=debug

test: ## Run tests
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	$(GO) tool cover -html=coverage.out

test-visual: ## Run tests and generate visual HTML outputs
	$(GO) test -v -run TestGenerateAllVisualOutputs
	@echo ""
	@echo "✓ Visual test outputs generated!"
	@echo "  Open test_output/index.html in your browser to view results"

test-clean: ## Clean test output directory
	rm -rf test_output
	rm -f coverage.out

test-view: test-visual ## Generate tests and open in browser (macOS/Linux)
	@if command -v xdg-open > /dev/null; then \
		xdg-open test_output/index.html; \
	elif command -v open > /dev/null; then \
		open test_output/index.html; \
	else \
		echo "Please open test_output/index.html in your browser"; \
	fi

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint (requires golangci-lint installation)
	golangci-lint run

tidy: ## Tidy go.mod
	$(GO) mod tidy

clean: ## Clean build artifacts and cache
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out

all: clean install fmt vet build test ## Run all checks and build

quick: ## Quick build and run without tests
	$(GO) run main.go -log=debug
