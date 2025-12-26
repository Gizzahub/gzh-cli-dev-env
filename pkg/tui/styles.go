// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

// Package tui provides a TUI dashboard for development environment management.
package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors (Nord-inspired).
var (
	ColorPrimary    = lipgloss.Color("#88C0D0")
	ColorSecondary  = lipgloss.Color("#81A1C1")
	ColorSuccess    = lipgloss.Color("#A3BE8C")
	ColorWarning    = lipgloss.Color("#EBCB8B")
	ColorError      = lipgloss.Color("#BF616A")
	ColorText       = lipgloss.Color("#ECEFF4")
	ColorSubtle     = lipgloss.Color("#4C566A")
	ColorBackground = lipgloss.Color("#2E3440")
	ColorBorder     = lipgloss.Color("#4C566A")
	ColorHighlight  = lipgloss.Color("#5E81AC")
)

// Base styles.
var (
	BaseStyle = lipgloss.NewStyle().
			Padding(1).
			Background(ColorBackground).
			Foreground(ColorText)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSubtle).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Padding(0, 1)
)

// Service status styles.
var (
	ServiceActiveStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	ServiceInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)

	ServiceWarningStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	ServiceErrorStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	ServiceUnknownStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)
)

// Table styles.
var (
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorBorder)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	TableSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorBackground).
				Background(ColorHighlight).
				Bold(true).
				Padding(0, 1)

	TableEvenRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3B4252"))

	TableOddRowStyle = lipgloss.NewStyle().
				Background(ColorBackground)
)

// Additional styles.
var (
	SpinnerStyle    = BaseStyle.Foreground(ColorPrimary)
	ErrorStyle      = BaseStyle.Foreground(ColorError).Bold(true)
	InfoStyle       = BaseStyle.Foreground(ColorPrimary).Bold(true)
	HelpHeaderStyle = BaseStyle.Foreground(ColorPrimary).Bold(true).Margin(1, 0)
)

// GetStatusIcon returns the appropriate icon for a service status.
func GetStatusIcon(status string) string {
	switch status {
	case "active", "connected", "running", "online":
		return "✅"
	case "inactive", "disconnected", "stopped", "offline":
		return "❌"
	case "warning", "degraded", "partial":
		return "⚠️"
	case "error", "failed", "critical":
		return "🔴"
	default:
		return "❓"
	}
}
