// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import (
	"testing"
)

// TestDefaultKeyMap tests the default keymap.
func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap

	// Test that essential keys are defined
	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit key should be defined")
	}

	if len(km.Up.Keys()) == 0 {
		t.Error("Up key should be defined")
	}

	if len(km.Down.Keys()) == 0 {
		t.Error("Down key should be defined")
	}

	if len(km.Refresh.Keys()) == 0 {
		t.Error("Refresh key should be defined")
	}

	if len(km.Help.Keys()) == 0 {
		t.Error("Help key should be defined")
	}
}

// TestKeyMap_ShortHelp tests ShortHelp method.
func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap
	bindings := km.ShortHelp()

	if len(bindings) == 0 {
		t.Error("ShortHelp() should return at least one binding")
	}
}

// TestKeyMap_FullHelp tests FullHelp method.
func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap
	bindings := km.FullHelp()

	if len(bindings) == 0 {
		t.Error("FullHelp() should return at least one binding group")
	}
}
