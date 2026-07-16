// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

// Switcher implements environment.ServiceSwitcher for SSH.
//
// Active config is ~/.ssh/config. When that path is a symlink (managed layout),
// Switch retargets it. When it is a regular file, Switch only succeeds if the
// requested path is already that same file — it never silently ignores config
// and never clobbers a user-owned regular file.
type Switcher struct{}

// NewSwitcher creates a new SSH switcher.
func NewSwitcher() *Switcher {
	return &Switcher{}
}

// Name returns the service name.
func (s *Switcher) Name() string {
	return serviceName
}

// Switch activates the specified SSH configuration path.
func (s *Switcher) Switch(ctx context.Context, config any) error {
	_ = ctx

	sshConfig, ok := config.(*environment.SSHConfig)
	if !ok {
		return fmt.Errorf("invalid SSH configuration type")
	}
	if sshConfig.Config == "" {
		return fmt.Errorf("SSH config path is required")
	}

	source, err := resolveConfigPath(sshConfig.Config)
	if err != nil {
		return err
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("SSH config not found at %s: %w", source, err)
	}
	if info.IsDir() {
		return fmt.Errorf("SSH config path is a directory: %s", source)
	}

	active, err := defaultConfigPath()
	if err != nil {
		return err
	}

	if samePath(source, active) {
		return nil
	}
	if sameFile(source, active) {
		return nil
	}

	return activateConfig(source, active)
}

// GetCurrentState reports the active SSH config path (empty when absent).
// Symlink targets are resolved so the returned path is the content source.
func (s *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	_ = ctx

	active, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(active)
	if err != nil {
		if os.IsNotExist(err) {
			return &environment.SSHConfig{Config: ""}, nil
		}
		return nil, fmt.Errorf("stat SSH config: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(active)
		if err != nil {
			return nil, fmt.Errorf("read SSH config symlink: %w", err)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(active), target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return nil, fmt.Errorf("resolve SSH config symlink target: %w", err)
		}
		return &environment.SSHConfig{Config: abs}, nil
	}

	abs, err := filepath.Abs(active)
	if err != nil {
		return nil, fmt.Errorf("resolve SSH config path: %w", err)
	}
	return &environment.SSHConfig{Config: abs}, nil
}

// Rollback restores a previous SSH configuration state.
func (s *Switcher) Rollback(ctx context.Context, previousState any) error {
	return s.Switch(ctx, previousState)
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func resolveConfigPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("SSH config path is required")
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve SSH config path %s: %w", path, err)
	}
	return abs, nil
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// activateConfig makes source the active SSH config at activePath.
// Only creates or retargets a symlink; never overwrites a regular file.
func activateConfig(source, activePath string) error {
	if err := os.MkdirAll(filepath.Dir(activePath), 0o700); err != nil {
		return fmt.Errorf("create .ssh directory: %w", err)
	}

	info, err := os.Lstat(activePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat active SSH config: %w", err)
		}
		if err := os.Symlink(source, activePath); err != nil {
			return fmt.Errorf("activate SSH config %s: %w", source, err)
		}
		return nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf(
			"cannot switch SSH config to %s: %s is a regular file (not a managed symlink); refusing to overwrite",
			source, activePath,
		)
	}

	if err := os.Remove(activePath); err != nil {
		return fmt.Errorf("remove SSH config symlink: %w", err)
	}
	if err := os.Symlink(source, activePath); err != nil {
		return fmt.Errorf("activate SSH config %s: %w", source, err)
	}
	return nil
}
