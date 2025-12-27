// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package status

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// mockChecker is a mock implementation of ServiceChecker for testing.
type mockChecker struct {
	name         string
	status       *ServiceStatus
	statusErr    error
	health       *HealthStatus
	healthErr    error
	checkCount   atomic.Int32
	healthCount  atomic.Int32
	delay        time.Duration
}

func newMockChecker(name string) *mockChecker {
	return &mockChecker{
		name: name,
		status: &ServiceStatus{
			Name:    name,
			Status:  StatusActive,
			Details: make(map[string]string),
		},
		health: &HealthStatus{
			Status:    StatusActive,
			CheckedAt: time.Now(),
			Details:   make(map[string]interface{}),
		},
	}
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) CheckStatus(ctx context.Context) (*ServiceStatus, error) {
	m.checkCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.status, nil
}

func (m *mockChecker) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	m.healthCount.Add(1)
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	return m.health, nil
}

// TestNewStatusCollector tests collector creation.
func TestNewStatusCollector(t *testing.T) {
	tests := []struct {
		name            string
		checkers        []ServiceChecker
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with default timeout",
			checkers:        []ServiceChecker{newMockChecker("test")},
			timeout:         0,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "with custom timeout",
			checkers:        []ServiceChecker{newMockChecker("test")},
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "with empty checkers",
			checkers:        nil,
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewStatusCollector(tt.checkers, tt.timeout)
			if collector == nil {
				t.Fatal("NewStatusCollector returned nil")
			}
			if collector.timeout != tt.expectedTimeout {
				t.Errorf("timeout = %v, want %v", collector.timeout, tt.expectedTimeout)
			}
		})
	}
}

// TestStatusCollector_AddChecker tests adding checkers.
func TestStatusCollector_AddChecker(t *testing.T) {
	collector := NewStatusCollector(nil, 5*time.Second)

	if len(collector.GetCheckers()) != 0 {
		t.Error("initial checkers should be empty")
	}

	mock1 := newMockChecker("service1")
	collector.AddChecker(mock1)

	if len(collector.GetCheckers()) != 1 {
		t.Error("should have 1 checker after adding")
	}

	mock2 := newMockChecker("service2")
	collector.AddChecker(mock2)

	if len(collector.GetCheckers()) != 2 {
		t.Error("should have 2 checkers after adding second")
	}
}

// TestStatusCollector_GetCheckers tests getting checkers.
func TestStatusCollector_GetCheckers(t *testing.T) {
	mock1 := newMockChecker("service1")
	mock2 := newMockChecker("service2")
	checkers := []ServiceChecker{mock1, mock2}

	collector := NewStatusCollector(checkers, 5*time.Second)
	got := collector.GetCheckers()

	if len(got) != 2 {
		t.Errorf("GetCheckers() returned %d checkers, want 2", len(got))
	}
}

// TestStatusCollector_CollectAll_Sequential tests sequential collection.
func TestStatusCollector_CollectAll_Sequential(t *testing.T) {
	mock1 := newMockChecker("service1")
	mock2 := newMockChecker("service2")
	mock2.status.Status = StatusInactive

	collector := NewStatusCollector([]ServiceChecker{mock1, mock2}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		Parallel: false,
		Timeout:  5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("CollectAll() returned %d results, want 2", len(results))
	}

	if results[0].Name != "service1" || results[0].Status != StatusActive {
		t.Errorf("result[0] = %v, want service1/Active", results[0])
	}

	if results[1].Name != "service2" || results[1].Status != StatusInactive {
		t.Errorf("result[1] = %v, want service2/Inactive", results[1])
	}
}

// TestStatusCollector_CollectAll_Parallel tests parallel collection.
func TestStatusCollector_CollectAll_Parallel(t *testing.T) {
	mock1 := newMockChecker("service1")
	mock2 := newMockChecker("service2")
	mock3 := newMockChecker("service3")

	collector := NewStatusCollector([]ServiceChecker{mock1, mock2, mock3}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		Parallel: true,
		Timeout:  5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("CollectAll() returned %d results, want 3", len(results))
	}

	// All should have been checked
	if mock1.checkCount.Load() != 1 || mock2.checkCount.Load() != 1 || mock3.checkCount.Load() != 1 {
		t.Error("not all checkers were called")
	}
}

