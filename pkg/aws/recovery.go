// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

const awsRecoveryVersion = 1

// CaptureState 는 요청이 바꿀 profile 파일과 대상 region만 복구 상태로 보존한다.
func (a *Switcher) CaptureState(ctx context.Context, value any) (any, error) {
	request, err := validatedRequest(value)
	if err != nil {
		return nil, err
	}
	state := &environment.AWSRecoveryState{Version: awsRecoveryVersion, Request: *request}
	if request.Profile == "" && request.Region == "" {
		return state, nil
	}
	if err := ctx.Err(); err != nil {
		if request.Profile != "" {
			return nil, fmt.Errorf("failed to set AWS profile: %w", err)
		}
		return nil, fmt.Errorf("failed to set AWS region: %w", err)
	}
	path, err := activeProfilePath()
	if err != nil {
		return nil, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	before, err := readProfileFile(path)
	if err != nil {
		return nil, err
	}
	state.ProfileBefore = before
	state.ProfileExpected = before
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = strings.TrimSpace(before.Raw)
	}
	if request.Profile != "" {
		profile = request.Profile
		state.ProfileExpected = environment.AWSProfileFileState{Path: path, Present: true, Raw: request.Profile + "\n", Mode: 0o600}
		if before.Present {
			state.ProfileExpected.Mode = before.Mode
		}
	}
	if profile == "" {
		profile = DefaultProfile
	}
	if err := validProfile(profile); err != nil {
		return nil, err
	}
	state.TargetProfile = profile
	if request.Region != "" {
		state.ConfigPath, err = awsConfigPath()
		if err != nil {
			return nil, err
		}
		if err := rejectResourceAlias(path, state.ConfigPath); err != nil {
			return nil, err
		}
		data, present, mode, err := readRegular(state.ConfigPath)
		if err != nil {
			return nil, err
		}
		state.ConfigExisted = present
		state.ConfigMode = uint32(mode.Perm())
		state.RegionBefore, err = readRegion(data, profile)
		if err != nil {
			return nil, err
		}
		state.RegionExpected = environment.AWSRegionState{Present: true, Value: request.Region}
	}
	return state, nil
}

func validatedRequest(value any) (*environment.AWSConfig, error) {
	request, ok := value.(*environment.AWSConfig)
	if !ok || request == nil {
		return nil, fmt.Errorf("invalid AWS configuration type")
	}
	if request.AccountID != "" {
		return nil, fmt.Errorf("AWS AccountID switching is unsupported")
	}
	if request.Profile != "" {
		if err := validProfile(request.Profile); err != nil {
			return nil, err
		}
	}
	if env := os.Getenv("AWS_PROFILE"); env != "" && request.Profile != "" && env != request.Profile {
		return nil, fmt.Errorf("AWS_PROFILE conflicts with requested profile")
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region != "" && request.Region != "" && region != request.Region {
		return nil, fmt.Errorf("AWS region environment conflicts with requested region")
	}
	if validRegionValue(request.Region) != nil {
		return nil, fmt.Errorf("unsupported AWS region value")
	}
	return request, nil
}

func validRegionValue(value string) error {
	if strings.ContainsAny(value, "\r\n\x00") || strings.TrimSpace(value) != value {
		return fmt.Errorf("unsupported AWS region value")
	}
	return nil
}

func validProfile(profile string) error {
	if strings.TrimSpace(profile) != profile || strings.ContainsAny(profile, "\r\n\x00[]=;#\"'\\") || strings.IndexFunc(profile, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return fmt.Errorf("unsupported AWS profile name")
	}
	return nil
}

func awsConfigPath() (string, error) {
	path := os.Getenv("AWS_CONFIG_FILE")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, ".aws", "config")
	}
	return filepath.Abs(path)
}

func readRegular(path string) (data []byte, present bool, mode os.FileMode, resultErr error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, false, 0, nil
	}
	if err != nil {
		return nil, false, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, 0, fmt.Errorf("refusing non-regular AWS state file: %s", path)
	}
	data, err = os.ReadFile(path)
	return data, true, info.Mode(), err
}

func readProfileFile(path string) (environment.AWSProfileFileState, error) {
	data, present, mode, err := readRegular(path)
	if err != nil {
		return environment.AWSProfileFileState{}, err
	}
	if present {
		profile := strings.TrimSpace(string(data))
		if err := validProfile(profile); err != nil {
			return environment.AWSProfileFileState{}, err
		}
		if len(data) > 4096 {
			return environment.AWSProfileFileState{}, fmt.Errorf("AWS profile state too large")
		}
	}
	return environment.AWSProfileFileState{Path: path, Present: present, Raw: string(data), Mode: uint32(mode.Perm())}, nil
}

