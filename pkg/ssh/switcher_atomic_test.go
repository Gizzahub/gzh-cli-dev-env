// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

func sshFixture(t *testing.T) (string, string, string, os.FileInfo) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".ssh")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old, next := filepath.Join(dir, "old"), filepath.Join(dir, "next")
	for _, p := range []string{old, next} {
		if err := os.WriteFile(p, []byte(p), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	active := filepath.Join(dir, "config")
	if err := os.Symlink(old, active); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(active)
	if err != nil {
		t.Fatal(err)
	}
	return old, next, active, info
}

// 커밋 이전 모든 실패는 실제 기존 링크 inode와 내용을 보존한다.
func TestActivateConfigFailurePreservesLink(t *testing.T) {
	for _, mode := range []string{"stage directory", "stage link", "rename", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			old, next, active, before := sshFixture(t)
			fs := operatingLinkFS()
			sentinel := errors.New("injected failure")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "stage directory":
				fs.mkdirTemp = func(string, string) (string, error) { return "", sentinel }
			case "stage link":
				fs.symlink = func(string, string) error { return sentinel }
			case "rename":
				fs.rename = func(string, string) error { return sentinel }
			case "cancel":
				fs.symlink = func(a, b string) error { err := os.Symlink(a, b); cancel(); return err }
			}
			err := activateConfig(ctx, next, active, fs)
			if mode == "cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
			} else if !errors.Is(err, sentinel) {
				t.Fatalf("error=%v", err)
			}
			after, err := os.Lstat(active)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before, after) {
				t.Fatal("previous link inode changed before commit")
			}
			target, err := os.Readlink(active)
			if err != nil || target != old {
				t.Fatalf("target=%s,error=%v", target, err)
			}
			leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(active), ".config-switch-*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("staging remains: %v,%v", leftovers, err)
			}
			content, err := os.ReadFile(old)
			if err != nil || string(content) != old {
				t.Fatalf("old content changed: %v", err)
			}
		})
	}
}

func TestActivateConfigFreshCheckRejectsUnmanagedFile(t *testing.T) {
	_, next, active, _ := sshFixture(t)
	fs := operatingLinkFS()
	fs.symlink = func(a, b string) error {
		if err := os.Symlink(a, b); err != nil {
			return err
		}
		if err := os.Remove(active); err != nil {
			return err
		}
		return os.WriteFile(active, []byte("user-owned"), 0o600)
	}
	if err := activateConfig(context.Background(), next, active, fs); err == nil {
		t.Fatal("new regular file must not be overwritten")
	}
	b, err := os.ReadFile(active)
	if err != nil || string(b) != "user-owned" {
		t.Fatalf("unmanaged bytes changed: %s,%v", b, err)
	}
}

// rename 성공 뒤 정리 실패도 숨기지 않으며 활성 상태의 실제 결과를 확인한다.
func TestActivateConfigCleanupFailureVisible(t *testing.T) {
	_, next, active, _ := sshFixture(t)
	fs := operatingLinkFS()
	sentinel := errors.New("cleanup denied")
	var owned string
	fs.mkdirTemp = func(a, b string) (string, error) { p, err := os.MkdirTemp(a, b); owned = p; return p, err }
	fs.remove = func(p string) error {
		if p == owned {
			return sentinel
		}
		return os.Remove(p)
	}
	err := activateConfig(context.Background(), next, active, fs)
	if !errors.Is(err, sentinel) {
		t.Fatalf("cleanup failure hidden: %v", err)
	}
	target, err := os.Readlink(active)
	if err != nil || target != next {
		t.Fatalf("committed target=%s,error=%v", target, err)
	}
	if err := os.Remove(owned); err != nil {
		t.Fatal(err)
	}
}

func TestSwitcherCanceledEntryAndTypedNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	config := &environment.SSHConfig{Config: filepath.Join(home, "unused")}
	s := NewSwitcher()
	if err := s.Switch(ctx, config); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.GetCurrentState(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.Rollback(ctx, &environment.SSHConfig{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatalf("entry cancellation changed HOME: %v,%v", entries, err)
	}
	var typedNil *environment.SSHConfig
	if err := s.Switch(context.Background(), typedNil); err == nil {
		t.Fatal("typed nil accepted")
	}
	if err := s.Rollback(context.Background(), typedNil); err == nil {
		t.Fatal("typed nil rollback accepted")
	}
}

func TestSwitcherRollbackSerializableAbsenceAndTarget(t *testing.T) {
	old, next, active, _ := sshFixture(t)
	previous, err := NewSwitcher().GetCurrentState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	var restored environment.SSHConfig
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: next}); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Rollback(context.Background(), &restored); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(active)
	if err != nil || target != old {
		t.Fatalf("rollback target=%s,error=%v", target, err)
	}
	if err := os.Remove(active); err != nil {
		t.Fatal(err)
	}
	absent, err := NewSwitcher().GetCurrentState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(absent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: next}); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Rollback(context.Background(), &restored); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Rollback(context.Background(), &restored); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(active); !os.IsNotExist(err) {
		t.Fatalf("absence not restored: %v", err)
	}
	if content, err := os.ReadFile(next); err != nil || string(content) != next {
		t.Fatalf("target contents changed: %v", err)
	}
	for _, directory := range []bool{false, true} {
		if directory {
			if err := os.Mkdir(active, 0o700); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(active, []byte("private"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := NewSwitcher().Rollback(context.Background(), &restored); err == nil {
			t.Fatal("unmanaged active path removed")
		}
		if err := os.Remove(active); err != nil {
			t.Fatal(err)
		}
	}
}

// 실제 typed journal 복구가 새 switcher에서도 부재를 복원하고 성공 뒤에만 journal을 지운다.
func TestSSHJournalRecoveryAbsenceAndCollision(t *testing.T) {
	_, next, active, _ := sshFixture(t)
	if err := os.Remove(active); err != nil {
		t.Fatal(err)
	}
	state, err := NewSwitcher().GetCurrentState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(t.TempDir(), "journal.json")
	if err := environment.WriteJournal(journal, false, "fixture", map[string]any{"ssh": state}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte("private-user-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	recovery := environment.NewEnvironmentSwitcher()
	recovery.Register(NewSwitcher())
	if err := recovery.RecoverFromJournal(context.Background(), journal); err == nil {
		t.Fatal("collision recovery should fail")
	}
	after, err := os.ReadFile(journal)
	if err != nil || string(before) != string(after) {
		t.Fatalf("failed recovery changed journal: %v", err)
	}
	content, err := os.ReadFile(active)
	if err != nil || string(content) != "private-user-config" {
		t.Fatalf("regular config changed: %v", err)
	}
	if err := os.Remove(active); err != nil {
		t.Fatal(err)
	}
	if err := NewSwitcher().Switch(context.Background(), &environment.SSHConfig{Config: next}); err != nil {
		t.Fatal(err)
	}
	recovery = environment.NewEnvironmentSwitcher()
	recovery.Register(NewSwitcher())
	if err := recovery.RecoverFromJournal(context.Background(), journal); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(active); !os.IsNotExist(err) {
		t.Fatalf("active should be absent: %v", err)
	}
	if _, err := os.Stat(journal); !os.IsNotExist(err) {
		t.Fatalf("successful recovery did not clear journal: %v", err)
	}
}

func TestActivateConfigReportsPrimaryAndCleanupFailures(t *testing.T) {
	old, next, active, before := sshFixture(t)
	fs := operatingLinkFS()
	primary, cleanup := errors.New("rename denied"), errors.New("cleanup denied")
	fs.rename = func(string, string) error { return primary }
	fs.remove = func(p string) error {
		if filepath.Base(p) == "config" {
			return cleanup
		}
		return os.Remove(p)
	}
	err := activateConfig(context.Background(), next, active, fs)
	if !errors.Is(err, primary) || !errors.Is(err, cleanup) {
		t.Fatalf("lost failure causes: %v", err)
	}
	after, err := os.Lstat(active)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("old link changed: %v", err)
	}
	target, err := os.Readlink(active)
	if err != nil || target != old {
		t.Fatalf("old target changed: %s,%v", target, err)
	}
	dirs, err := filepath.Glob(filepath.Join(filepath.Dir(active), ".config-switch-*"))
	if err != nil || len(dirs) != 1 {
		t.Fatalf("expected owned cleanup residue: %v,%v", dirs, err)
	}
	if err := os.Remove(filepath.Join(dirs[0], "config")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dirs[0]); err != nil {
		t.Fatal(err)
	}
}

// nil 맥락 호환성은 상태 직렬화 테스트와 분리해 실제 공개 API로 검증한다.
func TestSwitcherNilContextCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name string
		ctx  context.Context
	}{{name: "nil"}, {name: "background", ctx: context.Background()}} {
		t.Run(tt.name, func(t *testing.T) {
			old, next, active, _ := sshFixture(t)
			previous, err := NewSwitcher().GetCurrentState(tt.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := NewSwitcher().Switch(tt.ctx, &environment.SSHConfig{Config: next}); err != nil {
				t.Fatal(err)
			}
			if err := NewSwitcher().Rollback(tt.ctx, previous); err != nil {
				t.Fatal(err)
			}
			target, err := os.Readlink(active)
			if err != nil || target != old {
				t.Fatalf("context compatibility target=%s,error=%v", target, err)
			}
			if err := NewSwitcher().Rollback(tt.ctx, &environment.SSHConfig{}); err != nil {
				t.Fatal(err)
			}
			if err := NewSwitcher().Rollback(tt.ctx, &environment.SSHConfig{}); err != nil {
				t.Fatal(err)
			}
			state, err := NewSwitcher().GetCurrentState(tt.ctx)
			if err != nil {
				t.Fatal(err)
			}
			config, ok := state.(*environment.SSHConfig)
			if !ok || config.Config != "" {
				t.Fatalf("absence state=%#v", state)
			}
		})
	}
}
