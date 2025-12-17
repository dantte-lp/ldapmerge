# ldapmerge Makefile
# Cross-compilation for Linux, Windows, macOS

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build settings
BINARY_NAME := ldapmerge
CMD_PATH := ./cmd/ldapmerge
BUILD_DIR := ./build

# Go settings
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X 'ldapmerge/internal/version.Version=$(VERSION)' \
	-X 'ldapmerge/internal/version.Commit=$(COMMIT)' \
	-X 'ldapmerge/internal/version.BuildDate=$(BUILD_DATE)'

# Platforms
PLATFORMS := linux/amd64 windows/amd64 darwin/arm64

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
CYAN := \033[0;36m
NC := \033[0m

.PHONY: all build clean test lint deps help version
.PHONY: build-linux build-windows build-darwin build-all

# Default target
all: build

# Show version info
version:
	@echo "$(CYAN)Version:$(NC)    $(VERSION)"
	@echo "$(CYAN)Commit:$(NC)     $(COMMIT)"
	@echo "$(CYAN)Build Date:$(NC) $(BUILD_DATE)"

# Install dependencies
deps:
	@echo "$(YELLOW)► Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

# Build for current platform
build: deps
	@echo "$(YELLOW)► Building $(BINARY_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

# Build for Linux amd64
build-linux: deps
	@echo "$(YELLOW)► Building for Linux amd64...$(NC)"
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64$(NC)"

# Build for Windows amd64
build-windows: deps
	@echo "$(YELLOW)► Building for Windows amd64...$(NC)"
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe$(NC)"

# Build for macOS ARM64 (Apple Silicon)
build-darwin: deps
	@echo "$(YELLOW)► Building for macOS ARM64...$(NC)"
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64$(NC)"

# Build for all platforms
build-all: clean build-linux build-windows build-darwin
	@echo ""
	@echo "$(GREEN)✓ All binaries built successfully!$(NC)"
	@echo ""
	@ls -lh $(BUILD_DIR)/
	@echo ""

# Run tests
test:
	@echo "$(YELLOW)► Running tests...$(NC)"
	$(GO) test -v -race -cover ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Run linter
lint:
	@echo "$(YELLOW)► Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi
	@echo "$(GREEN)✓ Lint complete$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)► Cleaning...$(NC)"
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)
	@echo "$(GREEN)✓ Cleaned$(NC)"

# Run the application
run: build
	@echo "$(YELLOW)► Running $(BINARY_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME)

# Install locally
install: build
	@echo "$(YELLOW)► Installing $(BINARY_NAME)...$(NC)"
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

# Create release archives
release: build-all
	@echo "$(YELLOW)► Creating release archives...$(NC)"
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "$(GREEN)✓ Release archives created$(NC)"
	@ls -lh $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip

# Help
help:
	@echo ""
	@echo "$(CYAN)ldapmerge - LDAP Configuration Merger for VMware NSX$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make [target]"
	@echo ""
	@echo "$(YELLOW)Targets:$(NC)"
	@echo "  $(GREEN)build$(NC)          Build for current platform"
	@echo "  $(GREEN)build-all$(NC)      Build for all platforms (Linux, Windows, macOS)"
	@echo "  $(GREEN)build-linux$(NC)    Build for Linux amd64"
	@echo "  $(GREEN)build-windows$(NC)  Build for Windows amd64"
	@echo "  $(GREEN)build-darwin$(NC)   Build for macOS ARM64"
	@echo "  $(GREEN)test$(NC)           Run tests"
	@echo "  $(GREEN)lint$(NC)           Run linter"
	@echo "  $(GREEN)clean$(NC)          Clean build artifacts"
	@echo "  $(GREEN)deps$(NC)           Install dependencies"
	@echo "  $(GREEN)run$(NC)            Build and run"
	@echo "  $(GREEN)install$(NC)        Install to /usr/local/bin"
	@echo "  $(GREEN)release$(NC)        Create release archives"
	@echo "  $(GREEN)version$(NC)        Show version info"
	@echo "  $(GREEN)help$(NC)           Show this help"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make build-all              # Build all binaries"
	@echo "  make VERSION=1.0.0 release  # Create release with version"
	@echo ""
