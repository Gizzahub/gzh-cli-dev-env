// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// commandContext builds an exec.Cmd. Overridable in tests to avoid real kubectl.
var commandContext = exec.CommandContext

// lookPath locates an executable. Overridable in tests.
var lookPath = exec.LookPath

// Switcher implements environment.ServiceSwitcher for Kubernetes.
type Switcher struct{}

// NewSwitcher creates a new Kubernetes switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (k *Switcher) Name() string {
	return serviceName
}

// Switch switches to the specified Kubernetes configuration.
func (k *Switcher) Switch(ctx context.Context, config any) error {
	kubernetesConfig, ok := config.(*environment.KubernetesConfig)
	if !ok {
		return fmt.Errorf("invalid Kubernetes configuration type")
	}

	// Set Kubernetes context
	if kubernetesConfig.Context != "" {
		cmd := commandContext(ctx, "kubectl", "config", "use-context", kubernetesConfig.Context)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Kubernetes context: %w", err)
		}
	}

	// Set Kubernetes namespace
	if kubernetesConfig.Namespace != "" {
		cmd := commandContext(ctx, "kubectl", "config", "set-context", "--current", "--namespace", kubernetesConfig.Namespace)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set Kubernetes namespace: %w", err)
		}
	}

	return nil
}

// GetCurrentState retrieves the current Kubernetes configuration state.
func (k *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	return currentStateWithCommand(ctx, commandContext)
}

func currentStateWithCommand(ctx context.Context, build func(context.Context, string, ...string) *exec.Cmd) (any, error) {
	contextOutput, err := stateQuery(ctx, build, "context", "config", "current-context")
	if err != nil {
		return nil, err
	}
	namespaceOutput, err := stateQuery(ctx, build, "namespace", "config", "view", "--minify", "--output", "jsonpath={..namespace}")
	if err != nil {
		return nil, err
	}
	return &environment.KubernetesConfig{
		Context:   strings.TrimSpace(string(contextOutput)),
		Namespace: strings.TrimSpace(string(namespaceOutput)),
	}, nil
}

// 각 query 실패 시 부분 상태를 반환하지 않고 실행 오류와 취소 원인을 보존한다.
func stateQuery(ctx context.Context, build func(context.Context, string, ...string) *exec.Cmd, query string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("kubernetes %s query: %w", query, err)
	}
	output, err := build(ctx, "kubectl", args...).Output()
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("kubernetes %s query: %w", query, err)
	}
	return output, nil
}

// Rollback rolls back to the previous Kubernetes configuration.
func (k *Switcher) Rollback(ctx context.Context, previousState any) error {
	return k.Switch(ctx, previousState)
}
