// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withTempConfigDir redirects userConfigDir to a temp dir for the duration of t.
func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = prev })
	return dir
}

func journalPathIn(dir string) string {
	return filepath.Join(dir, journalAppDir, journalFileName)
}

// journalProbeSwitcher records whether the journal file existed when Switch ran.
type journalProbeSwitcher struct {
	name        string
	state       any
	switchError error
	sawJournal  bool
	journalPath string
}

func (m *journalProbeSwitcher) Name() string { return m.name }

func (m *journalProbeSwitcher) Switch(ctx context.Context, config any) error {
	if m.journalPath != "" {
		if _, err := os.Stat(m.journalPath); err == nil {
			m.sawJournal = true
		}
	}
	return m.switchError
}

func (m *journalProbeSwitcher) GetCurrentState(ctx context.Context) (any, error) {
	return m.state, nil
}

func (m *journalProbeSwitcher) Rollback(ctx context.Context, previousState any) error {
	return nil
}

// TestSwitchEnvironment_Journal writes the journal before Switch and clears it on success.
func TestSwitchEnvironment_Journal(t *testing.T) {
	dir := withTempConfigDir(t)
	path := journalPathIn(dir)

	probe := &journalProbeSwitcher{
		name:        "aws",
		state:       &AWSConfig{Profile: "staging", Region: "us-east-1"},
		journalPath: path,
	}
	es := NewEnvironmentSwitcher()
	es.Register(probe)

	env := &Environment{
		Name: "prod",
		Services: map[string]ServiceConfig{
			"aws": {AWS: &AWSConfig{Profile: "prod", Region: "us-west-2"}},
		},
	}

	ctx := context.Background()
	result, err := es.SwitchEnvironment(ctx, env, SwitchOptions{})
	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}
	if !result.Success {
		t.Fatal("SwitchEnvironment() should succeed")
	}
	if !probe.sawJournal {
		t.Error("journal must exist before Switch (crash-recovery artifact)")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("journal must be cleared on successful completion; stat err = %v", err)
	}
}

// TestSwitchEnvironment_Journal_FailureLeavesJournal asserts a failed switch leaves
// the journal for DetectStaleJournal / RecoverFromJournal.
func TestSwitchEnvironment_Journal_FailureLeavesJournal(t *testing.T) {
	dir := withTempConfigDir(t)
	path := journalPathIn(dir)

	probe := &journalProbeSwitcher{
		name:        "aws",
		state:       &AWSConfig{Profile: "staging", Region: "us-east-1"},
		switchError: errors.New("switch failed"),
		journalPath: path,
	}
	es := NewEnvironmentSwitcher()
	es.Register(probe)

	env := &Environment{
		Name: "prod",
		Services: map[string]ServiceConfig{
			"aws": {AWS: &AWSConfig{Profile: "prod"}},
		},
	}

	_, err := es.SwitchEnvironment(context.Background(), env, SwitchOptions{})
	if err == nil {
		t.Fatal("expected switch error")
	}
	if !probe.sawJournal {
		t.Error("journal must be written before the failing Switch")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("journal should remain after failure: %v", err)
	}
	var journal RollbackJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		t.Fatalf("parse journal: %v", err)
	}
	if journal.Environment != "prod" {
		t.Errorf("Environment = %q, want prod", journal.Environment)
	}
	entry, ok := journal.Services["aws"]
	if !ok {
		t.Fatal("journal missing aws service")
	}
	if entry.Type != "AWSConfig" {
		t.Errorf("Type = %q, want AWSConfig", entry.Type)
	}
	if strings.Contains(string(entry.Data), "secret") || strings.Contains(string(entry.Data), "password") {
		t.Error("journal must not contain secret-like fields")
	}
	var cfg AWSConfig
	if err := json.Unmarshal(entry.Data, &cfg); err != nil {
		t.Fatalf("unmarshal AWSConfig: %v", err)
	}
	if cfg.Profile != "staging" || cfg.Region != "us-east-1" {
		t.Errorf("stored state = %+v, want staging/us-east-1", cfg)
	}
}

