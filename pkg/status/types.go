// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package status

import (
	"context"
	"time"
)

// StatusType represents the current status of a service.
type StatusType string

const (
	StatusActive   StatusType = "active"
	StatusInactive StatusType = "inactive"
	StatusError    StatusType = "error"
	StatusUnknown  StatusType = "unknown"
)

// ServiceStatus represents the current status of a development environment service.
type ServiceStatus struct {
	Name        string            `json:"name"`
	Status      StatusType        `json:"status"`
	Current     CurrentConfig     `json:"current"`
	Credentials CredentialStatus  `json:"credentials"`
	LastUsed    time.Time         `json:"lastUsed"`
	HealthCheck *HealthStatus     `json:"healthCheck,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
}

// CurrentConfig holds the current configuration details for a service.
type CurrentConfig struct {
	Profile   string `json:"profile,omitempty"`
	Region    string `json:"region,omitempty"`
	Project   string `json:"project,omitempty"`
	Context   string `json:"context,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Account   string `json:"account,omitempty"`
}

// CredentialStatus represents the status of service credentials.
type CredentialStatus struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Type      string    `json:"type"`
	Warning   string    `json:"warning,omitempty"`
}

// HealthStatus represents detailed health check information.
type HealthStatus struct {
	Status    StatusType             `json:"status"`
	Message   string                 `json:"message,omitempty"`
	CheckedAt time.Time              `json:"checkedAt"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// StatusOptions configures how status information is collected.
type StatusOptions struct {
	Services     []string      `json:"services,omitempty"`
	CheckHealth  bool          `json:"checkHealth"`
	Timeout      time.Duration `json:"timeout"`
	Parallel     bool          `json:"parallel"`
	IncludeCache bool          `json:"includeCache"`
}

// ServiceChecker interface for checking service status.
type ServiceChecker interface {
	Name() string
	CheckStatus(ctx context.Context) (*ServiceStatus, error)
	CheckHealth(ctx context.Context) (*HealthStatus, error)
}

// StatusFormatter interface for formatting status output.
type StatusFormatter interface {
	Format(statuses []ServiceStatus) (string, error)
}
