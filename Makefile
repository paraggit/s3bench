.PHONY: build test docker clean lint fmt help

# Build variables
BINARY_NAME=s3-workload
VERSION?=v0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -w -s"

# Docker variables
DOCKER_REGISTRY?=ghcr.io
DOCKER_IMAGE?=$(DOCKER_REGISTRY)/paragkamble/s3-workload
DOCKER_TAG?=$(VERSION)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/s3-workload

build-local: ## Build the binary for local OS
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/s3-workload

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## Run tests and generate coverage report
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

docker: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .

docker-push: docker ## Build and push Docker image
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.txt coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

run-local: build-local ## Run locally with example config
	./bin/$(BINARY_NAME) --help

.DEFAULT_GOAL := help

