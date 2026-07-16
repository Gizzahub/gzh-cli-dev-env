// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package devenv

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func TestNewRootCmd_UseAndSilence(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "dev-env" {
		t.Errorf("Use = %q, want dev-env", cmd.Use)
	}
	if !cmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
	if cmd.Short == "" {
		t.Error("Short description empty")
	}
}

func TestNewRootCmd_SubcommandFlags(t *testing.T) {
	root := NewRootCmd()

	statusCmd, _, err := root.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Find status: %v", err)
	}
	for _, name := range []string{"service", "format", "check-health", "watch", "timeout", "no-color"} {
		if statusCmd.Flags().Lookup(name) == nil {
			t.Errorf("status missing flag %q", name)
		}
	}

	switchCmd, _, err := root.Find([]string{"switch-all"})
	if err != nil {
		t.Fatalf("Find switch-all: %v", err)
	}
	for _, name := range []string{"env", "from-file", "dry-run", "force", "interactive", "parallel", "timeout"} {
		if switchCmd.Flags().Lookup(name) == nil {
			t.Errorf("switch-all missing flag %q", name)
		}
	}

	tuiCmd, _, err := root.Find([]string{"tui"})
	if err != nil {
		t.Fatalf("Find tui: %v", err)
	}
	if tuiCmd.Flags().Lookup("verbose") == nil {
		t.Error("tui missing verbose flag")
	}
}

func TestCreateServiceCheckers_AllDefault(t *testing.T) {
	checkers := createServiceCheckers(nil)
	if len(checkers) != 6 {
		t.Fatalf("default checkers = %d, want 6", len(checkers))
	}
	names := map[string]bool{}
	for _, c := range checkers {
		names[c.Name()] = true
	}
	for _, want := range []string{"aws", "gcp", "azure", "docker", "kubernetes", "ssh"} {
		if !names[want] {
			t.Errorf("missing checker %q", want)
		}
	}
}

func TestCreateServiceCheckers_FilterAndAlias(t *testing.T) {
	checkers := createServiceCheckers([]string{" AWS ", "k8s", "unknown"})
	if len(checkers) != 2 {
		t.Fatalf("filtered checkers = %d, want 2", len(checkers))
	}
	if checkers[0].Name() != "aws" {
		t.Errorf("first = %q", checkers[0].Name())
	}
	if checkers[1].Name() != "kubernetes" {
		t.Errorf("second = %q", checkers[1].Name())
	}
}

func TestCreateServiceCheckers_EmptyAfterFilter(t *testing.T) {
	checkers := createServiceCheckers([]string{"nope"})
	if len(checkers) != 0 {
		t.Errorf("expected 0 checkers, got %d", len(checkers))
	}
}

