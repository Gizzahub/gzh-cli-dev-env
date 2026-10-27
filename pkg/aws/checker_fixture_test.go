// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package aws

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
		Output         string
		Exit           int
		Wait           bool
		ConfigContents *string
	}
	checkerInvocation struct {
		Name string
		Args []string
	}
)

func TestMain(m *testing.M) {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if os.Getenv("GZ_CHECKER_FIXTURE") == "native-v1" && name == "aws" {
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
		if release := os.Getenv("GZ_CHECKER_RELEASE"); release != "" {
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat(release); err == nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		} else {
			time.Sleep(30 * time.Second)
		}
	}
	if response.ConfigContents != nil {
		if err := os.WriteFile(os.Getenv("AWS_CONFIG_FILE"), []byte(*response.ConfigContents), 0o600); err != nil {
			return 92
		}
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
	t.Setenv("GZ_CHECKER_RELEASE", "")
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
	name := "aws"
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
			responses["sts get-caller-identity --output json --profile fixture"] = checkerResponse{Exit: 23}
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
				assertCheckerCalls(t, trace, "aws sts get-caller-identity --output json --profile fixture")
			} else {
				assertCheckerCalls(t, trace)
			}
		})
	}
	trace := checkerFixture(t, map[string]checkerResponse{"sts get-caller-identity --output json --profile fixture": {Wait: true}})
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
	assertCheckerCalls(t, trace, "aws sts get-caller-identity --output json --profile fixture")
}

func defaultCheckerResponses() map[string]checkerResponse {
	return map[string]checkerResponse{
		"configure get region --profile fixture":                  {Output: " fixture-region\n"},
		"sts get-caller-identity --profile fixture":               {},
		"sts get-caller-identity --output json --profile fixture": {Output: `{"Account":"fixture-account"}`},
	}
}

func TestCheckerFixtureActiveAndNoIssuance(t *testing.T) {
	trace := checkerFixture(t, nil)
	state, err := NewChecker().CheckStatus(context.Background())
	if err != nil || state.Status != status.StatusActive || !state.Credentials.Valid || state.Credentials.Type != "aws-credentials" || state.Current.Profile != "fixture" || state.Current.Region != "fixture-region" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace, "aws configure get region --profile fixture", "aws sts get-caller-identity --profile fixture")
	trace = checkerFixture(t, nil)
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil || health.Status != status.StatusActive || health.Details["caller_identity"] != `{"Account":"fixture-account"}` {
		t.Fatalf("health=%#v err=%v", health, err)
	}
	assertCheckerCalls(t, trace, "aws sts get-caller-identity --output json --profile fixture")
}

func TestCheckerFixtureCredentialFailureAndNoProfile(t *testing.T) {
	responses := defaultCheckerResponses()
	responses["sts get-caller-identity --profile fixture"] = checkerResponse{Exit: 23}
	trace := checkerFixture(t, responses)
	state, err := NewChecker().CheckStatus(context.Background())
	if err != nil || state.Status != status.StatusInactive || state.Credentials.Valid || state.Credentials.Warning != CredentialsExpiredMsg || state.Credentials.Type != "aws-credentials" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace, "aws configure get region --profile fixture", "aws sts get-caller-identity --profile fixture")
	trace = checkerFixture(t, nil)
	t.Setenv("AWS_PROFILE", "")
	state, err = NewChecker().CheckStatus(context.Background())
	if err != nil || state.Status != status.StatusInactive || state.Details["error"] != "No AWS profile configured" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace)
}

func TestCheckerFixtureRegionPrecedenceAndCanceledCredentials(t *testing.T) {
	trace := checkerFixture(t, nil)
	t.Setenv("AWS_REGION", "explicit-region")
	state, err := NewChecker().CheckStatus(context.Background())
	if err != nil || state.Current.Region != "explicit-region" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace, "aws sts get-caller-identity --profile fixture")
	trace = checkerFixture(t, nil)
	t.Setenv("AWS_DEFAULT_REGION", "explicit-default-region")
	state, err = NewChecker().CheckStatus(context.Background())
	if err != nil || state.Current.Region != "explicit-default-region" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace, "aws sts get-caller-identity --profile fixture")
	trace = checkerFixture(t, nil)
	t.Setenv("AWS_REGION", "explicit-region")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, err = NewChecker().CheckStatus(ctx)
	if err != nil || state.Credentials.Valid || state.Credentials.Warning == "" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	assertCheckerCalls(t, trace)
}

