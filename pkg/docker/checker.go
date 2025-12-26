// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// DefaultContext is the default Docker context name.
const DefaultContext = "default"

// Checker implements status.ServiceChecker for Docker.
type Checker struct{}

// NewChecker creates a new Docker status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (d *Checker) Name() string {
	return "docker"
}

// CheckStatus checks Docker current status.
func (d *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        "docker",
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if Docker CLI is available
	if !d.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "Docker CLI not found"
		return st, nil
	}

	// Check if Docker daemon is running
	if !d.isDaemonRunning(ctx) {
		st.Status = status.StatusInactive
		st.Details["error"] = "Docker daemon not running"
		return st, nil
	}

	// Get current context
	dockerCtx, err := d.getCurrentContext(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["error"] = fmt.Sprintf("Failed to get Docker context: %v", err)
		return st, nil
	}

	st.Current.Context = dockerCtx
	st.Status = status.StatusActive

	// Docker doesn't typically have credential expiration like cloud services
	st.Credentials = status.CredentialStatus{
		Valid: true,
		Type:  "docker-socket",
	}

	return st, nil
}

// CheckHealth performs detailed health check for Docker.
func (d *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]interface{}),
	}

	// Test Docker connectivity with docker info
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to connect to Docker daemon: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "Docker daemon is running and accessible"
	health.Details["server_version"] = strings.TrimSpace(string(output))

	// Get additional Docker info
	cmd = exec.CommandContext(ctx, "docker", "system", "df", "--format", "table")
	dfOutput, err := cmd.Output()
	if err == nil {
		health.Details["disk_usage"] = string(dfOutput)
	}

	// Check running containers count
	cmd = exec.CommandContext(ctx, "docker", "ps", "-q")
	psOutput, err := cmd.Output()
	if err == nil {
		containerCount := len(strings.Split(strings.TrimSpace(string(psOutput)), "\n"))
		if strings.TrimSpace(string(psOutput)) == "" {
			containerCount = 0
		}
		health.Details["running_containers"] = containerCount
	}

	return health, nil
}

// isCLIAvailable checks if Docker CLI is installed.
func (d *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// isDaemonRunning checks if Docker daemon is running.
func (d *Checker) isDaemonRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	err := cmd.Run()
	return err == nil
}

// getCurrentContext gets the current Docker context.
func (d *Checker) getCurrentContext(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		// If context command fails, assume default context
		return DefaultContext, nil
	}
	return strings.TrimSpace(string(output)), nil
}
