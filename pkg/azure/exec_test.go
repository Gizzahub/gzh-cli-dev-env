// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package azure

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
	keys := make([]string, 0, len(responses))
	for k := range responses {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if len(keys[j]) > len(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var calls [][]string
	orig := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		full := append([]string{name}, args...)
		calls = append(calls, full)
		key := strings.Join(full, " ")
		for _, k := range keys {
			if strings.Contains(key, k) {
				return responses[k].toCmd(ctx)
			}
		}
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { commandContext = orig })
	return &calls
}

type azureAccountRoutes struct {
	name     string
	userName string
	userType string
	showOK   bool
}

func withAzureAccountRouter(t *testing.T, r azureAccountRoutes) {
	t.Helper()
	orig := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "user.name"):
			return exec.CommandContext(ctx, "printf", "%s", r.userName)
		case strings.Contains(joined, "user.type"):
			return exec.CommandContext(ctx, "printf", "%s", r.userType)
		case strings.Contains(joined, "--query name"):
			return exec.CommandContext(ctx, "printf", "%s", r.name)
		case strings.Contains(joined, "account show") && !strings.Contains(joined, "--query"):
			if r.showOK {
				return exec.CommandContext(ctx, "true")
			}
			return exec.CommandContext(ctx, "false")
		default:
			return exec.CommandContext(ctx, "true")
		}
	}
	t.Cleanup(func() { commandContext = orig })
}

func TestSwitcher_Switch_SetsSubscription(t *testing.T) {
	calls := withCommandMap(t, map[string]cmdResponse{
		"account set": {code: 0},
	})
	err := NewSwitcher().Switch(context.Background(), &environment.AzureConfig{
		Subscription: "sub-123",
	})
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 az call, got %d: %v", len(*calls), *calls)
	}
	got := strings.Join((*calls)[0], " ")
	if !strings.Contains(got, "account set") || !strings.Contains(got, "sub-123") {
		t.Errorf("unexpected argv: %v", (*calls)[0])
	}
}

func TestSwitcher_Switch_SubscriptionFailure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"account set": {code: 1},
	})
	err := NewSwitcher().Switch(context.Background(), &environment.AzureConfig{
		Subscription: "bad-sub",
	})
	if err == nil {
		t.Fatal("Switch() expected error on az failure")
	}
	if !strings.Contains(err.Error(), "failed to set Azure subscription") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSwitcher_GetCurrentState_ParsesOutput(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"--query id":       {stdout: "sub-id-1\n"},
		"--query tenantId": {stdout: "tenant-id-1\n"},
	})
	state, err := NewSwitcher().GetCurrentState(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentState() error = %v", err)
	}
	cfg := state.(*environment.AzureConfig)
	if cfg.Subscription != "sub-id-1" {
		t.Errorf("Subscription = %q, want sub-id-1", cfg.Subscription)
	}
	if cfg.Tenant != "tenant-id-1" {
		t.Errorf("Tenant = %q, want tenant-id-1", cfg.Tenant)
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
	if st.Details["error"] != "Azure CLI not found" {
		t.Errorf("error detail = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_SubscriptionError(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"--query name": {code: 1},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusError {
		t.Errorf("Status = %v, want error", st.Status)
	}
	if !strings.Contains(st.Details["error"], "Failed to get current subscription") {
		t.Errorf("error detail = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_EmptySubscription(t *testing.T) {
	withLookPath(t, true)
	withCommandMap(t, map[string]cmdResponse{
		"--query name": {stdout: "  \n"},
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Details["error"] != "No Azure subscription configured" {
		t.Errorf("error detail = %q", st.Details["error"])
	}
}

func TestChecker_CheckStatus_ActiveUserAccount(t *testing.T) {
	withLookPath(t, true)
	withAzureAccountRouter(t, azureAccountRoutes{
		name:     "My Subscription",
		userName: "user@example.com",
		userType: "user",
		showOK:   true,
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusActive {
		t.Errorf("Status = %v, want active", st.Status)
	}
	if st.Current.Project != "My Subscription" {
		t.Errorf("Project = %q", st.Current.Project)
	}
	if st.Current.Account != "user@example.com" {
		t.Errorf("Account = %q", st.Current.Account)
	}
	if st.Credentials.Type != "user-account" {
		t.Errorf("Credentials.Type = %q, want user-account", st.Credentials.Type)
	}
	if !st.Credentials.Valid {
		t.Error("Credentials.Valid = false, want true")
	}
}

func TestChecker_CheckStatus_ServicePrincipal(t *testing.T) {
	withLookPath(t, true)
	withAzureAccountRouter(t, azureAccountRoutes{
		name:     "sp-sub",
		userName: "sp-app-id",
		userType: "servicePrincipal",
		showOK:   true,
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Credentials.Type != "service-principal" {
		t.Errorf("Credentials.Type = %q, want service-principal", st.Credentials.Type)
	}
}

func TestChecker_CheckStatus_InvalidCredentials(t *testing.T) {
	withLookPath(t, true)
	withAzureAccountRouter(t, azureAccountRoutes{
		name:     "sub",
		userName: "user",
		showOK:   false,
	})
	st, err := NewChecker().CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus() error = %v", err)
	}
	if st.Status != status.StatusInactive {
		t.Errorf("Status = %v, want inactive", st.Status)
	}
	if st.Credentials.Valid {
		t.Error("Credentials.Valid = true, want false")
	}
	if st.Credentials.Warning != "Credentials invalid or expired" {
		t.Errorf("Warning = %q", st.Credentials.Warning)
	}
}

func TestChecker_CheckHealth_Success(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"--output json": {stdout: `{"id":"sub"}`},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusActive {
		t.Errorf("Status = %v, want active", health.Status)
	}
	if !strings.Contains(health.Details["account_info"].(string), "sub") {
		t.Errorf("account_info = %v", health.Details["account_info"])
	}
}

func TestChecker_CheckHealth_Failure(t *testing.T) {
	withCommandMap(t, map[string]cmdResponse{
		"--output json": {code: 1},
	})
	health, err := NewChecker().CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	if health.Status != status.StatusError {
		t.Errorf("Status = %v, want error", health.Status)
	}
	if !strings.Contains(health.Message, "Failed to check Azure authentication") {
		t.Errorf("Message = %q", health.Message)
	}
}

func TestChecker_checkCredentials_DefaultUserType(t *testing.T) {
	withAzureAccountRouter(t, azureAccountRoutes{
		userType: "managedIdentity",
		showOK:   true,
	})
	cred, err := NewChecker().checkCredentials(context.Background())
	if err != nil {
		t.Fatalf("checkCredentials() error = %v", err)
	}
	if cred.Type != "managedIdentity" {
		t.Errorf("Type = %q, want managedIdentity", cred.Type)
	}
}
