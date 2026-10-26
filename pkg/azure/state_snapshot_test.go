// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func TestStateSnapshotProcess(t *testing.T) {
	if os.Getenv("GZ_SNAPSHOT_FIXTURE") != "1" {
		return
	}
	switch os.Getenv("GZ_SNAPSHOT_MODE") {
	case "exit":
		os.Exit(23)
	case "wait":
		time.Sleep(30 * time.Second)
	default:
		fmt.Print(os.Getenv("GZ_SNAPSHOT_OUTPUT"))
	}
	os.Exit(0)
}

func snapshotFixtureCommand(t *testing.T, ctx context.Context, mode, output string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestStateSnapshotProcess$", "--")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GZ_SNAPSHOT_FIXTURE=1", "GZ_SNAPSHOT_MODE="+mode, "GZ_SNAPSHOT_OUTPUT="+output, "GORACE=atexit_sleep_ms=0")
	return cmd
}

func TestStateSnapshotSuccessAndEmpty(t *testing.T) {
	for _, blank := range []bool{false, true} {
		calls := 0
		ctx := context.Background()
		state, err := currentStateWithCommand(ctx, func(got context.Context, name string, args ...string) *exec.Cmd {
			if got != ctx {
				t.Fatal("context identity changed")
			}
			if name != "az" {
				t.Fatalf("unexpected value: %v; want %v", name, "az")
			}
			expected := [][]string{{"account", "show", "--query", "id", "-o", "tsv"}, {"account", "show", "--query", "tenantId", "-o", "tsv"}}
			if calls >= len(expected) || !slices.Equal(expected[calls], args) {
				t.Fatalf("unexpected args: %v", args)
			}
			calls++
			output := fmt.Sprintf("  value%d \n", calls)
			if blank {
				output = " \n\t"
			}
			return snapshotFixtureCommand(t, got, "success", output)
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("unexpected value: %v; want %v", calls, 2)
		}
		config, ok := state.(*environment.AzureConfig)
		if !ok {
			t.Fatalf("unexpected state type: %T", state)
		}
		if blank {
			if config.Subscription != "" {
				t.Fatalf("expected empty value: %q", config.Subscription)
			}
			if config.Tenant != "" {
				t.Fatalf("expected empty value: %q", config.Tenant)
			}
		} else {
			if config.Subscription != "value1" {
				t.Fatalf("unexpected value: %v; want %v", config.Subscription, "value1")
			}
			if config.Tenant != "value2" {
				t.Fatalf("unexpected value: %v; want %v", config.Tenant, "value2")
			}
		}
	}
}

func TestStateSnapshotFailuresStopQueries(t *testing.T) {
	for _, position := range []int{1, 2} {
		for _, mode := range []string{"exit", "missing", "cancel"} {
			t.Run(fmt.Sprintf("%s-%d", mode, position), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				state, err := currentStateWithCommand(ctx, func(got context.Context, name string, _ ...string) *exec.Cmd {
					if got != ctx {
						t.Fatal("context identity changed")
					}
					if name != "az" {
						t.Fatalf("unexpected value: %v; want %v", name, "az")
					}
					calls++
					if calls != position {
						return snapshotFixtureCommand(t, got, "success", "previous\n")
					}
					switch mode {
					case "missing":
						return exec.CommandContext(got, filepath.Join(t.TempDir(), "absent-tool"))
					case "cancel":
						cancel()
						return snapshotFixtureCommand(t, got, "wait", "")
					default:
						return snapshotFixtureCommand(t, got, "exit", "")
					}
				})
				if err == nil {
					t.Fatal("expected query error")
				}
				if state != nil {
					t.Fatalf("partial state returned: %#v", state)
				}
				if calls != position {
					t.Fatalf("unexpected value: %v; want %v", calls, position)
				}
				query := []string{"subscription", "tenant"}[position-1]
				if !strings.Contains(err.Error(), "azure "+query+" query") {
					t.Fatalf("unexpected query error: %v", err)
				}
				switch mode {
				case "exit":
					var exitErr *exec.ExitError
					if !errors.As(err, &exitErr) {
						t.Fatalf("missing exit error: %v", err)
					}
					if exitErr.ExitCode() != 23 {
						t.Fatalf("unexpected exit code: %d", exitErr.ExitCode())
					}
				case "missing":
					if !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("error %v does not wrap %v", err, os.ErrNotExist)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("error %v does not wrap %v", err, context.Canceled)
					}
				}
			})
		}
	}
}

func TestStateSnapshotCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	state, err := currentStateWithCommand(ctx, func(context.Context, string, ...string) *exec.Cmd { calls++; return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v does not wrap %v", err, context.Canceled)
	}
	if state != nil {
		t.Fatalf("partial state returned: %#v", state)
	}
	if calls != 0 {
		t.Fatalf("unexpected calls: %d", calls)
	}
	for _, position := range []int{1, 2} {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		calls = 0
		state, err = currentStateWithCommand(ctx, func(got context.Context, _ string, _ ...string) *exec.Cmd {
			calls++
			if calls == position {
				return snapshotFixtureCommand(t, got, "wait", "")
			}
			return snapshotFixtureCommand(t, got, "success", "first")
		})
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error %v does not wrap %v", err, context.DeadlineExceeded)
		}
		if state != nil {
			t.Fatalf("partial state returned: %#v", state)
		}
		if calls != position {
			t.Fatalf("unexpected value: %v; want %v", calls, position)
		}
	}
}

func TestStateSnapshotPublicEntryPreCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, err := NewSwitcher().GetCurrentState(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v does not wrap %v", err, context.Canceled)
	}
	if state != nil {
		t.Fatalf("partial state returned: %#v", state)
	}
}
