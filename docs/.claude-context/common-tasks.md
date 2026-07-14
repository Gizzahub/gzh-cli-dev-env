# Common Tasks - gzh-cli-dev-env

## Architecture: Switcher Pattern

Each service (Docker, Azure, etc.) follows the same interface:

```go
type Switcher interface {
    Name() string
    Status() (Status, error)
    Start() error
    Stop() error
    Restart() error
}
```

### Package layout

```text
pkg/<service>/
├── switcher.go    # implements Switcher interface
├── checker.go     # health/status checks
└── doc.go
```

## Supported Services

| Package | Service | Status |
|---------|---------|--------|
| `pkg/docker` | Docker daemon | Start/Stop/Restart/Status |
| `pkg/azure` | Azure CLI / resources | Login/Logout/Status |
| `pkg/config` | Dev env config | Load/Save profiles |

## CLI Commands

| Command | Purpose |
|---------|---------|
| `devenv status` | Show all service statuses |
| `devenv switch docker start` | Start Docker |
| `devenv switch docker stop` | Stop Docker |
| `devenv switch azure login` | Azure CLI login |
| `devenv switch all status` | All services status |

## Adding a New Service Switcher

1. Create `pkg/<service>/switcher.go` implementing `Switcher`
2. Create `pkg/<service>/checker.go` for health checks
3. Add command in `cmd/devenv/service_switchers.go`
4. Register in the root command

## TUI Mode

The `devenv tui` command provides a terminal UI for managing services:

```go
// pkg/tui/ contains Bubble Tea model
model := tui.NewModel(switchers)
p := tea.NewProgram(model)
```

## Configuration

Dev env profiles stored under:

```
~/.config/gzh-manager/dev-env/config.yaml
```

Override with `GZH_CONFIG_DIR` environment variable.
