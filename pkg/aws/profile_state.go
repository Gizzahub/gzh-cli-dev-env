// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// configAppDir is the application config directory under the user config root.
	configAppDir = "gzh-dev-env"
	// activeProfileFileName stores the last switched AWS profile name.
	activeProfileFileName = "aws-active-profile"
)

// userConfigDir resolves the user config directory.
// Overridable in tests.
var userConfigDir = os.UserConfigDir

// activeProfilePath returns the path of the active-profile state file.
func activeProfilePath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, configAppDir, activeProfileFileName), nil
}

// writeActiveProfile persists the active AWS profile name to the state file.
func writeActiveProfile(profile string) error {
	path, err := activeProfilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(profile)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write active profile: %w", err)
	}
	return nil
}

// readActiveProfile reads the active AWS profile from the state file.
// Returns empty string when the file is missing.
func readActiveProfile() (string, error) {
	path, err := activeProfilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read active profile: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// resolveActiveProfile returns the active profile in priority order:
// 1. AWS_PROFILE environment variable
// 2. state file written by Switch
// 3. empty string.
func resolveActiveProfile() string {
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		return profile
	}
	profile, err := readActiveProfile()
	if err != nil || profile == "" {
		return ""
	}
	return profile
}
