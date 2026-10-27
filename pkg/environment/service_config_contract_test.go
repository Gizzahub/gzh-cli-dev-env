// Copyright (c) 2026 Gizzahub
// SPDX-License-Identifier: MIT

package environment_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/azure"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/docker"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/gcp"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/kubernetes"
)

func serviceConfigFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			files[relative] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[relative] = "file:" + string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// 공개 Switch와 Rollback은 잘못된 설정을 실행·저장 전에 같은 오류로 거부한다.
func TestServiceConfigPublicContract(t *testing.T) {
	cases := []struct {
		name            string
		newSwitcher     func() environment.ServiceSwitcher
		typedNil, empty any
		invalidMessage  string
	}{
		{"aws", func() environment.ServiceSwitcher { return aws.NewSwitcher() }, (*environment.AWSConfig)(nil), &environment.AWSConfig{}, "invalid AWS configuration type"},
		{"gcp", func() environment.ServiceSwitcher { return gcp.NewSwitcher() }, (*environment.GCPConfig)(nil), &environment.GCPConfig{}, "invalid GCP configuration type"},
		{"azure", func() environment.ServiceSwitcher { return azure.NewSwitcher() }, (*environment.AzureConfig)(nil), &environment.AzureConfig{}, "invalid Azure configuration type"},
		{"docker", func() environment.ServiceSwitcher { return docker.NewSwitcher() }, (*environment.DockerConfig)(nil), &environment.DockerConfig{}, "invalid Docker configuration type"},
		{"kubernetes", func() environment.ServiceSwitcher { return kubernetes.NewSwitcher() }, (*environment.KubernetesConfig)(nil), &environment.KubernetesConfig{}, "invalid Kubernetes configuration type"},
	}
	for _, service := range cases {
		t.Run(service.name, func(t *testing.T) {
			root := t.TempDir()
			for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
				t.Setenv(key, root)
			}
			t.Setenv("PATH", t.TempDir())
			t.Setenv("AWS_PROFILE", "")
			t.Setenv("AWS_DEFAULT_PROFILE", "")
			t.Setenv("AWS_CONFIG_FILE", filepath.Join(root, "aws-config"))
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(root, "aws-credentials"))
			t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
			t.Setenv("CLOUDSDK_CONFIG", filepath.Join(root, "gcloud-config"))
			sentinel := filepath.Join(root, "existing-config")
			if err := os.WriteFile(sentinel, []byte("unchanged fixture\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := serviceConfigFiles(t, root)
			for _, canceled := range []bool{false, true} {
				ctx, cancel := context.WithCancel(context.Background())
				if canceled {
					cancel()
				}
				switcher := service.newSwitcher()
				operations := []struct {
					name string
					run  func(context.Context, any) error
				}{
					{"Switch", switcher.Switch}, {"Rollback", switcher.Rollback},
				}
				for _, operation := range operations {
					for _, input := range []struct {
						name  string
						value any
						valid bool
					}{
						{name: "typed-nil", value: service.typedNil},
						{name: "untyped-nil"},
						{name: "wrong-type", value: "invalid-config"},
						{name: "valid-empty", value: service.empty, valid: true},
					} {
						err := operation.run(ctx, input.value)
						if input.valid {
							if err != nil {
								t.Fatalf("%s/%s canceled=%v: %v", operation.name, input.name, canceled, err)
							}
						} else if err == nil || err.Error() != service.invalidMessage {
							t.Fatalf("%s/%s canceled=%v: error=%v want=%q", operation.name, input.name, canceled, err, service.invalidMessage)
						}
						if after := serviceConfigFiles(t, root); !reflect.DeepEqual(before, after) {
							t.Fatalf("%s/%s changed owned configuration: before=%v after=%v", operation.name, input.name, before, after)
						}
					}
				}
				cancel()
			}
		})
	}
}

// 취소된 profile-only 요청은 새 상태를 만들거나 기존 상태를 덮어쓰지 않는다.
func TestAWSProfileCancellationPreservesState(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, rollback := range []bool{false, true} {
			root := t.TempDir()
			for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
				t.Setenv(key, root)
			}
			t.Setenv("PATH", t.TempDir())
			directory, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			relative, err := filepath.Rel(root, directory)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
				t.Fatal("config directory escaped owned fixture")
			}
			stateFile := filepath.Join(directory, "gzh-dev-env", "aws-active-profile")
			if existing {
				if err := os.MkdirAll(filepath.Dir(stateFile), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(stateFile, []byte("old-profile\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before := serviceConfigFiles(t, root)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			switcher := aws.NewSwitcher()
			config := &environment.AWSConfig{Profile: "new-profile"}
			if rollback {
				err = switcher.Rollback(ctx, config)
			} else {
				err = switcher.Switch(ctx, config)
			}
			if !errors.Is(err, context.Canceled) || !strings.HasPrefix(err.Error(), "failed to set AWS profile:") {
				t.Fatalf("existing=%v rollback=%v: unexpected error %v", existing, rollback, err)
			}
			if after := serviceConfigFiles(t, root); !reflect.DeepEqual(before, after) {
				t.Fatalf("existing=%v rollback=%v: state changed before=%v after=%v", existing, rollback, before, after)
			}
		}
	}
}
