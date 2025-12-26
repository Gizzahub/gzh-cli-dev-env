// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"testing"
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
