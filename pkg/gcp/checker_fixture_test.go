// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

type (
	checkerResponse struct {
		Output string
		Exit   int
		Wait   bool
	}
	checkerInvocation struct {
		Name string
		Args []string
	}
)

func TestMain(m *testing.M) {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if os.Getenv("GZ_CHECKER_FIXTURE") == "native-v1" && name == "gcloud" {
		os.Exit(runCheckerFixture(name))
	}
	os.Exit(m.Run())
}

func runCheckerFixture(name string) int {
	file, err := os.OpenFile(os.Getenv("GZ_CHECKER_TRACE"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 90
	}
	err = json.NewEncoder(file).Encode(checkerInvocation{name, os.Args[1:]})
	if errors.Join(err, file.Close()) != nil {
		return 90
	}
	var responses map[string]checkerResponse
	if json.Unmarshal([]byte(os.Getenv("GZ_CHECKER_RESPONSES")), &responses) != nil {
		return 90
	}
	response, ok := responses[strings.Join(os.Args[1:], " ")]
	if !ok {
		return 91
	}
	if response.Wait {
		time.Sleep(30 * time.Second)
	}
	if response.Exit != 0 {
		fmt.Fprint(os.Stderr, "controlled checker failure")
	}
	fmt.Print(response.Output)
	return response.Exit
}

func checkerFixture(t *testing.T, responses map[string]checkerResponse) string {
	t.Helper()
	root := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(key, root)
	}
	t.Setenv("AWS_PROFILE", "fixture")
	t.Setenv("AWS_DEFAULT_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(root, "aws-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(root, "aws-credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("CLOUDSDK_CONFIG", filepath.Join(root, "gcloud-config"))
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	t.Setenv("GZ_CHECKER_FIXTURE", "native-v1")
	trace := filepath.Join(root, "trace.jsonl")
	t.Setenv("GZ_CHECKER_TRACE", trace)
	if responses == nil {
		responses = defaultCheckerResponses()
	}
	data, err := json.Marshal(responses)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GZ_CHECKER_RESPONSES", string(data))
	input, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	name := "gcloud"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	output, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		t.Fatal(errors.Join(err, source.Close()))
	}
	_, copyErr := io.Copy(output, source)
	if err := errors.Join(copyErr, source.Close(), output.Close()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	t.Setenv("PATHEXT", ".EXE")
	return trace
}

func assertCheckerCalls(t *testing.T, trace string, want ...string) {
	t.Helper()
	data, err := os.ReadFile(trace)
	if os.IsNotExist(err) && len(want) == 0 {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	var calls []string
	for {
		var call checkerInvocation
		err := decoder.Decode(&call)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		calls = append(calls, strings.Join(append([]string{call.Name}, call.Args...), " "))
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v want=%v", calls, want)
	}
}

func TestCheckerFixtureMissingCLI(t *testing.T) {
	trace := checkerFixture(t, nil)
	t.Setenv("PATH", t.TempDir())
	state, err := NewChecker().CheckStatus(context.Background())
	if err != nil || state.Status != status.StatusInactive || state.Details["error"] == "" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace)
}

func TestCheckerFixtureHealthFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			responses := defaultCheckerResponses()
			responses["auth list --format=json"] = checkerResponse{Exit: 23}
			trace := checkerFixture(t, responses)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			health, err := NewChecker().CheckHealth(ctx)
			if err != nil || health.Status != status.StatusError || health.Message == "" {
				t.Fatalf("health=%#v err=%v", health, err)
			}
			if mode == "failure" {
				if health.Details["stderr"] != "controlled checker failure" {
					t.Fatalf("stderr=%v", health.Details)
				}
				assertCheckerCalls(t, trace, "gcloud auth list --format=json")
			} else {
				assertCheckerCalls(t, trace)
			}
		})
	}
	trace := checkerFixture(t, map[string]checkerResponse{"auth list --format=json": {Wait: true}})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	type completion struct {
		health *status.HealthStatus
		err    error
	}
	done := make(chan completion, 1)
	go func() { health, err := NewChecker().CheckHealth(ctx); done <- completion{health, err} }()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	ready := false
	for !ready && ctx.Err() == nil {
		if data, err := os.ReadFile(trace); err == nil && strings.HasSuffix(string(data), "\n") {
			ready = true
			break
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
		}
	}
	cancel()
	result := <-done
	if !ready || result.err != nil || result.health.Status != status.StatusError {
		t.Fatalf("ready=%v health=%#v err=%v", ready, result.health, result.err)
	}
	assertCheckerCalls(t, trace, "gcloud auth list --format=json")
}

