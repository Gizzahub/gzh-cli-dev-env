// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"os"
	"runtime"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// observedProfilePermissions는 Windows의 읽기 전용 속성만으로 보고되는 mode를 모델링한다.
// POSIX 권한이나 Windows ACL을 서로 같은 권한 체계로 취급하지 않는다.
func observedProfilePermissions(mode uint32, platform string) uint32 {
	if platform == "windows" {
		if mode&0o200 != 0 {
			return 0o666
		}
		return 0o444
	}
	return mode
}

func newProfileExpectedMode() uint32 { return observedProfilePermissions(0o600, runtime.GOOS) }

// expectedProfileAfterRequest는 실제 파일에서 읽은 before mode를 정규화하지 않는다.
func expectedProfileAfterRequest(state *environment.AWSRecoveryState, platform string) environment.AWSProfileFileState {
	expected := state.ProfileBefore
	if state.Request.Profile != "" {
		expected.Present = true
		expected.Raw = state.Request.Profile + "\n"
		if !state.ProfileBefore.Present {
			expected.Mode = observedProfilePermissions(0o600, platform)
		}
	}
	return expected
}

// compatibleProfileExpected는 v0.2.0 Windows의 알려진 신규 파일 예상값만 해석한다.
// 반환값만 조정하며 저장된 journal, 실제 current, 캡처된 before는 변경하지 않는다.
func compatibleProfileExpected(state *environment.AWSRecoveryState, platform string) environment.AWSProfileFileState {
	actual := state.ProfileExpected
	if platform != "windows" || state.Version != awsRecoveryVersion || state.Request.Profile == "" || state.Request.AccountID != "" || validProfile(state.Request.Profile) != nil {
		return actual
	}
	if state.ProfileBefore.Present || state.ProfileBefore.Raw != "" || state.ProfileBefore.Mode != 0 {
		return actual
	}
	expected := expectedProfileAfterRequest(state, platform)
	expected.Mode = 0o600
	if actual == expected {
		actual.Mode = observedProfilePermissions(0o600, platform)
	}
	return actual
}

func effectiveProfileExpected(state *environment.AWSRecoveryState) environment.AWSProfileFileState {
	return compatibleProfileExpected(state, runtime.GOOS)
}

// profileApplyPermissions는 부재 파일에 0600을 요청하고 기존 파일의 mode는 보존한다.
func profileApplyPermissions(state *environment.AWSRecoveryState) os.FileMode {
	if !state.ProfileBefore.Present {
		return 0o600
	}
	return os.FileMode(state.ProfileBefore.Mode)
}
