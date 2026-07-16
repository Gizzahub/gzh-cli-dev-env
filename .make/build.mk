# Makefile.build - Build and Installation targets for gzh-cli
# Build, compilation, and installation management

# ==============================================================================
# Build Configuration
# ==============================================================================

# Go environment configuration
# Use system Go installation

# Detect OS-specific executable extension (e.g., .exe on Windows)
BINEXT := $(shell go env GOEXE)
BINARY := $(executablename)$(BINEXT)
GOBIN := $(shell go env GOBIN)
GOPATH := $(shell go env GOPATH)

# OS-specific path separator and binary install dir
ifeq ($(OS),Windows_NT)
SEP := \\\\
else
SEP := /
endif

USERBIN := $(HOME)$(SEP).local$(SEP)bin

ifeq ($(strip $(GOBIN)),)
  ifeq ($(OS),Windows_NT)
    BINDIR := $(GOPATH)$(SEP)bin
  else
    BINDIR := $(GOPATH)$(SEP)bin
  endif
else
  BINDIR := $(GOBIN)
endif

# ==============================================================================
# Build Targets
# ==============================================================================

.PHONY: build install run bootstrap clean release-dry-run release-snapshot release-check deploy

# This repo ships a library, not a binary: there is no main package and
# cmd/devenv is mounted by gzh-cli via NewRootCmd(). See PRODUCT.md Non-Goals.
build: ## compile all packages (library — produces no binary)
	@printf "$(CYAN)Building %s...$(RESET)\n" "$(projectname)"
	@go build ./...
	@printf "$(GREEN)Built %s successfully$(RESET)\n" "$(projectname)"


install: build ## install golang binary
	@printf "$(CYAN)Installing $(BINARY) $(VERSION) to %s$(RESET)\n" "$(BINDIR)$(SEP)$(BINARY)"
	@mkdir -p "$(BINDIR)"
	@mkdir -p "$(USERBIN)"
	@echo @mkdir -p "$(USERBIN)"
	@cp $(BINARY) "$(USERBIN)"/"$(BINARY)"
	@echo @cp $(BINARY) "$(USERBIN)"/"$(BINARY)"
	@mv $(BINARY) "$(BINDIR)"/
	@printf "$(GREEN)✅ Installed $(BINARY) $(VERSION) to %s$(RESET)\n" "$(BINDIR)$(SEP)$(BINARY)"
	@echo ""
	@printf "$(CYAN)Verifying installation...$(RESET)\n"
	@"$(BINDIR)$(SEP)$(BINARY)" --version || echo -e "$(YELLOW)Note: Binary installed but --version flag not implemented$(RESET)"
	@echo ""
	@printf "$(GREEN)🎉 Installation complete! Run '$(BINARY) --help' to get started.$(RESET)\n"

run: ## run the application (usage: make run [args...] or ARGS="args" make run)
	@echo -e "$(CYAN)Running application with version $(VERSION)...$(RESET)"
	@if [ "$(words $(MAKECMDGOALS))" -gt 1 ]; then \
		ARGS="$(filter-out run,$(MAKECMDGOALS))"; \
		echo -e "$(YELLOW)Arguments: $$ARGS$(RESET)"; \
		go run -ldflags "-X main.version=$(VERSION)" ./cmd/gzh-git $$ARGS; \
	elif [ -n "$(ARGS)" ]; then \
		echo -e "$(YELLOW)Arguments: $(ARGS)$(RESET)"; \
		go run -ldflags "-X main.version=$(VERSION)" ./cmd/gzh-git $(ARGS); \
	else \
		go run -ldflags "-X main.version=$(VERSION)" ./cmd/gzh-git; \
	fi

# Prevent make from interpreting arguments as targets
%:
	@:

bootstrap: ## install build dependencies
	@echo -e "$(CYAN)Installing build dependencies...$(RESET)"
	go generate -tags tools tools/tools.go
	@echo -e "$(GREEN)✅ Build dependencies installed$(RESET)"

clean: ## clean up environment
	@echo -e "$(CYAN)Cleaning up build artifacts...$(RESET)"
	@rm -rf coverage.out coverage.html dist/ $(executablename) $(BINARY)
	@rm -f $(shell go env GOPATH)/bin/$(executablename)
	@rm -f $(shell go env GOPATH)/bin/$(BINARY)
	@rm -f lint-report.json gosec-report.json
	@echo -e "$(GREEN)✅ Cleanup completed$(RESET)"

# ==============================================================================
# Release Targets
# ==============================================================================

release-dry-run: ## run goreleaser in dry-run mode
	@echo -e "$(CYAN)Running goreleaser in dry-run mode...$(RESET)"
	@command -v goreleaser >/dev/null 2>&1 || { echo -e "$(RED)goreleaser not found. Install with: make install-goreleaser$(RESET)"; exit 1; }
	@goreleaser release --snapshot --clean --skip=publish

release-snapshot: ## create a snapshot release
	@echo -e "$(CYAN)Creating snapshot release...$(RESET)"
	@command -v goreleaser >/dev/null 2>&1 || { echo -e "$(RED)goreleaser not found. Install with: make install-goreleaser$(RESET)"; exit 1; }
	@goreleaser release --snapshot --clean

release-check: ## check goreleaser configuration
	@echo -e "$(CYAN)Checking goreleaser configuration...$(RESET)"
	@command -v goreleaser >/dev/null 2>&1 || { echo -e "$(RED)goreleaser not found. Install with: make install-goreleaser$(RESET)"; exit 1; }
	@goreleaser check

deploy: release-dry-run ## alias for release-dry-run

# ==============================================================================
# Build Information
# ==============================================================================

.PHONY: build-info

build-info: ## show build information and current configuration
	@echo -e "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo -e "║                         $(YELLOW)Build Information$(CYAN)                              ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo -e "$(RESET)"
	@echo -e "$(GREEN)📋 Project Details:$(RESET)"
	@echo -e "  Name:           $(YELLOW)$(projectname)$(RESET)"
	@echo -e "  Executable:     $(YELLOW)$(BINARY)$(RESET)"
	@echo -e "  Version:        $(YELLOW)$(VERSION)$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🏗️  Build Environment:$(RESET)"
	@echo "  Go Version:     $$(go version | cut -d' ' -f3)"
	@echo -e "  GOPROXY:        $(GOPROXY)"
	@echo -e "  GOSUMDB:        $(GOSUMDB)"
	@echo "  GOPATH:         $$(go env GOPATH)"
	@echo "  GOROOT:         $$(go env GOROOT)"
	@echo ""
	@echo -e "$(GREEN)🎯 Build Targets:$(RESET)"
	@echo -e "  • $(CYAN)build$(RESET)               Build golang binary"
	@echo -e "  • $(CYAN)install$(RESET)             Install golang binary to GOPATH/bin"
	@echo -e "  • $(CYAN)run$(RESET)                 Run the application"
	@echo -e "  • $(CYAN)bootstrap$(RESET)           Install build dependencies"
	@echo -e "  • $(CYAN)clean$(RESET)               Clean up build artifacts"
	@echo -e "  • $(CYAN)release-dry-run$(RESET)     Test goreleaser configuration"
	@echo -e "  • $(CYAN)release-snapshot$(RESET)    Create snapshot release"
	@echo -e "  • $(CYAN)release-check$(RESET)       Check goreleaser configuration"