// ValidateRecoveryState 는 legacy 표시 DTO를 journal 복구로 오인하지 않도록 거절한다.
func (a *Switcher) ValidateRecoveryState(value any) error {
	state, ok := value.(*environment.AWSRecoveryState)
	if !ok || state == nil || state.Version != awsRecoveryVersion {
		return fmt.Errorf("invalid AWS recovery state/version")
	}
	request := state.Request
	if request.AccountID != "" || validProfile(request.Profile) != nil || validRegionValue(request.Region) != nil {
		return fmt.Errorf("invalid AWS recovery request")
	}
	if request.Profile == "" && request.Region == "" {
		if !reflect.DeepEqual(state, &environment.AWSRecoveryState{Version: awsRecoveryVersion, Request: request}) {
			return fmt.Errorf("invalid empty AWS recovery state")
		}
		return nil
	}
	if err := validateProfileRecovery(state); err != nil {
		return err
	}
	return validateRegionRecovery(state)
}

func validateProfileRecovery(state *environment.AWSRecoveryState) error {
	request := state.Request
	path, err := activeProfilePath()
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	if state.ProfileBefore.Path != path || state.ProfileExpected.Path != path {
		return fmt.Errorf("AWS recovery profile path mismatch")
	}
	if err := validProfile(strings.TrimSpace(state.ProfileBefore.Raw)); err != nil {
		return err
	}
	if !state.ProfileBefore.Present && (state.ProfileBefore.Raw != "" || state.ProfileBefore.Mode != 0) {
		return fmt.Errorf("invalid absent AWS profile state")
	}
	expected := state.ProfileBefore
	if request.Profile != "" {
		expected.Present = true
		expected.Raw = request.Profile + "\n"
		if !state.ProfileBefore.Present {
			expected.Mode = 0o600
		}
	}
	if expected != state.ProfileExpected {
		return fmt.Errorf("invalid expected AWS profile state")
	}
	if state.ProfileBefore.Mode > 0o777 || state.ProfileExpected.Mode > 0o777 {
		return fmt.Errorf("invalid AWS profile mode")
	}
	if state.TargetProfile == "" || validProfile(state.TargetProfile) != nil {
		return fmt.Errorf("invalid AWS target profile")
	}
	if request.Profile != "" && state.TargetProfile != request.Profile {
		return fmt.Errorf("AWS target profile mismatch")
	}
	return nil
}

func validateRegionRecovery(state *environment.AWSRecoveryState) error {
	request := state.Request
	if validRegionValue(state.RegionBefore.Value) != nil {
		return fmt.Errorf("invalid prior AWS region")
	}
	if !state.ConfigExisted && (state.ConfigMode != 0 || state.RegionBefore.Present) {
		return fmt.Errorf("invalid absent AWS config state")
	}
	if request.Region != "" {
		path, err := awsConfigPath()
		if err != nil {
			return err
		}
		if err := rejectResourceAlias(state.ProfileBefore.Path, state.ConfigPath); err != nil {
			return err
		}
		if state.ConfigPath != path || state.RegionExpected != (environment.AWSRegionState{Present: true, Value: request.Region}) || state.ConfigMode > 0o777 {
			return fmt.Errorf("invalid AWS region recovery state")
		}
		if !state.RegionBefore.Present && state.RegionBefore.Value != "" {
			return fmt.Errorf("invalid absent AWS region")
		}
	}
	if request.Region == "" && (state.ConfigPath != "" || state.ConfigExisted || state.ConfigMode != 0 || state.RegionBefore != (environment.AWSRegionState{}) || state.RegionExpected != (environment.AWSRegionState{})) {
		return fmt.Errorf("unexpected AWS region recovery data")
	}
	return nil
}

func sameProfile(a, b environment.AWSProfileFileState) bool { return a == b }

