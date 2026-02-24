# ghent Makefile — command interface for all operations
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := gh-ghent
MODULE := github.com/indrasvat/gh-ghent
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

# Directories
BIN_DIR := bin
COVERAGE_DIR := coverage
GH_EXT_DIR := $(HOME)/.local/share/gh/extensions/$(BINARY_NAME)

# Tools
GOLANGCI_LINT := golangci-lint
GOTESTSUM := $(shell command -v gotestsum 2> /dev/null)

# Colors
COLOR_RESET := \033[0m
COLOR_GREEN := \033[32m
COLOR_BLUE := \033[34m

# ─── Build ───────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build binary
	@printf "$(COLOR_BLUE)>> Building $(BINARY_NAME)...$(COLOR_RESET)\n"
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/ghent
	@printf "$(COLOR_GREEN)>> Built $(BIN_DIR)/$(BINARY_NAME)$(COLOR_RESET)\n"

.PHONY: install
install: build ## Install as gh extension (symlink into gh extensions dir)
	@printf "$(COLOR_BLUE)>> Installing $(BINARY_NAME) as gh extension...$(COLOR_RESET)\n"
	@mkdir -p $(GH_EXT_DIR)
	@ln -sf $(CURDIR)/$(BIN_DIR)/$(BINARY_NAME) $(GH_EXT_DIR)/$(BINARY_NAME)
	@printf "$(COLOR_GREEN)>> Installed: gh ghent$(COLOR_RESET)\n"

.PHONY: clean
clean: ## Remove build artifacts
	@printf "$(COLOR_BLUE)>> Cleaning...$(COLOR_RESET)\n"
	rm -rf $(BIN_DIR) $(COVERAGE_DIR)
	@printf "$(COLOR_GREEN)>> Clean$(COLOR_RESET)\n"

# ─── Test ────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run unit tests
	@printf "$(COLOR_BLUE)>> Running tests...$(COLOR_RESET)\n"
ifdef GOTESTSUM
	gotestsum --format pkgname-and-test-fails --format-icons hivis -- -race -shuffle=on ./...
else
	go test -race -shuffle=on ./...
endif
	@printf "$(COLOR_GREEN)>> Tests passed$(COLOR_RESET)\n"

.PHONY: test-race
test-race: ## Run tests with race detector (verbose)
	go test -race -shuffle=on -v ./...

.PHONY: coverage
coverage: ## Generate coverage report
	@mkdir -p $(COVERAGE_DIR)
	go test -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@printf "$(COLOR_GREEN)>> Coverage report: $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)\n"

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -race -shuffle=on -tags=integration ./...

.PHONY: test-binary
test-binary: build ## L3: Run binary execution tests
	@printf "$(COLOR_BLUE)>> Running binary tests...$(COLOR_RESET)\n"
	bash scripts/test-binary.sh
	@printf "$(COLOR_GREEN)>> Binary tests passed$(COLOR_RESET)\n"

.PHONY: test-visual
test-visual: build ## L4: Run visual tests (iterm2-driver)
	@printf "$(COLOR_BLUE)>> Running visual tests...$(COLOR_RESET)\n"
	bash scripts/verify-visual-tests.sh
	@printf "$(COLOR_GREEN)>> Visual tests passed$(COLOR_RESET)\n"

.PHONY: test-agent
test-agent: build ## L5: Run agent workflow tests
	@printf "$(COLOR_BLUE)>> Running agent workflow tests...$(COLOR_RESET)\n"
	bash scripts/test-agent-workflow.sh
	@printf "$(COLOR_GREEN)>> Agent tests passed$(COLOR_RESET)\n"

.PHONY: test-all
test-all: test test-integration test-binary test-visual test-agent ## Run all test levels

# ─── Lint & Format ───────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint
	@printf "$(COLOR_BLUE)>> Linting...$(COLOR_RESET)\n"
	$(GOLANGCI_LINT) run ./...
	@printf "$(COLOR_GREEN)>> Lint passed$(COLOR_RESET)\n"

.PHONY: lint-fix
lint-fix: ## Auto-fix lint issues
	$(GOLANGCI_LINT) run --fix ./...

.PHONY: fmt
fmt: ## Format code with gofumpt
	@printf "$(COLOR_BLUE)>> Formatting...$(COLOR_RESET)\n"
	$(GOLANGCI_LINT) fmt ./...
	@printf "$(COLOR_GREEN)>> Formatted$(COLOR_RESET)\n"

.PHONY: vet
vet: ## Run go vet
	@printf "$(COLOR_BLUE)>> Vetting...$(COLOR_RESET)\n"
	go vet ./...
	@printf "$(COLOR_GREEN)>> Vet passed$(COLOR_RESET)\n"

# ─── Dependencies ────────────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: verify
verify: ## Run go mod verify
	go mod verify

# ─── CI ──────────────────────────────────────────────────────────────────────

.PHONY: ci
ci: lint test vet ## Full CI pipeline (lint + test + vet)
	@printf "$(COLOR_GREEN)>> CI passed$(COLOR_RESET)\n"

.PHONY: ci-fast
ci-fast: fmt vet test ## Quick CI (fmt + vet + test)
	@printf "$(COLOR_GREEN)>> CI-fast passed$(COLOR_RESET)\n"

# ─── Tools ───────────────────────────────────────────────────────────────────

.PHONY: tools
tools: ## Install development tools
	@printf "$(COLOR_BLUE)>> Installing tools...$(COLOR_RESET)\n"
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0
	go install gotest.tools/gotestsum@latest
	go install github.com/evilmartians/lefthook@latest
	@printf "$(COLOR_GREEN)>> Tools installed$(COLOR_RESET)\n"

.PHONY: hooks
hooks: ## Install git hooks via lefthook
	lefthook install

# ─── Info ────────────────────────────────────────────────────────────────────

.PHONY: version
version: ## Show version info
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
