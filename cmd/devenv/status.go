// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package devenv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/azure"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/docker"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/gcp"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/kubernetes"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/ssh"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// newStatusCmd creates the dev-env status command.
func newStatusCmd() *cobra.Command {
	var (
		services    []string
		format      string
		checkHealth bool
		watch       bool
		timeout     time.Duration
		noColor     bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show development environment status",
		Long: `Display the current status of all development environment services.

This command shows the status of various development environment services:
- AWS: Current profile, region, and credential status
- GCP: Current project, account, and credential status
- Azure: Current subscription and credential status (if available)
- Docker: Current context and daemon status
- Kubernetes: Current context, namespace, and cluster connectivity
- SSH: SSH agent status and loaded keys

The command provides color-coded status indicators, credential expiration
warnings, and optional health checks for detailed service validation.

Examples:
  # Show status of all services
  dev-env status

  # Show status of specific services only
  dev-env status --service aws,kubernetes

  # Show status with detailed health checks
  dev-env status --check-health

  # Output status in JSON format
  dev-env status --format json

  # Watch status in real-time (updates every 30 seconds)
  dev-env status --watch

  # Show status without colors (for scripting)
  dev-env status --no-color`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusCmd(services, format, checkHealth, watch, timeout, !noColor)
		},
	}

	cmd.Flags().StringSliceVarP(&services, "service", "s", nil, "Services to check (aws,gcp,azure,docker,kubernetes,ssh)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table,json,yaml)")
	cmd.Flags().BoolVar(&checkHealth, "check-health", false, "Perform detailed health checks")
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch mode - continuously update status")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for status checks")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	return cmd
}

// runStatusCmd executes the status command.
func runStatusCmd(services []string, format string, checkHealth, watch bool, timeout time.Duration, useColor bool) error {
	ctx := context.Background()

	// Create service checkers
	checkers := createServiceCheckers(services)
	if len(checkers) == 0 {
		return fmt.Errorf("no valid services specified")
	}

	// Create status collector
	collector := status.NewStatusCollector(checkers, timeout)

	// Create formatter
	formatter, err := createFormatter(format, useColor)
	if err != nil {
		return fmt.Errorf("invalid format: %w", err)
	}

	if watch {
		return runWatchMode(ctx, collector, formatter, checkHealth, timeout)
	}

	return runSingleCheck(ctx, collector, formatter, checkHealth)
}

// createServiceCheckers creates the appropriate service checkers.
func createServiceCheckers(services []string) []status.ServiceChecker {
	var checkers []status.ServiceChecker

	// If no services specified, use all available services
	if len(services) == 0 {
		services = []string{"aws", "gcp", "azure", "docker", "kubernetes", "ssh"}
	}

	serviceSet := make(map[string]bool)
	for _, service := range services {
		serviceSet[strings.ToLower(strings.TrimSpace(service))] = true
	}

	if serviceSet["aws"] {
		checkers = append(checkers, aws.NewChecker())
	}
	if serviceSet["gcp"] {
		checkers = append(checkers, gcp.NewChecker())
	}
	if serviceSet["azure"] {
		checkers = append(checkers, azure.NewChecker())
	}
	if serviceSet["docker"] {
		checkers = append(checkers, docker.NewChecker())
	}
	if serviceSet["kubernetes"] || serviceSet["k8s"] {
		checkers = append(checkers, kubernetes.NewChecker())
	}
	if serviceSet["ssh"] {
		checkers = append(checkers, ssh.NewChecker())
	}

	return checkers
}

// createFormatter creates the appropriate output formatter.
func createFormatter(format string, useColor bool) (status.StatusFormatter, error) {
	switch strings.ToLower(format) {
	case "table":
		return status.NewStatusTableFormatter(useColor), nil
	case "json":
		return status.NewStatusJSONFormatter(true), nil
	case "yaml", "yml":
		return status.NewStatusYAMLFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: table, json, yaml)", format)
	}
}

// runSingleCheck performs a single status check.
func runSingleCheck(ctx context.Context, collector *status.StatusCollector, formatter status.StatusFormatter, checkHealth bool) error {
	options := status.StatusOptions{
		CheckHealth: checkHealth,
		Parallel:    true,
	}

	statuses, err := collector.CollectAll(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to collect status: %w", err)
	}

	output, err := formatter.Format(statuses)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(output)
	return nil
}

// runWatchMode runs the status command in watch mode.
func runWatchMode(ctx context.Context, collector *status.StatusCollector, formatter status.StatusFormatter, checkHealth bool, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Clear screen function
	clearScreen := func() {
		fmt.Print("\033[2J\033[H")
	}

	options := status.StatusOptions{
		CheckHealth: checkHealth,
		Parallel:    true,
	}

	for {
		clearScreen()

		// Show current time
		fmt.Printf("Last updated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

		statuses, err := collector.CollectAll(ctx, options)
		if err != nil {
			fmt.Printf("Error collecting status: %v\n", err)
		} else {
			output, err := formatter.Format(statuses)
			if err != nil {
				fmt.Printf("Error formatting output: %v\n", err)
			} else {
				fmt.Print(output)
			}
		}

		fmt.Println("\nPress Ctrl+C to exit watch mode")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue loop
		}
	}
}
