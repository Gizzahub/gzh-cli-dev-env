// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

// Package devenv provides CLI commands for development environment management.
//
// This package exports a root command that can be used directly as a standalone CLI
// or integrated into a larger CLI application like gzh-cli.
//
// Usage as standalone:
//
//	cmd := devenv.NewRootCmd()
//	if err := cmd.Execute(); err != nil {
//	    os.Exit(1)
//	}
//
// Usage in wrapper:
//
//	import devenv "github.com/gizzahub/gzh-cli-dev-env/cmd/devenv"
//
//	func NewDevEnvCmd() *cobra.Command {
//	    cmd := devenv.NewRootCmd()
//	    cmd.Use = "dev-env"
//	    return cmd
//	}
package devenv
