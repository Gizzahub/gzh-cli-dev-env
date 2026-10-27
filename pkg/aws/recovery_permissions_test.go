// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func permissionState() *environment.AWSRecoveryState {
	return &environment.AWSRecoveryState{Version: awsRecoveryVersion, Request: environment.AWSConfig{Profile: "C"}, ProfileBefore: environment.AWSProfileFileState{Path: "owned"}, ProfileExpected: environment.AWSProfileFileState{Path: "owned", Present: true, Raw: "C\n", Mode: 0o600}, TargetProfile: "C"}
}

func TestProfilePermissionPlatformModel(t *testing.T) {
	for _, tc := range []struct {
		platform   string
		input      uint32
		observable uint32
	}{
		{"windows", 0o600, 0o666},
		{"windows", 0o666, 0o666},
		{"windows", 0o400, 0o444},
		{"windows", 0o444, 0o444},
		{"darwin", 0o600, 0o600},
		{"linux", 0o640, 0o640},
		{"linux", 0o666, 0o666},
	} {
		t.Run(tc.platform+"/"+os.FileMode(tc.input).String(), func(t *testing.T) {
			recoveryEqual(t, tc.observable, observedProfilePermissions(tc.input, tc.platform))
		})
	}
}

func TestProfileLegacyCompatibilityIsNarrow(t *testing.T) {
	state := permissionState()
	before := *state
	effective := compatibleProfileExpected(state, "windows")
	recoveryEqual(t, uint32(0o666), effective.Mode)
	recoveryEqual(t, before, *state)
	recoveryEqual(t, uint32(0o600), compatibleProfileExpected(state, "darwin").Mode)
	for _, tc := range []struct {
		name   string
		change func(*environment.AWSRecoveryState)
	}{
		{"mode640", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o640 }},
		{"mode777", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o777 }},
		{"readonly", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o444 }},
		{"path", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Path = "other" }},
		{"raw", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Raw = "other\n" }},
		{"presence", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Present = false }},
		{"before-present", func(s *environment.AWSRecoveryState) { s.ProfileBefore.Present = true }},
		{"before-raw", func(s *environment.AWSRecoveryState) { s.ProfileBefore.Raw = "B" }},
		{"before-mode", func(s *environment.AWSRecoveryState) { s.ProfileBefore.Mode = 0o600 }},
		{"version", func(s *environment.AWSRecoveryState) { s.Version = 99 }},
		{"empty-profile", func(s *environment.AWSRecoveryState) { s.Request.Profile = "" }},
		{"account", func(s *environment.AWSRecoveryState) { s.Request.AccountID = "unsupported" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := permissionState()
			tc.change(s)
			recoveryEqual(t, s.ProfileExpected, compatibleProfileExpected(s, "windows"))
		})
	}
	readonly := effective
	readonly.Mode = 0o444
	recoveryFalse(t, sameProfile(readonly, effective))
}

func TestProfileNativeExpectedAndAbsenceRecovery(t *testing.T) {
	path, _ := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	state := captured(t, &environment.AWSConfig{Profile: "C"})
	recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), state))
	actual, err := readProfileFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, state.ProfileExpected, actual)
	if runtime.GOOS != "windows" {
		recoveryEqual(t, uint32(0o600), actual.Mode)
	} else {
		recoveryEqual(t, uint32(0o666), actual.Mode)
	}
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
	_, err = os.Lstat(path)
	recoveryTrue(t, os.IsNotExist(err))
}

func TestProfileNativeReadonlyConflict(t *testing.T) {
	path, _ := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	state := captured(t, &environment.AWSConfig{Profile: "C"})
	recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), state))
	recoveryNoError(t, os.Chmod(path, 0o400))
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
	})
	recoveryErrorContains(t, NewSwitcher().Rollback(context.Background(), state), "conflict")
	data, err := os.ReadFile(path)
	recoveryNoError(t, err)
	recoveryEqual(t, "C\n", string(data))
}

func TestProfileLegacySerializedJournalRecovery(t *testing.T) {
	path, _ := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	state := captured(t, &environment.AWSConfig{Profile: "C"})
	// 구 v0.2.0의 신규 파일 metadata를 그대로 보존한다.
	state.ProfileExpected.Mode = 0o600
	recoveryNoError(t, NewSwitcher().ValidateRecoveryState(state))
	recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), state))
	journal := filepath.Join(t.TempDir(), "journal.json")
	recoveryNoError(t, environment.WriteJournal(journal, false, "legacy-windows", map[string]any{"aws": state}))
	data, err := os.ReadFile(journal)
	recoveryNoError(t, err)
	loaded, err := environment.LoadJournal(journal)
	recoveryNoError(t, err)
	var stored environment.AWSRecoveryState
	recoveryNoError(t, json.Unmarshal(loaded.Services["aws"].Data, &stored))
	recoveryEqual(t, uint32(0o600), stored.ProfileExpected.Mode)
	// 복구 전 검증은 파일을 재작성하지 않는다.
	recoveryNoError(t, NewSwitcher().ValidateRecoveryState(&stored))
	after, err := os.ReadFile(journal)
	recoveryNoError(t, err)
	recoveryEqual(t, string(data), string(after))
	manager := environment.NewEnvironmentSwitcher()
	manager.Register(NewSwitcher())
	recoveryNoError(t, manager.RecoverFromJournal(context.Background(), journal))
	_, err = os.Lstat(path)
	recoveryTrue(t, os.IsNotExist(err))
	_, err = os.Lstat(journal)
	recoveryTrue(t, os.IsNotExist(err))
}

func TestProfileApplyPermissionsPreserveIntent(t *testing.T) {
	state := permissionState()
	state.ProfileExpected.Mode = 0o666
	recoveryEqual(t, os.FileMode(0o600), profileApplyPermissions(state))
	for _, mode := range []uint32{0o600, 0o640, 0o666, 0o444} {
		state.ProfileBefore.Present = true
		state.ProfileBefore.Mode = mode
		recoveryEqual(t, os.FileMode(mode), profileApplyPermissions(state))
	}
}

func TestProfileMalformedLegacyRefusesMutation(t *testing.T) {
	path, _ := recoveryPaths(t)
	t.Setenv("AWS_PROFILE", "")
	state := captured(t, &environment.AWSConfig{Profile: "C"})
	recoveryNoError(t, NewSwitcher().ApplyState(context.Background(), state))
	for _, tc := range []struct {
		name   string
		change func(*environment.AWSRecoveryState)
	}{
		{"mode640", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o640 }},
		{"mode777", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o777 }},
		{"readonly444", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Mode = 0o444 }},
		{"wrong-path", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Path += "-other" }},
		{"wrong-raw", func(s *environment.AWSRecoveryState) { s.ProfileExpected.Raw = "other\n" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := *state
			tc.change(&bad)
			recoveryError(t, NewSwitcher().ValidateRecoveryState(&bad))
			recoveryError(t, NewSwitcher().Rollback(context.Background(), &bad))
			data, err := os.ReadFile(path)
			recoveryNoError(t, err)
			recoveryEqual(t, "C\n", string(data))
		})
	}
	recoveryNoError(t, NewSwitcher().Rollback(context.Background(), state))
}
