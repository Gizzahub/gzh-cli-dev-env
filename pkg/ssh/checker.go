// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// Checker implements status.ServiceChecker for SSH.
type Checker struct{}

// NewChecker creates a new SSH status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (s *Checker) Name() string {
	return "ssh"
}

// CheckStatus checks SSH current status.
func (s *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        "ssh",
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if SSH is available
	if !s.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "SSH not found"
		return st, nil
	}

	// Check SSH agent status
	agentStatus := s.checkSSHAgent()
	if !agentStatus {
		st.Status = status.StatusInactive
		st.Details["error"] = "SSH agent not running"
		return st, nil
	}

	// Get loaded keys
	keys, err := s.getLoadedKeys(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["error"] = fmt.Sprintf("Failed to get SSH keys: %v", err)
		return st, nil
	}

	if len(keys) == 0 {
		st.Status = status.StatusInactive
		st.Details["error"] = "No SSH keys loaded"
		return st, nil
	}

	st.Status = status.StatusActive
	st.Current.Context = fmt.Sprintf("%d keys loaded", len(keys))

	// Check SSH key validity
	credStatus := s.checkSSHKeys(keys)
	st.Credentials = *credStatus

	return st, nil
}

// CheckHealth performs detailed health check for SSH.
func (s *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]interface{}),
	}

	// Check SSH agent connectivity
	cmd := exec.CommandContext(ctx, "ssh-add", "-l")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to connect to SSH agent: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "SSH agent is running with loaded keys"
	health.Details["loaded_keys"] = string(output)

	// Check SSH config file
	configPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	if _, err := os.Stat(configPath); err == nil {
		health.Details["config_file"] = configPath
	}

	return health, nil
}

// isCLIAvailable checks if SSH is installed.
func (s *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("ssh")
	return err == nil
}

// checkSSHAgent checks if SSH agent is running.
func (s *Checker) checkSSHAgent() bool {
	// Check SSH_AUTH_SOCK environment variable
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return false
	}

	// Try to connect to SSH agent
	cmd := exec.Command("ssh-add", "-l")
	err := cmd.Run()
	// ssh-add -l returns 0 if keys are loaded, 1 if no keys, 2 if agent not running
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() != 2
	}
	return err == nil
}

// getLoadedKeys gets the list of loaded SSH keys.
func (s *Checker) getLoadedKeys(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's "no keys loaded" vs actual error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return []string{}, nil // No keys loaded, but agent is running
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var keys []string
	for _, line := range lines {
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// checkSSHKeys checks the status of SSH keys.
func (s *Checker) checkSSHKeys(keys []string) *status.CredentialStatus {
	credStatus := &status.CredentialStatus{
		Valid: len(keys) > 0,
		Type:  "ssh-keys",
	}

	if len(keys) == 0 {
		credStatus.Warning = "No SSH keys loaded"
		return credStatus
	}

	// Check for common key types and potential issues
	hasRSA := false
	hasEd25519 := false
	for _, key := range keys {
		if strings.Contains(key, "RSA") {
			hasRSA = true
		}
		if strings.Contains(key, "ED25519") {
			hasEd25519 = true
		}
	}

	if hasRSA && !hasEd25519 {
		credStatus.Warning = "Consider using Ed25519 keys for better security"
	}

	return credStatus
}
