// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/azure"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/docker"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/gcp"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/kubernetes"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/ssh"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// Model represents the main TUI application model.
type Model struct {
	state       AppState
	currentView ViewType
	keymap      KeyMap
	help        help.Model
	width       int
	height      int

	// View models
	dashboardModel *DashboardModel

	// Status management
	statusCollector *status.StatusCollector
	lastUpdate      time.Time
	updateInterval  time.Duration

	// Application state
	ctx      context.Context
	quitting bool
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context) *Model {
	// Create all available service checkers
	checkers := []status.ServiceChecker{
		aws.NewChecker(),
		gcp.NewChecker(),
		azure.NewChecker(),
		docker.NewChecker(),
		kubernetes.NewChecker(),
		ssh.NewChecker(),
	}

	return &Model{
		state:           StateLoading,
		currentView:     ViewDashboard,
		keymap:          DefaultKeyMap,
		help:            help.New(),
		dashboardModel:  NewDashboardModel(),
		statusCollector: status.NewStatusCollector(checkers, 10*time.Second),
		updateInterval:  5 * time.Second,
		ctx:             ctx,
	}
}

// Init initializes the TUI application.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshStatus(),
		m.startUpdateTicker(),
		tea.EnterAltScreen,
	)
}

// Update handles all messages in the TUI.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.handleGlobalKeys(msg) {
			return m, tea.Quit
		}

		// Delegate to current view
		cmd := m.updateCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update all view models with new size
		cmd := m.updateCurrentView(WindowSizeMsg{Width: msg.Width, Height: msg.Height})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case TickMsg:
		// Periodic status update
		cmds = append(cmds, m.refreshStatus())
		cmds = append(cmds, m.startUpdateTicker())

	case StatusUpdateMsg:
		m.lastUpdate = time.Now()
		m.state = StateDashboard

		// Update current view with status data
		cmd := m.updateCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ErrorMsg:
		m.state = StateError
		cmd := m.updateCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case NavigationMsg:
		m.currentView = msg.View
		m.updateStateFromView()

	case ServiceSelectedMsg:
		m.currentView = ViewServiceDetail
		m.state = StateServiceDetail

	case RefreshMsg:
		cmds = append(cmds, m.refreshStatus())

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	default:
		// Delegate to current view
		cmd := m.updateCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view.
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye! 👋\n"
	}

	switch m.currentView {
	case ViewDashboard:
		return m.dashboardModel.View()
	case ViewServiceDetail:
		return m.renderServiceDetail()
	case ViewEnvironmentSwitch:
		return m.renderEnvironmentSwitch()
	case ViewSettings:
		return m.renderSettings()
	case ViewLogs:
		return m.renderLogs()
	case ViewHelp:
		return m.renderHelp()
	case ViewSearch:
		return m.renderSearch()
	default:
		return m.dashboardModel.View()
	}
}

// handleGlobalKeys handles global keyboard shortcuts.
func (m *Model) handleGlobalKeys(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.currentView == ViewDashboard {
			return true // Quit
		} else {
			// Navigate back to dashboard
			m.currentView = ViewDashboard
			m.state = StateDashboard
			return false
		}
	case "esc":
		if m.currentView != ViewDashboard {
			m.currentView = ViewDashboard
			m.state = StateDashboard
		}
		return false
	default:
		return false
	}
}

// updateCurrentView updates the current view with a message.
func (m *Model) updateCurrentView(msg tea.Msg) tea.Cmd {
	switch m.currentView {
	case ViewDashboard:
		var cmd tea.Cmd
		m.dashboardModel, cmd = m.dashboardModel.Update(msg)
		return cmd
	case ViewServiceDetail:
		return nil
	case ViewEnvironmentSwitch:
		return nil
	case ViewSettings:
		return nil
	case ViewLogs:
		return nil
	case ViewHelp:
		return nil
	case ViewSearch:
		return nil
	default:
		return nil
	}
}

// updateStateFromView updates the app state based on current view.
func (m *Model) updateStateFromView() {
	switch m.currentView {
	case ViewDashboard:
		m.state = StateDashboard
	case ViewServiceDetail:
		m.state = StateServiceDetail
	case ViewEnvironmentSwitch:
		m.state = StateEnvironmentSwitch
	case ViewSettings:
		m.state = StateSettings
	case ViewLogs:
		m.state = StateLogs
	case ViewHelp:
		m.state = StateHelp
	case ViewSearch:
		m.state = StateSearch
	}
}

// refreshStatus refreshes the development environment status.
func (m *Model) refreshStatus() tea.Cmd {
	return func() tea.Msg {
		options := status.StatusOptions{
			Parallel:    true,
			CheckHealth: true,
			Timeout:     10 * time.Second,
		}

		statuses, err := m.statusCollector.CollectAll(m.ctx, options)
		if err != nil {
			return ErrorMsg{Error: err}
		}

		return StatusUpdateMsg{Statuses: statuses}
	}
}

// startUpdateTicker starts the periodic update ticker.
func (m *Model) startUpdateTicker() tea.Cmd {
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

// Placeholder view implementations.

func (m *Model) renderServiceDetail() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		InfoStyle.Render("Service Detail View\n(Coming Soon)\n\nPress 'esc' to go back"),
	)
}

func (m *Model) renderEnvironmentSwitch() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		InfoStyle.Render("Environment Switch View\n(Coming Soon)\n\nPress 'esc' to go back"),
	)
}

func (m *Model) renderSettings() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		InfoStyle.Render("Settings View\n(Coming Soon)\n\nPress 'esc' to go back"),
	)
}

func (m *Model) renderLogs() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		InfoStyle.Render("Logs View\n(Coming Soon)\n\nPress 'esc' to go back"),
	)
}

func (m *Model) renderHelp() string {
	helpContent := `GZH Development Environment Manager - Help

Navigation:
  ↑/k, ↓/j     Navigate up/down
  ←/h, →/l     Navigate left/right
  Enter        Select/confirm
  Esc          Go back
  q            Quit (from dashboard)

Views:
  s            Switch environment
  L            View logs
  P            Settings/preferences
  ?            Toggle help

Actions:
  r            Refresh status
  /            Search
  f            Filter
  1,2,3        Quick actions

Press 'esc' to go back to dashboard`

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		HelpHeaderStyle.Render(helpContent),
	)
}

func (m *Model) renderSearch() string {
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		InfoStyle.Render("Search View\n(Coming Soon)\n\nPress 'esc' to go back"),
	)
}
