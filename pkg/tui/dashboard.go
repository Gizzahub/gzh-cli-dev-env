// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// DashboardModel represents the main dashboard view.
type DashboardModel struct {
	table      table.Model
	help       help.Model
	keymap     KeyMap
	services   []status.ServiceStatus
	lastUpdate time.Time
	width      int
	height     int
	currentEnv string
	loading    bool
	errorMsg   string
}

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel() *DashboardModel {
	// Create table columns
	columns := []table.Column{
		{Title: "Service", Width: 12},
		{Title: "Status", Width: 12},
		{Title: "Current", Width: 25},
		{Title: "Credentials", Width: 15},
		{Title: "", Width: 3},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	// Apply table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(ColorBackground).
		Background(ColorHighlight).
		Bold(false)
	t.SetStyles(s)

	return &DashboardModel{
		table:      t,
		help:       help.New(),
		keymap:     DefaultKeyMap,
		services:   []status.ServiceStatus{},
		lastUpdate: time.Now(),
		currentEnv: "production",
		loading:    true,
	}
}

// Init initializes the dashboard model.
func (m *DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the dashboard.
func (m *DashboardModel) Update(msg tea.Msg) (*DashboardModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			m.table, cmd = m.table.Update(msg)
		case key.Matches(msg, m.keymap.Down):
			m.table, cmd = m.table.Update(msg)
		case key.Matches(msg, m.keymap.Enter):
			return m, m.selectService()
		case key.Matches(msg, m.keymap.Refresh):
			return m, m.refreshStatus()
		case key.Matches(msg, m.keymap.SwitchEnv):
			return m, func() tea.Msg {
				return NavigationMsg{View: ViewEnvironmentSwitch}
			}
		case key.Matches(msg, m.keymap.ViewLogs):
			return m, func() tea.Msg {
				return NavigationMsg{View: ViewLogs}
			}
		case key.Matches(msg, m.keymap.ViewSettings):
			return m, func() tea.Msg {
				return NavigationMsg{View: ViewSettings}
			}
		case key.Matches(msg, m.keymap.Search):
			return m, func() tea.Msg {
				return NavigationMsg{View: ViewSearch}
			}
		case key.Matches(msg, m.keymap.QuickAction1):
			return m, m.handleQuickAction(1)
		case key.Matches(msg, m.keymap.QuickAction2):
			return m, m.handleQuickAction(2)
		case key.Matches(msg, m.keymap.QuickAction3):
			return m, m.handleQuickAction(3)
		default:
			m.table, cmd = m.table.Update(msg)
		}

	case StatusUpdateMsg:
		m.updateServices(msg.Statuses)
		m.loading = false
		m.errorMsg = ""
		m.lastUpdate = time.Now()

	case ErrorMsg:
		m.loading = false
		m.errorMsg = msg.Error.Error()

	case LoadingMsg:
		m.loading = msg.Loading

	case WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableSize()

	default:
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

// View renders the dashboard.
func (m *DashboardModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.errorMsg != "" {
		return m.renderError()
	}

	return m.renderDashboard()
}

// renderDashboard renders the main dashboard view.
func (m *DashboardModel) renderDashboard() string {
	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Service table
	tableView := m.table.View()
	b.WriteString(tableView)
	b.WriteString("\n")

	// Quick actions
	quickActions := m.renderQuickActions()
	b.WriteString(quickActions)
	b.WriteString("\n")

	// Help
	helpView := m.help.View(m.keymap)
	b.WriteString(helpView)

	return b.String()
}

// renderHeader renders the dashboard header.
func (m *DashboardModel) renderHeader() string {
	title := "GZH Development Environment Manager"
	env := fmt.Sprintf("Current Environment: %s", m.currentEnv)
	updated := fmt.Sprintf("Updated: %s", m.lastUpdate.Format("15:04:05"))

	titleStyle := TitleStyle.Width(m.width - 2).Align(lipgloss.Center)
	headerStyle := HeaderStyle.Width(m.width - 2)

	padding := m.width - len(env) - len(updated) - 4
	if padding < 0 {
		padding = 0
	}

	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		env,
		strings.Repeat(" ", padding),
		updated,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		headerStyle.Render(headerContent),
	)
}

// renderQuickActions renders the quick actions bar.
func (m *DashboardModel) renderQuickActions() string {
	actions := []string{
		"[1] Switch Environment",
		"[2] Refresh Status",
		"[3] View Logs",
		"[q] Quit",
	}

	secondRow := []string{
		"[s] Search",
		"[f] Filter",
		"[?] Help",
		"[Enter] Service Details",
	}

	style := FooterStyle.Width(m.width - 2)

	firstLine := "Quick Actions: " + strings.Join(actions, "  ")
	secondLine := strings.Join(secondRow, "  ")

	return style.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		firstLine,
		secondLine,
	))
}

