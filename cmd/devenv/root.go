// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package devenv

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for development environment management.
// This command is designed to be used directly or wrapped by a parent CLI.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev-env",
		Short: "Manage development environment configurations",
		Long: `Inspect and switch development environments for AWS, Google Cloud,
Azure, Docker, Kubernetes, and SSH.

Use status to inspect service state, tui to view the dashboard, and switch-all
to apply a named YAML environment across configured services.

Examples:
  # Show status of all development environment services
  dev-env status

  # Launch interactive TUI dashboard
  dev-env tui

  # Switch all services to a named environment
  dev-env switch-all --env production`,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newTUICmd())
	cmd.AddCommand(newSwitchAllCmd())

	return cmd
}
