// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const hookFixtureName = "gzh-env-hook-fixture"

// TestMain은 고정된 helper 실행 파일만 처리하고 일반 시험 실행은 유지한다.
func TestMain(m *testing.M) {
	base := filepath.Base(os.Args[0])
	if os.Getenv("GZH_ENV_HOOK_FIXTURE") == "native-v1" && (base == hookFixtureName || base == hookFixtureName+".exe") {
		if len(os.Args) != 2 || (os.Args[1] != "pre-hook-ran" && os.Args[1] != "post-hook-ran") {
			os.Exit(91)
		}
		path := filepath.Join(os.Getenv("GZH_ENV_HOOK_DIRECTORY"), os.Args[1])
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			os.Exit(92)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// nativeHookFixture는 PATH의 소유 helper만 실행하므로 Unix touch나 경로 문법에 의존하지 않는다.
func nativeHookFixture(t *testing.T, directory string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	name := hookFixtureName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, name), data, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("PATHEXT", ".EXE")
	t.Setenv("GZH_ENV_HOOK_FIXTURE", "native-v1")
	t.Setenv("GZH_ENV_HOOK_DIRECTORY", directory)
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	return hookFixtureName
}