// renderLoading renders the loading state.
func (m *DashboardModel) renderLoading() string {
	loadingText := "Loading development environment status..."
	spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	spinnerChar := string(spinner[int(time.Now().UnixNano()/100000000)%len(spinner)])

	content := fmt.Sprintf("%s %s", spinnerChar, loadingText)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		SpinnerStyle.Render(content),
	)
}

// renderError renders the error state.
func (m *DashboardModel) renderError() string {
	errorContent := fmt.Sprintf("Error: %s\n\nPress 'r' to retry or 'q' to quit", m.errorMsg)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		ErrorStyle.Render(errorContent),
	)
}

// updateServices updates the service list and table rows.
func (m *DashboardModel) updateServices(services []status.ServiceStatus) {
	m.services = services

	rows := make([]table.Row, len(services))
	for i, service := range services {
		statusIcon := GetStatusIcon(strings.ToLower(string(service.Status)))
		statusText := fmt.Sprintf("%s %s", statusIcon, string(service.Status))

		// Format current context
		current := service.Current.Context
		if len(current) > 22 {
			current = current[:19] + "..."
		}

		// Format credentials status
		var credStatus string
		if service.Credentials.Valid {
			credStatus = "✅ Valid"
			// Check if credentials are expiring soon
			if !service.Credentials.ExpiresAt.IsZero() {
				timeUntilExpiry := time.Until(service.Credentials.ExpiresAt)
				if timeUntilExpiry < 0 {
					credStatus = "❌ Expired"
				} else if timeUntilExpiry < 2*time.Hour {
					credStatus = fmt.Sprintf("⚠️ Expires %s", formatDuration(timeUntilExpiry))
				} else {
					credStatus = fmt.Sprintf("✅ Valid (%s)", formatDuration(timeUntilExpiry))
				}
			}
		} else {
			if service.Credentials.Warning != "" {
				credStatus = fmt.Sprintf("⚠️ %s", service.Credentials.Warning)
			} else {
				credStatus = "❌ Invalid"
			}
		}

		rows[i] = table.Row{
			service.Name,
			statusText,
			current,
			credStatus,
			"→",
		}
	}

	m.table.SetRows(rows)
}

// updateTableSize updates the table size based on terminal dimensions.
func (m *DashboardModel) updateTableSize() {
	if m.width < 80 {
		// Adjust column widths for smaller terminals
		columns := []table.Column{
			{Title: "Service", Width: 10},
			{Title: "Status", Width: 10},
			{Title: "Current", Width: 20},
			{Title: "Creds", Width: 12},
			{Title: "", Width: 2},
		}
		m.table.SetColumns(columns)
	}

	// Adjust table height
	availableHeight := m.height - 8 // Reserve space for header, footer, help
	if availableHeight < 5 {
		availableHeight = 5
	}
	if availableHeight > 15 {
		availableHeight = 15
	}

	m.table.SetHeight(availableHeight)
}

// selectService handles service selection.
func (m *DashboardModel) selectService() tea.Cmd {
	selectedRow := m.table.SelectedRow()
	if selectedRow == nil {
		return nil
	}

	serviceName := selectedRow[0]
	var selectedService *status.ServiceStatus

	for _, service := range m.services {
		if service.Name == serviceName {
			s := service
			selectedService = &s
			break
		}
	}

	return func() tea.Msg {
		return ServiceSelectedMsg{
			Service: serviceName,
			Status:  selectedService,
		}
	}
}

// refreshStatus triggers a status refresh.
func (m *DashboardModel) refreshStatus() tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{}
	}
}

// handleQuickAction handles quick action buttons.
func (m *DashboardModel) handleQuickAction(action int) tea.Cmd {
	switch action {
	case 1: // Switch Environment
		return func() tea.Msg {
			return NavigationMsg{View: ViewEnvironmentSwitch}
		}
	case 2: // Refresh Status
		return m.refreshStatus()
	case 3: // View Logs
		return func() tea.Msg {
			return NavigationMsg{View: ViewLogs}
		}
	default:
		return nil
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
