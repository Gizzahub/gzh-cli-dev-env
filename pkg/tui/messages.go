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
		Data any
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
	// ViewDashboard is the main dashboard view.
	ViewDashboard ViewType = iota
	// ViewServiceDetail is the service detail view.
	ViewServiceDetail
	// ViewEnvironmentSwitch is the environment switch view.
	ViewEnvironmentSwitch
	// ViewSettings is the settings view.
	ViewSettings
	// ViewLogs is the logs view.
	ViewLogs
	// ViewHelp is the help view.
	ViewHelp
	// ViewSearch is the search view.
	ViewSearch
)

const (
	dashboardViewName            = "Dashboard"
	serviceDetailViewName        = "Service Detail"
	environmentSwitchViewName    = "Environment Switch"
	settingsViewName             = "Settings"
	logsViewName                 = "Logs"
	helpViewName                 = "Help"
	searchViewName               = "Search"
	unknownViewName              = "Unknown"
)

// String returns the string representation of a ViewType.
func (v ViewType) String() string {
	switch v {
	case ViewDashboard:
		return dashboardViewName
	case ViewServiceDetail:
		return serviceDetailViewName
	case ViewEnvironmentSwitch:
		return environmentSwitchViewName
	case ViewSettings:
		return settingsViewName
	case ViewLogs:
		return logsViewName
	case ViewHelp:
		return helpViewName
	case ViewSearch:
		return searchViewName
	default:
		return unknownViewName
	}
}

// AppState represents the overall application state.
type AppState int

const (
	// StateLoading is the loading state.
	StateLoading AppState = iota
	// StateDashboard is the dashboard state.
	StateDashboard
	// StateServiceDetail is the service detail state.
	StateServiceDetail
	// StateEnvironmentSwitch is the environment switch state.
	StateEnvironmentSwitch
	// StateSettings is the settings state.
	StateSettings
	// StateLogs is the logs state.
	StateLogs
	// StateError is the error state.
	StateError
	// StateHelp is the help state.
	StateHelp
	// StateSearch is the search state.
	StateSearch
)

// String returns the string representation of an AppState.
func (s AppState) String() string {
	switch s {
	case StateLoading:
		return "Loading"
	case StateDashboard:
		return dashboardViewName
	case StateServiceDetail:
		return serviceDetailViewName
	case StateEnvironmentSwitch:
		return environmentSwitchViewName
	case StateSettings:
		return settingsViewName
	case StateLogs:
		return logsViewName
	case StateError:
		return "Error"
	case StateHelp:
		return helpViewName
	case StateSearch:
		return searchViewName
	default:
		return unknownViewName
	}
}
