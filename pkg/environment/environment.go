// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadEnvironment loads an environment configuration from YAML bytes.
func LoadEnvironment(data []byte) (*Environment, error) {
	var env Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse environment configuration: %w", err)
	}

	// Validate required fields
	if env.Name == "" {
		return nil, fmt.Errorf("environment name is required")
	}

	return &env, nil
}

// LoadEnvironmentFromFile loads an environment configuration from a file.
func LoadEnvironmentFromFile(filepath string) (*Environment, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %w", err)
	}

	return LoadEnvironment(data)
}

// Validate validates the environment configuration.
func (e *Environment) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("environment name is required")
	}

	if len(e.Services) == 0 {
		return fmt.Errorf("at least one service must be configured")
	}

	// Validate dependencies
	for _, dep := range e.Dependencies {
		// Dependencies are parsed later, just check they're not empty
		if dep == "" {
			return fmt.Errorf("empty dependency string found")
		}
	}

	return nil
}

// GetServiceNames returns a list of configured service names.
func (e *Environment) GetServiceNames() []string {
	services := make([]string, 0, len(e.Services))
	for name := range e.Services {
		services = append(services, name)
	}
	return services
}

// HasService checks if a service is configured in this environment.
func (e *Environment) HasService(serviceName string) bool {
	_, exists := e.Services[serviceName]
	return exists
}

// ToYAML serializes the environment to YAML bytes.
func (e *Environment) ToYAML() ([]byte, error) {
	return yaml.Marshal(e)
}
