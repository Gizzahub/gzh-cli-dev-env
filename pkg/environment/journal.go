// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// journalAppDir matches pkg/aws active-profile layout under user config.
	journalAppDir = "gzh-dev-env"
	// journalFileName is the on-disk rollback journal file.
	journalFileName = "rollback-journal.json"
	// journalVersion is the schema version of RollbackJournal.
	journalVersion = 1
)

// userConfigDir resolves the OS user config directory. Overridable in tests.
var userConfigDir = os.UserConfigDir

// JournalEntry is one service's previous state in the rollback journal.
// Only safe fields (profiles, paths, regions) are stored in Data.
type JournalEntry struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// RollbackJournal is the disk-persisted previous-state snapshot for crash recovery.
type RollbackJournal struct {
	Version     int                     `json:"version"`
	Environment string                  `json:"environment"`
	StartedAt   time.Time               `json:"startedAt"`
	Services    map[string]JournalEntry `json:"services"`
}

// DefaultJournalPath returns `{UserConfigDir}/gzh-dev-env/rollback-journal.json`.
func DefaultJournalPath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, journalAppDir, journalFileName), nil
}

// WriteJournal persists previousStates to path.
//
// Dry-run must not create a journal file: the guard lives here (not only at call
// sites) so every writer path inherits it — same ownership rule as runHooks /
// rollbackServices.
func WriteJournal(path string, dryRun bool, envName string, previousStates map[string]any) error {
	if dryRun {
		return nil
	}
	if path == "" {
		return fmt.Errorf("journal path is empty")
	}
	if len(previousStates) == 0 {
		return nil
	}

	journal := &RollbackJournal{
		Version:     journalVersion,
		Environment: envName,
		StartedAt:   time.Now().UTC(),
		Services:    make(map[string]JournalEntry, len(previousStates)),
	}
	for name, state := range previousStates {
		entry, err := serializeState(state)
		if err != nil {
			return fmt.Errorf("serialize state for %s: %w", name, err)
		}
		journal.Services[name] = entry
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create journal dir: %w", err)
	}

	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal journal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write journal: %w", err)
	}
	return nil
}

// ClearJournal removes the journal file. Missing file is not an error.
func ClearJournal(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove journal: %w", err)
	}
	return nil
}

// LoadJournal reads a journal from path. Returns (nil, nil) when the file is absent.
func LoadJournal(path string) (*RollbackJournal, error) {
	if path == "" {
		return nil, nil //nolint:nilnil // absent journal is not an error
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil // missing file is an empty journal
		}
		return nil, fmt.Errorf("read journal: %w", err)
	}
	var journal RollbackJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return nil, fmt.Errorf("parse journal: %w", err)
	}
	return &journal, nil
}

// DetectStaleJournal checks for a leftover journal after a crashed/incomplete switch.
// Returns the journal (if any) and an optional CLI warning string.
func DetectStaleJournal(path string) (*RollbackJournal, string, error) {
	if path == "" {
		var err error
		path, err = DefaultJournalPath()
		if err != nil {
			return nil, "", err
		}
	}
	journal, err := LoadJournal(path)
	if err != nil {
		return nil, "", err
	}
	if journal == nil || len(journal.Services) == 0 {
		return nil, "", nil
	}
	warning := fmt.Sprintf(
		"warning: incomplete environment switch detected (journal from %s for env %q, %d service(s)). "+
			"Previous states are on disk — call RecoverFromJournal to restore, or ClearJournal to discard.",
		journal.StartedAt.Format(time.RFC3339),
		journal.Environment,
		len(journal.Services),
	)
	return journal, warning, nil
}

// RecoverFromJournal loads the journal, rolls each service back via registered
// switchers, then clears the journal on full success.
func (es *EnvironmentSwitcher) RecoverFromJournal(ctx context.Context, path string) error {
	if path == "" {
		var err error
		path, err = DefaultJournalPath()
		if err != nil {
			return err
		}
	}
	journal, err := LoadJournal(path)
	if err != nil {
		return err
	}
	if journal == nil || len(journal.Services) == 0 {
		return fmt.Errorf("no rollback journal found at %s", path)
	}

	var errs []string
	for serviceName, entry := range journal.Services {
		es.mu.RLock()
		switcher, exists := es.serviceSwitchers[serviceName]
		es.mu.RUnlock()
		if !exists {
			errs = append(errs, fmt.Sprintf("no switcher for %s", serviceName))
			continue
		}
		state, err := deserializeState(entry)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: deserialize: %v", serviceName, err))
			continue
		}
		if err := switcher.Rollback(ctx, state); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", serviceName, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("journal recovery incomplete: %s", strings.Join(errs, "; "))
	}
	return ClearJournal(path)
}
