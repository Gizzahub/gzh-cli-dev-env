// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

const dockerCheckerMarker = "gzh-dev-env-docker-checker-fixture-v1"

func TestMain(m *testing.M) {
	name := filepath.Base(os.Args[0])
	if os.Getenv("GZH_DOCKER_CHECKER_CHILD") == dockerCheckerMarker && (name == "docker" || name == "docker.exe") {
		os.Exit(runDockerCheckerChild())
	}
	os.Exit(m.Run())
}

// 복사된 테스트 실행체만 명령별 응답을 제공하며 Docker나 daemon은 실행하지 않는다.
func runDockerCheckerChild() int {
	args := os.Args[1:]
	trace, err := os.OpenFile(os.Getenv("GZH_DOCKER_CHECKER_TRACE"), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 25
	}
	encodeErr := json.NewEncoder(trace).Encode(args)
	closeErr := trace.Close()
	if encodeErr != nil || closeErr != nil {
		return 25
	}
	mode := os.Getenv("GZH_DOCKER_CHECKER_MODE")
	var output string
	switch strings.Join(args, "|") {
	case "info":
		if mode == "daemon-fail" {
			return 23
		}
	case "context|show":
		if mode == "context-fail" {
			return 23
		}
		if mode != "blank-context" {
			output = "fixture-context\n"
		}
	case "info|--format|{{.ServerVersion}}":
		if mode == "health-fail" {
			_, _ = fmt.Fprint(os.Stderr, "controlled daemon failure")
			return 23
		}
		output = "28.1-fixture\n"
	case "system|df|--format|table":
		if mode == "optional-fail" {
			return 24
		}
		output = "fixture disk data\n"
	case "ps|-q":
		if mode == "optional-fail" {
			return 24
		}
		if mode != "blank-count" {
			output = "first\nsecond\n"
		}
	default:
		return 26
	}
	if _, err := fmt.Fprint(os.Stdout, output); err != nil {
		return 25
	}
	return 0
}

func dockerCheckerFixture(t *testing.T, mode string) func([][]string) {
	t.Helper()
	root := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "DOCKER_CONFIG"} {
		t.Setenv(key, root)
	}
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := "docker"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := copyDockerCheckerExecutable(source, filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(root, "trace.jsonl")
	if err := os.WriteFile(trace, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	t.Setenv("GZH_DOCKER_CHECKER_CHILD", dockerCheckerMarker)
	t.Setenv("GZH_DOCKER_CHECKER_TRACE", trace)
	t.Setenv("GZH_DOCKER_CHECKER_MODE", mode)
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	return func(want [][]string) {
		file, err := os.Open(trace)
		if err != nil {
			t.Fatal(err)
		}
		decoder := json.NewDecoder(file)
		var got [][]string
		var decodeErr error
		for {
			var args []string
			decodeErr = decoder.Decode(&args)
			if decodeErr != nil {
				break
			}
			got = append(got, args)
		}
		closeErr := file.Close()
		if !errors.Is(decodeErr, io.EOF) {
			t.Fatal(decodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("Docker calls = %v, want %v", got, want)
		}
	}
}

func copyDockerCheckerExecutable(source, destination string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, output.Close()) }()
	_, err = io.Copy(output, input)
	return err
}

func TestDockerCheckerStatusContract(t *testing.T) {
	for _, mode := range []string{"normal", "blank-context", "daemon-fail", "context-fail"} {
		t.Run(mode, func(t *testing.T) {
			assertCalls := dockerCheckerFixture(t, mode)
			st, err := NewChecker().CheckStatus(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			want := status.StatusActive
			contextName := "fixture-context"
			if mode == "blank-context" {
				contextName = ""
			}
			if mode == "daemon-fail" {
				want = status.StatusInactive
				contextName = ""
			}
			if mode == "context-fail" {
				want = status.StatusError
				contextName = ""
			}
			if st.Status != want || st.Current.Context != contextName {
				t.Fatalf("status = %+v", st)
			}
			if mode == "context-fail" && !strings.Contains(st.Details["error"], "query Docker context") {
				t.Fatalf("missing query error: %+v", st)
			}
			calls := [][]string{{"info"}}
			if mode != "daemon-fail" {
				calls = append(calls, []string{"context", "show"})
			}
			assertCalls(calls)
		})
	}
}

func TestDockerCheckerHealthContract(t *testing.T) {
	for _, mode := range []string{"normal", "blank-count", "optional-fail", "health-fail"} {
		t.Run(mode, func(t *testing.T) {
			assertCalls := dockerCheckerFixture(t, mode)
			health, err := NewChecker().CheckHealth(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			calls := [][]string{{"info", "--format", "{{.ServerVersion}}"}}
			if mode == "health-fail" {
				if health.Status != status.StatusError || health.Details["stderr"] != "controlled daemon failure" {
					t.Fatalf("health = %+v", health)
				}
			} else {
				if health.Status != status.StatusActive || health.Details["server_version"] != "28.1-fixture" {
					t.Fatalf("health = %+v", health)
				}
				calls = append(calls, []string{"system", "df", "--format", "table"}, []string{"ps", "-q"})
				if mode == "optional-fail" {
					if _, ok := health.Details["disk_usage"]; ok {
						t.Fatal("failed disk query produced a field")
					}
					if _, ok := health.Details["running_containers"]; ok {
						t.Fatal("failed count query produced a field")
					}
				} else {
					want := 2
					if mode == "blank-count" {
						want = 0
					}
					if health.Details["running_containers"] != want || health.Details["disk_usage"] != "fixture disk data\n" {
						t.Fatalf("details = %+v", health.Details)
					}
				}
			}
			assertCalls(calls)
		})
	}
}

func TestDockerCheckerContextCauseAndPrecancel(t *testing.T) {
	assertCalls := dockerCheckerFixture(t, "context-fail")
	value, err := NewChecker().getCurrentContext(context.Background())
	var exitErr *exec.ExitError
	if value != "" || !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 {
		t.Fatalf("context = %q, %v", value, err)
	}
	assertCalls([][]string{{"context", "show"}})
	assertNoCalls := dockerCheckerFixture(t, "normal")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	value, err = NewChecker().getCurrentContext(ctx)
	if value != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context = %q, %v", value, err)
	}
	health, err := NewChecker().CheckHealth(ctx)
	if err != nil || health.Status != status.StatusError {
		t.Fatalf("canceled health = %+v, %v", health, err)
	}
	assertNoCalls(nil)
}

func TestDockerCheckerMissingCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", root)
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil || st.Status != status.StatusInactive || st.Details["error"] != "Docker CLI not found" || st.Current.Context != "" {
		t.Fatalf("missing CLI status = %+v, %v", st, err)
	}
}
