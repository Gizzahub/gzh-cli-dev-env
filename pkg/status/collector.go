// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package status

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StatusCollector collects status information from multiple services.
type StatusCollector struct {
	checkers []ServiceChecker
	timeout  time.Duration
}

// NewStatusCollector creates a new status collector.
func NewStatusCollector(checkers []ServiceChecker, timeout time.Duration) *StatusCollector {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &StatusCollector{
		checkers: checkers,
		timeout:  timeout,
	}
}

// CollectAll collects status from all registered services.
func (sc *StatusCollector) CollectAll(ctx context.Context, options StatusOptions) ([]ServiceStatus, error) {
	checkers := sc.filterCheckers(options.Services)
	if len(checkers) == 0 {
		return nil, fmt.Errorf("no services found to check")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, options.Timeout)
	if options.Timeout == 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, sc.timeout)
	}
	defer cancel()

	if options.Parallel {
		return sc.collectParallel(ctxWithTimeout, checkers, options)
	}
	return sc.collectSequential(ctxWithTimeout, checkers, options)
}

// collectParallel collects status information in parallel.
func (sc *StatusCollector) collectParallel(ctx context.Context, checkers []ServiceChecker, options StatusOptions) ([]ServiceStatus, error) {
	var wg sync.WaitGroup
	results := make([]ServiceStatus, len(checkers))
	errors := make([]error, len(checkers))

	for i, checker := range checkers {
		wg.Add(1)
		go func(index int, c ServiceChecker) {
			defer wg.Done()
			status, err := sc.checkService(ctx, c, options)
			if err != nil {
				errors[index] = fmt.Errorf("failed to check %s: %w", c.Name(), err)
				results[index] = ServiceStatus{
					Name:   c.Name(),
					Status: StatusError,
					Details: map[string]string{
						"error": err.Error(),
					},
				}
			} else {
				results[index] = *status
			}
		}(i, checker)
	}

	wg.Wait()
	return results, nil
}

// collectSequential collects status information sequentially.
func (sc *StatusCollector) collectSequential(ctx context.Context, checkers []ServiceChecker, options StatusOptions) ([]ServiceStatus, error) {
	results := make([]ServiceStatus, 0, len(checkers))

	for _, checker := range checkers {
		status, err := sc.checkService(ctx, checker, options)
		if err != nil {
			results = append(results, ServiceStatus{
				Name:   checker.Name(),
				Status: StatusError,
				Details: map[string]string{
					"error": err.Error(),
				},
			})
			continue
		}
		results = append(results, *status)
	}

	return results, nil
}

// checkService checks a single service status.
func (sc *StatusCollector) checkService(ctx context.Context, checker ServiceChecker, options StatusOptions) (*ServiceStatus, error) {
	status, err := checker.CheckStatus(ctx)
	if err != nil {
		return nil, err
	}

	if options.CheckHealth {
		healthStatus, healthErr := checker.CheckHealth(ctx)
		if healthErr == nil {
			status.HealthCheck = healthStatus
		} else {
			if status.Details == nil {
				status.Details = make(map[string]string)
			}
			status.Details["health_check_error"] = healthErr.Error()
		}
	}

	return status, nil
}

// filterCheckers filters checkers based on requested service names.
func (sc *StatusCollector) filterCheckers(services []string) []ServiceChecker {
	if len(services) == 0 {
		return sc.checkers
	}

	serviceSet := make(map[string]bool)
	for _, service := range services {
		serviceSet[service] = true
	}

	var filtered []ServiceChecker
	for _, checker := range sc.checkers {
		if serviceSet[checker.Name()] {
			filtered = append(filtered, checker)
		}
	}

	return filtered
}

// AddChecker adds a checker to the collector.
func (sc *StatusCollector) AddChecker(checker ServiceChecker) {
	sc.checkers = append(sc.checkers, checker)
}

// GetCheckers returns all registered checkers.
func (sc *StatusCollector) GetCheckers() []ServiceChecker {
	return sc.checkers
}
