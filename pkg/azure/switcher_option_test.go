// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func TestSwitch_RejectsOptionLikeSubscriptionBeforeAnyCommand(t *testing.T) {
	for _, value := range []string{"-h", "--help"} {
		t.Run(value, func(t *testing.T) {
			calls := withCommandMap(t, nil)
			err := NewSwitcher().Switch(context.Background(), &environment.AzureConfig{Subscription: value})
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("Switch() error = %v, want ErrOptionLikeValue", err)
			}
			if len(*calls) != 0 {
				t.Fatalf("az ran despite rejection: %v", *calls)
			}
		})
	}
}

func TestRollback_RejectsOptionLikeSnapshot(t *testing.T) {
	calls := withCommandMap(t, nil)
	err := NewSwitcher().Rollback(context.Background(), &environment.AzureConfig{Subscription: "-x"})
	if !errors.Is(err, environment.ErrOptionLikeValue) {
		t.Fatalf("Rollback() error = %v, want ErrOptionLikeValue", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("az ran despite rejection: %v", *calls)
	}
}
