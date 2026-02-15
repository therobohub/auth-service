.PHONY: help build test run docker-build docker-up docker-down clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o robohub-auth ./cmd/robohub-auth

test: ## Run tests
	go test ./... -v

test-coverage: ## Run tests with coverage
	go test ./... -cover

run: ## Run the service locally (requires ROBOHUB_JWT_SECRET)
	go run ./cmd/robohub-auth/main.go

docker-build: ## Build Docker image
	docker build -t robohub-auth:latest .

docker-up: ## Start service with docker-compose
	docker compose up --build

docker-down: ## Stop docker-compose services
	docker compose down

clean: ## Clean build artifacts
	rm -f robohub-auth
	go clean

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

lint-install: ## Install golangci-lint
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin)

fmt: ## Format code
	go fmt ./...

tidy: ## Tidy dependencies
	go mod tidy

.DEFAULT_GOAL := help
