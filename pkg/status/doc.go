// Package status provides service status checking and monitoring functionality.
//
// The main abstractions are:
//   - ServiceChecker: Interface for checking individual service status
//   - StatusCollector: Aggregates status from multiple checkers
//   - Formatter: Formats status output for display
//
// Example usage:
//
//	collector := status.NewCollector(
//	    aws.NewChecker(),
//	    gcp.NewChecker(),
//	)
//	statuses, err := collector.Collect(ctx)
package status
