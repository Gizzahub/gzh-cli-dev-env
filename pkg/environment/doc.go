// Package environment provides core interfaces and implementations for
// unified development environment switching.
//
// The main abstractions are:
//   - ServiceSwitcher: Interface for switching individual services (AWS, GCP, etc.)
//   - EnvironmentSwitcher: Orchestrates multiple service switches atomically
//   - DependencyResolver: Handles service dependencies and ordering
//
// Example usage:
//
//	switcher := environment.NewEnvironmentSwitcher()
//	switcher.Register(aws.NewSwitcher())
//	switcher.Register(gcp.NewSwitcher())
//
//	err := switcher.SwitchEnvironment(ctx, env)
package environment
