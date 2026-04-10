# Variables
BINARY_NAME=agent
DOCKER_COMPOSE=docker-compose
PROTOC=protoc
GOPATH_BIN=$(shell go env GOPATH)/bin

# Add GOPATH/bin to PATH for the current shell session in Makefile
export PATH := $(PATH):$(GOPATH_BIN):/opt/homebrew/bin

.PHONY: all build gen up down restart logs test clean help install-deps

all: gen build test

## Help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

## Install dependencies: Install protoc plugins
install-deps: ## Install protoc-gen-go and protoc-gen-go-grpc
	@echo "Installing gRPC plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

## Generate: Rebuild gRPC code from proto files
gen: ## Generate Go code and tidy dependencies
	@echo "Generating gRPC code..."
	$(PROTOC) --go_out=. --go-grpc_out=. proto/data_agent.proto
	@echo "Tidying go.mod..."
	go mod tidy

## Build: Build agent binary
build: ## Build the agent binary
	@echo "Building agent..."
	go build -o bin/$(BINARY_NAME) cmd/agent/main.go

## Docker: Manage infrastructure
up: ## Start all services in docker (detached)
	$(DOCKER_COMPOSE) up --build -d

down: ## Stop all services in docker
	$(DOCKER_COMPOSE) down

restart: down up ## Restart all services

logs: ## View logs from docker containers
	$(DOCKER_COMPOSE) logs -f

## Test: Run project tests
test: ## Run all tests
	go test -v ./internal/...

## Clean: Remove binaries and temporary files
clean: ## Remove build artifacts
	rm -rf bin/
	find . -name ".DS_Store" -delete

## Run: Run agent locally
run-agent: build ## Build and run agent locally
	./bin/$(BINARY_NAME) --url 'amqp://guest:guest@localhost:5672/' --interval 2
