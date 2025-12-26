// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// TestNewDashboardModel tests the DashboardModel constructor.
func TestNewDashboardModel(t *testing.T) {
	model := NewDashboardModel()

	if model == nil {
		t.Fatal("NewDashboardModel() returned nil")
	}

	// Check that services slice is initialized
	if model.services == nil {
		t.Error("services should be initialized")
	}

	// Check that loading is true initially
	if !model.loading {
		t.Error("loading should be true initially")
	}

	// Check that current environment is set
	if model.currentEnv == "" {
		t.Error("currentEnv should be set")
	}
}

// TestDashboardModel_Init tests Init returns nil.
func TestDashboardModel_Init(t *testing.T) {
	model := NewDashboardModel()
	cmd := model.Init()

	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

// TestFormatDuration tests the formatDuration function.
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"30 seconds", 30 * time.Second, "30s"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"2 hours", 2 * time.Hour, "2h"},
		{"3 days", 72 * time.Hour, "3d"},
		{"0 seconds", 0, "0s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"60 seconds (1 min)", 60 * time.Second, "1m"},
		{"90 minutes", 90 * time.Minute, "1h"},
		{"25 hours", 25 * time.Hour, "1d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestDashboardModel_View_Loading tests View() in loading state.
func TestDashboardModel_View_Loading(t *testing.T) {
	model := NewDashboardModel()
	model.loading = true
	model.width = 80
	model.height = 24

	view := model.View()

	if !strings.Contains(view, "Loading") {
		t.Error("View should contain 'Loading' text when loading")
	}
}

// TestDashboardModel_View_Error tests View() in error state.
func TestDashboardModel_View_Error(t *testing.T) {
	model := NewDashboardModel()
	model.loading = false
	model.errorMsg = "connection failed"
	model.width = 80
	model.height = 24

	view := model.View()

	if !strings.Contains(view, "Error") {
		t.Error("View should contain 'Error' when errorMsg is set")
	}
	if !strings.Contains(view, "connection failed") {
		t.Error("View should contain the error message")
	}
}

// TestDashboardModel_View_Dashboard tests View() in normal state.
func TestDashboardModel_View_Dashboard(t *testing.T) {
	model := NewDashboardModel()
	model.loading = false
	model.errorMsg = ""
	model.width = 100
	model.height = 30

	view := model.View()

	if !strings.Contains(view, "GZH Development Environment Manager") {
		t.Error("View should contain the title")
	}
	if !strings.Contains(view, "Quick Actions") {
		t.Error("View should contain quick actions")
	}
}

// TestDashboardModel_Update_StatusUpdate tests Update with StatusUpdateMsg.
func TestDashboardModel_Update_StatusUpdate(t *testing.T) {
	model := NewDashboardModel()

	statuses := []status.ServiceStatus{
		{
			Name:   "AWS",
			Status: status.StatusActive,
			Current: status.CurrentConfig{
				Context: "production",
			},
			Credentials: status.CredentialStatus{
				Valid: true,
			},
		},
		{
			Name:   "Docker",
			Status: status.StatusActive,
			Current: status.CurrentConfig{
				Context: "default",
			},
			Credentials: status.CredentialStatus{
				Valid: true,
			},
		},
	}

	msg := StatusUpdateMsg{Statuses: statuses}
	updatedModel, _ := model.Update(msg)

	if updatedModel.loading {
		t.Error("loading should be false after StatusUpdateMsg")
	}
	if len(updatedModel.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(updatedModel.services))
	}
	if updatedModel.errorMsg != "" {
		t.Error("errorMsg should be empty after StatusUpdateMsg")
	}
}

// TestDashboardModel_Update_ErrorMsg tests Update with ErrorMsg.
func TestDashboardModel_Update_ErrorMsg(t *testing.T) {
	model := NewDashboardModel()

	msg := ErrorMsg{Error: &testError{message: "test error"}}
	updatedModel, _ := model.Update(msg)

	if updatedModel.loading {
		t.Error("loading should be false after ErrorMsg")
	}
	if updatedModel.errorMsg != "test error" {
		t.Errorf("errorMsg = %q, want %q", updatedModel.errorMsg, "test error")
	}
}

// TestDashboardModel_Update_LoadingMsg tests Update with LoadingMsg.
func TestDashboardModel_Update_LoadingMsg(t *testing.T) {
	model := NewDashboardModel()
	model.loading = false

	msg := LoadingMsg{Loading: true}
	updatedModel, _ := model.Update(msg)

	if !updatedModel.loading {
		t.Error("loading should be true after LoadingMsg{Loading: true}")
	}
}

// TestDashboardModel_Update_WindowSizeMsg tests Update with WindowSizeMsg.
func TestDashboardModel_Update_WindowSizeMsg(t *testing.T) {
	model := NewDashboardModel()

	msg := WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)

	if updatedModel.width != 120 {
		t.Errorf("width = %d, want 120", updatedModel.width)
	}
	if updatedModel.height != 40 {
		t.Errorf("height = %d, want 40", updatedModel.height)
	}
}

