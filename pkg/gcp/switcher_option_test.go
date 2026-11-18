// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// fakeGcloud puts a gcloud on PATH that appends its argv to a log and returns
// the log path. Reading the log tells whether Switch ran any command.
func fakeGcloud(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake gcloud is a POSIX shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + log + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gcloud"), []byte(script), 0o700); err != nil { // #nosec G306 -- test fixture executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return log
}

func calls(t *testing.T, log string) string {
	t.Helper()
	data, err := os.ReadFile(log) // #nosec G304 -- path from t.TempDir in fakeGcloud
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSwitch_RunsGcloudForPlainValues(t *testing.T) {
	log := fakeGcloud(t)
	cfg := &environment.GCPConfig{Project: "p1", Account: "a@example.com", Region: "asia-northeast3"}
	if err := NewSwitcher().Switch(context.Background(), cfg); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	want := "config set project p1\nconfig set account a@example.com\nconfig set compute/region asia-northeast3\n"
	if got := calls(t, log); got != want {
		t.Fatalf("gcloud calls = %q, want %q", got, want)
	}
}

func TestSwitch_RejectsOptionLikeValuesBeforeAnyCommand(t *testing.T) {
	tests := map[string]*environment.GCPConfig{
		"project": {Project: "--impersonate-service-account=x", Account: "a", Region: "r"},
		"account": {Project: "p", Account: "-a", Region: "r"},
		"region":  {Project: "p", Account: "a", Region: "--verbosity=debug"},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			log := fakeGcloud(t)
			err := NewSwitcher().Switch(context.Background(), cfg)
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("Switch() error = %v, want ErrOptionLikeValue", err)
			}
			if got := calls(t, log); got != "" {
				t.Fatalf("gcloud ran despite rejection: %q", got)
			}
		})
	}
}

func TestRollback_RejectsOptionLikeSnapshot(t *testing.T) {
	log := fakeGcloud(t)
	err := NewSwitcher().Rollback(context.Background(), &environment.GCPConfig{Project: "-p"})
	if !errors.Is(err, environment.ErrOptionLikeValue) {
		t.Fatalf("Rollback() error = %v, want ErrOptionLikeValue", err)
	}
	if got := calls(t, log); got != "" {
		t.Fatalf("gcloud ran despite rejection: %q", got)
	}
}