func defaultCheckerResponses() map[string]checkerResponse {
	return map[string]checkerResponse{
		"config get-value project":        {Output: " project-1\n"},
		"config get-value account":        {Output: " user@example.invalid\n"},
		"config get-value compute/region": {Output: " region-1\n"},
		"auth print-access-token":         {},
		"auth list --format=json":         {Output: `[{"account":"user@example.invalid"}]`},
	}
}

func TestCheckerFixtureActiveAccountTypes(t *testing.T) {
	for _, account := range []string{"user@example.invalid", "fixture@project.iam.gserviceaccount.com"} {
		responses := defaultCheckerResponses()
		responses["config get-value account"] = checkerResponse{Output: " " + account + "\n"}
		trace := checkerFixture(t, responses)
		state, err := NewChecker().CheckStatus(context.Background())
		wantType := "user-account"
		if strings.Contains(account, ".iam.gserviceaccount.com") {
			wantType = "service-account"
		}
		if err != nil || state.Status != status.StatusActive || !state.Credentials.Valid || state.Credentials.Type != wantType || state.Current.Project != "project-1" || state.Current.Account != account || state.Current.Region != "region-1" {
			t.Fatalf("state=%#v err=%v", state, err)
		}
		assertCheckerCalls(t, trace, "gcloud config get-value project", "gcloud config get-value account", "gcloud config get-value compute/region", "gcloud auth print-access-token", "gcloud config get-value account")
	}
	trace := checkerFixture(t, nil)
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil || health.Status != status.StatusActive || health.Message != "gcloud authentication configuration is readable" || health.Details["auth_list"] != `[{"account":"user@example.invalid"}]` {
		t.Fatalf("health=%#v err=%v", health, err)
	}
	assertCheckerCalls(t, trace, "gcloud auth list --format=json")
}

func TestCheckerFixtureFailuresAndEarlyStop(t *testing.T) {
	for _, query := range []string{"config get-value project", "config get-value account", "config get-value compute/region", "auth print-access-token"} {
		responses := defaultCheckerResponses()
		responses[query] = checkerResponse{Exit: 23}
		trace := checkerFixture(t, responses)
		state, err := NewChecker().CheckStatus(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		switch query {
		case "config get-value project":
			if state.Status != status.StatusError || state.Details["error"] == "" {
				t.Fatalf("state=%#v", state)
			}
			assertCheckerCalls(t, trace, "gcloud config get-value project")
		case "auth print-access-token":
			if state.Credentials.Valid || state.Status != status.StatusInactive || state.Credentials.Warning == "" {
				t.Fatalf("state=%#v", state)
			}
			assertCheckerCalls(t, trace, "gcloud config get-value project", "gcloud config get-value account", "gcloud config get-value compute/region", "gcloud auth print-access-token")
		default:
			if state.Status != status.StatusActive || !state.Credentials.Valid {
				t.Fatalf("state=%#v", state)
			}
			if query == "config get-value account" && (state.Current.Account != "" || state.Credentials.Type != "gcp-credentials") {
				t.Fatalf("state=%#v", state)
			}
			if query == "config get-value compute/region" && state.Current.Region != "" {
				t.Fatalf("state=%#v", state)
			}
			assertCheckerCalls(t, trace, "gcloud config get-value project", "gcloud config get-value account", "gcloud config get-value compute/region", "gcloud auth print-access-token", "gcloud config get-value account")
		}
	}
	responses := defaultCheckerResponses()
	responses["config get-value project"] = checkerResponse{Output: " \n"}
	trace := checkerFixture(t, responses)
	state, err := NewChecker().CheckStatus(context.Background())
	if err != nil || state.Status != status.StatusInactive || state.Details["error"] != "No GCP project configured" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace, "gcloud config get-value project")
	trace = checkerFixture(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, err = NewChecker().CheckStatus(ctx)
	if err != nil || state.Status != status.StatusError {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace)
}
