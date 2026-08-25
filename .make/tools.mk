# Makefile.tools - Tool Installation and Management for gzh-cli
# Development tools, linters, formatters, and utilities

# ==============================================================================
# Tool Configuration
# ==============================================================================

GOLANGCI_LINT_VERSION := v2.12.2
GOLANGCI_LINT_RELEASE_VERSION := $(patsubst v%,%,$(GOLANGCI_LINT_VERSION))
GOLANGCI_LINT_DIR := $(CURDIR)/bin/tools
GOLANGCI_LINT := $(GOLANGCI_LINT_DIR)/golangci-lint$(shell go env GOEXE)

# ==============================================================================
# Core Tool Installation
# ==============================================================================

.PHONY: install-tools install-format-tools install-analysis-tools install-goreleaser
.PHONY: install-golangci-lint install-pre-commit-tools install-docs-tools

install-tools: install-format-tools install-analysis-tools install-golangci-lint install-goreleaser ## install all development tools
	@echo -e "$(GREEN)✅ All development tools installed!$(RESET)"

install-format-tools: ## install advanced formatting tools
	@echo -e "$(CYAN)Installing formatting tools...$(RESET)"
	@which gofumpt > /dev/null || (echo "Installing gofumpt..." && go install mvdan.cc/gofumpt@latest)
	@which gci > /dev/null || (echo "Installing gci..." && go install github.com/daixiang0/gci@latest)
	@echo -e "$(GREEN)✅ All formatting tools installed!$(RESET)"

install-analysis-tools: ## install code analysis tools
	@echo -e "$(CYAN)Installing code analysis tools...$(RESET)"
	@command -v gocyclo >/dev/null 2>&1 || { echo "Installing gocyclo..." && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; }
	@command -v ineffassign >/dev/null 2>&1 || { echo "Installing ineffassign..." && go install github.com/gordonklaus/ineffassign@latest; }
	@command -v dupl >/dev/null 2>&1 || { echo "Installing dupl..." && go install github.com/mibk/dupl@latest; }
	@command -v staticcheck >/dev/null 2>&1 || { echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest; }
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	@echo -e "$(GREEN)✅ All analysis tools installed!$(RESET)"

install-golangci-lint: ## install the pinned golangci-lint v2 release
	@echo -e "$(CYAN)Ensuring golangci-lint $(GOLANGCI_LINT_VERSION)...$(RESET)"
	@mkdir -p "$(GOLANGCI_LINT_DIR)"
	@if [ ! -x "$(GOLANGCI_LINT)" ] || ! "$(GOLANGCI_LINT)" --version 2>/dev/null | grep -Fq "version $(GOLANGCI_LINT_RELEASE_VERSION) "; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		GOBIN="$(GOLANGCI_LINT_DIR)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@"$(GOLANGCI_LINT)" --version

install-goreleaser: ## install goreleaser
	@echo -e "$(CYAN)Installing goreleaser...$(RESET)"
	@go install github.com/goreleaser/goreleaser@latest
	@echo -e "$(GREEN)✅ goreleaser installed$(RESET)"

# ==============================================================================
# Mock and Generation Tools
# ==============================================================================

.PHONY: install-mock-tools generate-mocks clean-mocks regenerate-mocks

install-mock-tools: ## install mock generation tools
	@echo -e "$(CYAN)Installing mock generation tools...$(RESET)"
	@command -v mockgen >/dev/null 2>&1 || { echo "Installing mockgen..." && go install go.uber.org/mock/mockgen@latest; }
	@echo -e "$(GREEN)✅ Mock generation tools installed!$(RESET)"

