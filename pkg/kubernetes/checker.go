// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

// DefaultNamespace is the default Kubernetes namespace.
const DefaultNamespace = "default"

// Checker implements status.ServiceChecker for Kubernetes.
type Checker struct{}

// NewChecker creates a new Kubernetes status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Name returns the service name.
func (k *Checker) Name() string {
	return "kubernetes"
}

// CheckStatus checks Kubernetes current status.
func (k *Checker) CheckStatus(ctx context.Context) (*status.ServiceStatus, error) {
	st := &status.ServiceStatus{
		Name:        "kubernetes",
		Status:      status.StatusUnknown,
		Current:     status.CurrentConfig{},
		Credentials: status.CredentialStatus{},
		LastUsed:    time.Now(),
		Details:     make(map[string]string),
	}

	// Check if kubectl is available
	if !k.isCLIAvailable() {
		st.Status = status.StatusInactive
		st.Details["error"] = "kubectl not found"
		return st, nil
	}

	// Get current context
	k8sCtx, err := k.getCurrentContext(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["error"] = fmt.Sprintf("Failed to get current context: %v", err)
		return st, nil
	}

	if k8sCtx == "" {
		st.Status = status.StatusInactive
		st.Details["error"] = "No Kubernetes context set"
		return st, nil
	}

	st.Current.Context = k8sCtx

	// Get current namespace
	namespace, err := k.getCurrentNamespace(ctx)
	if err == nil {
		st.Current.Namespace = namespace
	}

	// Check cluster connectivity
	credStatus, err := k.checkClusterAccess(ctx)
	if err != nil {
		st.Status = status.StatusError
		st.Details["connectivity_error"] = err.Error()
		return st, nil
	}

	st.Credentials = *credStatus
	if credStatus.Valid {
		st.Status = status.StatusActive
	} else {
		st.Status = status.StatusInactive
	}

	return st, nil
}

// CheckHealth performs detailed health check for Kubernetes.
func (k *Checker) CheckHealth(ctx context.Context) (*status.HealthStatus, error) {
	start := time.Now()
	health := &status.HealthStatus{
		Status:    status.StatusUnknown,
		CheckedAt: start,
		Details:   make(map[string]interface{}),
	}

	// Test cluster connectivity with kubectl cluster-info
	cmd := exec.CommandContext(ctx, "kubectl", "cluster-info", "--request-timeout=10s")
	output, err := cmd.Output()
	health.Duration = time.Since(start)

	if err != nil {
		health.Status = status.StatusError
		health.Message = fmt.Sprintf("Failed to connect to Kubernetes cluster: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			health.Details["stderr"] = string(exitErr.Stderr)
		}
		return health, nil
	}

	health.Status = status.StatusActive
	health.Message = "Kubernetes cluster is accessible"
	health.Details["cluster_info"] = string(output)

	// Additional check: get node status
	cmd = exec.CommandContext(ctx, "kubectl", "get", "nodes", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.conditions[?(@.type==\"Ready\")].status")
	nodeOutput, err := cmd.Output()
	if err == nil {
		health.Details["node_status"] = string(nodeOutput)
	}

	return health, nil
}

// isCLIAvailable checks if kubectl is installed.
func (k *Checker) isCLIAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

// getCurrentContext gets the current Kubernetes context.
func (k *Checker) getCurrentContext(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentNamespace gets the current Kubernetes namespace.
func (k *Checker) getCurrentNamespace(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "config", "view", "--minify", "--output", "jsonpath={..namespace}")
	output, err := cmd.Output()
	if err != nil {
		return DefaultNamespace, nil // Default to "default" namespace
	}

	namespace := strings.TrimSpace(string(output))
	if namespace == "" {
		return DefaultNamespace, nil
	}
	return namespace, nil
}

// checkClusterAccess checks if we can access the Kubernetes cluster.
func (k *Checker) checkClusterAccess(ctx context.Context) (*status.CredentialStatus, error) {
	credStatus := &status.CredentialStatus{
		Valid: false,
		Type:  "kubeconfig",
	}

	// Test cluster access with a simple API call
	cmd := exec.CommandContext(ctx, "kubectl", "auth", "can-i", "get", "pods", "--request-timeout=10s")
	err := cmd.Run()
	if err != nil {
		credStatus.Warning = "Cannot access Kubernetes cluster"
		return credStatus, nil
	}

	credStatus.Valid = true

	// Check if credentials have expiration (for OIDC/cloud providers)
	currentUser := k.getCurrentUser(ctx)
	jsonPath := fmt.Sprintf("{.users[?(@.name==%q)].user}", currentUser)
	cmd = exec.CommandContext(ctx, "kubectl", "config", "view", "--raw", "-o", "jsonpath="+jsonPath) // #nosec G204 - validated kubectl command with controlled arguments
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "expiry") {
		credStatus.Type = "oidc-token"
		credStatus.Warning = "Token may expire - check manually"
	}

	return credStatus, nil
}

// getCurrentUser gets the current Kubernetes user.
func (k *Checker) getCurrentUser(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "kubectl", "config", "view", "--minify", "--output", "jsonpath={.contexts[0].context.user}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
