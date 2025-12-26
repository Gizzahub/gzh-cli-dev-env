// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keyboard shortcuts for the TUI.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Enter        key.Binding
	Back         key.Binding
	Quit         key.Binding
	Help         key.Binding
	Refresh      key.Binding
	Search       key.Binding
	Filter       key.Binding
	SwitchEnv    key.Binding
	ViewLogs     key.Binding
	ViewSettings key.Binding
	QuickAction1 key.Binding
	QuickAction2 key.Binding
	QuickAction3 key.Binding
}

// DefaultKeyMap provides the default keyboard shortcuts.
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/confirm"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r", "ctrl+r"),
		key.WithHelp("r", "refresh"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	SwitchEnv: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "switch environment"),
	),
	ViewLogs: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "view logs"),
	),
	ViewSettings: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "preferences/settings"),
	),
	QuickAction1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "quick action 1"),
	),
	QuickAction2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "quick action 2"),
	),
	QuickAction3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "quick action 3"),
	),
}

// ShortHelp returns key bindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns key bindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},                  // navigation
		{k.Enter, k.Back, k.Quit, k.Help},                // actions
		{k.Refresh, k.Search, k.Filter},                  // utilities
		{k.SwitchEnv, k.ViewLogs, k.ViewSettings},        // views
		{k.QuickAction1, k.QuickAction2, k.QuickAction3}, // quick actions
	}
}

// Enabled returns whether the keymap is enabled.
func (k KeyMap) Enabled() bool {
	return true
}