func TestCheckerFixtureProfileAlignment(t *testing.T) {
	for _, environmentProfile := range []string{"fixture", ""} {
		responses := defaultCheckerResponses()
		responses["configure get region --profile persisted"] = checkerResponse{Output: "persisted-region\n"}
		responses["sts get-caller-identity --profile persisted"] = checkerResponse{}
		responses["sts get-caller-identity --output json --profile persisted"] = checkerResponse{Output: `{"Account":"persisted"}`}
		trace := checkerFixture(t, responses)
		path, err := activeProfilePath()
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(os.Getenv("HOME"), path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
			t.Fatal("profile path escaped owned fixture")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("persisted\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AWS_PROFILE", environmentProfile)
		profile := "fixture"
		if environmentProfile == "" {
			profile = "persisted"
		}
		state, err := NewChecker().CheckStatus(context.Background())
		if err != nil || state.Status != status.StatusActive || state.Current.Profile != profile {
			t.Fatalf("state=%#v err=%v", state, err)
		}
		assertCheckerCalls(t, trace, "aws configure get region --profile "+profile, "aws sts get-caller-identity --profile "+profile)
		if err := os.Remove(trace); err != nil {
			t.Fatal(err)
		}
		health, err := NewChecker().CheckHealth(context.Background())
		if err != nil || health.Status != status.StatusActive {
			t.Fatalf("health=%#v err=%v", health, err)
		}
		assertCheckerCalls(t, trace, "aws sts get-caller-identity --output json --profile "+profile)
	}
	responses := defaultCheckerResponses()
	responses["sts get-caller-identity --output json"] = checkerResponse{Output: `{"Account":"default-chain"}`}
	trace := checkerFixture(t, responses)
	t.Setenv("AWS_PROFILE", "")
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil || health.Status != status.StatusActive {
		t.Fatalf("health=%#v err=%v", health, err)
	}
	assertCheckerCalls(t, trace, "aws sts get-caller-identity --output json")
}

// 이전의 비맥락 실행이 돌아와도 release 파일로 소유 child를 풀고 반드시 회수한다.
func TestCheckerFixtureRegionCancellation(t *testing.T) {
	responses := defaultCheckerResponses()
	responses["configure get region --profile fixture"] = checkerResponse{Wait: true, Output: "fixture-region\n"}
	trace := checkerFixture(t, responses)
	release := filepath.Join(filepath.Dir(trace), "release")
	t.Setenv("GZ_CHECKER_RELEASE", release)
	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		state *status.ServiceStatus
		err   error
	}
	done := make(chan result, 1)
	completed := false
	t.Cleanup(func() {
		cancel()
		if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
			t.Error(err)
		}
		if !completed {
			select {
			case <-done:
			case <-time.After(15 * time.Second):
				t.Error("owned checker did not complete after helper release")
			}
		}
	})
	go func() { state, err := NewChecker().CheckStatus(ctx); done <- result{state, err} }()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if data, err := os.ReadFile(trace); err == nil && strings.HasSuffix(string(data), "\n") {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("region helper did not reach readiness")
		}
	}
	cancel()
	select {
	case got := <-done:
		completed = true
		if got.err != nil || got.state == nil || got.state.Credentials.Valid || got.state.Status != status.StatusInactive {
			t.Fatalf("state=%#v err=%v", got.state, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CheckStatus ignored caller cancellation while reading region")
	}
	if _, err := os.Stat(release); !os.IsNotExist(err) {
		t.Fatal("cancellation required releasing the helper")
	}
	assertCheckerCalls(t, trace, "aws configure get region --profile fixture")
}
