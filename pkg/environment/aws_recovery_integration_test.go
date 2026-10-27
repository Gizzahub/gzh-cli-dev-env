// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

type afterAWSFailure struct {
	sawProfile bool
	path       string
}

func (p *afterAWSFailure) Name() string { return "docker" }
func (p *afterAWSFailure) GetCurrentState(context.Context) (any, error) {
	return &environment.DockerConfig{Context: "old"}, nil
}

func (p *afterAWSFailure) Switch(context.Context, any) error {
	raw, err := os.ReadFile(p.path)
	p.sawProfile = err == nil && string(raw) == "C\n"
	return errors.New("later service failed")
}
func (p *afterAWSFailure) Rollback(context.Context, any) error { return nil }

// The real manager and AWS owner restore absent/empty/raw selection state after
// a later service fails. Profile-only switching must never invoke a vendor CLI.
func TestAWSManagerRestoresExactSelectionAfterLaterFailure(t *testing.T) {
	for _, original := range []string{"absent", "", "B"} {
		t.Run(original, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("APPDATA", filepath.Join(home, "AppData"))
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
			t.Setenv("PATH", "")
			for _, name := range []string{"AWS_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION"} {
				t.Setenv(name, "")
			}
			configRoot, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(configRoot, "gzh-dev-env", "aws-active-profile")
			var originalMode os.FileMode
			if original != "absent" {
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				originalMode = info.Mode().Perm()
			}
			later := &afterAWSFailure{path: path}
			manager := environment.NewEnvironmentSwitcher()
			manager.Register(aws.NewSwitcher())
			manager.Register(later)
			result, err := manager.SwitchEnvironment(context.Background(), &environment.Environment{
				Name: "recovery", Services: map[string]environment.ServiceConfig{
					"aws": {AWS: &environment.AWSConfig{Profile: "C"}}, "docker": {Docker: &environment.DockerConfig{Context: "new"}},
				}, Dependencies: []string{"aws -> docker"},
			}, environment.SwitchOptions{RollbackOnError: true})
			if err == nil || result == nil || result.Success || !result.RollbackPerformed || !later.sawProfile {
				t.Fatalf("result=%+v err=%v sawProfile=%v", result, err, later.sawProfile)
			}
			assertSelection := func() {
				t.Helper()
				raw, readErr := os.ReadFile(path)
				if original == "absent" {
					if !os.IsNotExist(readErr) {
						t.Fatalf("absence not restored: %q %v", raw, readErr)
					}
					return
				}
				if readErr != nil || string(raw) != original {
					t.Fatalf("raw state=%q err=%v", raw, readErr)
				}
				info, statErr := os.Stat(path)
				if statErr != nil || info.Mode().Perm() != originalMode {
					t.Fatalf("mode not restored: %v %v", info, statErr)
				}
			}
			assertSelection()
			journalPath, err := environment.DefaultJournalPath()
			if err != nil {
				t.Fatal(err)
			}
			journal, err := environment.LoadJournal(journalPath)
			if err != nil || journal == nil || journal.Version != 2 {
				t.Fatalf("journal=%+v err=%v", journal, err)
			}
			// Retained failure journal is replayable even after in-process rollback.
			if err := manager.RecoverFromJournal(context.Background(), journalPath); err != nil {
				t.Fatal(err)
			}
			assertSelection()
			if _, err := os.Stat(journalPath); !os.IsNotExist(err) {
				t.Fatalf("journal retained after full recovery: %v", err)
			}
		})
	}
}
