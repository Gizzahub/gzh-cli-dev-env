# Makefile - gzh-cli-dev-env Library
# Development Environment Management Library
# Modular Makefile structure

# ==============================================================================
# Project Configuration
# ==============================================================================

# Project metadata
projectname := gzh-cli-dev-env
executablename := gzh-devenv
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0-alpha")

# Go configuration
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org

# Colors for output
export CYAN := \033[36m
export GREEN := \033[32m
export YELLOW := \033[33m
export RED := \033[31m
export BLUE := \033[34m
export MAGENTA := \033[35m
export RESET := \033[0m

# ==============================================================================
# Include Modular Makefiles
# ==============================================================================

include .make/deps.mk
include .make/build.mk
include .make/test.mk
include .make/quality.mk
include .make/tools.mk
include .make/dev.mk
include .make/docker.mk

# ==============================================================================
# Help System
# ==============================================================================

.DEFAULT_GOAL := help

.PHONY: help

help: ## show main help menu
	@echo -e "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo -e "║                      $(MAGENTA)gzh-cli-dev-env Makefile Help$(CYAN)                        ║"
	@echo -e "║              $(YELLOW)Development Environment Management Library$(CYAN)                    ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo -e "$(RESET)"
	@echo -e "$(GREEN)📋 Quick Commands:$(RESET)"
	@echo -e "  $(CYAN)make build$(RESET)         Build the library"
	@echo -e "  $(CYAN)make test$(RESET)          Run all tests"
	@echo -e "  $(CYAN)make fmt$(RESET)           Format code"
	@echo -e "  $(CYAN)make lint$(RESET)          Run linter"
	@echo -e "  $(CYAN)make dev$(RESET)           Development workflow (fmt + lint + test)"
	@echo -e "  $(CYAN)make clean$(RESET)         Clean build artifacts"
	@echo ""
	@echo -e "$(GREEN)📦 Library Structure:$(RESET)"
	@echo "  pkg/environment    - Core switching interfaces and logic"
	@echo "  pkg/status         - Status checking subsystem"
	@echo "  pkg/aws            - AWS switcher and checker"
	@echo "  pkg/gcp            - GCP switcher and checker"
	@echo "  pkg/azure          - Azure switcher and checker"
	@echo "  pkg/docker         - Docker switcher and checker"
	@echo "  pkg/kubernetes     - Kubernetes switcher and checker"
	@echo "  pkg/ssh            - SSH switcher, checker, and parser"
	@echo "  pkg/config         - Configuration management"
	@echo "  pkg/tui            - Terminal UI dashboard"
	@echo ""
	@echo -e "$(BLUE)📖 Documentation: $(RESET)See CLAUDE.md for LLM context"

# ==============================================================================
# Project Information
# ==============================================================================

.PHONY: info

info: ## show project information
	@echo -e "$(CYAN)"
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo -e "║                      $(MAGENTA)gzh-cli-dev-env Project Info$(CYAN)                         ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo -e "$(RESET)"
	@echo -e "$(GREEN)📋 Project Details:$(RESET)"
	@echo -e "  Name:           $(YELLOW)$(projectname)$(RESET)"
	@echo -e "  Version:        $(YELLOW)$(VERSION)$(RESET)"
	@echo ""
	@echo -e "$(GREEN)🏗️  Build Environment:$(RESET)"
	@echo "  Go Version:     $$(go version | cut -d' ' -f3)"
	@echo ""
	@echo -e "$(GREEN)📁 Key Features:$(RESET)"
	@echo "  • Cloud platform switching (AWS, GCP, Azure)"
	@echo "  • Container environment management (Docker, Kubernetes)"
	@echo "  • SSH configuration management"
	@echo "  • Unified environment switching with rollback"
	@echo "  • Status monitoring and TUI dashboard"
