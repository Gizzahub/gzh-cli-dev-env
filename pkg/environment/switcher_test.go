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
