// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func TestSwitch_RejectsOptionLikeValuesBeforeAnyCommand(t *testing.T) {
	tests := map[string]*environment.KubernetesConfig{
		"context":   {Context: "--kubeconfig=/tmp/x", Namespace: "default"},
		"namespace": {Context: "dev", Namespace: "-n"},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			calls := withCommandMap(t, nil)
			err := NewSwitcher().Switch(context.Background(), cfg)
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("Switch() error = %v, want ErrOptionLikeValue", err)
			}
			if len(*calls) != 0 {
				t.Fatalf("kubectl ran despite rejection: %v", *calls)
			}
		})
	}
}

func TestRollback_RejectsOptionLikeSnapshot(t *testing.T) {
	calls := withCommandMap(t, nil)
	err := NewSwitcher().Rollback(context.Background(), &environment.KubernetesConfig{Context: "-h"})
	if !errors.Is(err, environment.ErrOptionLikeValue) {
		t.Fatalf("Rollback() error = %v, want ErrOptionLikeValue", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("kubectl ran despite rejection: %v", *calls)
	}
}