// TestStatusCollector_CollectAll_NoServices tests error when no services.
func TestStatusCollector_CollectAll_NoServices(t *testing.T) {
	collector := NewStatusCollector(nil, 5*time.Second)

	_, err := collector.CollectAll(context.Background(), StatusOptions{})

	if err == nil {
		t.Error("CollectAll() should return error when no services")
	}
}

// TestStatusCollector_CollectAll_FilteredNoMatch tests error when filter has no matches.
func TestStatusCollector_CollectAll_FilteredNoMatch(t *testing.T) {
	mock := newMockChecker("service1")
	collector := NewStatusCollector([]ServiceChecker{mock}, 5*time.Second)

	_, err := collector.CollectAll(context.Background(), StatusOptions{
		Services: []string{"nonexistent"},
	})

	if err == nil {
		t.Error("CollectAll() should return error when no matching services")
	}
}

// TestStatusCollector_CollectAll_FilteredServices tests filtering by service name.
func TestStatusCollector_CollectAll_FilteredServices(t *testing.T) {
	mock1 := newMockChecker("service1")
	mock2 := newMockChecker("service2")
	mock3 := newMockChecker("service3")

	collector := NewStatusCollector([]ServiceChecker{mock1, mock2, mock3}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		Services: []string{"service1", "service3"},
		Timeout:  5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("CollectAll() returned %d results, want 2", len(results))
	}

	// service2 should not have been called
	if mock2.checkCount.Load() != 0 {
		t.Error("service2 should not have been checked")
	}
}

// TestStatusCollector_CollectAll_WithHealthCheck tests health checking.
func TestStatusCollector_CollectAll_WithHealthCheck(t *testing.T) {
	mock := newMockChecker("service1")
	collector := NewStatusCollector([]ServiceChecker{mock}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		CheckHealth: true,
		Timeout:     5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("CollectAll() returned %d results, want 1", len(results))
	}

	if results[0].HealthCheck == nil {
		t.Error("HealthCheck should be populated when CheckHealth is true")
	}

	if mock.healthCount.Load() != 1 {
		t.Error("CheckHealth should have been called")
	}
}

// TestStatusCollector_CollectAll_HealthCheckError tests health check error handling.
func TestStatusCollector_CollectAll_HealthCheckError(t *testing.T) {
	mock := newMockChecker("service1")
	mock.healthErr = errors.New("health check failed")

	collector := NewStatusCollector([]ServiceChecker{mock}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		CheckHealth: true,
		Timeout:     5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("CollectAll() returned %d results, want 1", len(results))
	}

	// Should have error in details, not fail the whole collection
	if results[0].Details["health_check_error"] == "" {
		t.Error("health_check_error should be in details")
	}
}

// TestStatusCollector_CollectAll_StatusError tests status check error handling.
func TestStatusCollector_CollectAll_StatusError(t *testing.T) {
	mock := newMockChecker("service1")
	mock.statusErr = errors.New("status check failed")

	collector := NewStatusCollector([]ServiceChecker{mock}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("CollectAll() returned %d results, want 1", len(results))
	}

	if results[0].Status != StatusError {
		t.Errorf("status = %v, want StatusError", results[0].Status)
	}

	if results[0].Details["error"] == "" {
		t.Error("error should be in details")
	}
}