func TestCreateFormatter_Formats(t *testing.T) {
	for _, format := range []string{"table", "json", "yaml", "yml", "TABLE"} {
		f, err := createFormatter(format, false)
		if err != nil {
			t.Errorf("createFormatter(%q) error = %v", format, err)
			continue
		}
		if f == nil {
			t.Errorf("createFormatter(%q) returned nil", format)
		}
	}
	_, err := createFormatter("xml", true)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestRunStatusCmd_NoValidServices(t *testing.T) {
	err := runStatusCmd([]string{"nope"}, "table", false, false, time.Second, false)
	if err == nil {
		t.Fatal("expected error for no valid services")
	}
	if !strings.Contains(err.Error(), "no valid services") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestRunStatusCmd_InvalidFormat(t *testing.T) {
	err := runStatusCmd([]string{"aws"}, "xml", false, false, time.Second, false)
	if err == nil {
		t.Fatal("expected invalid format error")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestIsYAMLFile(t *testing.T) {
	cases := map[string]bool{
		"prod.yaml": true,
		"prod.yml":  true,
		"prod.json": false,
		"prod":      false,
		"prod.YAML": false,
	}
	for name, want := range cases {
		if got := isYAMLFile(name); got != want {
			t.Errorf("isYAMLFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestLoadEnvironment_RequiresSelector(t *testing.T) {
	opts := &switchAllOptions{}
	_, err := opts.loadEnvironment()
	if err == nil {
		t.Fatal("expected error when no selector")
	}
	if !strings.Contains(err.Error(), "must specify") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestLoadEnvironment_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prod.yaml")
	content := []byte("name: production\ndescription: prod env\nservices: {}\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &switchAllOptions{fromFile: path}
	env, err := opts.loadEnvironment()
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if env.Name != "production" {
		t.Errorf("Name = %q", env.Name)
	}
}

func TestLoadEnvironment_FromFileMissing(t *testing.T) {
	opts := &switchAllOptions{fromFile: filepath.Join(t.TempDir(), "missing.yaml")}
	_, err := opts.loadEnvironment()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadEnvironment_EnvNameFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	envDir := filepath.Join(home, ".gzh", "dev-env", "environments")
	if err := os.MkdirAll(envDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(envDir, "staging.yaml")
	if err := os.WriteFile(path, []byte("name: staging\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &switchAllOptions{env: "staging"}
	env, err := opts.loadEnvironment()
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if env.Name != "staging" {
		t.Errorf("Name = %q", env.Name)
	}
}

func TestLoadEnvironment_EnvNameMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	opts := &switchAllOptions{env: "no-such-env"}
	_, err := opts.loadEnvironment()
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestFindEnvironmentFile_LocalDir(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("local.yml", []byte("name: local\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &switchAllOptions{}
	got := opts.findEnvironmentFile("local")
	if got == "" {
		t.Fatal("expected to find local.yml")
	}
}

func TestFindAvailableEnvironments(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	envDir := filepath.Join(home, ".gzh", "dev-env", "environments")
	if err := os.MkdirAll(envDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "a.yaml"), []byte("name: a\nservices: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "skip.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "bad.yaml"), []byte(":::"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(envDir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}

	opts := &switchAllOptions{}
	envs, err := opts.findAvailableEnvironments()
	if err != nil {
		t.Fatalf("findAvailableEnvironments() error = %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "a" {
		t.Errorf("envs = %+v, want single env a", envs)
	}
}

func TestFindAvailableEnvironments_MissingDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := (&switchAllOptions{}).findAvailableEnvironments()
	if err == nil {
		t.Fatal("expected error for missing environments dir")
	}
}

func TestDisplaySkippedHooks(t *testing.T) {
	opts := &switchAllOptions{}
	env := &environment.Environment{
		Name: "x",
		PreHooks: []environment.Hook{
			{Command: "echo pre"},
		},
		PostHooks: []environment.Hook{
			{Command: "echo post"},
		},
	}
	out := captureStdout(t, func() {
		opts.displaySkippedHooks(env)
	})
	if !strings.Contains(out, "echo pre") || !strings.Contains(out, "echo post") {
		t.Errorf("output missing hooks: %q", out)
	}

	out = captureStdout(t, func() {
		opts.displaySkippedHooks(&environment.Environment{Name: "empty"})
	})
	if out != "" {
		t.Errorf("expected no output for empty hooks, got %q", out)
	}
}

func TestReportProgress(t *testing.T) {
	opts := &switchAllOptions{}
	out := captureStdout(t, func() {
		opts.reportProgress(environment.SwitchProgress{
			TotalServices:     4,
			CompletedServices: 2,
			CurrentService:    "aws",
			Status:            "switching",
		})
	})
	if !strings.Contains(out, "50.0%") || !strings.Contains(out, "aws") {
		t.Errorf("progress output = %q", out)
	}
}

func TestDisplayResults(t *testing.T) {
	opts := &switchAllOptions{}
	out := captureStdout(t, func() {
		opts.displayResults(&environment.SwitchResult{
			Success:           true,
			Duration:          time.Second,
			SwitchedServices:  []string{"aws"},
			FailedServices:    []string{"gcp"},
			RollbackPerformed: true,
			Errors: []environment.SwitchError{
				{Service: "gcp", Error: "boom", Time: time.Now()},
			},
		})
	})
	for _, want := range []string{"aws", "gcp", "Rollback", "boom"} {
		if !strings.Contains(out, want) {
			t.Errorf("results missing %q in %q", want, out)
		}
	}
}

func TestRegisterDefaultSwitchers(t *testing.T) {
	sw := environment.NewEnvironmentSwitcher()
	registerDefaultSwitchers(sw)
	env := &environment.Environment{
		Name: "dev",
		Services: map[string]environment.ServiceConfig{
			"aws": {AWS: &environment.AWSConfig{Profile: "default", Region: "us-east-1"}},
		},
	}
	// Dry-run never calls Switch; registration is enough for the path to run.
	result, err := sw.SwitchEnvironment(context.Background(), env, environment.SwitchOptions{
		DryRun:  true,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("SwitchEnvironment after register: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success for dry-run, got %+v", result)
	}
}

func TestSwitchAll_Run_MissingEnv(t *testing.T) {
	opts := &switchAllOptions{env: "missing", force: true, timeout: time.Second}
	err := opts.run(t.Context())
	if err == nil {
		t.Fatal("expected error for missing env")
	}
}

func TestSwitchAll_Run_FromFileDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dev.yaml")
	content := []byte(`
name: dev
description: development
services:
  aws:
    aws:
      profile: default
      region: us-east-1
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &switchAllOptions{
		fromFile: path,
		dryRun:   true,
		force:    true,
		timeout:  time.Second,
	}
	out := captureStdout(t, func() {
		if err := opts.run(t.Context()); err != nil {
			t.Fatalf("run() error = %v", err)
		}
	})
	if !strings.Contains(out, "DRY-RUN") {
		t.Errorf("output missing DRY-RUN: %q", out)
	}
	if !strings.Contains(out, "Successfully switched") {
		t.Errorf("output missing success: %q", out)
	}
}

func TestSelectEnvironmentInteractively_None(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := (&switchAllOptions{}).selectEnvironmentInteractively()
	if err == nil {
		t.Fatal("expected error when no environments")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}
