// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
)

// ServiceSwitcher interface for switching individual services.
// Implementations should be stateless and thread-safe.
type ServiceSwitcher interface {
	// Name returns the service name (e.g., "aws", "gcp", "docker").
	Name() string

	// Switch switches the service to the specified configuration.
	// The config parameter type depends on the service implementation.
	Switch(ctx context.Context, config interface{}) error

	// GetCurrentState returns the current state of the service.
	// This state can be used for rollback operations.
	GetCurrentState(ctx context.Context) (interface{}, error)

	// Rollback restores the service to a previous state.
	Rollback(ctx context.Context, previousState interface{}) error
}
