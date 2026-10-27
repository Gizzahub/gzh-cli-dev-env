// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 기존 journal의 형태와 직렬/병렬 실행에 관계없이 dry-run은 디스크 상태를 보존한다.
func TestSwitchEnvironment_DryRunPreservesJournal(t *testing.T) {
	for _, execution := range []struct {
		name     string
		parallel bool
	}{{"serial", false}, {"parallel", true}} {
		for _, kind := range []string{"absent", "valid", "read-only", "malformed"} {
			t.Run(execution.name+"/"+kind, func(t *testing.T) {
				dir := withTempConfigDir(t)
				path := journalPathIn(dir)
				var before []byte
				var beforeMode os.FileMode
				if kind != "absent" {
					if err := WriteJournal(path, false, "previous-interrupted-switch", map[string]any{"docker": &DockerConfig{Context: "previous"}}); err != nil {
						t.Fatal(err)
					}
					if kind == "malformed" {
						if err := os.WriteFile(path, []byte("{ invalid journal }\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					mode := os.FileMode(0o640)
					if kind == "read-only" {
						mode = 0o400
					}
					if err := os.Chmod(path, mode); err != nil {
						t.Fatal(err)
					}
					// Windows에서도 이 시험이 만든 readonly 파일만 정리할 수 있게 복원한다.
					t.Cleanup(func() {
						if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
							t.Errorf("restore owned fixture permissions: %v", err)
						}
					})
					var err error
					before, err = os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					info, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					beforeMode = info.Mode().Perm()
				}

				awsMock, dockerMock := newMockSwitcher("aws"), newMockSwitcher("docker")
				manager := NewEnvironmentSwitcher()
				manager.Register(awsMock)
				manager.Register(dockerMock)
				env := &Environment{Name: "preview", Services: map[string]ServiceConfig{
					"aws":    {AWS: &AWSConfig{Profile: "next"}},
					"docker": {Docker: &DockerConfig{Context: "next"}},
				}}
				result, err := manager.SwitchEnvironment(context.Background(), env, SwitchOptions{DryRun: true, Parallel: execution.parallel, RollbackOnError: true})
				if err != nil || result == nil || !result.Success || len(result.Errors) != 0 {
					t.Fatalf("dry-run result=%+v err=%v", result, err)
				}
				if awsMock.switchCalled || dockerMock.switchCalled || awsMock.rollbackCalled || dockerMock.rollbackCalled {
					t.Fatal("dry-run called Switch or Rollback")
				}
				if kind == "absent" {
					if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
						t.Fatalf("dry-run created journal directory or stat failed: %v", err)
					}
					return
				}
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("existing journal missing after dry-run: %v", err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) || info.Mode().Perm() != beforeMode {
					t.Fatalf("journal changed: bytes equal=%v mode=%v want=%v", bytes.Equal(before, after), info.Mode().Perm(), beforeMode)
				}
			})
		}
	}
}