// TestSwitchEnvironment_DryRun_NoJournal proves dry-run creates no journal file.
// The guard is inside WriteJournal; without it this test fails because
// SwitchEnvironment still captures previousStates and would call the writer.
func TestSwitchEnvironment_DryRun_NoJournal(t *testing.T) {
	dir := withTempConfigDir(t)
	path := journalPathIn(dir)

	es := NewEnvironmentSwitcher()
	es.Register(newMockSwitcher("aws"))

	env := &Environment{
		Name: "test-env",
		Services: map[string]ServiceConfig{
			"aws": {AWS: &AWSConfig{Profile: "test"}},
		},
	}

	result, err := es.SwitchEnvironment(context.Background(), env, SwitchOptions{DryRun: true})
	if err != nil {
		t.Fatalf("SwitchEnvironment() error = %v", err)
	}
	if !result.Success {
		t.Error("DryRun should succeed")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not create journal at %s (guard must live in WriteJournal)", path)
	}

	// Direct proof: WriteJournal with dryRun=true is a no-op; dryRun=false creates the file.
	states := map[string]any{"aws": &AWSConfig{Profile: "x"}}
	if err := WriteJournal(path, true, "env", states); err != nil {
		t.Fatalf("WriteJournal(dryRun=true) error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("WriteJournal(dryRun=true) must not create a file — guard is missing")
	}
	if err := WriteJournal(path, false, "env", states); err != nil {
		t.Fatalf("WriteJournal(dryRun=false) error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("WriteJournal(dryRun=false) should create file: %v", err)
	}
}

// TestDetectStaleJournal covers detection, warning text, and RecoverFromJournal.
func TestDetectStaleJournal(t *testing.T) {
	dir := withTempConfigDir(t)
	path := journalPathIn(dir)

	t.Run("absent", func(t *testing.T) {
		j, warning, err := DetectStaleJournal(path)
		if err != nil {
			t.Fatalf("DetectStaleJournal() error = %v", err)
		}
		if j != nil || warning != "" {
			t.Errorf("expected empty detection, got journal=%v warning=%q", j, warning)
		}
	})

	t.Run("present", func(t *testing.T) {
		states := map[string]any{
			"aws": &AWSConfig{Profile: "old", Region: "eu-west-1"},
			"gcp": &GCPConfig{Project: "old-proj", Region: "europe-west1"},
		}
		if err := WriteJournal(path, false, "staging", states); err != nil {
			t.Fatalf("WriteJournal: %v", err)
		}

		j, warning, err := DetectStaleJournal(path)
		if err != nil {
			t.Fatalf("DetectStaleJournal() error = %v", err)
		}
		if j == nil {
			t.Fatal("expected journal")
		}
		if j.Environment != "staging" {
			t.Errorf("Environment = %q, want staging", j.Environment)
		}
		if len(j.Services) != 2 {
			t.Errorf("Services len = %d, want 2", len(j.Services))
		}
		if warning == "" {
			t.Fatal("expected non-empty warning for CLI")
		}
		if !strings.Contains(warning, "incomplete environment switch") {
			t.Errorf("warning missing key phrase: %q", warning)
		}
		if !strings.Contains(warning, "staging") {
			t.Errorf("warning should mention env name: %q", warning)
		}
	})

	t.Run("recover", func(t *testing.T) {
		// Fresh journal for recovery path.
		_ = os.Remove(path)
		states := map[string]any{
			"aws": &AWSConfig{Profile: "recover-me", Region: "ap-northeast-1"},
		}
		if err := WriteJournal(path, false, "crash-env", states); err != nil {
			t.Fatalf("WriteJournal: %v", err)
		}

		rollbackCalled := false
		mock := &recoverMockSwitcher{
			name: "aws",
			onRollback: func(state any) error {
				rollbackCalled = true
				cfg, ok := state.(*AWSConfig)
				if !ok {
					t.Errorf("rollback state type = %T, want *AWSConfig", state)
					return nil
				}
				if cfg.Profile != "recover-me" {
					t.Errorf("Profile = %q, want recover-me", cfg.Profile)
				}
				return nil
			},
		}
		es := NewEnvironmentSwitcher()
		es.Register(mock)

		if err := es.RecoverFromJournal(context.Background(), path); err != nil {
			t.Fatalf("RecoverFromJournal() error = %v", err)
		}
		if !rollbackCalled {
			t.Error("RecoverFromJournal must call Rollback")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("RecoverFromJournal must clear journal on success")
		}
	})

	t.Run("default_path", func(t *testing.T) {
		// DetectStaleJournal("") uses DefaultJournalPath under our temp config dir.
		states := map[string]any{"docker": &DockerConfig{Context: "desktop-linux"}}
		if err := WriteJournal(path, false, "env", states); err != nil {
			t.Fatalf("WriteJournal: %v", err)
		}
		j, warning, err := DetectStaleJournal("")
		if err != nil {
			t.Fatalf("DetectStaleJournal(\"\") error = %v", err)
		}
		if j == nil || warning == "" {
			t.Fatal("expected journal + warning via default path")
		}
	})
}

type recoverMockSwitcher struct {
	name       string
	onRollback func(state any) error
}

func (m *recoverMockSwitcher) Name() string { return m.name }
func (m *recoverMockSwitcher) Switch(ctx context.Context, config any) error {
	return nil
}
func (m *recoverMockSwitcher) GetCurrentState(ctx context.Context) (any, error) {
	return &AWSConfig{}, nil
}
func (m *recoverMockSwitcher) Rollback(ctx context.Context, previousState any) error {
	if m.onRollback != nil {
		return m.onRollback(previousState)
	}
	return nil
}

func TestSerializeState_StripsSecrets(t *testing.T) {
	type opaque struct {
		Profile  string
		Password string
		Token    string
		Region   string
	}
	entry, err := serializeState(&opaque{
		Profile:  "ok",
		Password: "s3cret",
		Token:    "tok",
		Region:   "us-east-1",
	})
	if err != nil {
		t.Fatalf("serializeState: %v", err)
	}
	raw := string(entry.Data)
	if strings.Contains(raw, "s3cret") || strings.Contains(raw, "tok") {
		t.Errorf("secret values leaked into journal data: %s", raw)
	}
	if !strings.Contains(raw, "ok") || !strings.Contains(raw, "us-east-1") {
		t.Errorf("safe fields missing: %s", raw)
	}
}

func TestWriteJournal_EmptyStatesNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "j.json")
	if err := WriteJournal(path, false, "env", map[string]any{}); err != nil {
		t.Fatalf("WriteJournal empty: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("empty previousStates must not create a journal file")
	}
}

func TestClearJournal_MissingOK(t *testing.T) {
	if err := ClearJournal(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatalf("ClearJournal missing: %v", err)
	}
}

func TestLoadJournal_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "j.json")
	started := time.Now().UTC().Truncate(time.Second)
	states := map[string]any{
		"ssh": &SSHConfig{Config: "/home/u/.ssh/config.work"},
	}
	if err := WriteJournal(path, false, "work", states); err != nil {
		t.Fatalf("WriteJournal: %v", err)
	}
	j, err := LoadJournal(path)
	if err != nil {
		t.Fatalf("LoadJournal: %v", err)
	}
	if j.Version != journalVersion {
		t.Errorf("Version = %d, want %d", j.Version, journalVersion)
	}
	if j.Environment != "work" {
		t.Errorf("Environment = %q", j.Environment)
	}
	if j.StartedAt.Before(started.Add(-time.Minute)) {
		t.Errorf("StartedAt unexpected: %v", j.StartedAt)
	}
	entry := j.Services["ssh"]
	state, err := deserializeState(entry)
	if err != nil {
		t.Fatalf("deserializeState: %v", err)
	}
	cfg, ok := state.(*SSHConfig)
	if !ok || cfg.Config != "/home/u/.ssh/config.work" {
		t.Errorf("deserialized = %#v", state)
	}
}
