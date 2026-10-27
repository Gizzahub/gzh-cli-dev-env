// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func recoveryPaths(t *testing.T) (string, string) {
	t.Helper()
	dir := withTempConfigDir(t)
	profile := filepath.Join(dir, configAppDir, activeProfileFileName)
	config := filepath.Join(dir, "aws-config")
	t.Setenv("AWS_CONFIG_FILE", config)
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	return profile, config
}

func captured(t *testing.T, request *environment.AWSConfig) *environment.AWSRecoveryState {
	t.Helper()
	value, err := NewSwitcher().CaptureState(context.Background(), request)
	recoveryNoError(t, err)
	state, ok := value.(*environment.AWSRecoveryState)
	recoveryTrue(t, ok)
	recoveryNoError(t, NewSwitcher().ValidateRecoveryState(state))
	return state
}

func putProfile(t *testing.T, path, value string) {
	t.Helper()
	recoveryNoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	recoveryNoError(t, os.WriteFile(path, []byte(value), 0o640))
}

func TestRecoveryProfileMatrix(t *testing.T) {
	checkerFixture(t, nil)
	for _, env := range []string{"absent", "", "A"} {
		for _, file := range []string{"absent", "", "B"} {
			for _, request := range []string{"", "B", "C"} {
				t.Run(env+"/"+file+"/"+request, func(t *testing.T) {
					path, _ := recoveryPaths(t)
					t.Setenv("AWS_PROFILE", env)
					if env == "absent" {
						recoveryNoError(t, os.Unsetenv("AWS_PROFILE"))
					}
					if file != "absent" {
						putProfile(t, path, file)
					}
					before, err := readProfileFile(path)
					recoveryNoError(t, err)
					value, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: request})
					if env == "A" && request != "" {
						recoveryErrorContains(t, err, "conflicts")
						after, readErr := readProfileFile(path)
						recoveryNoError(t, readErr)
						recoveryEqual(t, before, after)
						return
					}
					recoveryNoError(t, err)
					recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), value))
					recoveryNoError(t, NewSwitcher().Rollback(context.Background(), value))
					after, err := readProfileFile(path)
					recoveryNoError(t, err)
					recoveryEqual(t, before, after)
				})
			}
		}
	}
}

func TestRecoveryTargetRegionAndOtherKeys(t *testing.T) {
	text := "[profile A]\nregion = a-old\n[profile C]\nregion = c-old\noutput = json\n"
	written := "[profile A]\nregion = a-old\n[profile C]\nregion = new-region\noutput = json\n"
	trace := checkerFixture(t, map[string]checkerResponse{"configure set region new-region --profile C": {ConfigContents: &written}})
	path, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, path, "A")
	recoveryNoError(t, os.WriteFile(config, []byte(text), 0o600))
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new-region"})
	recoveryEqual(t, "C", state.TargetProfile)
	recoveryEqual(t, "c-old", state.RegionBefore.Value)
	recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), state))
	calls := readCheckerTrace(t, trace)
	recoveryLen(t, calls, 1)
	recoveryEqual(t, []string{"configure", "set", "region", "new-region", "--profile", "C"}, calls[0].Args)
	// 변경하지 않은 다른 key는 복원 후에도 유지한다.
	recoveryNoError(t, os.WriteFile(config, []byte(written+"format = safe\n"), 0o600))
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	data, err := os.ReadFile(config)
	recoveryNoError(t, err)
	recoveryContains(t, string(data), "region = c-old")
	recoveryContains(t, string(data), "format = safe")
	profile, err := os.ReadFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, "A", string(profile))
}

func TestRecoveryRegionOnlyAndAbsence(t *testing.T) {
	for _, initial := range []string{"missing-file", "missing-key", "empty-value"} {
		t.Run(initial, func(t *testing.T) {
			checkerFixture(t, nil)
			path, config := recoveryPaths(t)
			t.Setenv("AWS_PROFILE", "")
			putProfile(t, path, "B")
			text := ""
			switch initial {
			case "missing-key":
				text = "[profile B]\noutput = json\n"
			case "empty-value":
				text = "[profile B]\nregion = \n"
			}
			if initial != "missing-file" {
				recoveryNoError(t, os.WriteFile(config, []byte(text), 0o640))
			}
			state := captured(t, &environment.AWSConfig{Region: "new-region"})
			recoveryEqual(t, "B", state.TargetProfile)
			written, err := editRegion([]byte(text), "B", state.RegionExpected)
			recoveryNoError(t, err)
			recoveryNoError(t, os.WriteFile(config, written, 0o640))
			recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
			data, exists, _, err := readRegular(config)
			recoveryNoError(t, err)
			recoveryEqual(t, initial != "missing-file", exists)
			region, err := readRegion(data, "B")
			recoveryNoError(t, err)
			recoveryEqual(t, state.RegionBefore, region)
		})
	}
}

