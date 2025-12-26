// Copyright (c) 2025 Archmagece
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
		Long: `Save and load development environment configurations.

This command helps you backup, restore, and manage various development
environment configurations including:
- Kubernetes configurations (kubeconfig)
- Docker configurations
- AWS configurations and credentials
- AWS profile management with SSO support
- Google Cloud (GCloud) configurations and credentials
- GCP project management and gcloud configurations
- Azure subscription management with multi-tenant support
- SSH configurations
- And more...

This is useful when setting up new development machines, switching between
projects, or maintaining consistent environments across multiple machines.

Examples:
  # Show status of all development environment services
  dev-env status

  # Launch interactive TUI dashboard
  dev-env tui

  # Switch all services to a named environment
  dev-env switch-all --env production

  # Save current kubeconfig
  dev-env kubeconfig save --name my-cluster

  # Manage AWS profiles with SSO support
  dev-env aws-profile list
  dev-env aws-profile switch production`,
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newTUICmd())
	cmd.AddCommand(newSwitchAllCmd())

	return cmd
}