generate-mocks: install-mock-tools ## generate all mock files using gomock
	@echo -e "$(CYAN)Generating mocks...$(RESET)"
	@echo "Generating GitHub interface mocks..."
	@if [ -f "pkg/github/interfaces.go" ]; then \
		mockgen -source=pkg/github/interfaces.go -destination=pkg/github/mocks/github_mocks.go -package=mocks; \
		echo "  ✅ GitHub mocks generated"; \
	else \
		echo "  ⚠️  pkg/github/interfaces.go not found"; \
	fi
	@echo "Generating filesystem interface mocks..."
	@if [ -f "internal/filesystem/interfaces.go" ]; then \
		mockgen -source=internal/filesystem/interfaces.go -destination=internal/filesystem/mocks/filesystem_mocks.go -package=mocks; \
		echo "  ✅ Filesystem mocks generated"; \
	else \
		echo "  ⚠️  internal/filesystem/interfaces.go not found"; \
	fi
	@echo "Generating HTTP client interface mocks..."
	@if [ -f "internal/httpclient/interfaces.go" ]; then \
		mockgen -source=internal/httpclient/interfaces.go -destination=internal/httpclient/mocks/httpclient_mocks.go -package=mocks; \
		echo "  ✅ HTTP client mocks generated"; \
	else \
		echo "  ⚠️  internal/httpclient/interfaces.go not found"; \
	fi
	@echo "Generating Git interface mocks..."
	@if [ -f "internal/git/interfaces.go" ]; then \
		mockgen -source=internal/git/interfaces.go -destination=internal/git/mocks/git_mocks.go -package=mocks; \
		echo "  ✅ Git mocks generated"; \
	else \
		echo "  ⚠️  internal/git/interfaces.go not found"; \
	fi
	@echo -e "$(GREEN)✅ Mock generation complete!$(RESET)"

clean-mocks: ## remove all generated mock files
	@echo -e "$(CYAN)Cleaning generated mocks...$(RESET)"
	@rm -f pkg/github/mocks/github_mocks.go
	@rm -f internal/filesystem/mocks/filesystem_mocks.go
	@rm -f internal/httpclient/mocks/httpclient_mocks.go
	@rm -f internal/git/mocks/git_mocks.go
	@echo -e "$(GREEN)✅ Mock cleanup complete!$(RESET)"

regenerate-mocks: clean-mocks generate-mocks ## clean and regenerate all mocks

# ==============================================================================
# Pre-commit and Git Hooks
# ==============================================================================

.PHONY: install-pre-commit-tools pre-commit-update

install-pre-commit-tools: ## install pre-commit and related tools
	@echo -e "$(CYAN)Installing pre-commit tools...$(RESET)"
	@command -v pre-commit >/dev/null 2>&1 || { echo -e "$(RED)pre-commit not found. Install with: pip install pre-commit$(RESET)"; }
	@echo -e "$(GREEN)✅ Pre-commit tools ready!$(RESET)"


# ==============================================================================
# Documentation Tools
# ==============================================================================

.PHONY: install-docs-tools

install-docs-tools: ## install documentation tools
	@echo -e "$(CYAN)Installing documentation tools...$(RESET)"
	@go install golang.org/x/perf/cmd/benchstat@latest
	@go install golang.org/x/tools/cmd/godoc@latest
	@which git-chglog >/dev/null 2>&1 || echo -e "$(YELLOW)Consider installing git-chglog for changelog generation$(RESET)"
	@which mkdocs >/dev/null 2>&1 || echo -e "$(YELLOW)Consider installing mkdocs for documentation: pip install mkdocs mkdocs-material$(RESET)"
	@echo -e "$(GREEN)✅ Documentation tools installed$(RESET)"

# ==============================================================================
# Security Tools
# ==============================================================================

.PHONY: install-security-tools

install-security-tools: ## install security analysis tools
	@echo -e "$(CYAN)Installing security tools...$(RESET)"
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	@echo -e "$(GREEN)✅ Security tools installed!$(RESET)"

# ==============================================================================
# Vulnerability Scanning
# ==============================================================================

.PHONY: install-vuln-tools

