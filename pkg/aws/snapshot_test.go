// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func isolateSnapshotHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(key, home)
	}
	t.Setenv("AWS_PROFILE", "")
}

func successfulSnapshotOutput(context.Context, string, ...string) ([]byte, error) { return nil, nil }

func TestSnapshotFailuresAndValues(t *testing.T) {
	isolateSnapshotHome(t)
	expectedArgs := [][]string{{"configure", "get", "region"}}
	queries := []string{"region"}
	for position := 1; position <= len(queries); position++ {
		failure := errors.New("controlled query failure")
		calls := 0
		state, err := NewSwitcher().getCurrentState(context.Background(), func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls++
			snapshotCheck(t, reflect.DeepEqual("aws", name), "Equal assertion")
			snapshotCheck(t, reflect.DeepEqual(expectedArgs[calls-1], args), "Equal assertion")
			if calls == position {
				return []byte("partial output must be discarded"), failure
			}
			return []byte("value\n"), nil
		})
		snapshotCheck(t, errors.Is(err, failure), "ErrorIs assertion")
		snapshotCheck(t, err != nil && strings.Contains(err.Error(), queries[position-1]), "ErrorContains assertion")
		snapshotCheck(t, state == nil, "Nil assertion")
		snapshotCheck(t, reflect.DeepEqual(position, calls), "Equal assertion")
	}
	for _, value := range []string{"  fixture-value \n", "\n"} {
		state, err := NewSwitcher().getCurrentState(context.Background(), func(context.Context, string, ...string) ([]byte, error) { return []byte(value), nil })
		snapshotCheck(t, err == nil, "NoError assertion")
		config, ok := state.(*environment.AWSConfig)
		snapshotCheck(t, ok, "AWSConfig type")
		want := ""
		if value != "\n" {
			want = "fixture-value"
		}
		snapshotCheck(t, reflect.DeepEqual(&environment.AWSConfig{Region: want}, config), "AWS queried region and unqueried metadata")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	state, err := NewSwitcher().getCurrentState(ctx, func(context.Context, string, ...string) ([]byte, error) { calls++; return nil, nil })
	snapshotCheck(t, errors.Is(err, context.Canceled), "ErrorIs assertion")
	snapshotCheck(t, state == nil, "Nil assertion")
	snapshotCheck(t, calls == 0, "Zero assertion")
	ctx, cancel = context.WithCancel(context.Background())
	failure := errors.New("failure during cancel")
	state, err = NewSwitcher().getCurrentState(ctx, func(context.Context, string, ...string) ([]byte, error) { cancel(); return nil, failure })
	snapshotCheck(t, errors.Is(err, context.Canceled), "ErrorIs assertion")
	snapshotCheck(t, errors.Is(err, failure), "ErrorIs assertion")
	snapshotCheck(t, state == nil, "Nil assertion")
}

func TestSnapshotMissingExecutable(t *testing.T) {
	isolateSnapshotHome(t)
	t.Setenv("PATH", t.TempDir())
	state, err := NewSwitcher().GetCurrentState(context.Background())
	snapshotCheck(t, err != nil, "Error assertion")
	snapshotCheck(t, state == nil, "Nil assertion")
	snapshotCheck(t, err != nil && strings.Contains(err.Error(), "aws"), "ErrorContains assertion")
}

