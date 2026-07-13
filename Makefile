.PHONY: help build run test test-cover test-integration bench vet lint tidy docker-build compose-up compose-down clean

APP        := tsc_forex
PKG        := github.com/tsc-forex/tsc_forex
VERSION    ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
  -X $(PKG)/internal/utils.Version=$(VERSION) \
  -X $(PKG)/internal/utils.Commit=$(COMMIT) \
  -X $(PKG)/internal/utils.BuildTime=$(BUILD_TIME)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Build the server binary
	go build -ldflags="$(LDFLAGS)" -o bin/server ./cmd/server

run: ## Run the server locally
	APP_CONFIG_FILE=configs/config.yaml go run ./cmd/server

test: ## Run unit tests
	go test ./... -count=1 -timeout 120s

test-cover: ## Run tests with coverage report
	go test ./... -count=1 -covermode=atomic -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html

test-integration: ## Run integration tests (needs Postgres)
	go test -tags=integration ./... -count=1 -timeout 120s

bench: ## Run benchmarks
	go test ./tests/benchmark/ -bench=. -benchmem -run=^$$

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy modules
	go mod tidy

docker-build: ## Build Docker image
	docker build -f tsc_docker/Dockerfile -t $(APP):$(VERSION) \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) .

compose-up: ## Start full stack (postgres, redis, app x2, nginx, prometheus, grafana)
	docker compose -f tsc_docker/docker-compose.yml up -d --build

compose-down: ## Stop the stack
	docker compose -f tsc_docker/docker-compose.yml down

clean: ## Remove build artifacts
	rm -rf bin dist coverage.out coverage.html
