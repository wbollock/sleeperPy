.PHONY: help install build run dev test clean fmt vet lint all

# Variables
BINARY_NAME=sleeperpy
GO=go
PORT?=8080
LOG_LEVEL?=debug

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

dev-watch: ## Run with hot reload (requires air: go install github.com/air-verse/air@latest)
	air

test: ## Run tests
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	$(GO) tool cover -html=coverage.out

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
