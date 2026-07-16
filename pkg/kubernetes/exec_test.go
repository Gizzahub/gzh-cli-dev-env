// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package kubernetes

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/status"
)

func withLookPath(t *testing.T, available bool) {
	t.Helper()
	orig := lookPath
	lookPath = func(file string) (string, error) {
		if available {
			return "/usr/bin/" + file, nil
		}
		return "", fmt.Errorf("%s not found", file)
	}
	t.Cleanup(func() { lookPath = orig })
}

type cmdResponse struct {
	stdout string
	code   int
}

func (r cmdResponse) toCmd(ctx context.Context) *exec.Cmd {
	if r.code == 0 {
		return exec.CommandContext(ctx, "printf", "%s", r.stdout)
	}
	return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("exit %d", r.code))
}

func withCommandMap(t *testing.T, responses map[string]cmdResponse) *[][]string {
	t.Helper()
	var calls [][]string
	orig := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		full := append([]string{name}, args...)
		calls = append(calls, full)
		key := strings.Join(full, " ")
		for prefix, resp := range responses {
			if strings.Contains(key, prefix) {
				return resp.toCmd(ctx)
			}
		}
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { commandContext = orig })
	return &calls
}

func TestSwitcher_Switch_ContextAndNamespace(t *testing.T) {
	calls := withCommandMap(t, map[string]cmdResponse{
		"use-context": {code: 0},
		"set-context": {code: 0},
	})
	err := NewSwitcher().Switch(context.Background(), &environment.KubernetesConfig{
		Context:   "prod-ctx",
		Namespace: "apps",
	})
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("expected 2 kubectl calls, got %d: %v", len(*calls), *calls)
	}
	joined := strings.Join((*calls)[0], " ") + " | " + strings.Join((*calls)[1], " ")
	if !strings.Contains(joined, "use-context prod-ctx") {
		t.Errorf("missing use-context: %s", joined)
	}
	if !strings.Contains(joined, "set-context") || !strings.Contains(joined, "apps") {
		t.Errorf("missing set-context namespace: %s", joined)
	}
}

func TestSwitcher_Switch_ContextFailure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"use-context": {code: 1},
	})
	err := NewSwitcher().Switch(context.Background(), &environment.KubernetesConfig{
		Context: "missing",
	})
	if err == nil {
		t.Fatal("Switch() expected error")
	}
	if !strings.Contains(err.Error(), "failed to set Kubernetes context") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSwitcher_Switch_NamespaceFailure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"use-context": {code: 0},
		"set-context": {code: 1},
	})
	err := NewSwitcher().Switch(context.Background(), &environment.KubernetesConfig{
		Context:   "ctx",
		Namespace: "ns",
	})
	if err == nil {
		t.Fatal("Switch() expected error")
	}
	if !strings.Contains(err.Error(), "failed to set Kubernetes namespace") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSwitcher_GetCurrentState_ParsesOutput(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"current-context": {stdout: "dev-ctx\n"},
		"jsonpath={..namespace}": {stdout: "staging\n"},
	})
	state, err := NewSwitcher().GetCurrentState(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}
	cfg := state.(*environment.KubernetesConfig)
	if cfg.Context != "dev-ctx" {
		t.Errorf("Context = %q", cfg.Context)
	}
	if cfg.Namespace != "staging" {
		t.Errorf("Namespace = %q", cfg.Namespace)
	}
}

func TestChecker_CheckStatus_CLIMissing(t *testing.T) {
	withLookPath(t, false)
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Details["error"] != "kubectl not found" {
		t.Errorf("error detail = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_ContextError(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"current-context": {code: 1},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusError {
		t.Errorf("Status = %v, want error", st.Status)
	}
}

func TestChecker_CheckStatus_EmptyContext(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"current-context": {stdout: "\n"},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Details["error"] != "No Kubernetes context set" {
		t.Errorf("error detail = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_ActiveCluster(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"current-context":        {stdout: "minikube"},
		"jsonpath={..namespace}": {stdout: "kube-system"},
		"auth can-i":             {code: 0},
		"jsonpath={.contexts":    {stdout: "admin"},
		"config view --raw":      {stdout: `{"token":"x"}`},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusActive {
		t.Errorf("Status = %v, want active (details=%v)", st.Status, st.Details)
	}
	if st.Current.Context != "minikube" {
		t.Errorf("Context = %q", st.Current.Context)
	}
	if st.Current.Namespace != "kube-system" {
		t.Errorf("Namespace = %q", st.Current.Namespace)
	}
	if !st.Credentials.Valid {
		t.Error("Credentials.Valid = false")
	}
}

func TestChecker_CheckStatus_ClusterInaccessible(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"current-context":        {stdout: "ctx"},
		"jsonpath={..namespace}": {stdout: "default"},
		"auth can-i":             {code: 1},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Credentials.Warning != "Cannot access Kubernetes cluster" {
		t.Errorf("Warning = %q", st.Credentials.Warning)
	}
}

func TestChecker_CheckStatus_OIDCToken(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"current-context":        {stdout: "oidc-ctx"},
		"jsonpath={..namespace}": {stdout: ""},
		"auth can-i":             {code: 0},
		"jsonpath={.contexts":    {stdout: "oidc-user"},
		"config view --raw":      {stdout: `{"expiry":"2026-01-01"}`},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Current.Namespace != DefaultNamespace {
		t.Errorf("Namespace = %q, want default for empty", st.Current.Namespace)
	}
	if st.Credentials.Type != "oidc-token" {
		t.Errorf("Type = %q, want oidc-token", st.Credentials.Type)
	}
}

func TestChecker_getCurrentNamespace_OnErrorDefaults(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"jsonpath={..namespace}": {code: 1},
	})
	ns, err := NewChecker().getCurrentNamespace(context.Background())
	if err != nil {
		t.Fatalf("getCurrentNamespace() error = %v", err)
	}
	if ns != DefaultNamespace {
		t.Errorf("namespace = %q, want %q", ns, DefaultNamespace)
	}
}

func TestChecker_CheckHealth_SuccessWithNodes(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"cluster-info": {stdout: "Kubernetes control plane is running"},
		"get nodes":    {stdout: "node1 True"},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusActive {
		t.Errorf("Status = %v", health.Status)
	}
	if health.Details["node_status"] == nil {
		t.Error("expected node_status detail")
	}
}

func TestChecker_CheckHealth_Failure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"cluster-info": {code: 1},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusError {
		t.Errorf("Status = %v, want error", health.Status)
	}
}

func TestChecker_getCurrentUser_Error(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"jsonpath={.contexts": {code: 1},
	})
	if got := NewChecker().getCurrentUser(context.Background()); got != "" {
		t.Errorf("getCurrentUser() = %q, want empty", got)
	}
}
