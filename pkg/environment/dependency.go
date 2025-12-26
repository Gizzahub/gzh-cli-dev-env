// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package environment

import (
	"fmt"
	"sort"
)

// DependencyResolver handles service dependency resolution and execution ordering.
type DependencyResolver struct {
	services     map[string]ServiceConfig
	dependencies []string
}

// NewDependencyResolver creates a new dependency resolver.
func NewDependencyResolver(services map[string]ServiceConfig, dependencies []string) *DependencyResolver {
	return &DependencyResolver{
		services:     services,
		dependencies: dependencies,
	}
}

// ResolveDependencies resolves service dependencies and returns execution order.
func (dr *DependencyResolver) ResolveDependencies() ([]ServiceGroup, error) {
	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all services with zero in-degree
	for serviceName := range dr.services {
		inDegree[serviceName] = 0
		graph[serviceName] = []string{}
	}

	// Parse dependencies and build graph
	for _, dep := range dr.dependencies {
		parts := parseDependency(dep)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid dependency format: %s (expected format: 'service1 -> service2')", dep)
		}

		from, to := parts[0], parts[1]

		// Validate that both services exist
		if _, exists := dr.services[from]; !exists {
			return nil, fmt.Errorf("dependency source service '%s' not found", from)
		}
		if _, exists := dr.services[to]; !exists {
			return nil, fmt.Errorf("dependency target service '%s' not found", to)
		}

		// Add edge and update in-degree
		graph[from] = append(graph[from], to)
		inDegree[to]++
	}

	// Check for cycles
	if err := dr.detectCycles(graph); err != nil {
		return nil, err
	}

	// Perform topological sort with level grouping
	return dr.topologicalSort(graph, inDegree)
}

// parseDependency parses a dependency string like "aws -> kubernetes".
func parseDependency(dep string) []string {
	parts := []string{}
	current := ""
	i := 0

	for i < len(dep) {
		if i+3 < len(dep) && dep[i:i+3] == " ->" {
			if current != "" {
				parts = append(parts, trim(current))
				current = ""
			}
			i += 3
			for i < len(dep) && dep[i] == ' ' {
				i++
			}
		} else {
			current += string(dep[i])
			i++
		}
	}

	if current != "" {
		parts = append(parts, trim(current))
	}

	return parts
}

// trim removes leading and trailing whitespace.
func trim(s string) string {
	start := 0
	end := len(s)

	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}

// detectCycles uses DFS to detect cycles in the dependency graph.
func (dr *DependencyResolver) detectCycles(graph map[string][]string) error {
	white := make(map[string]bool) // unvisited
	gray := make(map[string]bool)  // visiting
	black := make(map[string]bool) // visited

	// Initialize all nodes as white (unvisited)
	for service := range dr.services {
		white[service] = true
	}

	// DFS from each unvisited node
	for service := range white {
		if white[service] {
			if err := dr.dfsVisit(service, graph, white, gray, black); err != nil {
				return err
			}
		}
	}

	return nil
}

// dfsVisit performs DFS visit for cycle detection.
func (dr *DependencyResolver) dfsVisit(service string, graph map[string][]string, white, gray, black map[string]bool) error {
	white[service] = false
	gray[service] = true

	for _, neighbor := range graph[service] {
		if gray[neighbor] {
			return fmt.Errorf("circular dependency detected involving services: %s -> %s", service, neighbor)
		}
		if white[neighbor] {
			if err := dr.dfsVisit(neighbor, graph, white, gray, black); err != nil {
				return err
			}
		}
	}

	gray[service] = false
	black[service] = true

	return nil
}

// topologicalSort performs topological sorting with level grouping.
func (dr *DependencyResolver) topologicalSort(graph map[string][]string, inDegree map[string]int) ([]ServiceGroup, error) {
	var groups []ServiceGroup
	level := 0
	remaining := make(map[string]int)

	for service, degree := range inDegree {
		remaining[service] = degree
	}

	for len(remaining) > 0 {
		var currentLevel []string
		for service, degree := range remaining {
			if degree == 0 {
				currentLevel = append(currentLevel, service)
			}
		}

		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected - no services with zero in-degree")
		}

		sort.Strings(currentLevel)

		groups = append(groups, ServiceGroup{
			Services: currentLevel,
			Level:    level,
		})

		for _, service := range currentLevel {
			delete(remaining, service)

			for _, dependent := range graph[service] {
				if _, exists := remaining[dependent]; exists {
					remaining[dependent]--
				}
			}
		}

		level++
	}

	return groups, nil
}

// GetExecutionOrder returns a flattened list of services in execution order.
func (dr *DependencyResolver) GetExecutionOrder() ([]string, error) {
	groups, err := dr.ResolveDependencies()
	if err != nil {
		return nil, err
	}

	var order []string
	for _, group := range groups {
		order = append(order, group.Services...)
	}

	return order, nil
}

// GetParallelGroups returns groups of services that can be executed in parallel.
func (dr *DependencyResolver) GetParallelGroups() ([]ServiceGroup, error) {
	return dr.ResolveDependencies()
}

// ValidateDependencies validates that all dependencies are satisfiable.
func (dr *DependencyResolver) ValidateDependencies() error {
	_, err := dr.ResolveDependencies()
	return err
}