// TestStatusCollector_CollectAll_ParallelError tests parallel collection with error.
func TestStatusCollector_CollectAll_ParallelError(t *testing.T) {
	mock1 := newMockChecker("service1")
	mock2 := newMockChecker("service2")
	mock2.statusErr = errors.New("service2 failed")

	collector := NewStatusCollector([]ServiceChecker{mock1, mock2}, 5*time.Second)

	results, err := collector.CollectAll(context.Background(), StatusOptions{
		Parallel: true,
		Timeout:  5 * time.Second,
	})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("CollectAll() returned %d results, want 2", len(results))
	}

	// First should succeed
	if results[0].Status != StatusActive {
		t.Errorf("result[0].Status = %v, want Active", results[0].Status)
	}

	// Second should be error
	if results[1].Status != StatusError {
		t.Errorf("result[1].Status = %v, want Error", results[1].Status)
	}
}

// TestStatusCollector_CollectAll_DefaultTimeout tests default timeout usage.
func TestStatusCollector_CollectAll_DefaultTimeout(t *testing.T) {
	mock := newMockChecker("service1")
	collector := NewStatusCollector([]ServiceChecker{mock}, 10*time.Second)

	// No timeout in options, should use collector's default
	results, err := collector.CollectAll(context.Background(), StatusOptions{})

	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("CollectAll() returned %d results, want 1", len(results))
	}
}

// TestStatusCollector_filterCheckers tests filter logic.
func TestStatusCollector_filterCheckers(t *testing.T) {
	mock1 := newMockChecker("aws")
	mock2 := newMockChecker("gcp")
	mock3 := newMockChecker("azure")

	collector := NewStatusCollector([]ServiceChecker{mock1, mock2, mock3}, 5*time.Second)

	tests := []struct {
		name     string
		services []string
		want     int
	}{
		{"empty filter returns all", nil, 3},
		{"empty slice returns all", []string{}, 3},
		{"single match", []string{"aws"}, 1},
		{"multiple matches", []string{"aws", "azure"}, 2},
		{"no matches", []string{"kubernetes"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collector.filterCheckers(tt.services)
			if len(got) != tt.want {
				t.Errorf("filterCheckers(%v) = %d checkers, want %d", tt.services, len(got), tt.want)
			}
		})
	}
}

// TestStatusCollector_checkService tests single service check.
func TestStatusCollector_checkService(t *testing.T) {
	mock := newMockChecker("test")
	collector := NewStatusCollector(nil, 5*time.Second)

	// Test without health check
	status, err := collector.checkService(context.Background(), mock, StatusOptions{})
	if err != nil {
		t.Fatalf("checkService() error = %v", err)
	}
	if status.Name != "test" {
		t.Errorf("status.Name = %q, want %q", status.Name, "test")
	}

	// Test with health check
	status, err = collector.checkService(context.Background(), mock, StatusOptions{CheckHealth: true})
	if err != nil {
		t.Fatalf("checkService() with health error = %v", err)
	}
	if status.HealthCheck == nil {
		t.Error("HealthCheck should be set when CheckHealth is true")
	}
}

// TestStatusCollector_checkService_HealthError tests health error handling in checkService.
func TestStatusCollector_checkService_HealthError(t *testing.T) {
	mock := newMockChecker("test")
	mock.healthErr = errors.New("health failed")
	collector := NewStatusCollector(nil, 5*time.Second)

	status, err := collector.checkService(context.Background(), mock, StatusOptions{CheckHealth: true})
	if err != nil {
		t.Fatalf("checkService() error = %v", err)
	}

	if status.Details == nil {
		t.Fatal("Details should be initialized")
	}

	if status.Details["health_check_error"] == "" {
		t.Error("health_check_error should be set")
	}
}

// TestStatusCollector_checkService_NilDetails tests nil details handling.
func TestStatusCollector_checkService_NilDetails(t *testing.T) {
	mock := newMockChecker("test")
	mock.status.Details = nil // Force nil details
	mock.healthErr = errors.New("health failed")
	collector := NewStatusCollector(nil, 5*time.Second)

	status, err := collector.checkService(context.Background(), mock, StatusOptions{CheckHealth: true})
	if err != nil {
		t.Fatalf("checkService() error = %v", err)
	}

	// Should have created details map
	if status.Details == nil {
		t.Error("Details should be initialized even if originally nil")
	}
}
