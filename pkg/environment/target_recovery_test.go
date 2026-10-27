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
)

type targetProbe struct {
	journalProbeSwitcher
	captureErr    error
	captured      any
	queried       bool
	switched      bool
	captureReturn any
}

type applyingProbe struct {
	targetProbe
	applied      any
	journalPath  string
	journalState any
}

func (p *applyingProbe) ApplyState(_ context.Context, state any) error {
	p.applied = state
	j, err := LoadJournal(p.journalPath)
	if err != nil {
		return err
	}
	if j == nil {
		return errors.New("missing persisted plan")
	}
	p.journalState, err = deserializeState(j.Services["aws"])
	return err
}

func TestManagerAppliesPersistedCapture(t *testing.T) {
	for _, mode := range []string{"apply", "dry-run", "capture-error"} {
		t.Run(mode, func(t *testing.T) {
			dir := withTempConfigDir(t)
			captured := &AWSRecoveryState{Version: 1, Request: AWSConfig{Profile: "C"}}
			p := &applyingProbe{targetProbe: targetProbe{captureReturn: captured}, journalPath: journalPathIn(dir)}
			if mode == "capture-error" {
				p.captureErr = errors.New("capture failure")
			}
			es := NewEnvironmentSwitcher()
			es.RegisterServiceSwitcher("aws", p)
			_, err := es.SwitchEnvironment(context.Background(), &Environment{Name: "plan", Services: map[string]ServiceConfig{"aws": {AWS: &AWSConfig{Profile: "C"}}}}, SwitchOptions{DryRun: mode == "dry-run"})
			if (err != nil) != (mode == "capture-error") || p.switched || p.queried {
				t.Fatalf("err=%v switched=%v queried=%v", err, p.switched, p.queried)
			}
			if mode == "apply" {
				if p.applied != captured || p.journalState == nil || *p.journalState.(*AWSRecoveryState) != *captured {
					t.Fatalf("plan replaced applied=%v journal=%v", p.applied, p.journalState)
				}
			} else if p.applied != nil || p.journalState != nil {
				t.Fatal("unexpected application")
			}
		})
	}
}

func (p *targetProbe) CaptureState(_ context.Context, request any) (any, error) {
	p.captured = request
	if p.captureReturn == nil {
		p.captureReturn = &AWSRecoveryState{Version: 1}
	}
	return p.captureReturn, p.captureErr
}

func (p *targetProbe) GetCurrentState(context.Context) (any, error) {
	p.queried = true
	return nil, errors.New("display snapshot must not be used")
}

func (p *targetProbe) Switch(_ context.Context, _ any) error {
	p.switched = true
	return nil
}

func TestTargetCaptureBeforeJournalAndSwitch(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "preflight-failure"}[fail], func(t *testing.T) {
			dir := withTempConfigDir(t)
			path := journalPathIn(dir)
			p := &targetProbe{}
			if fail {
				p.captureErr = errors.New("ENV conflict")
			}
			es := NewEnvironmentSwitcher()
			es.RegisterServiceSwitcher("aws", p)
			request := &AWSConfig{Profile: "C"}
			_, err := es.SwitchEnvironment(context.Background(), &Environment{Name: "test", Services: map[string]ServiceConfig{"aws": {AWS: request}}}, SwitchOptions{})
			if (err != nil) != fail || p.queried || p.captured != request || p.switched == fail {
				t.Fatalf("err=%v queried=%v captured=%p switched=%v", err, p.queried, p.captured, p.switched)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("journal unexpectedly remains: %v", err)
			}
		})
	}
}

func TestTargetJournalWireRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	state := &AWSRecoveryState{Version: 1, Request: AWSConfig{Profile: "C"}}
	if err := WriteJournal(path, false, "test", map[string]any{"aws": state}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The v0.1.1 reader has this exact int field and ordinary json.Unmarshal.
	var legacy struct {
		Version  int                     `json:"version"`
		Services map[string]JournalEntry `json:"services"`
	}
	if err := json.Unmarshal(raw, &legacy); err == nil {
		t.Fatal("legacy int reader accepted target journal")
	}
	j, err := LoadJournal(path)
	if err != nil || j.Version != 2 {
		t.Fatalf("journal=%v err=%v", j, err)
	}
	decoded, err := deserializeState(j.Services["aws"])
	if err != nil || decoded.(*AWSRecoveryState).Request.Profile != "C" {
		t.Fatalf("state=%v err=%v", decoded, err)
	}
	reserialized, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(reserialized, &legacy); err == nil {
		t.Fatal("reserialization downgraded discriminator")
	}
	for _, version := range []string{"2", `"unknown"`, "null"} {
		if err := json.Unmarshal([]byte(`{"version":`+version+`}`), &RollbackJournal{}); err == nil {
			t.Fatalf("accepted version %s", version)
		}
	}
}

func TestRecoveryPreflightsWholeMixedJournal(t *testing.T) {
	good, _ := marshalTyped("DockerConfig", &DockerConfig{Context: "old"})
	legacy, _ := marshalTyped("AWSConfig", &AWSConfig{Profile: "old"})
	validTarget, _ := marshalTyped("AWSRecoveryState", &AWSRecoveryState{Version: 1})
	for _, bad := range []JournalEntry{legacy, validTarget, {Type: "AWSRecoveryState", Data: json.RawMessage(`{"Version":1}`)}, {Type: "SSHConfig", Data: json.RawMessage(`{}`)}} {
		path := filepath.Join(t.TempDir(), "journal.json")
		j := RollbackJournal{Version: 2, Services: map[string]JournalEntry{"docker": good, "aws": bad}}
		raw, err := json.Marshal(j)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		calls := 0
		es := NewEnvironmentSwitcher()
		for _, name := range []string{"docker", "aws"} {
			es.RegisterServiceSwitcher(name, &recoverMockSwitcher{name: name, onRollback: func(any) error { calls++; return nil }})
		}
		if err := es.RecoverFromJournal(context.Background(), path); err == nil || !strings.Contains(err.Error(), "preflight") {
			t.Fatalf("expected preflight refusal: %v", err)
		}
		if calls != 0 {
			t.Fatalf("mutated %d services before refusal", calls)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("must retain journal: %v", err)
		}
	}
}

