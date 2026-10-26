// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"errors"
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
	if err := contextErr(ctx); err != nil {
		return err
	}

	sshConfig, ok := config.(*environment.SSHConfig)
	if !ok || sshConfig == nil {
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

	return activateConfig(ctx, source, active, operatingLinkFS())
}

// GetCurrentState reports the active SSH config path (empty when absent).
// Symlink targets are resolved so the returned path is the content source.
func (s *Switcher) GetCurrentState(ctx context.Context) (any, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

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
	if err := contextErr(ctx); err != nil {
		return err
	}
	config, ok := previousState.(*environment.SSHConfig)
	if !ok || config == nil {
		return fmt.Errorf("invalid SSH configuration type")
	}
	if config.Config != "" {
		return s.Switch(ctx, config)
	}
	active, err := defaultConfigPath()
	if err != nil {
		return err
	}
	info, err := os.Lstat(active)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat active SSH config: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to remove unmanaged SSH config: %s", active)
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := os.Remove(active); err != nil {
		return fmt.Errorf("remove SSH config symlink: %w", err)
	}
	return nil
}

// nil은 기존 호출과 같이 제한 없는 맥락으로 취급한다.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
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

// linkFS는 호출마다 전달하는 테스트 경계이며 switcher 상태나 전역 재정의를 만들지 않는다.
type linkFS struct {
	mkdirTemp func(string, string) (string, error)
	symlink   func(string, string) error
	rename    func(string, string) error
	remove    func(string) error
	lstat     func(string) (os.FileInfo, error)
}

func operatingLinkFS() linkFS {
	return linkFS{mkdirTemp: os.MkdirTemp, symlink: os.Symlink, rename: os.Rename, remove: os.Remove, lstat: os.Lstat}
}

// activateConfig는 같은 파일시스템의 임시 링크를 rename하여 이전 링크를 먼저 지우지 않는다.
// 다른 프로세스와의 동시 변경이나 모든 OS에서의 원자성을 보장하지 않는다.
func activateConfig(ctx context.Context, source, activePath string, fs linkFS) (resultErr error) {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := ensureManagedLink(activePath, fs); err != nil {
		return err
	}
	parent := filepath.Dir(activePath)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create .ssh directory: %w", err)
	}
	stageDir, err := fs.mkdirTemp(parent, ".config-switch-*")
	if err != nil {
		return fmt.Errorf("stage SSH config directory: %w", err)
	}
	stage := filepath.Join(stageDir, "config")
	defer func() {
		if err := fs.remove(stage); err != nil && !os.IsNotExist(err) {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean staged SSH config: %w", err))
		}
		if err := fs.remove(stageDir); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean SSH staging directory: %w", err))
		}
	}()
	if err := fs.symlink(source, stage); err != nil {
		return fmt.Errorf("stage SSH config %s: %w", source, err)
	}
	if err := ensureManagedLink(activePath, fs); err != nil {
		return err
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := fs.rename(stage, activePath); err != nil {
		return fmt.Errorf("activate SSH config %s: %w", source, err)
	}
	return nil
}

func ensureManagedLink(active string, fs linkFS) error {
	info, err := fs.lstat(active)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat active SSH config: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("cannot switch SSH config: %s is not a managed symlink; refusing to overwrite", active)
	}
	return nil
}
