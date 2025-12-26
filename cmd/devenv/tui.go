// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package devenv

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/tui"
)

// newTUICmd creates a new TUI command for interactive development environment management.
func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI dashboard for development environment management",
		Long: `Launch an interactive Terminal User Interface (TUI) dashboard for managing
development environments. This provides a real-time view of all development
services including AWS, GCP, Azure, Docker, Kubernetes, and SSH configurations.

The TUI includes:
- Real-time service status monitoring
- Interactive service management
- Environment switching capabilities
- Service logs and details view
- Quick actions and keyboard shortcuts

Navigation:
  ↑/k, ↓/j     Navigate up/down
  ←/h, →/l     Navigate left/right
  Enter        Select/confirm action
  Esc          Go back to previous view
  q            Quit (from dashboard)
  r            Refresh status
  s            Switch environment
  L            View logs
  P            Settings/preferences
  /            Search
  ?            Toggle help

Examples:
  # Launch the TUI dashboard
  dev-env tui

  # Launch TUI with verbose logging (for debugging)
  dev-env tui --verbose`,
		SilenceUsage: true,
		RunE:         runTUI,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging for debugging")

	return cmd
}

// runTUI executes the TUI command.
func runTUI(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Set up context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create TUI model
	model := tui.NewModel(ctx)

	// Configure tea options
	var opts []tea.ProgramOption
	// Enable alt screen for both verbose and normal operation
	opts = append(opts, tea.WithAltScreen())

	// Create and run the TUI program
	p := tea.NewProgram(model, opts...)

	// Handle interrupt signals gracefully
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	// Check if the program exited due to an error
	if m, ok := finalModel.(*tui.Model); ok {
		if verbose {
			fmt.Fprintf(os.Stderr, "TUI exited successfully\n")
		}
		_ = m // Use the final model if needed for cleanup
	}

	return nil
}