func TestRecoveryRejectsBeforeMutation(t *testing.T) {
	checkerFixture(t, nil)
	path, config := recoveryPaths(t)
	putProfile(t, path, "B")
	t.Setenv("AWS_PROFILE", "")
	for _, tc := range []struct {
		name    string
		env     map[string]string
		request environment.AWSConfig
	}{
		{"profile conflict", map[string]string{"AWS_PROFILE": "A"}, environment.AWSConfig{Profile: "C"}},
		{"region conflict", map[string]string{"AWS_REGION": "R"}, environment.AWSConfig{Region: "other"}},
		{"default region conflict", map[string]string{"AWS_DEFAULT_REGION": "R"}, environment.AWSConfig{Region: "other"}},
		{"account unsupported", nil, environment.AWSConfig{AccountID: "account"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			before, err := os.ReadFile(path)
			recoveryNoError(t, err)
			_, err = NewSwitcher().CaptureState(context.Background(), &tc.request)
			recoveryError(t, err)
			after, err := os.ReadFile(path)
			recoveryNoError(t, err)
			recoveryEqual(t, before, after)
			_, err = os.Stat(config)
			recoveryTrue(t, os.IsNotExist(err))
		})
	}
	t.Setenv("AWS_REGION", "R")
	t.Setenv("AWS_DEFAULT_REGION", "other")
	_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Region: "R"})
	recoveryNoError(t, err)
}

func TestRecoveryConflictAndPartialFailure(t *testing.T) {
	trace := checkerFixture(t, map[string]checkerResponse{"configure set region new --profile C": {Exit: 23}})
	path, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, path, "B")
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = old\n"), 0o600))
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	err := NewSwitcher().ApplyState(context.Background(), state)
	recoveryError(t, err)
	recoveryLen(t, readCheckerTrace(t, trace), 1)
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	data, err := os.ReadFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, "B", string(data))
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = concurrent\n"), 0o600))
	recoveryErrorContains(t, NewSwitcher().ApplyState(context.Background(), state), "changed")
	recoveryErrorContains(t, NewSwitcher().Rollback(context.Background(), state), "conflict")
	after, err := os.ReadFile(config)
	recoveryNoError(t, err)
	recoveryContains(t, string(after), "concurrent")
}

func TestRecoveryCancellationValidationAndUnsafeFiles(t *testing.T) {
	checkerFixture(t, nil)
	path, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	state := captured(t, &environment.AWSConfig{Profile: "C"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recoveryErrorIs(t, NewSwitcher().ApplyState(ctx, state), context.Canceled)
	_, err := os.Lstat(path)
	recoveryTrue(t, os.IsNotExist(err))
	recoveryErrorIs(t, NewSwitcher().Rollback(ctx, state), context.Canceled)
	recoveryError(t, NewSwitcher().ValidateRecoveryState(&environment.AWSConfig{}))
	recoveryError(t, NewSwitcher().ValidateRecoveryState((*environment.AWSRecoveryState)(nil)))
	copy := *state
	copy.ProfileBefore.Path = config
	recoveryError(t, NewSwitcher().ValidateRecoveryState(&copy))
	recoveryNoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	target := filepath.Join(filepath.Dir(path), "real")
	recoveryNoError(t, os.WriteFile(target, []byte("B"), 0o600))
	recoveryNoError(t, os.Symlink(target, path))
	_, err = NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C"})
	recoveryErrorContains(t, err, "non-regular")
	recoveryNoError(t, os.Remove(path))
	recoveryNoError(t, os.Mkdir(path, 0o700))
	_, err = NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C"})
	recoveryError(t, err)
}

func TestRecoveryINIAndJournalDTO(t *testing.T) {
	for _, text := range []string{"[profile C]\nregion = a\nregion = b\n", "[profile C]\n[profile C]\n", "bad data", "[profile C]\nregion = a\n continuation\n"} {
		_, err := readRegion([]byte(text), "C")
		recoveryError(t, err)
	}
	checkerFixture(t, nil)
	_, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = old\nsecret = never-journal-this\n"), 0o600))
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	data, err := json.Marshal(state)
	recoveryNoError(t, err)
	recoveryNotContains(t, string(data), "never-journal-this")
	var round environment.AWSRecoveryState
	recoveryNoError(t, json.Unmarshal(data, &round))
	recoveryTrue(t, reflect.DeepEqual(state, &round))
	recoveryNoError(t, NewSwitcher().ValidateRecoveryState(&round))
}

func readCheckerTrace(t *testing.T, path string) []checkerInvocation {
	t.Helper()
	data, err := os.ReadFile(path)
	recoveryNoError(t, err)
	var calls []checkerInvocation
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			var call checkerInvocation
			recoveryNoError(t, json.Unmarshal([]byte(line), &call))
			calls = append(calls, call)
		}
	}
	return calls
}

func recoveryNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func recoveryError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
}

func recoveryErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error %v does not contain %q", err, want)
	}
}

func recoveryErrorIs(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error %v does not wrap %v", err, want)
	}
}

func recoveryEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func recoveryTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatal("expected true")
	}
}

func recoveryFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatal("expected false")
	}
}

func recoveryNotNil(t *testing.T, value any) {
	t.Helper()
	if value == nil || reflect.ValueOf(value).IsNil() {
		t.Fatal("unexpected nil")
	}
}

func recoveryLen(t *testing.T, value any, length int) {
	t.Helper()
	if reflect.ValueOf(value).Len() != length {
		t.Fatalf("unexpected length: %#v", value)
	}
}

func recoveryContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("%q lacks %q", text, want)
	}
}

func recoveryNotContains(t *testing.T, text, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Fatalf("%q contains %q", text, unwanted)
	}
}

func TestRecoveryAfterPartialRegionWrite(t *testing.T) {
	written := "[profile C]\nregion = new\noutput = json\n"
	checkerFixture(t, map[string]checkerResponse{"configure set region new --profile C": {Exit: 23, ConfigContents: &written}})
	path, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, path, "B")
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = old\noutput = json\n"), 0o600))
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	recoveryError(t, NewSwitcher().ApplyState(context.Background(), state))
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	data, err := os.ReadFile(config)
	recoveryNoError(t, err)
	recoveryContains(t, string(data), "region = old")
	recoveryContains(t, string(data), "output = json")
	profile, err := os.ReadFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, "B", string(profile))
}

func TestRecoveryCancellationAfterProfileWrite(t *testing.T) {
	trace := checkerFixture(t, map[string]checkerResponse{"configure set region new --profile C": {Wait: true}})
	path, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, path, "B")
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = old\n"), 0o600))
	release := filepath.Join(t.TempDir(), "release")
	t.Setenv("GZ_CHECKER_RELEASE", release)
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- NewSwitcher().ApplyState(ctx, state) }()
	finished := false
	t.Cleanup(func() {
		cancel()
		if !finished {
			_ = os.WriteFile(release, nil, 0o600)
			select {
			case <-done:
			case <-time.After(12 * time.Second):
				t.Error("owned region helper did not finish")
			}
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, _ := os.ReadFile(trace)
		if len(data) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owned region helper did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 준비된 CLI까지 도달한 뒤 취소하므로 시작 시간 추정에 의존하지 않는다.
	cancel()
	select {
	case err := <-done:
		finished = true
		recoveryErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("region helper did not stop after cancellation")
	}
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	profile, err := os.ReadFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, "B", string(profile))
	data, err := os.ReadFile(config)
	recoveryNoError(t, err)
	recoveryContains(t, string(data), "region = old")
}

func TestRecoveryRejectsAmbiguousINIAndAliases(t *testing.T) {
	checkerFixture(t, nil)
	for _, text := range []string{"[profile \"C\"]\nregion = old\n", "[profile C]\nRegion = old\n", "[profile C]\nREGION = old\n", "[profile C]\n region = old\n"} {
		t.Run(text, func(t *testing.T) {
			profile, config := recoveryPaths(t)
			t.Setenv("AWS_PROFILE", "")
			putProfile(t, profile, "B")
			recoveryNoError(t, os.WriteFile(config, []byte(text), 0o600))
			_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C", Region: "new"})
			recoveryError(t, err)
			unchanged, err := os.ReadFile(profile)
			recoveryNoError(t, err)
			recoveryEqual(t, "B", string(unchanged))
		})
	}
	for _, kind := range []string{"same-path", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			profile, config := recoveryPaths(t)
			t.Setenv("AWS_PROFILE", "")
			putProfile(t, profile, "B")
			if kind == "same-path" {
				t.Setenv("AWS_CONFIG_FILE", profile)
			} else {
				recoveryNoError(t, os.Link(profile, config))
			}
			_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C", Region: "new"})
			recoveryErrorContains(t, err, "alias")
			unchanged, err := os.ReadFile(profile)
			recoveryNoError(t, err)
			recoveryEqual(t, "B", string(unchanged))
		})
	}
}

