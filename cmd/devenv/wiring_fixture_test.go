// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package devenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

const wiringAWSChildMarker = "gzh-dev-env-wiring-aws-fixture-v1"

// 복사된 테스트 실행체를 vendor 대신 실행한다. 실제 aws나 네트워크는 호출하지 않는다.
func TestMain(m *testing.M) {
	if os.Getenv("GZH_WIRING_AWS_CHILD") == wiringAWSChildMarker && (filepath.Base(os.Args[0]) == "aws" || filepath.Base(os.Args[0]) == "aws.exe") {
		args := os.Args[1:]
		trace, err := os.OpenFile(os.Getenv("GZH_WIRING_AWS_TRACE"), os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			os.Exit(25)
		}
		encodeErr := json.NewEncoder(trace).Encode(args)
		closeErr := trace.Close()
		if encodeErr != nil || closeErr != nil {
			os.Exit(25)
		}
		if !reflect.DeepEqual(args, []string{"configure", "get", "region"}) {
			os.Exit(24)
		}
		if os.Getenv("GZH_WIRING_AWS_FAIL") == "1" {
			os.Exit(23)
		}
		if _, err := fmt.Fprintln(os.Stdout, "wiring-fixture-region"); err != nil {
			os.Exit(25)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func wiringAWSFixture(t *testing.T, fail bool) func() {
	t.Helper()
	root := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(key, root)
	}
	for _, key := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION"} {
		t.Setenv(key, "")
	}
	for _, item := range [][2]string{{"AWS_CONFIG_FILE", "config"}, {"AWS_SHARED_CREDENTIALS_FILE", "credentials"}} {
		path := filepath.Join(root, item[1])
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(item[0], path)
	}
	name := "aws"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := copyWiringExecutable(executable, filepath.Join(root, name)); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(root, "trace.jsonl")
	if err := os.WriteFile(trace, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	t.Setenv("GZH_WIRING_AWS_CHILD", wiringAWSChildMarker)
	t.Setenv("GZH_WIRING_AWS_TRACE", trace)
	t.Setenv("GZH_WIRING_AWS_FAIL", "0")
	if fail {
		t.Setenv("GZH_WIRING_AWS_FAIL", "1")
	}
	// 자식 race runtime 종료 지연을 없애며 검사 자체나 생산 timeout은 변경하지 않는다.
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	return func() {
		data, err := os.ReadFile(trace)
		if err != nil {
			t.Fatal(err)
		}
		var args []string
		if err := json.Unmarshal(data, &args); err != nil {
			t.Fatalf("expected exactly one query record: %v", err)
		}
		if !reflect.DeepEqual(args, []string{"configure", "get", "region"}) {
			t.Fatalf("unexpected AWS calls: %v", args)
		}
	}
}

func copyWiringExecutable(source, destination string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, output.Close()) }()
	_, err = io.Copy(output, input)
	return err
}

func TestDefaultWiringSnapshotFailureStopsDryRun(t *testing.T) {
	assertQuery := wiringAWSFixture(t, true)
	switcher := environment.NewEnvironmentSwitcher()
	registerDefaultSwitchers(switcher)
	env := &environment.Environment{Name: "fixture", Services: map[string]environment.ServiceConfig{"aws": {AWS: &environment.AWSConfig{Profile: "target", Region: "target-region"}}}}
	result, err := switcher.SwitchEnvironment(context.Background(), env, environment.SwitchOptions{DryRun: true, Timeout: 5 * time.Second})
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 {
		t.Fatalf("expected preserved fixture exit23, got %v", err)
	}
	if result == nil || result.Success {
		t.Fatalf("failed snapshot must not report success: %+v", result)
	}
	assertQuery()
}

func TestSwitchAllWiringSnapshotFailureIsVisible(t *testing.T) {
	assertQuery := wiringAWSFixture(t, true)
	path := filepath.Join(t.TempDir(), "fixture.yaml")
	data := []byte("name: fixture\nservices:\n  aws:\n    aws:\n      profile: target\n      region: target-region\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &switchAllOptions{fromFile: path, dryRun: true, force: true, timeout: 5 * time.Second}
	var runErr error
	output := captureStdout(t, func() { runErr = opts.run(context.Background()) })
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 23 {
		t.Fatalf("expected preserved fixture exit23, got %v", runErr)
	}
	if strings.Contains(output, "Successfully switched") {
		t.Fatalf("failure reported success: %q", output)
	}
	assertQuery()
}
