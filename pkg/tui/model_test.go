// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewModel tests the Model constructor.
func TestNewModel(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	if model == nil {
		t.Fatal("NewModel() returned nil")
	}

	if model.state != StateLoading {
		t.Errorf("Initial state = %v, want StateLoading", model.state)
	}

	if model.currentView != ViewDashboard {
		t.Errorf("Initial view = %v, want ViewDashboard", model.currentView)
	}

	if model.statusCollector == nil {
		t.Error("statusCollector should be initialized")
	}

	if model.dashboardModel == nil {
		t.Error("dashboardModel should be initialized")
	}
}

// TestModel_Init tests the Init method.
func TestModel_Init(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init() should return a batch command")
	}
}

// TestAppState_Values tests AppState constants.
func TestAppState_Values(t *testing.T) {
	states := []AppState{
		StateLoading,
		StateDashboard,
		StateServiceDetail,
		StateError,
	}

	for _, state := range states {
		if state < 0 {
			t.Errorf("State %v should be non-negative", state)
		}
	}
}

// TestViewType_Values tests ViewType constants.
func TestViewType_Values(t *testing.T) {
	views := []ViewType{
		ViewDashboard,
		ViewServiceDetail,
		ViewSettings,
		ViewHelp,
	}

	for _, view := range views {
		if view < 0 {
			t.Errorf("View %v should be non-negative", view)
		}
	}
}

// TestAppState_String tests AppState String method.
func TestAppState_String(t *testing.T) {
	tests := []struct {
		state    AppState
		expected string
	}{
		{StateLoading, "Loading"},
		{StateDashboard, "Dashboard"},
		{StateError, "Error"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("AppState(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

// TestViewType_String tests ViewType String method.
func TestViewType_String(t *testing.T) {
	tests := []struct {
		view     ViewType
		expected string
	}{
		{ViewDashboard, "Dashboard"},
		{ViewServiceDetail, "Service Detail"},
		{ViewHelp, "Help"},
	}

	for _, tt := range tests {
		if got := tt.view.String(); got != tt.expected {
			t.Errorf("ViewType(%d).String() = %q, want %q", tt.view, got, tt.expected)
		}
	}
}

// TestModel_View tests the View method.
func TestModel_View(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test quitting view
	model.quitting = true
	view := model.View()
	if view != "Goodbye! 👋\n" {
		t.Errorf("View() when quitting = %q, want goodbye message", view)
	}

	// Reset and test dashboard view
	model.quitting = false
	model.currentView = ViewDashboard
	view = model.View()
	if view == "" {
		t.Error("View() should return non-empty string for dashboard")
	}
}

// TestModel_View_AllViews tests View for all view types.
func TestModel_View_AllViews(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)
	model.width = 100
	model.height = 30

	views := []ViewType{
		ViewDashboard,
		ViewServiceDetail,
		ViewEnvironmentSwitch,
		ViewSettings,
		ViewLogs,
		ViewHelp,
		ViewSearch,
	}

	for _, viewType := range views {
		t.Run(viewType.String(), func(t *testing.T) {
			model.currentView = viewType
			view := model.View()
			if view == "" {
				t.Errorf("View() for %v should not be empty", viewType)
			}
		})
	}
}

// TestModel_HandleGlobalKeys tests handleGlobalKeys method.
func TestModel_HandleGlobalKeys(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	tests := []struct {
		name        string
		key         string
		currentView ViewType
		shouldQuit  bool
	}{
		{"q from dashboard", "q", ViewDashboard, true},
		{"ctrl+c from dashboard", "ctrl+c", ViewDashboard, true},
		{"q from service detail", "q", ViewServiceDetail, false},
		{"esc from service detail", "esc", ViewServiceDetail, false},
		{"esc from dashboard", "esc", ViewDashboard, false},
		{"other key", "a", ViewDashboard, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.currentView = tt.currentView
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}
			if tt.key == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			result := model.handleGlobalKeys(msg)
			if result != tt.shouldQuit {
				t.Errorf("handleGlobalKeys(%q) = %v, want %v", tt.key, result, tt.shouldQuit)
			}
		})
	}
}

// TestModel_UpdateStateFromView tests updateStateFromView method.
func TestModel_UpdateStateFromView(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	tests := []struct {
		view     ViewType
		expected AppState
	}{
		{ViewDashboard, StateDashboard},
		{ViewServiceDetail, StateServiceDetail},
		{ViewEnvironmentSwitch, StateEnvironmentSwitch},
		{ViewSettings, StateSettings},
		{ViewLogs, StateLogs},
		{ViewHelp, StateHelp},
		{ViewSearch, StateSearch},
	}

	for _, tt := range tests {
		t.Run(tt.view.String(), func(t *testing.T) {
			model.currentView = tt.view
			model.updateStateFromView()
			if model.state != tt.expected {
				t.Errorf("state = %v, want %v", model.state, tt.expected)
			}
		})
	}
}

// TestModel_Update_KeyMsg tests Update with key messages.
func TestModel_Update_KeyMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)
	model.currentView = ViewDashboard

	// Test regular key that doesn't quit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	newModel, _ := model.Update(msg)
	if newModel == nil {
		t.Error("Update should return model")
	}
}

