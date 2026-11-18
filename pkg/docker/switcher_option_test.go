// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// fakeDocker puts a docker on PATH that appends its argv to a log and returns
// the log path. Reading the log tells whether Switch ran any command.
func fakeDocker(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake docker is a POSIX shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + log + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o700); err != nil { // #nosec G306 -- test fixture executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return log
}

func dockerCalls(t *testing.T, log string) string {
	t.Helper()
	data, err := os.ReadFile(log) // #nosec G304 -- path from t.TempDir in fakeDocker
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSwitch_RunsDockerForPlainContext(t *testing.T) {
	log := fakeDocker(t)
	if err := NewSwitcher().Switch(context.Background(), &environment.DockerConfig{Context: "desktop-linux"}); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	if got, want := dockerCalls(t, log), "context use desktop-linux\n"; got != want {
		t.Fatalf("docker calls = %q, want %q", got, want)
	}
}

func TestSwitch_RejectsOptionLikeContextBeforeAnyCommand(t *testing.T) {
	for _, value := range []string{"-h", "--help"} {
		t.Run(value, func(t *testing.T) {
			log := fakeDocker(t)
			err := NewSwitcher().Switch(context.Background(), &environment.DockerConfig{Context: value})
			if !errors.Is(err, environment.ErrOptionLikeValue) {
				t.Fatalf("Switch() error = %v, want ErrOptionLikeValue", err)
			}
			if got := dockerCalls(t, log); got != "" {
				t.Fatalf("docker ran despite rejection: %q", got)
			}
		})
	}
}

func TestRollback_RejectsOptionLikeSnapshot(t *testing.T) {
	log := fakeDocker(t)
	err := NewSwitcher().Rollback(context.Background(), &environment.DockerConfig{Context: "-x"})
	if !errors.Is(err, environment.ErrOptionLikeValue) {
		t.Fatalf("Rollback() error = %v, want ErrOptionLikeValue", err)
	}
	if got := dockerCalls(t, log); got != "" {
		t.Fatalf("docker ran despite rejection: %q", got)
	}
}
