// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

// AWSRecoveryState records the request's write targets, separately from display state.
// It contains only profile selection and region metadata, never credentials or a
// complete AWS config file. Version identifies the recovery payload contract.
type AWSRecoveryState struct {
	Version         int
	Request         AWSConfig
	ProfileBefore   AWSProfileFileState
	ProfileExpected AWSProfileFileState
	ConfigPath      string
	ConfigExisted   bool
	ConfigMode      uint32
	TargetProfile   string
	RegionBefore    AWSRegionState
	RegionExpected  AWSRegionState
}

// AWSProfileFileState distinguishes a missing selection file from an empty one.
type AWSProfileFileState struct {
	Path    string
	Present bool
	Raw     string
	Mode    uint32
}

// AWSRegionState distinguishes an absent region key from a present value.
type AWSRegionState struct {
	Present bool
	Value   string
}
