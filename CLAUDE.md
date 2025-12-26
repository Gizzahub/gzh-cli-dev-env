# CLAUDE.md

This file provides LLM-optimized guidance for Claude Code when working with this repository.

---

## Quick Start (30s scan)

**Library**: `gzh-cli-dev-env` (Development Environment Management)
**Architecture**: Interface-driven design (follows gzh-cli-git pattern)
**Go Version**: 1.24+
**Main Branch**: `master`

Core principle: ServiceSwitcher interface for unified cloud/container/SSH environment switching.

---

## Top 10 Commands

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `make build` | Build library | After changes |
| `make test` | Run all tests | Pre-commit validation |
| `make fmt && make lint` | Format + lint | Before every commit |
| `make dev` | Quick dev cycle | Rapid iteration |
| `make cover` | Coverage report | Check coverage |
| `go test ./pkg/... -v` | Test specific package | Focused testing |
| `make clean` | Clean artifacts | Fresh start |
| `make info` | Show project info | Quick reference |

---

## Directory Structure

```
.
├── pkg/
│   ├── environment/       # Core interfaces + switching logic
│   │   ├── interfaces.go  # ServiceSwitcher interface
│   │   ├── types.go       # Environment, ServiceConfig types
│   │   ├── switcher.go    # EnvironmentSwitcher implementation
│   │   └── dependency.go  # DependencyResolver
│   ├── status/            # Status checking subsystem
│   │   ├── interfaces.go  # ServiceChecker interface
│   │   ├── collector.go   # StatusCollector
│   │   └── formatter.go   # Output formatters
│   ├── aws/               # AWS-specific implementations
│   ├── gcp/               # GCP-specific implementations
│   ├── azure/             # Azure-specific implementations
│   ├── docker/            # Docker-specific implementations
│   ├── kubernetes/        # Kubernetes-specific implementations
│   ├── ssh/               # SSH-specific implementations
│   ├── config/            # Configuration save/load
│   └── tui/               # Terminal UI dashboard
├── internal/
│   ├── exec/              # Command execution utilities
│   └── testutil/          # Test helpers and mocks
├── cmd/gzh-devenv/        # Optional standalone CLI
├── go.mod
├── Makefile
└── CLAUDE.md
```

---

## Core Interfaces

### ServiceSwitcher (pkg/environment/interfaces.go)

```go
type ServiceSwitcher interface {
    Name() string
    Switch(ctx context.Context, config interface{}) error
    GetCurrentState(ctx context.Context) (interface{}, error)
    Rollback(ctx context.Context, previousState interface{}) error
}
```

Implementations: AWSSwitcher, GCPSwitcher, AzureSwitcher, DockerSwitcher, KubernetesSwitcher, SSHSwitcher

### ServiceChecker (pkg/status/interfaces.go)

```go
type ServiceChecker interface {
    Name() string
    CheckStatus(ctx context.Context) (*ServiceStatus, error)
    CheckHealth(ctx context.Context) (*HealthStatus, error)
}
```

---

## Absolute Rules (DO/DON'T)

### DO
- ✅ Use interfaces for all service implementations
- ✅ Run `make fmt && make lint` before every commit
- ✅ Maintain 80%+ test coverage for core logic
- ✅ Keep pkg/ dependencies minimal (stdlib preferred)
- ✅ Use context.Context for all operations

### DON'T
- ❌ Add external dependencies to pkg/ without review
- ❌ Put CLI-specific code in pkg/ (that goes in cmd/)
- ❌ Skip error handling for cloud API calls
- ❌ Store credentials in code or config

---

## Integration with gzh-cli

This library is consumed by gzh-cli as a wrapper:

```go
// In gzh-cli/cmd/dev_env_wrapper.go
import (
    "github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
    "github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
)

func newDevEnvCmd() *cobra.Command {
    switcher := environment.NewEnvironmentSwitcher()
    switcher.Register(aws.NewSwitcher())
    // ...
}
```

---

## Git Commit Format

```
{type}({scope}): {description}

Model: claude-{model}
Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types**: feat, fix, docs, refactor, test, chore
**Scope**: environment, status, aws, gcp, azure, docker, k8s, ssh, config, tui

---

## Context Documentation

| Guide | Purpose |
|-------|---------|
| [Architecture Guide](docs/.claude-context/architecture-guide.md) | Integration pattern, extensions |
| [Testing Guide](docs/.claude-context/testing-guide.md) | Test organization, mocking |

---

**Last Updated**: 2025-12-26
**Status**: Initial scaffolding
