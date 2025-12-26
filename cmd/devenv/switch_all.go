// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package devenv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// switchAllOptions contains options for the switch-all command.
type switchAllOptions struct {
	env         string
	fromFile    string
	dryRun      bool
	force       bool
	interactive bool
	parallel    bool
	timeout     time.Duration
}

// newSwitchAllCmd creates the switch-all command.
func newSwitchAllCmd() *cobra.Command {
	opts := &switchAllOptions{
		timeout: 5 * time.Minute,
	}

	cmd := &cobra.Command{
		Use:   "switch-all",
		Short: "Switch all development environments atomically",
		Long: `Switch multiple cloud services and development environments to a target state.

This command provides atomic environment switching across all configured services:
- AWS profiles, regions, and accounts
- GCP projects and accounts
- Azure subscriptions and tenants
- Docker contexts
- Kubernetes clusters and namespaces
- SSH configurations

All services are switched atomically - either all succeed or all are rolled back.

Examples:
  # Switch to production environment
  dev-env switch-all --env production

  # Preview changes without applying
  dev-env switch-all --env production --dry-run

  # Switch using environment file
  dev-env switch-all --from-file production.yaml

  # Interactive environment selection
  dev-env switch-all --interactive

  # Force switch without confirmation
  dev-env switch-all --env dev --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&opts.env, "env", "", "Environment name to switch to")
	cmd.Flags().StringVar(&opts.fromFile, "from-file", "", "Environment configuration file")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Force switch without confirmation")
	cmd.Flags().BoolVar(&opts.interactive, "interactive", false, "Interactive environment selection")
	cmd.Flags().BoolVar(&opts.parallel, "parallel", false, "Enable parallel service switching")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", opts.timeout, "Operation timeout")

	// Make env and from-file mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("env", "from-file", "interactive")

	return cmd
}

// run executes the switch-all command.
func (opts *switchAllOptions) run(ctx context.Context) error {
	// Load environment configuration
	env, err := opts.loadEnvironment()
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Initialize environment switcher
	switcher := environment.NewEnvironmentSwitcher()

	// Register service switchers
	registerDefaultSwitchers(switcher)

	// Set up progress reporting
	switcher.SetProgressCallback(opts.reportProgress)

	// Prepare switch options
	switchOptions := environment.SwitchOptions{
		DryRun:          opts.dryRun,
		Force:           opts.force,
		Parallel:        opts.parallel,
		RollbackOnError: true,
		Timeout:         opts.timeout,
	}

	// Confirm operation if not forced or dry-run
	if !opts.force && !opts.dryRun {
		if err := opts.confirmSwitch(env); err != nil {
			return err
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	// Perform the switch
	fmt.Printf("🔄 Switching to environment: %s\n", env.Name)
	if opts.dryRun {
		fmt.Println("👁️  DRY-RUN MODE: No changes will be made")
	}

	result, err := switcher.SwitchEnvironment(ctx, env, switchOptions)
	if err != nil {
		return fmt.Errorf("environment switch failed: %w", err)
	}

	// Display results
	opts.displayResults(result)

	if !result.Success {
		return fmt.Errorf("environment switch completed with errors")
	}

	fmt.Printf("✅ Successfully switched to environment: %s\n", env.Name)
	return nil
}

// loadEnvironment loads the environment configuration.
func (opts *switchAllOptions) loadEnvironment() (*environment.Environment, error) {
	var data []byte
	var err error

	switch {
	case opts.interactive:
		return opts.selectEnvironmentInteractively()
	case opts.fromFile != "":
		data, err = os.ReadFile(opts.fromFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read environment file %s: %w", opts.fromFile, err)
		}
	case opts.env != "":
		envFile := opts.findEnvironmentFile(opts.env)
		if envFile == "" {
			return nil, fmt.Errorf("environment '%s' not found", opts.env)
		}
		data, err = os.ReadFile(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read environment file %s: %w", envFile, err)
		}
	default:
		return nil, fmt.Errorf("must specify --env, --from-file, or --interactive")
	}

	env, err := environment.LoadEnvironment(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment configuration: %w", err)
	}

	return env, nil
}

// findEnvironmentFile finds the environment configuration file.
func (opts *switchAllOptions) findEnvironmentFile(envName string) string {
	// Search paths for environment files
	searchPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".gzh", "dev-env", "environments"),
		filepath.Join(".", "environments"),
		".",
	}

	extensions := []string{".yaml", ".yml"}

	for _, searchPath := range searchPaths {
		for _, ext := range extensions {
			filename := filepath.Join(searchPath, envName+ext)
			if _, err := os.Stat(filename); err == nil {
				return filename
			}
		}
	}

	return ""
}

// selectEnvironmentInteractively allows interactive environment selection.
func (opts *switchAllOptions) selectEnvironmentInteractively() (*environment.Environment, error) {
	// Find available environments
	environments, err := opts.findAvailableEnvironments()
	if err != nil {
		return nil, fmt.Errorf("failed to find available environments: %w", err)
	}

	if len(environments) == 0 {
		return nil, fmt.Errorf("no environments found")
	}

	// Display available environments
	fmt.Println("Available environments:")
	for i, env := range environments {
		fmt.Printf("  %d. %s", i+1, env.Name)
		if env.Description != "" {
			fmt.Printf(" - %s", env.Description)
		}
		fmt.Println()
	}

	// Get user selection
	fmt.Print("Select environment (1-", len(environments), "): ")
	var selection int
	if _, err := fmt.Scanf("%d", &selection); err != nil {
		return nil, fmt.Errorf("invalid selection: %w", err)
	}

	if selection < 1 || selection > len(environments) {
		return nil, fmt.Errorf("selection out of range")
	}

	return &environments[selection-1], nil
}

// findAvailableEnvironments finds all available environment configurations.
func (opts *switchAllOptions) findAvailableEnvironments() ([]environment.Environment, error) {
	envDir := filepath.Join(os.Getenv("HOME"), ".gzh", "dev-env", "environments")

	entries, err := os.ReadDir(envDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	environments := make([]environment.Environment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !isYAMLFile(filename) {
			continue
		}

		envPath := filepath.Join(envDir, filename)
		data, err := os.ReadFile(envPath)
		if err != nil {
			continue // Skip files that can't be read
		}

		env, err := environment.LoadEnvironment(data)
		if err != nil {
			continue // Skip invalid environment files
		}

		environments = append(environments, *env)
	}

	return environments, nil
}

// isYAMLFile checks if a filename has a YAML extension.
func isYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}

// confirmSwitch asks for user confirmation.
func (opts *switchAllOptions) confirmSwitch(env *environment.Environment) error {
	fmt.Printf("🔄 About to switch to environment: %s\n", env.Name)
	if env.Description != "" {
		fmt.Printf("   Description: %s\n", env.Description)
	}

	services := env.GetServiceNames()
	fmt.Printf("   Services: %v\n", services)

	fmt.Print("Continue? [y/N]: ")
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" && response != "yes" {
		return fmt.Errorf("operation canceled by user")
	}

	return nil
}

// reportProgress reports switching progress.
func (opts *switchAllOptions) reportProgress(progress environment.SwitchProgress) {
	percentage := float64(progress.CompletedServices) / float64(progress.TotalServices) * 100
	fmt.Printf("⏳ Progress: %.1f%% (%d/%d) - %s\n",
		percentage,
		progress.CompletedServices,
		progress.TotalServices,
		progress.Status)

	if progress.CurrentService != "" {
		fmt.Printf("   Current: %s\n", progress.CurrentService)
	}
}

// displayResults displays the switching results.
func (opts *switchAllOptions) displayResults(result *environment.SwitchResult) {
	fmt.Printf("\n📊 Switch Results:\n")
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Success: %v\n", result.Success)

	if len(result.SwitchedServices) > 0 {
		fmt.Printf("   ✅ Switched: %v\n", result.SwitchedServices)
	}

	if len(result.FailedServices) > 0 {
		fmt.Printf("   ❌ Failed: %v\n", result.FailedServices)
	}

	if result.RollbackPerformed {
		fmt.Printf("   🔄 Rollback: Performed\n")
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("   [%s] %s: %s\n", err.Time.Format("15:04:05"), err.Service, err.Error)
		}
	}
}
