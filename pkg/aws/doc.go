// Package aws provides AWS-specific implementations for environment switching
// and status checking.
//
// This package implements:
//   - AWSSwitcher: Switches AWS profiles, regions, and credentials
//   - AWSChecker: Checks AWS service status and health
//
// Example usage:
//
//	switcher := aws.NewSwitcher()
//	err := switcher.Switch(ctx, &aws.Config{Profile: "production"})
package aws
