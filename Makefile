# gymctl Makefile

# Variables
BINARY_NAME=gymctl
BUILD_DIR=build
CMD_PATH=cmd/gymctl
GO=go
GOFLAGS=-v

# Version info
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)/main.go

# Run the application
.PHONY: run
run:
	$(GO) run $(CMD_PATH)/main.go

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -cover ./...

# Generate test coverage report
.PHONY: coverage
coverage:
	@echo "Generating coverage report..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	$(GO) vet ./...

# Install dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Cross-compilation targets
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)/main.go

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)/main.go

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)/main.go

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Development workflow - format, vet, test, build
.PHONY: dev
dev: fmt vet test build

# CI workflow - comprehensive checks
.PHONY: ci
ci: deps fmt vet lint test build

# Docker variables
DOCKER_IMAGE ?= gymctl
DOCKER_TAG ?= latest
DOCKER_REGISTRY ?= ghcr.io/shart
DOCKER_FULL_IMAGE = $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-build-alpine
docker-build-alpine:
	@echo "Building minimal Alpine image..."
	docker build -f Dockerfile.alpine -t $(DOCKER_IMAGE):alpine .

.PHONY: docker-build-debian
docker-build-debian:
	@echo "Building full-featured Debian image..."
	docker build -f Dockerfile.debian -t $(DOCKER_IMAGE):debian .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -it --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v gymctl-data:/home/gymuser/.gym \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-push
docker-push:
	@echo "Pushing image to registry..."
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_FULL_IMAGE)
	docker push $(DOCKER_FULL_IMAGE)

.PHONY: compose-up
compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d --build

.PHONY: compose-down
compose-down:
	@echo "Stopping docker-compose services..."
	docker-compose down

.PHONY: compose-shell
compose-shell:
	@echo "Opening shell in gymctl container..."
	docker-compose exec gymctl bash

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Go Build Targets:"
	@echo "  all           - Build the project (default)"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the application"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  coverage      - Generate coverage report"
	@echo "  clean         - Remove build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  vet           - Vet code"
	@echo "  deps          - Download dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  tidy          - Tidy dependencies"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-darwin  - Build for macOS"
	@echo "  build-windows - Build for Windows"
	@echo "  build-all     - Build for all platforms"
	@echo "  dev           - Development workflow (fmt, vet, test, build)"
	@echo "  ci            - CI workflow (deps, fmt, vet, lint, test, build)"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build        - Build Docker image"
	@echo "  docker-build-alpine - Build minimal Alpine image"
	@echo "  docker-build-debian - Build full-featured Debian image"
	@echo "  docker-run          - Run Docker container interactively"
	@echo "  docker-push         - Push image to registry"
	@echo "  compose-up          - Start with docker-compose"
	@echo "  compose-down        - Stop docker-compose services"
	@echo "  compose-shell       - Get shell in running container"
	@echo ""
	@echo "  help          - Show this help message"