func TestRecoveryRejectsInjectedJournalRegionAndPreservesEnvironmentChanges(t *testing.T) {
	checkerFixture(t, nil)
	profile, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, profile, "B")
	recoveryNoError(t, os.WriteFile(config, []byte("[profile C]\nregion = old\n"), 0o600))
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	corrupt := *state
	corrupt.RegionBefore.Value = "old\ncredential = injected"
	recoveryError(t, NewSwitcher().ValidateRecoveryState(&corrupt))
	t.Setenv("AWS_PROFILE", "different")
	// 복원 대상은 현재 환경 선택으로 바꾸지 않고 기록한 C를 사용한다.
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	data, err := os.ReadFile(config)
	recoveryNoError(t, err)
	recoveryContains(t, string(data), "region = old")
}

func TestRecoveryRejectsUnsupportedProfileNames(t *testing.T) {
	checkerFixture(t, nil)
	profile, _ := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	putProfile(t, profile, "B")
	for _, name := range []string{"A B", "A\tB", "A\"B", "A'B", "A\\B", "A\x01B"} {
		_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: name, Region: "new"})
		recoveryError(t, err)
		data, readErr := os.ReadFile(profile)
		recoveryNoError(t, readErr)
		recoveryEqual(t, "B", string(data))
	}
}

func TestRecoveryRejectsNoncanonicalSectionWhitespace(t *testing.T) {
	checkerFixture(t, nil)
	for _, section := range []string{"profile  C", "profile\tC", " profile C ", "profile C ", " profile C", "profile\nC", "profile C\t"} {
		t.Run(section, func(t *testing.T) {
			profile, config := recoveryPaths(t)
			t.Setenv("AWS_PROFILE", "")
			putProfile(t, profile, "B")
			text := "[" + section + "]\nregion = old\n"
			recoveryNoError(t, os.WriteFile(config, []byte(text), 0o600))
			_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C", Region: "new"})
			recoveryError(t, err)
			after, err := os.ReadFile(profile)
			recoveryNoError(t, err)
			recoveryEqual(t, "B", string(after))
			unchanged, err := os.ReadFile(config)
			recoveryNoError(t, err)
			recoveryEqual(t, text, string(unchanged))
		})
	}
	region, err := readRegion([]byte("[profile C]\nregion = old\n"), "C")
	recoveryNoError(t, err)
	recoveryEqual(t, environment.AWSRegionState{Present: true, Value: "old"}, region)
}

func TestRecoveryRejectsMissingLeafParentAlias(t *testing.T) {
	profile, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	recoveryNoError(t, os.MkdirAll(filepath.Dir(profile), 0o700))
	aliasParent := filepath.Join(filepath.Dir(config), "linked-parent")
	if err := os.Symlink(filepath.Dir(profile), aliasParent); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("owned symlink fixture unavailable: %v", err)
		}
		t.Fatal(err)
	}
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	alias := filepath.Join(aliasParent, filepath.Base(profile))
	t.Setenv("AWS_CONFIG_FILE", alias)
	_, err := NewSwitcher().CaptureState(context.Background(), &environment.AWSConfig{Profile: "C", Region: "new"})
	recoveryErrorContains(t, err, "alias")
	state.ConfigPath = alias
	recoveryErrorContains(t, NewSwitcher().ValidateRecoveryState(state), "alias")
	for _, path := range []string{profile, alias} {
		_, err := os.Lstat(path)
		recoveryTrue(t, os.IsNotExist(err))
	}
}

func TestRecoveryDistinctMissingParentsRemainValid(t *testing.T) {
	profile, config := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	distinct := filepath.Join(filepath.Dir(config), "missing-parent", "nested", "config")
	t.Setenv("AWS_CONFIG_FILE", distinct)
	state := captured(t, &environment.AWSConfig{Profile: "C", Region: "new"})
	recoveryEqual(t, distinct, state.ConfigPath)
	for _, path := range []string{profile, distinct} {
		_, err := os.Lstat(path)
		recoveryTrue(t, os.IsNotExist(err))
	}
}

func TestRecoveryUnknownAncestorResolutionFailsClosed(t *testing.T) {
	root := t.TempDir()
	loop := filepath.Join(root, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("owned symlink fixture unavailable: %v", err)
		}
		t.Fatal(err)
	}
	recoveryError(t, rejectResourceAlias(filepath.Join(root, "profile"), filepath.Join(loop, "config")))
}