// TestModel_Update_WindowSizeMsg tests Update with window size.
func TestModel_Update_WindowSizeMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := model.Update(msg)

	m := newModel.(*Model)
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

// TestModel_Update_NavigationMsg tests Update with navigation.
func TestModel_Update_NavigationMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := NavigationMsg{View: ViewSettings}
	newModel, _ := model.Update(msg)

	m := newModel.(*Model)
	if m.currentView != ViewSettings {
		t.Errorf("currentView = %v, want ViewSettings", m.currentView)
	}
}

// TestModel_Update_ServiceSelectedMsg tests Update with service selection.
func TestModel_Update_ServiceSelectedMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := ServiceSelectedMsg{Service: "AWS"}
	newModel, _ := model.Update(msg)

	m := newModel.(*Model)
	if m.currentView != ViewServiceDetail {
		t.Errorf("currentView = %v, want ViewServiceDetail", m.currentView)
	}
	if m.state != StateServiceDetail {
		t.Errorf("state = %v, want StateServiceDetail", m.state)
	}
}

// TestModel_Update_RefreshMsg tests Update with refresh message.
func TestModel_Update_RefreshMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := RefreshMsg{}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("RefreshMsg should produce a command")
	}
}

// TestModel_Update_QuitMsg tests Update with quit message.
func TestModel_Update_QuitMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := QuitMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(*Model)
	if !m.quitting {
		t.Error("quitting should be true after QuitMsg")
	}
	if cmd == nil {
		t.Error("QuitMsg should produce tea.Quit command")
	}
}

// TestModel_Update_ErrorMsg tests Update with error message.
func TestModel_Update_ErrorMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	testErr := &modelTestError{msg: "test error"}
	msg := ErrorMsg{Error: testErr}
	newModel, _ := model.Update(msg)

	m := newModel.(*Model)
	if m.state != StateError {
		t.Errorf("state = %v, want StateError", m.state)
	}
}

// TestModel_Update_TickMsg tests Update with tick message.
func TestModel_Update_TickMsg(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	msg := TickMsg{Time: time.Now()}
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("TickMsg should produce commands")
	}
}

// TestModel_UpdateCurrentView tests updateCurrentView for different views.
func TestModel_UpdateCurrentView(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	views := []ViewType{
		ViewDashboard,
		ViewServiceDetail,
		ViewEnvironmentSwitch,
		ViewSettings,
		ViewLogs,
		ViewHelp,
		ViewSearch,
	}

	for _, view := range views {
		t.Run(view.String(), func(t *testing.T) {
			model.currentView = view
			// Just ensure no panic
			model.updateCurrentView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		})
	}
}

// modelTestError is a simple error type for testing.
type modelTestError struct {
	msg string
}

func (e *modelTestError) Error() string {
	return e.msg
}
