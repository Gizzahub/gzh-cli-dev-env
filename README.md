# gzh-cli-dev-env

Development environment management library for gzh-cli.

## Overview

This library provides unified management of development environment configurations including:

- **AWS** - Profile, region, credentials management
- **GCP** - Project, account, service account management
- **Azure** - Subscription, tenant management
- **Docker** - Context management
- **Kubernetes** - Context, namespace management
- **SSH** - Configuration management

## Installation

```bash
go get github.com/gizzahub/gzh-cli-dev-env
```

## Usage

### Status Checking

```go
import (
    "context"

    "github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
    "github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

func main() {
    ctx := context.Background()

    // Create checkers
    checkers := []status.ServiceChecker{
        aws.NewChecker(),
        gcp.NewChecker(),
        docker.NewChecker(),
        kubernetes.NewChecker(),
    }

    // Create collector
    collector := status.NewStatusCollector(checkers, 30*time.Second)

    // Collect status
    opts := status.StatusOptions{
        Parallel:    true,
        CheckHealth: true,
    }

    statuses, err := collector.CollectAll(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Format output
    formatter := status.NewStatusTableFormatter(true)
    output, _ := formatter.Format(statuses)
    fmt.Print(output)
}
```

### Environment Switching

```go
import (
    "github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
    "github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
)

func main() {
    // Create switcher
    switcher := environment.NewEnvironmentSwitcher()

    // Register service switchers
    switcher.RegisterServiceSwitcher("aws", aws.NewSwitcher())

    // Define environment
    env := &environment.Environment{
        Name: "production",
        Services: map[string]environment.ServiceConfig{
            "aws": {
                AWS: &environment.AWSConfig{
                    Profile: "prod",
                    Region:  "us-west-2",
                },
            },
        },
    }

    // Switch
    opts := environment.SwitchOptions{
        Parallel:        true,
        RollbackOnError: true,
    }

    result, err := switcher.SwitchEnvironment(ctx, env, opts)
    if err != nil {
        log.Fatal(err)
    }
}
```

### TUI Dashboard

```go
import (
    "context"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/gizzahub/gzh-cli-dev-env/pkg/tui"
)

func main() {
    ctx := context.Background()
    model := tui.NewModel(ctx)

    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Package Structure

```
pkg/
├── environment/     # Core interfaces and environment switcher
├── status/          # Status collection and formatting
├── aws/             # AWS checker and switcher
├── gcp/             # GCP checker and switcher
├── azure/           # Azure checker and switcher
├── docker/          # Docker checker and switcher
├── kubernetes/      # Kubernetes checker and switcher
├── ssh/             # SSH checker and switcher
├── config/          # Configuration management
└── tui/             # Bubbletea TUI dashboard
```

## Key Interfaces

### ServiceChecker

```go
type ServiceChecker interface {
    Name() string
    CheckStatus(ctx context.Context) (*ServiceStatus, error)
    CheckHealth(ctx context.Context) (*HealthStatus, error)
}
```

### ServiceSwitcher

```go
type ServiceSwitcher interface {
    Name() string
    Switch(ctx context.Context, config interface{}) error
    GetCurrentState(ctx context.Context) (interface{}, error)
    Rollback(ctx context.Context, previousState interface{}) error
}
```

## License

MIT License - Copyright (c) 2025 Archmagece
