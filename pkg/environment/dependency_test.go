// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"strings"
	"testing"
)

// TestNewDependencyResolver tests constructor.
func TestNewDependencyResolver(t *testing.T) {
	services := map[string]ServiceConfig{
		"aws": {},
		"gcp": {},
	}
	deps := []string{"aws -> gcp"}

	resolver := NewDependencyResolver(services, deps)

	if resolver == nil {
		t.Fatal("NewDependencyResolver returned nil")
	}
	if len(resolver.services) != 2 {
		t.Errorf("services count = %d, want 2", len(resolver.services))
	}
	if len(resolver.dependencies) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(resolver.dependencies))
	}
}

// TestDependencyResolver_ResolveDependencies tests basic resolution.
func TestDependencyResolver_ResolveDependencies(t *testing.T) {
	tests := []struct {
		name         string
		services     map[string]ServiceConfig
		dependencies []string
		wantLevels   int
		wantErr      bool
		errContains  string
	}{
		{
			name: "no dependencies",
			services: map[string]ServiceConfig{
				"aws":        {},
				"gcp":        {},
				"kubernetes": {},
			},
			dependencies: nil,
			wantLevels:   1, // All in same level
			wantErr:      false,
		},
		{
			name: "linear dependency chain",
			services: map[string]ServiceConfig{
				"aws":        {},
				"gcp":        {},
				"kubernetes": {},
			},
			dependencies: []string{
				"aws -> gcp",
				"gcp -> kubernetes",
			},
			wantLevels: 3, // aws, then gcp, then kubernetes
			wantErr:    false,
		},
		{
			name: "parallel services with shared dependency",
			services: map[string]ServiceConfig{
				"base":    {},
				"service1": {},
				"service2": {},
			},
			dependencies: []string{
				"base -> service1",
				"base -> service2",
			},
			wantLevels: 2, // base first, then service1 & service2 in parallel
			wantErr:    false,
		},
		{
			name: "invalid dependency format",
			services: map[string]ServiceConfig{
				"aws": {},
			},
			dependencies: []string{"invalid"},
			wantErr:     true,
			errContains: "invalid dependency format",
		},
		{
			name: "source service not found",
			services: map[string]ServiceConfig{
				"gcp": {},
			},
			dependencies: []string{"aws -> gcp"},
			wantErr:     true,
			errContains: "source service",
		},
		{
			name: "target service not found",
			services: map[string]ServiceConfig{
				"aws": {},
			},
			dependencies: []string{"aws -> gcp"},
			wantErr:     true,
			errContains: "target service",
		},
		{
			name: "circular dependency",
			services: map[string]ServiceConfig{
				"aws": {},
				"gcp": {},
			},
			dependencies: []string{
				"aws -> gcp",
				"gcp -> aws",
			},
			wantErr:     true,
			errContains: "circular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewDependencyResolver(tt.services, tt.dependencies)
			groups, err := resolver.ResolveDependencies()

			if tt.wantErr {
				if err == nil {
					t.Error("ResolveDependencies() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveDependencies() error = %v", err)
			}

			if len(groups) != tt.wantLevels {
				t.Errorf("got %d levels, want %d", len(groups), tt.wantLevels)
			}
		})
	}
}

