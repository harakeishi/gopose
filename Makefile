# Makefile for gopose

# 変数定義
BINARY_NAME=gopose
VERSION?=dev
BUILD_DIR=build
DOCKER_IMAGE=gopose
GO_VERSION=1.21

# Go関連の設定
GO_BUILD_FLAGS=-ldflags "-X main.version=$(VERSION)"
GO_TEST_FLAGS=-v -race -coverprofile=coverage.out

# デフォルトターゲット
.PHONY: all
all: clean test build

# ビルド関連
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

.PHONY: build-windows
build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

.PHONY: build-all
build-all: build build-linux build-windows

# テスト関連
.PHONY: test
test:
	@echo "Running tests..."
	go test $(GO_TEST_FLAGS) ./...

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test $(GO_TEST_FLAGS) ./internal/... ./pkg/...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test $(GO_TEST_FLAGS) ./test/integration/...

.PHONY: test-e2e
test-e2e:
	@echo "Running e2e tests..."
	go test $(GO_TEST_FLAGS) ./test/e2e/...

.PHONY: test-coverage
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# コード品質関連
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: mod-tidy
mod-tidy:
	@echo "Tidying go modules..."
	go mod tidy

.PHONY: check
check: fmt vet lint test

# 依存関係管理
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify

.PHONY: deps-upgrade
deps-upgrade:
	@echo "Upgrading dependencies..."
	go get -u ./...
	go mod tidy

# 開発用
.PHONY: dev
dev: deps build
	@echo "Development build complete"

.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: run-help
run-help: build
	./$(BUILD_DIR)/$(BINARY_NAME) --help

.PHONY: run-up
run-up: build
	./$(BUILD_DIR)/$(BINARY_NAME) up --dry-run

# クリーンアップ
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all generated files..."
	go clean -cache
	go clean -modcache

# Docker関連
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it $(DOCKER_IMAGE):$(VERSION)

# インストール関連
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# プロジェクト情報
.PHONY: info
info:
	@echo "Project: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Build directory: $(BUILD_DIR)"

# ヘルプ
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for all platforms"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-coverage - Generate test coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  check         - Run all checks (fmt, vet, lint, test)"
	@echo "  deps          - Install dependencies"
	@echo "  dev           - Development build"
	@echo "  run           - Build and run the binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  help          - Show this help message" 
