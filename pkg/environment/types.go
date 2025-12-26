// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"time"
)

// Environment represents a complete development environment configuration.
type Environment struct {
	Name         string                   `yaml:"name"`
	Description  string                   `yaml:"description"`
	Services     map[string]ServiceConfig `yaml:"services"`
	Dependencies []string                 `yaml:"dependencies"`
	PreHooks     []Hook                   `yaml:"preHooks,omitempty"`
	PostHooks    []Hook                   `yaml:"postHooks,omitempty"`
}

// ServiceConfig contains configuration for a specific service.
type ServiceConfig struct {
	AWS        *AWSConfig        `yaml:"aws,omitempty"`
	GCP        *GCPConfig        `yaml:"gcp,omitempty"`
	Azure      *AzureConfig      `yaml:"azure,omitempty"`
	Docker     *DockerConfig     `yaml:"docker,omitempty"`
	Kubernetes *KubernetesConfig `yaml:"kubernetes,omitempty"`
	SSH        *SSHConfig        `yaml:"ssh,omitempty"`
}

// AWSConfig represents AWS service configuration.
type AWSConfig struct {
	Profile   string `yaml:"profile"`
	Region    string `yaml:"region"`
	AccountID string `yaml:"accountId,omitempty"`
}

// GCPConfig represents GCP service configuration.
type GCPConfig struct {
	Project string `yaml:"project"`
	Account string `yaml:"account,omitempty"`
	Region  string `yaml:"region,omitempty"`
}

// AzureConfig represents Azure service configuration.
type AzureConfig struct {
	Subscription string `yaml:"subscription"`
	Tenant       string `yaml:"tenant,omitempty"`
}

// DockerConfig represents Docker service configuration.
type DockerConfig struct {
	Context string `yaml:"context"`
}

// KubernetesConfig represents Kubernetes service configuration.
type KubernetesConfig struct {
	Context   string `yaml:"context"`
	Namespace string `yaml:"namespace,omitempty"`
}

// SSHConfig represents SSH service configuration.
type SSHConfig struct {
	Config string `yaml:"config"`
}

// Hook represents a command to execute before or after environment switching.
type Hook struct {
	Command string        `yaml:"command"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
	OnError string        `yaml:"onError,omitempty"` // continue, fail, rollback
}

// SwitchProgress represents the progress of environment switching.
type SwitchProgress struct {
	TotalServices     int           `json:"totalServices"`
	CompletedServices int           `json:"completedServices"`
	CurrentService    string        `json:"currentService"`
	Status            string        `json:"status"`
	StartTime         time.Time     `json:"startTime"`
	EstimatedEnd      time.Time     `json:"estimatedEnd"`
	Errors            []SwitchError `json:"errors,omitempty"`
}

// SwitchError represents an error during environment switching.
type SwitchError struct {
	Service string    `json:"service"`
	Error   string    `json:"error"`
	Time    time.Time `json:"time"`
}

// SwitchResult represents the result of environment switching.
type SwitchResult struct {
	Success           bool          `json:"success"`
	SwitchedServices  []string      `json:"switchedServices"`
	FailedServices    []string      `json:"failedServices"`
	RollbackPerformed bool          `json:"rollbackPerformed"`
	Duration          time.Duration `json:"duration"`
	Errors            []SwitchError `json:"errors,omitempty"`
}

// SwitchOptions contains options for environment switching.
type SwitchOptions struct {
	DryRun          bool
	Force           bool
	Parallel        bool
	RollbackOnError bool
	Timeout         time.Duration
}

// ServiceGroup represents a group of services that can be executed in parallel.
type ServiceGroup struct {
	Services []string
	Level    int
}