// TestDependencyResolver_GetExecutionOrder tests flattened order.
func TestDependencyResolver_GetExecutionOrder(t *testing.T) {
	services := map[string]ServiceConfig{
		"aws":        {},
		"gcp":        {},
		"kubernetes": {},
	}
	deps := []string{
		"aws -> gcp",
		"gcp -> kubernetes",
	}

	resolver := NewDependencyResolver(services, deps)
	order, err := resolver.GetExecutionOrder()

	if err != nil {
		t.Fatalf("GetExecutionOrder() error = %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("got %d services, want 3", len(order))
	}

	// aws must come before gcp, gcp must come before kubernetes
	awsIdx, gcpIdx, k8sIdx := -1, -1, -1
	for i, s := range order {
		switch s {
		case "aws":
			awsIdx = i
		case "gcp":
			gcpIdx = i
		case "kubernetes":
			k8sIdx = i
		}
	}

	if awsIdx >= gcpIdx {
		t.Error("aws should come before gcp")
	}
	if gcpIdx >= k8sIdx {
		t.Error("gcp should come before kubernetes")
	}
}

// TestDependencyResolver_GetExecutionOrder_Error tests error propagation.
func TestDependencyResolver_GetExecutionOrder_Error(t *testing.T) {
	services := map[string]ServiceConfig{
		"a": {},
		"b": {},
	}
	deps := []string{"a -> b", "b -> a"} // Circular

	resolver := NewDependencyResolver(services, deps)
	_, err := resolver.GetExecutionOrder()

	if err == nil {
		t.Error("GetExecutionOrder() should return error for circular dependency")
	}
}

// TestDependencyResolver_GetParallelGroups tests parallel grouping.
func TestDependencyResolver_GetParallelGroups(t *testing.T) {
	services := map[string]ServiceConfig{
		"base":     {},
		"service1": {},
		"service2": {},
		"final":    {},
	}
	deps := []string{
		"base -> service1",
		"base -> service2",
		"service1 -> final",
		"service2 -> final",
	}

	resolver := NewDependencyResolver(services, deps)
	groups, err := resolver.GetParallelGroups()

	if err != nil {
		t.Fatalf("GetParallelGroups() error = %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("got %d groups, want 3", len(groups))
	}

	// Level 0: base
	if len(groups[0].Services) != 1 || groups[0].Services[0] != "base" {
		t.Errorf("level 0 should be [base], got %v", groups[0].Services)
	}

	// Level 1: service1 and service2 (parallel)
	if len(groups[1].Services) != 2 {
		t.Errorf("level 1 should have 2 services, got %d", len(groups[1].Services))
	}

	// Level 2: final
	if len(groups[2].Services) != 1 || groups[2].Services[0] != "final" {
		t.Errorf("level 2 should be [final], got %v", groups[2].Services)
	}
}

// TestDependencyResolver_ValidateDependencies tests validation.
func TestDependencyResolver_ValidateDependencies(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]ServiceConfig
		deps     []string
		wantErr  bool
	}{
		{
			name: "valid",
			services: map[string]ServiceConfig{
				"a": {},
				"b": {},
			},
			deps:    []string{"a -> b"},
			wantErr: false,
		},
		{
			name: "circular",
			services: map[string]ServiceConfig{
				"a": {},
				"b": {},
			},
			deps:    []string{"a -> b", "b -> a"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewDependencyResolver(tt.services, tt.deps)
			err := resolver.ValidateDependencies()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseDependency tests dependency string parsing.
func TestParseDependency(t *testing.T) {
	tests := []struct {
		input    string
		wantLen  int
		wantFrom string
		wantTo   string
	}{
		{"aws -> gcp", 2, "aws", "gcp"},
		{"service1 -> service2", 2, "service1", "service2"},
		{"  spaced  ->  values  ", 2, "spaced", "values"},
		{"no-arrow", 1, "", ""}, // Single part, no arrow
		{"", 0, "", ""},         // Empty
		{"multi -> part -> chain", 3, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parts := parseDependency(tt.input)

			if len(parts) != tt.wantLen {
				t.Errorf("parseDependency(%q) = %d parts, want %d", tt.input, len(parts), tt.wantLen)
			}

			if tt.wantLen == 2 {
				if parts[0] != tt.wantFrom {
					t.Errorf("from = %q, want %q", parts[0], tt.wantFrom)
				}
				if parts[1] != tt.wantTo {
					t.Errorf("to = %q, want %q", parts[1], tt.wantTo)
				}
			}
		})
	}
}

// TestTrim tests whitespace trimming.
func TestTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"  hello", "hello"},
		{"hello  ", "hello"},
		{"  hello  ", "hello"},
		{"\thello\t", "hello"},
		{"\nhello\n", "hello"},
		{"  \t\n  ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trim(tt.input)
			if result != tt.expected {
				t.Errorf("trim(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDependencyResolver_ComplexGraph tests a more complex dependency graph.
func TestDependencyResolver_ComplexGraph(t *testing.T) {
	// Diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	services := map[string]ServiceConfig{
		"A": {},
		"B": {},
		"C": {},
		"D": {},
	}
	deps := []string{
		"A -> B",
		"A -> C",
		"B -> D",
		"C -> D",
	}

	resolver := NewDependencyResolver(services, deps)
	groups, err := resolver.GetParallelGroups()

	if err != nil {
		t.Fatalf("GetParallelGroups() error = %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("got %d groups, want 3", len(groups))
	}

	// Verify order
	order, _ := resolver.GetExecutionOrder()

	aIdx, bIdx, cIdx, dIdx := -1, -1, -1, -1
	for i, s := range order {
		switch s {
		case "A":
			aIdx = i
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		case "D":
			dIdx = i
		}
	}

	if aIdx >= bIdx || aIdx >= cIdx {
		t.Error("A should come before B and C")
	}
	if bIdx >= dIdx || cIdx >= dIdx {
		t.Error("B and C should come before D")
	}
}

// TestDependencyResolver_EmptyServices tests with no services.
func TestDependencyResolver_EmptyServices(t *testing.T) {
	resolver := NewDependencyResolver(nil, nil)
	groups, err := resolver.ResolveDependencies()

	if err != nil {
		t.Fatalf("ResolveDependencies() error = %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("got %d groups for empty services, want 0", len(groups))
	}
}

// TestDependencyResolver_SelfDependency tests self-referential dependency.
func TestDependencyResolver_SelfDependency(t *testing.T) {
	services := map[string]ServiceConfig{
		"service": {},
	}
	deps := []string{"service -> service"}

	resolver := NewDependencyResolver(services, deps)
	_, err := resolver.ResolveDependencies()

	if err == nil {
		t.Error("ResolveDependencies() should error on self-dependency")
	}
}