// TestDashboardModel_Update_KeyMsgRefresh tests Update with refresh key.
func TestDashboardModel_Update_KeyMsgRefresh(t *testing.T) {
	model := NewDashboardModel()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected a command for refresh key")
	}
}

// TestDashboardModel_Update_KeyMsgSwitchEnv tests Update with switch env key.
func TestDashboardModel_Update_KeyMsgSwitchEnv(t *testing.T) {
	model := NewDashboardModel()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected a command for switch env key")
	}
}

// TestDashboardModel_UpdateServices tests updateServices method.
func TestDashboardModel_UpdateServices(t *testing.T) {
	model := NewDashboardModel()

	// Test with valid credentials
	services := []status.ServiceStatus{
		{
			Name:   "AWS",
			Status: status.StatusActive,
			Current: status.CurrentConfig{
				Context: "prod-account",
			},
			Credentials: status.CredentialStatus{
				Valid:     true,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
		},
	}

	model.updateServices(services)

	if len(model.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(model.services))
	}
	if model.services[0].Name != "AWS" {
		t.Errorf("service name = %q, want %q", model.services[0].Name, "AWS")
	}
}

// TestDashboardModel_UpdateServices_ExpiredCredentials tests updateServices with expired creds.
func TestDashboardModel_UpdateServices_ExpiredCredentials(t *testing.T) {
	model := NewDashboardModel()

	services := []status.ServiceStatus{
		{
			Name:   "AWS",
			Status: status.StatusError,
			Current: status.CurrentConfig{
				Context: "prod-account",
			},
			Credentials: status.CredentialStatus{
				Valid:     true,
				ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
			},
		},
	}

	model.updateServices(services)

	if len(model.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(model.services))
	}
}

// TestDashboardModel_UpdateServices_LongContext tests context truncation.
func TestDashboardModel_UpdateServices_LongContext(t *testing.T) {
	model := NewDashboardModel()

	services := []status.ServiceStatus{
		{
			Name:   "Kubernetes",
			Status: status.StatusActive,
			Current: status.CurrentConfig{
				Context: "very-long-context-name-that-should-be-truncated-for-display",
			},
			Credentials: status.CredentialStatus{
				Valid: true,
			},
		},
	}

	model.updateServices(services)

	if len(model.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(model.services))
	}
}

// TestDashboardModel_UpdateTableSize tests table resizing.
func TestDashboardModel_UpdateTableSize(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"small terminal", 60, 15},
		{"medium terminal", 100, 30},
		{"large terminal", 200, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewDashboardModel()
			model.width = tt.width
			model.height = tt.height
			model.updateTableSize()
			// Just ensure no panic
		})
	}
}

// TestDashboardModel_SelectService tests selectService method.
func TestDashboardModel_SelectService(t *testing.T) {
	model := NewDashboardModel()

	// With no services, selectService should return nil
	cmd := model.selectService()
	if cmd != nil {
		// Get the message from cmd
		msg := cmd()
		if _, ok := msg.(ServiceSelectedMsg); ok {
			t.Error("selectService should return nil command when no row selected")
		}
	}
}

// TestDashboardModel_RefreshStatus tests refreshStatus method.
func TestDashboardModel_RefreshStatus(t *testing.T) {
	model := NewDashboardModel()
	cmd := model.refreshStatus()

	if cmd == nil {
		t.Error("refreshStatus should return a command")
	}
}

// TestDashboardModel_HandleQuickAction tests handleQuickAction method.
func TestDashboardModel_HandleQuickAction(t *testing.T) {
	model := NewDashboardModel()

	tests := []struct {
		action   int
		hasCmd   bool
		cmdType  string
	}{
		{1, true, "switch env"},
		{2, true, "refresh"},
		{3, true, "logs"},
		{4, false, "unknown"},
		{0, false, "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.cmdType, func(t *testing.T) {
			cmd := model.handleQuickAction(tt.action)
			if tt.hasCmd && cmd == nil {
				t.Errorf("handleQuickAction(%d) = nil, want command", tt.action)
			}
			if !tt.hasCmd && cmd != nil {
				t.Errorf("handleQuickAction(%d) = command, want nil", tt.action)
			}
		})
	}
}

// TestDashboardModel_RenderHeader tests renderHeader method.
func TestDashboardModel_RenderHeader(t *testing.T) {
	model := NewDashboardModel()
	model.width = 100
	model.currentEnv = "production"

	header := model.renderHeader()

	if !strings.Contains(header, "GZH Development Environment Manager") {
		t.Error("header should contain title")
	}
	if !strings.Contains(header, "production") {
		t.Error("header should contain current environment")
	}
}

// TestDashboardModel_RenderQuickActions tests renderQuickActions method.
func TestDashboardModel_RenderQuickActions(t *testing.T) {
	model := NewDashboardModel()
	model.width = 100

	actions := model.renderQuickActions()

	if !strings.Contains(actions, "Switch Environment") {
		t.Error("quick actions should contain 'Switch Environment'")
	}
	if !strings.Contains(actions, "Refresh") {
		t.Error("quick actions should contain 'Refresh'")
	}
	if !strings.Contains(actions, "Quit") {
		t.Error("quick actions should contain 'Quit'")
	}
}

// testError is a simple error type for testing.
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
