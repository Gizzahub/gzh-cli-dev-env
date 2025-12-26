// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// Message types for the TUI application.
type (
	// TickMsg represents a timer tick.
	TickMsg struct {
		Time time.Time
	}

	// StatusUpdateMsg represents an update to service statuses.
	StatusUpdateMsg struct {
		Statuses []status.ServiceStatus
	}

	// ErrorMsg represents an error.
	ErrorMsg struct {
		Error error
	}

	// LoadingMsg represents loading state changes.
	LoadingMsg struct {
		Loading bool
		Message string
	}

	// NavigationMsg represents navigation between views.
	NavigationMsg struct {
		View ViewType
		Data interface{}
	}

	// ServiceSelectedMsg represents a service being selected.
	ServiceSelectedMsg struct {
		Service string
		Status  *status.ServiceStatus
	}

	// EnvironmentSwitchMsg represents environment switching.
	EnvironmentSwitchMsg struct {
		Environment string
		Success     bool
		Error       error
	}

	// RefreshMsg represents a manual refresh request.
	RefreshMsg struct{}

	// QuitMsg represents a quit request.
	QuitMsg struct{}

	// WindowSizeMsg represents terminal window size changes.
	WindowSizeMsg struct {
		Width  int
		Height int
	}

	// HelpToggleMsg represents help display toggle.
	HelpToggleMsg struct{}

	// SearchMsg represents search functionality.
	SearchMsg struct {
		Query   string
		Results []SearchResult
	}

	// FilterMsg represents filter functionality.
	FilterMsg struct {
		Filter string
		Active bool
	}
)

// SearchResult represents a search result item.
type SearchResult struct {
	Type        string // "service", "action", "setting"
	Name        string
	Description string
	Action      func() error
}

// ViewType represents different views in the TUI.
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewServiceDetail
	ViewEnvironmentSwitch
	ViewSettings
	ViewLogs
	ViewHelp
	ViewSearch
)

// String returns the string representation of a ViewType.
func (v ViewType) String() string {
	switch v {
	case ViewDashboard:
		return "Dashboard"
	case ViewServiceDetail:
		return "Service Detail"
	case ViewEnvironmentSwitch:
		return "Environment Switch"
	case ViewSettings:
		return "Settings"
	case ViewLogs:
		return "Logs"
	case ViewHelp:
		return "Help"
	case ViewSearch:
		return "Search"
	default:
		return "Unknown"
	}
}

// AppState represents the overall application state.
type AppState int

const (
	StateLoading AppState = iota
	StateDashboard
	StateServiceDetail
	StateEnvironmentSwitch
	StateSettings
	StateLogs
	StateError
	StateHelp
	StateSearch
)

// String returns the string representation of an AppState.
func (s AppState) String() string {
	switch s {
	case StateLoading:
		return "Loading"
	case StateDashboard:
		return "Dashboard"
	case StateServiceDetail:
		return "Service Detail"
	case StateEnvironmentSwitch:
		return "Environment Switch"
	case StateSettings:
		return "Settings"
	case StateLogs:
		return "Logs"
	case StateError:
		return "Error"
	case StateHelp:
		return "Help"
	case StateSearch:
		return "Search"
	default:
		return "Unknown"
	}
}