// 현재 테스트 실행체만 고정 실행하며 실제 vendor CLI는 실행하지 않는다.
func TestSnapshotNativeCommand(t *testing.T) {
	isolateSnapshotHome(t)
	workingDir := t.TempDir()
	t.Chdir(workingDir)
	for _, mode := range []string{"success", "failure"} {
		output, err := nativeSnapshotCommand(context.Background(), "-test.run=^TestSnapshotCommandChild$", "--", "snapshot-child", mode)
		if mode == "success" {
			snapshotCheck(t, err == nil, "NoError assertion")
			snapshotCheck(t, reflect.DeepEqual("native fixture\n", string(output)), "Equal assertion")
		} else {
			var exitErr *exec.ExitError
			snapshotCheck(t, errors.As(err, &exitErr), "ErrorAs assertion")
			snapshotCheck(t, reflect.DeepEqual(17, exitErr.ExitCode()), "Equal assertion")
		}
	}
	marker := filepath.Join(t.TempDir(), "started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := os.Stat(marker); err == nil {
					observed <- true
					cancel()
					return
				}
			case <-deadline.C:
				observed <- false
				cancel()
				return
			}
		}
	}()
	state, err := NewSwitcher().getCurrentState(ctx, func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		return nativeSnapshotCommand(ctx, "-test.run=^TestSnapshotCommandChild$", "--", "snapshot-child", "wait", marker)
	})
	snapshotCheck(t, <-observed, "True assertion")
	cwd, readErr := os.ReadFile(marker)
	snapshotCheck(t, readErr == nil, "NoError assertion")
	expectedInfo, statErr := os.Stat(workingDir)
	snapshotCheck(t, statErr == nil, "NoError assertion")
	actualInfo, statErr := os.Stat(string(cwd))
	snapshotCheck(t, statErr == nil, "NoError assertion")
	snapshotCheck(t, os.SameFile(expectedInfo, actualInfo), "True assertion")
	snapshotCheck(t, errors.Is(err, context.Canceled), "ErrorIs assertion")
	snapshotCheck(t, state == nil, "Nil assertion")
}

func TestSnapshotCommandChild(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "snapshot-child" {
			continue
		}
		switch os.Args[i+1] {
		case "success":
			_, _ = fmt.Fprint(os.Stdout, "native fixture\n")
			os.Exit(0)
		case "failure":
			os.Exit(17)
		case "wait":
			cwd, err := os.Getwd()
			if err != nil {
				os.Exit(18)
			}
			if err := os.WriteFile(os.Args[i+2], []byte(cwd), 0o600); err != nil {
				os.Exit(18)
			}
			for {
				time.Sleep(time.Hour)
			}
		}
	}
}

func TestSnapshotAWSProfileResolutionIsStrictOnlyForTransactions(t *testing.T) {
	isolateSnapshotHome(t)
	profile, err := resolveActiveProfileStrict()
	snapshotCheck(t, err == nil, "NoError assertion")
	snapshotCheck(t, profile == "", "Empty assertion")
	snapshotCheck(t, writeActiveProfile("stored") == nil, "NoError assertion")
	profile, err = resolveActiveProfileStrict()
	snapshotCheck(t, err == nil, "NoError assertion")
	snapshotCheck(t, reflect.DeepEqual("stored", profile), "Equal assertion")
	path, err := activeProfilePath()
	snapshotCheck(t, err == nil, "NoError assertion")
	snapshotCheck(t, os.Remove(path) == nil, "NoError assertion")
	snapshotCheck(t, os.Mkdir(path, 0o700) == nil, "NoError assertion")
	calls := 0
	state, err := NewSwitcher().getCurrentState(context.Background(), func(context.Context, string, ...string) ([]byte, error) { calls++; return nil, nil })
	snapshotCheck(t, err != nil, "Error assertion")
	snapshotCheck(t, state == nil, "Nil assertion")
	snapshotCheck(t, calls == 0, "Zero assertion")
	var pathErr *os.PathError
	snapshotCheck(t, errors.As(err, &pathErr), "ErrorAs assertion")
	snapshotCheck(t, resolveActiveProfile() == "", "Empty assertion")
	t.Setenv("AWS_PROFILE", "environment-wins")
	state, err = NewSwitcher().getCurrentState(context.Background(), func(_ context.Context, name string, args ...string) ([]byte, error) {
		snapshotCheck(t, reflect.DeepEqual("aws", name), "Equal assertion")
		snapshotCheck(t, reflect.DeepEqual([]string{"configure", "get", "region", "--profile", "environment-wins"}, args), "Equal assertion")
		calls++
		return []byte("region\n"), nil
	})
	snapshotCheck(t, err == nil, "NoError assertion")
	snapshotCheck(t, reflect.DeepEqual(1, calls), "Equal assertion")
	config, ok := state.(*environment.AWSConfig)
	snapshotCheck(t, ok, "True assertion")
	snapshotCheck(t, reflect.DeepEqual("environment-wins", config.Profile), "Equal assertion")
}

func snapshotCheck(t *testing.T, condition bool, description string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s failed", description)
	}
}

// 고정된 현재 테스트 실행체로만 성공·실패·취소를 관찰한다.
func nativeSnapshotCommand(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, os.Args[0], args...).Output()
}
