// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"testing"
	"time"
)

func TestEnvironment_Fields(t *testing.T) {
	env := Environment{
		Name:        "production",
		Description: "Production environment",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{
					Profile: "prod",
					Region:  "us-west-2",
				},
			},
		},
		Dependencies: []string{"aws"},
	}

	if env.Name != "production" {
		t.Errorf("Name = %v, want production", env.Name)
	}
	if env.Description != "Production environment" {
		t.Errorf("Description mismatch")
	}
	if len(env.Services) != 1 {
		t.Errorf("Services count = %d, want 1", len(env.Services))
	}
	if env.Services["aws"].AWS.Profile != "prod" {
		t.Error("AWS Profile mismatch")
	}
}

func TestEnvironment_GetServiceNames(t *testing.T) {
	env := Environment{
		Services: map[string]ServiceConfig{
			"aws":        {},
			"gcp":        {},
			"kubernetes": {},
		},
	}

	names := env.GetServiceNames()
	if len(names) != 3 {
		t.Errorf("GetServiceNames() returned %d names, want 3", len(names))
	}
}

func TestAWSConfig_Fields(t *testing.T) {
	config := AWSConfig{
		Profile:   "default",
		Region:    "us-east-1",
		AccountID: "123456789012",
	}

	if config.Profile != "default" {
		t.Error("Profile mismatch")
	}
	if config.Region != "us-east-1" {
		t.Error("Region mismatch")
	}
	if config.AccountID != "123456789012" {
		t.Error("AccountID mismatch")
	}
}

func TestGCPConfig_Fields(t *testing.T) {
	config := GCPConfig{
		Project: "my-project",
		Account: "user@example.com",
		Region:  "us-central1",
	}

	if config.Project != "my-project" {
		t.Error("Project mismatch")
	}
	if config.Account != "user@example.com" {
		t.Error("Account mismatch")
	}
}

func TestAzureConfig_Fields(t *testing.T) {
	config := AzureConfig{
		Subscription: "sub-id-123",
		Tenant:       "tenant-id-456",
	}

	if config.Subscription != "sub-id-123" {
		t.Error("Subscription mismatch")
	}
	if config.Tenant != "tenant-id-456" {
		t.Error("Tenant mismatch")
	}
}

func TestDockerConfig_Fields(t *testing.T) {
	config := DockerConfig{
		Context: "default",
	}

	if config.Context != "default" {
		t.Error("Context mismatch")
	}
}

func TestKubernetesConfig_Fields(t *testing.T) {
	config := KubernetesConfig{
		Context:   "minikube",
		Namespace: "kube-system",
	}

	if config.Context != "minikube" {
		t.Error("Context mismatch")
	}
	if config.Namespace != "kube-system" {
		t.Error("Namespace mismatch")
	}
}

func TestSSHConfig_Fields(t *testing.T) {
	config := SSHConfig{
		Config: "~/.ssh/config",
	}

	if config.Config != "~/.ssh/config" {
		t.Error("Config mismatch")
	}
}

func TestHook_Fields(t *testing.T) {
	hook := Hook{
		Command: "echo 'pre-hook'",
		Timeout: 30 * time.Second,
		OnError: "fail",
	}

	if hook.Command != "echo 'pre-hook'" {
		t.Error("Command mismatch")
	}
	if hook.Timeout != 30*time.Second {
		t.Error("Timeout mismatch")
	}
	if hook.OnError != "fail" {
		t.Error("OnError mismatch")
	}
}

func TestSwitchProgress_Fields(t *testing.T) {
	progress := SwitchProgress{
		TotalServices:     5,
		CompletedServices: 3,
		CurrentService:    "kubernetes",
		Status:            "switching",
	}

	if progress.TotalServices != 5 {
		t.Error("TotalServices mismatch")
	}
	if progress.CompletedServices != 3 {
		t.Error("CompletedServices mismatch")
	}
	if progress.CurrentService != "kubernetes" {
		t.Error("CurrentService mismatch")
	}
}

func TestSwitchResult_Success(t *testing.T) {
	result := SwitchResult{
		Success:          true,
		SwitchedServices: []string{"aws", "gcp", "kubernetes"},
		FailedServices:   []string{},
		Duration:         5 * time.Second,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if len(result.SwitchedServices) != 3 {
		t.Errorf("SwitchedServices count = %d, want 3", len(result.SwitchedServices))
	}
	if len(result.FailedServices) != 0 {
		t.Error("FailedServices should be empty")
	}
}

func TestSwitchResult_Failure(t *testing.T) {
	result := SwitchResult{
		Success:           false,
		SwitchedServices:  []string{"aws"},
		FailedServices:    []string{"gcp"},
		RollbackPerformed: true,
		Errors: []SwitchError{
			{
				Service: "gcp",
				Error:   "authentication failed",
				Time:    time.Now(),
			},
		},
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if !result.RollbackPerformed {
		t.Error("RollbackPerformed should be true")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors count = %d, want 1", len(result.Errors))
	}
}

func TestSwitchOptions_Fields(t *testing.T) {
	opts := SwitchOptions{
		DryRun:          true,
		Force:           false,
		Parallel:        true,
		RollbackOnError: true,
		Timeout:         5 * time.Minute,
	}

	if !opts.DryRun {
		t.Error("DryRun should be true")
	}
	if opts.Force {
		t.Error("Force should be false")
	}
	if !opts.Parallel {
		t.Error("Parallel should be true")
	}
	if opts.Timeout != 5*time.Minute {
		t.Error("Timeout mismatch")
	}
}

func TestServiceGroup_Fields(t *testing.T) {
	group := ServiceGroup{
		Services: []string{"aws", "gcp"},
		Level:    1,
	}

	if len(group.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(group.Services))
	}
	if group.Level != 1 {
		t.Error("Level mismatch")
	}
}