func TestRecoveryRejectsUnregisteredServiceBeforeWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	if err := WriteJournal(path, false, "test", map[string]any{"docker": &DockerConfig{}, "ssh": &SSHConfig{}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	es := NewEnvironmentSwitcher()
	es.RegisterServiceSwitcher("docker", &recoverMockSwitcher{name: "docker", onRollback: func(any) error { calls++; return nil }})
	if err := es.RecoverFromJournal(context.Background(), path); err == nil {
		t.Fatal("accepted unregistered switcher")
	}
	if calls != 0 {
		t.Fatal("rollback before complete preflight")
	}
}

func TestLegacyCustomRecoveryRemainsSupported(t *testing.T) {
	for _, state := range []any{map[string]any{"Context": "custom"}, nil} {
		path := filepath.Join(t.TempDir(), "journal.json")
		if err := WriteJournal(path, false, "test", map[string]any{"custom": state}); err != nil {
			t.Fatal(err)
		}
		calls := 0
		es := NewEnvironmentSwitcher()
		es.RegisterServiceSwitcher("custom", &recoverMockSwitcher{name: "custom", onRollback: func(got any) error {
			calls++
			if state == nil && got != nil {
				t.Fatalf("nil became %v", got)
			}
			if state != nil && got.(map[string]any)["Context"] != "custom" {
				t.Fatalf("custom state=%v", got)
			}
			return nil
		}})
		if err := es.RecoverFromJournal(context.Background(), path); err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("calls=%d", calls)
		}
	}
}

type validatingRecoveryProbe struct{ recoverMockSwitcher }

func (*validatingRecoveryProbe) ValidateRecoveryState(any) error { return nil }

func TestCorruptTargetMetadataNeverStartsRecovery(t *testing.T) {
	good, err := marshalTyped(awsRecoveryType, &AWSRecoveryState{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"valid":            string(good.Data),
		"non-object":       `"invalid"`,
		"missing-presence": strings.Replace(string(good.Data), `"Present":false`, `"Omitted":false`, 1),
		"null-presence":    strings.Replace(string(good.Data), `"Present":false`, `"Present":null`, 1),
		"bad-request":      strings.Replace(string(good.Data), `"Request":{"Profile":"","Region":"","AccountID":""}`, `"Request":"invalid"`, 1),
		"bad-field-type":   strings.Replace(string(good.Data), `"Version":1`, `"Version":"invalid"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "journal.json")
			j := RollbackJournal{Version: 2, Services: map[string]JournalEntry{"aws": {Type: awsRecoveryType, Data: json.RawMessage(data)}}}
			raw, err := json.Marshal(j)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			es := NewEnvironmentSwitcher()
			es.RegisterServiceSwitcher("aws", &validatingRecoveryProbe{recoverMockSwitcher{name: "aws", onRollback: func(any) error { calls++; return nil }}})
			err = es.RecoverFromJournal(context.Background(), path)
			if name == "valid" {
				if err != nil || calls != 1 {
					t.Fatalf("valid recovery baseline: calls=%d err=%v", calls, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "deserialize") {
				t.Fatalf("expected metadata decode refusal: %v", err)
			}
			if calls != 0 {
				t.Fatal("rollback called before metadata validation")
			}
		})
	}
	for _, j := range []RollbackJournal{{Version: 99}, {Version: 1, Services: map[string]JournalEntry{"aws": good}}} {
		if _, err := json.Marshal(j); err == nil {
			t.Fatal("writer accepted unsupported/downgraded recovery format")
		}
	}
}

func TestTargetCaptureCannotProceedWithoutJournal(t *testing.T) {
	prev := userConfigDir
	userConfigDir = func() (string, error) { return "", errors.New("config root unavailable") }
	t.Cleanup(func() { userConfigDir = prev })
	p := &targetProbe{}
	es := NewEnvironmentSwitcher()
	es.RegisterServiceSwitcher("aws", p)
	_, err := es.SwitchEnvironment(context.Background(), &Environment{Name: "test", Services: map[string]ServiceConfig{"aws": {AWS: &AWSConfig{Profile: "C"}}}}, SwitchOptions{})
	if err == nil || p.switched {
		t.Fatalf("target mutation without recovery journal: %v", err)
	}
}

func TestRecoveryRejectsMisplacedTargetMetadata(t *testing.T) {
	entry, err := marshalTyped(awsRecoveryType, &AWSRecoveryState{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		version any
		service string
	}{
		{"legacy-envelope", 1, "aws"},
		{"foreign-service", targetJournalVersion, "custom"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "journal.json")
			// Deliberately bypass the safe writer to model a corrupt/foreign file.
			raw, err := json.Marshal(map[string]any{"version": tc.version, "services": map[string]JournalEntry{tc.service: entry}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			es := NewEnvironmentSwitcher()
			es.RegisterServiceSwitcher(tc.service, &validatingRecoveryProbe{recoverMockSwitcher{name: tc.service, onRollback: func(any) error { calls++; return nil }}})
			if err := es.RecoverFromJournal(context.Background(), path); err == nil {
				t.Fatal("accepted incompatible recovery metadata")
			}
			if calls != 0 {
				t.Fatal("rollback started before wire/service validation")
			}
		})
	}
}