install-vuln-tools: ## install vulnerability scanning tools
	@echo -e "$(CYAN)Installing vulnerability scanning tools...$(RESET)"
	@echo "govulncheck is available as: go run golang.org/x/vuln/cmd/govulncheck@latest"
	@echo -e "$(GREEN)✅ Vulnerability tools ready!$(RESET)"

# ==============================================================================
# Tool Status and Information
# ==============================================================================

.PHONY: tools-status tools-info

tools-status: ## show installed tool status
	@echo -e "$(CYAN)Checking development tool status...$(RESET)"
	@echo ""
	@echo -e "$(GREEN)📦 Core Tools:$(RESET)"
	@printf "  %-20s " "go:"; go version 2>/dev/null | cut -d' ' -f3 || echo -e "$(RED)Not found$(RESET)"
	@printf "  %-20s " "git:"; git --version 2>/dev/null | cut -d' ' -f3 || echo -e "$(RED)Not found$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🔧 Build Tools:$(RESET)"
	@printf "  %-20s " "goreleaser:"; goreleaser --version 2>/dev/null | head -1 | awk '{print $3}' || echo -e "$(RED)Not installed$(RESET)"
	@echo ""
	@echo -e "$(GREEN)✨ Format Tools:$(RESET)"
	@printf "  %-20s " "gofumpt:"; gofumpt --version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"
	@printf "  %-20s " "gci:"; gci --version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🔍 Lint Tools:$(RESET)"
	@printf "  %-20s " "golangci-lint:"; golangci-lint --version 2>/dev/null | cut -d' ' -f4 || echo -e "$(RED)Not installed$(RESET)"
	@printf "  %-20s " "staticcheck:"; staticcheck -version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🛡️  Security Tools:$(RESET)"
	@printf "  %-20s " "gosec:"; gosec -version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🎭 Mock Tools:$(RESET)"
	@printf "  %-20s " "mockgen:"; mockgen --version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🎣 Git Hooks:$(RESET)"
	@printf "  %-20s " "pre-commit:"; pre-commit --version 2>/dev/null || echo -e "$(RED)Not installed$(RESET)"

tools-info: ## show comprehensive tool information
	@echo -e "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo -e "║                         $(YELLOW)Development Tools Information$(CYAN)                   ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo -e "$(RESET)"
	@echo -e "$(GREEN)🔧 Available Tool Categories:$(RESET)"
	@echo -e "  • $(CYAN)Format Tools$(RESET)        Code formatting (gofumpt, gci)"
	@echo -e "  • $(CYAN)Analysis Tools$(RESET)      Static analysis (staticcheck, gosec)"
	@echo -e "  • $(CYAN)Lint Tools$(RESET)          Code linting (golangci-lint)"
	@echo -e "  • $(CYAN)Build Tools$(RESET)         Build and release (goreleaser)"
	@echo -e "  • $(CYAN)Mock Tools$(RESET)          Mock generation (mockgen)"
	@echo -e "  • $(CYAN)Security Tools$(RESET)      Security scanning (gosec, govulncheck)"
	@echo -e "  • $(CYAN)Git Hooks$(RESET)           Pre-commit hooks and validation"
	@echo -e "  • $(CYAN)Documentation$(RESET)       Documentation tools (godoc, benchstat)"
	@echo ""
	@echo -e "$(GREEN)🚀 Quick Installation:$(RESET)"
	@echo -e "  $(CYAN)make install-tools$(RESET)        Install all development tools"
	@echo -e "  $(CYAN)make tools-status$(RESET)         Check current tool installation status"
	@echo ""
	@echo -e "$(GREEN)💡 Individual Categories:$(RESET)"
	@echo -e "  $(CYAN)make install-format-tools$(RESET)     Format tools only"
	@echo -e "  $(CYAN)make install-analysis-tools$(RESET)   Analysis tools only"
	@echo -e "  $(CYAN)make install-security-tools$(RESET)   Security tools only"
	@echo -e "  $(CYAN)make install-mock-tools$(RESET)       Mock generation tools only"