// restoreRecovery는 before 또는 예상 write 결과인 자원만 되돌린다. 외부 변경은 덮어쓰지 않는다.
func (a *Switcher) restoreRecovery(ctx context.Context, state *environment.AWSRecoveryState) error {
	if err := a.ValidateRecoveryState(state); err != nil {
		return err
	}
	if state.Request.Profile == "" && state.Request.Region == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := checkRecoveryProfile(state)
	if err != nil {
		return err
	}
	config, exists, mode, err := prepareRegionRecovery(state)
	if err != nil {
		return err
	}
	if state.Request.Region != "" {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !state.ConfigExisted {
			if exists {
				if err := os.Remove(state.ConfigPath); err != nil {
					return err
				}
			}
		} else {
			if err := replaceRegular(ctx, state.ConfigPath, config, mode); err != nil {
				return err
			}
		}
	}
	if state.Request.Profile != "" && !sameProfile(current, state.ProfileBefore) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !state.ProfileBefore.Present {
			return os.Remove(state.ProfileBefore.Path)
		}
		return replaceRegular(ctx, state.ProfileBefore.Path, []byte(state.ProfileBefore.Raw), os.FileMode(state.ProfileBefore.Mode))
	}
	return nil
}

func checkRecoveryProfile(state *environment.AWSRecoveryState) (environment.AWSProfileFileState, error) {
	var current environment.AWSProfileFileState
	var err error
	if state.Request.Profile != "" {
		current, err = readProfileFile(state.ProfileBefore.Path)
		if err != nil {
			return current, err
		}
		if !sameProfile(current, state.ProfileBefore) && !sameProfile(current, state.ProfileExpected) {
			return current, fmt.Errorf("AWS profile recovery conflict")
		}
	}
	return current, nil
}

func prepareRegionRecovery(state *environment.AWSRecoveryState) (data []byte, present bool, permissions os.FileMode, resultErr error) {
	var config []byte
	var exists bool
	var mode os.FileMode
	var err error
	if state.Request.Region != "" {
		config, exists, mode, err = readRegular(state.ConfigPath)
		if err != nil {
			return nil, false, 0, err
		}
		region, readErr := readRegion(config, state.TargetProfile)
		if readErr != nil {
			return nil, false, 0, readErr
		}
		if region != state.RegionBefore && region != state.RegionExpected {
			return nil, false, 0, fmt.Errorf("AWS region recovery conflict")
		}
		if state.ConfigExisted && !exists {
			return nil, false, 0, fmt.Errorf("AWS config file removed during recovery")
		}
		if state.ConfigExisted && uint32(mode.Perm()) != state.ConfigMode {
			return nil, false, 0, fmt.Errorf("AWS config mode recovery conflict")
		}
		config, err = editRegion(config, state.TargetProfile, state.RegionBefore)
		if err != nil {
			return nil, false, 0, err
		}
		if !state.ConfigExisted && hasConfigValues(config) {
			return nil, false, 0, fmt.Errorf("AWS config absence recovery conflicts with other settings")
		}
	}
	return config, exists, mode, nil
}

func replaceRegular(ctx context.Context, path string, data []byte, mode os.FileMode) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, _, _, err := readRegular(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".aws-state-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			resultErr = errors.Join(resultErr, fmt.Errorf("AWS temporary cleanup: %w", err))
		}
	}()
	if err := f.Chmod(mode.Perm()); err != nil {
		_ = f.Close()
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, _, _, err := readRegular(path); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// rejectResourceAlias는 같은 파일을 두 복구 자원으로 취급하지 않도록 거절한다.
func rejectResourceAlias(profile, config string) error {
	profilePath, err := canonicalMissingPath(profile)
	if err != nil {
		return fmt.Errorf("resolve AWS profile resource: %w", err)
	}
	configPath, err := canonicalMissingPath(config)
	if err != nil {
		return fmt.Errorf("resolve AWS config resource: %w", err)
	}
	if profilePath == configPath {
		return fmt.Errorf("AWS profile/config resources alias")
	}
	a, ae := os.Stat(profile)
	b, be := os.Stat(config)
	if ae != nil && !os.IsNotExist(ae) {
		return ae
	}
	if be != nil && !os.IsNotExist(be) {
		return be
	}
	if ae == nil && be == nil && os.SameFile(a, b) {
		return fmt.Errorf("AWS profile/config resources hardlink alias")
	}
	return nil
}

// canonicalMissingPath는 기존 조상의 symlink를 해석한 뒤 아직 없는 경로를 붙인다.
func canonicalMissingPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	ancestor := absolute
	for {
		_, err := os.Lstat(ancestor)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(ancestor)
			if err != nil {
				return "", err
			}
			suffix, err := filepath.Rel(ancestor, absolute)
			if err != nil {
				return "", err
			}
			return filepath.Join(resolved, suffix), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", fmt.Errorf("no existing AWS resource ancestor")
		}
		ancestor = parent
	}
}
