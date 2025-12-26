// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package kubernetes

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for Kubernetes.
type Switcher struct{}

// NewSwitcher creates a new Kubernetes switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (k *Switcher) Name() string {
	return "kubernetes"
}

// Switch switches to the specified Kubernetes configuration.
func (k *Switcher) Switch(ctx context.Context, config interface{}) error {
	kubernetesConfig, ok := config.(*environment.KubernetesConfig)
	if !ok {
		return fmt.Errorf("invalid Kubernetes configuration type")
	}

	// Set Kubernetes context
	if kubernetesConfig.Context != "" {
		cmd := exec.CommandContext(ctx, "kubectl", "config", "use-context", kubernetesConfig.Context)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Kubernetes context: %w", err)
		}
	}

	// Set Kubernetes namespace
	if kubernetesConfig.Namespace != "" {
		cmd := exec.CommandContext(ctx, "kubectl", "config", "set-context", "--current", "--namespace", kubernetesConfig.Namespace)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Kubernetes namespace: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current Kubernetes configuration state.
func (k *Switcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	// Get current Kubernetes context
	cmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")
	contextOutput, _ := cmd.Output()

	// Get current namespace
	cmd = exec.CommandContext(ctx, "kubectl", "config", "view", "--minify", "--output", "jsonpath={..namespace}")
	namespaceOutput, _ := cmd.Output()

	return &environment.KubernetesConfig{
		Context:   strings.TrimSpace(string(contextOutput)),
		Namespace: strings.TrimSpace(string(namespaceOutput)),
	}, nil
}

// Rollback rolls back to the previous Kubernetes configuration.
func (k *Switcher) Rollback(ctx context.Context, previousState interface{}) error {
	return k.Switch(ctx, previousState)
}
