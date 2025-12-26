// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"testing"
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
