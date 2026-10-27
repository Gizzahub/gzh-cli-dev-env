// Copyright (c) 2025 Gizzahub
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
	Switch(ctx context.Context, config any) error

	// GetCurrentState returns the current state of the service.
	// This state can be used for rollback operations.
	// Failed queries and cancellation must return an error rather than an
	// unobserved state, so callers can stop before journaling or switching.
	GetCurrentState(ctx context.Context) (any, error)

	// Rollback restores the service to a previous state.
	Rollback(ctx context.Context, previousState any) error
}

// StateCapturer optionally captures the actual targets of a pending request.
// CaptureState must not mutate resources. Errors stop this service before its
// journal entry or Switch; hooks and previously completed services are separate.
type StateCapturer interface {
	CaptureState(ctx context.Context, config any) (any, error)
}

// StateApplier applies the same captured plan that was persisted to the journal,
// without resolving a new target between capture and mutation.
type StateApplier interface {
	ApplyState(ctx context.Context, state any) error
}

// RecoveryValidator checks decoded recovery metadata before any service is
// restored. This makes a mixed journal fail closed on unsupported snapshots.
type RecoveryValidator interface {
	ValidateRecoveryState(state any) error
}
