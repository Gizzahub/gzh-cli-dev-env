// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestSwitchEnvironment_StateReadFailureStopsBeforeJournalAndMutation(t *testing.T) {
	services := map[string]ServiceConfig{
		"aws":        {AWS: &AWSConfig{Profile: "wanted"}},
		"gcp":        {GCP: &GCPConfig{Project: "wanted"}},
		"azure":      {Azure: &AzureConfig{Subscription: "wanted"}},
		"docker":     {Docker: &DockerConfig{Context: "wanted"}},
		"kubernetes": {Kubernetes: &KubernetesConfig{Context: "wanted"}},
	}
	for name, config := range services {
		t.Run(name, func(t *testing.T) {
			dir := withTempConfigDir(t)
			cause := errors.New("state query failed")
			switcher := newMockSwitcher(name)
			switcher.stateError = cause
			manager := NewEnvironmentSwitcher()
			manager.Register(switcher)
			result, err := manager.SwitchEnvironment(context.Background(), &Environment{
				Name:     "snapshot-failure",
				Services: map[string]ServiceConfig{name: config},
			}, SwitchOptions{RollbackOnError: true})
			if !errors.Is(err, cause) {
				t.Fatalf("state query cause lost: %v", err)
			}
			if result == nil || result.Success || len(result.SwitchedServices) != 0 {
				t.Fatalf("unexpected result: %+v", result)
			}
			if switcher.switchCalled || switcher.rollbackCalled {
				t.Fatal("state query failure must stop before switch or rollback")
			}
			if _, statErr := os.Stat(journalPathIn(dir)); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("failed snapshot must not create journal: %v", statErr)
			}
		})
	}
}
