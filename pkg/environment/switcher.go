// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// EnvironmentSwitcher handles switching between different development environments.
type EnvironmentSwitcher struct {
	serviceSwitchers map[string]ServiceSwitcher
	progressCallback func(SwitchProgress)
	// mu guards serviceSwitchers only.
	mu sync.RWMutex
	// stateMu guards the previousStates map and the SwitchResult slices that
	// switchSingleService writes to, which parallel switching shares across goroutines.
	stateMu sync.Mutex
}

// NewEnvironmentSwitcher creates a new environment switcher.
func NewEnvironmentSwitcher() *EnvironmentSwitcher {
	return &EnvironmentSwitcher{
		serviceSwitchers: make(map[string]ServiceSwitcher),
	}
}

// RegisterServiceSwitcher registers a service switcher.
func (es *EnvironmentSwitcher) RegisterServiceSwitcher(name string, switcher ServiceSwitcher) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.serviceSwitchers[name] = switcher
}

// Register is an alias for RegisterServiceSwitcher that uses the switcher's name.
func (es *EnvironmentSwitcher) Register(switcher ServiceSwitcher) {
	es.RegisterServiceSwitcher(switcher.Name(), switcher)
}

// SetProgressCallback sets the progress callback function.
func (es *EnvironmentSwitcher) SetProgressCallback(callback func(SwitchProgress)) {
	es.progressCallback = callback
}

// SwitchEnvironment switches to the specified environment.
func (es *EnvironmentSwitcher) SwitchEnvironment(ctx context.Context, env *Environment, options SwitchOptions) (*SwitchResult, error) { //nolint:gocognit // journal-clear error path is one extra branch on an existing orchestration func
	startTime := time.Now()

	if err := env.Validate(); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	resolver := NewDependencyResolver(env.Services, env.Dependencies)
	groups, err := resolver.GetParallelGroups()
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	result := &SwitchResult{
		Success:          true,
		SwitchedServices: []string{},
		FailedServices:   []string{},
		Errors:           []SwitchError{},
	}

	previousStates := make(map[string]any)

	journalPath, journalPathErr := DefaultJournalPath()
	if journalPathErr != nil {
		// Non-fatal: switch can proceed without crash-recovery journal.
		journalPath = ""
	}

	if err := es.runHooks(ctx, env.PreHooks, "pre-hook", options); err != nil {
		return &SwitchResult{
			Success:  false,
			Duration: time.Since(startTime),
			Errors:   []SwitchError{{Service: "pre-hook", Error: err.Error(), Time: time.Now()}},
		}, err
	}

	totalServices := len(env.Services)
	completedServices := 0

	for _, group := range groups {
		if options.Parallel && len(group.Services) > 1 {
			if err := es.switchServicesParallel(ctx, env, group.Services, previousStates, result, options, journalPath); err != nil {
				if options.RollbackOnError {
					es.rollbackServices(ctx, previousStates, result, options)
				}
				result.Success = false
				result.Duration = time.Since(startTime)
				return result, err
			}
		} else {
			for _, serviceName := range group.Services {
				if err := es.switchSingleService(ctx, env, serviceName, previousStates, result, options, journalPath); err != nil {
					if options.RollbackOnError {
						es.rollbackServices(ctx, previousStates, result, options)
					}
					result.Success = false
					result.Duration = time.Since(startTime)
					return result, err
				}
			}
		}

		completedServices += len(group.Services)

		if es.progressCallback != nil {
			progress := SwitchProgress{
				TotalServices:     totalServices,
				CompletedServices: completedServices,
				Status:            fmt.Sprintf("Completed group %d", group.Level),
				StartTime:         startTime,
				EstimatedEnd:      startTime.Add(time.Duration(float64(time.Since(startTime)) * float64(totalServices) / float64(completedServices))),
			}
			es.progressCallback(progress)
		}
	}

	if err := es.runHooks(ctx, env.PostHooks, "post-hook", options); err != nil {
		result.Errors = append(result.Errors, SwitchError{
			Service: "post-hook",
			Error:   err.Error(),
			Time:    time.Now(),
		})
	}

	// Successful completion: remove crash-recovery journal.
	if journalPath != "" {
		if err := ClearJournal(journalPath); err != nil {
			result.Errors = append(result.Errors, SwitchError{
				Service: "journal",
				Error:   err.Error(),
				Time:    time.Now(),
			})
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// switchSingleService switches a single service.
// journalPath is the rollback journal location; empty disables disk persistence.
func (es *EnvironmentSwitcher) switchSingleService(ctx context.Context, env *Environment, serviceName string, previousStates map[string]any, result *SwitchResult, options SwitchOptions, journalPath string) error {
	es.mu.RLock()
	switcher, exists := es.serviceSwitchers[serviceName]
	es.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no switcher registered for service: %s", serviceName)
	}

	serviceConfig, exists := env.Services[serviceName]
	if !exists {
		return fmt.Errorf("service configuration not found: %s", serviceName)
	}

	currentState, err := switcher.GetCurrentState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current state for %s: %w", serviceName, err)
	}

	es.stateMu.Lock()
	previousStates[serviceName] = currentState
	// Snapshot under lock so parallel writers see a consistent map for the journal.
	statesSnapshot := make(map[string]any, len(previousStates))
	for k, v := range previousStates {
		statesSnapshot[k] = v
	}
	es.stateMu.Unlock()

	// Persist previous states before Switch so a crash mid-switch leaves a recovery artifact.
	// WriteJournal owns the dry-run guard (no file on DryRun).
	if journalPath != "" {
		if err := WriteJournal(journalPath, options.DryRun, env.Name, statesSnapshot); err != nil {
			return fmt.Errorf("failed to persist rollback journal for %s: %w", serviceName, err)
		}
	}

	var config any
	switch serviceName {
	case "aws":
		config = serviceConfig.AWS
	case "gcp":
		config = serviceConfig.GCP
	case "azure":
		config = serviceConfig.Azure
	case "docker":
		config = serviceConfig.Docker
	case "kubernetes":
		config = serviceConfig.Kubernetes
	case "ssh":
		config = serviceConfig.SSH
	default:
		return fmt.Errorf("unknown service type: %s", serviceName)
	}

	if config == nil {
		return fmt.Errorf("no configuration provided for service: %s", serviceName)
	}

	if !options.DryRun {
		if err := switcher.Switch(ctx, config); err != nil {
			es.stateMu.Lock()
			result.FailedServices = append(result.FailedServices, serviceName)
			result.Errors = append(result.Errors, SwitchError{
				Service: serviceName,
				Error:   err.Error(),
				Time:    time.Now(),
			})
			es.stateMu.Unlock()

			return fmt.Errorf("failed to switch %s: %w", serviceName, err)
		}
	}

	es.stateMu.Lock()
	result.SwitchedServices = append(result.SwitchedServices, serviceName)
	es.stateMu.Unlock()

	return nil
}

// switchServicesParallel switches multiple services in parallel.
func (es *EnvironmentSwitcher) switchServicesParallel(ctx context.Context, env *Environment, serviceNames []string, previousStates map[string]any, result *SwitchResult, options SwitchOptions, journalPath string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(serviceNames))

	for _, serviceName := range serviceNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := es.switchSingleService(ctx, env, name, previousStates, result, options, journalPath); err != nil {
				errChan <- err
			}
		}(serviceName)
	}

	wg.Wait()
	close(errChan)

	errs := make([]string, 0, len(serviceNames))
	for err := range errChan {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("parallel switch failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

// rollbackServices rolls back services to their previous states.
//
// A dry run never called Switch, so there is nothing to undo and Rollback would be a
// real side effect on the user's machine. previousStates is populated even in a dry
// run, so the guard belongs here rather than at the call sites — same reason runHooks
// owns its own check.
func (es *EnvironmentSwitcher) rollbackServices(ctx context.Context, previousStates map[string]any, result *SwitchResult, options SwitchOptions) {
	if options.DryRun {
		return
	}

	var rollbackErrors []string

	for serviceName, previousState := range previousStates {
		es.mu.RLock()
		switcher, exists := es.serviceSwitchers[serviceName]
		es.mu.RUnlock()

		if !exists {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("no switcher for %s", serviceName))
			continue
		}

		if err := switcher.Rollback(ctx, previousState); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s: %v", serviceName, err))
		}
	}

	result.RollbackPerformed = true
	if len(rollbackErrors) > 0 {
		result.Errors = append(result.Errors, SwitchError{
			Service: "rollback",
			Error:   strings.Join(rollbackErrors, "; "),
			Time:    time.Now(),
		})
	}
}

// runHooks executes hooks unless this is a dry run. Hooks are arbitrary shell
// commands, so running them would make --dry-run mutate the machine it claims to
// leave untouched. Keeping the check here means every hook call site inherits it.
func (es *EnvironmentSwitcher) runHooks(ctx context.Context, hooks []Hook, hookType string, options SwitchOptions) error {
	if options.DryRun {
		return nil
	}

	return es.executeHooks(ctx, hooks, hookType)
}

// executeHooks executes pre or post hooks.
func (es *EnvironmentSwitcher) executeHooks(ctx context.Context, hooks []Hook, hookType string) error {
	for i, hook := range hooks {
		if err := es.executeHook(ctx, hook, fmt.Sprintf("%s-%d", hookType, i)); err != nil {
			if hook.OnError == "continue" {
				continue
			}
			return fmt.Errorf("hook execution failed: %w", err)
		}
	}
	return nil
}

// executeHook executes a single hook with input validation.
// Commands run via direct exec (no shell) after ValidateHookCommand — same model as
// gzh-cli-gitforge hooks: pipes/redirects/variables are rejected at validation time.
func (es *EnvironmentSwitcher) executeHook(ctx context.Context, hook Hook, hookName string) error {
	if err := ValidateHookCommand(hook.Command); err != nil {
		return fmt.Errorf("hook '%s' validation failed: %w", hookName, err)
	}

	args := ParseHookCommand(hook.Command)
	if len(args) == 0 {
		return fmt.Errorf("hook '%s' validation failed: empty command after parse", hookName)
	}

	timeout := hook.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 -- argv is user config, validated, and not passed through a shell
	cmd := exec.CommandContext(hookCtx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook '%s' failed: %w (output: %s)", hookName, err, string(output))
	}

	return nil
}

// GetAvailableServices returns a list of available service switchers.
func (es *EnvironmentSwitcher) GetAvailableServices() []string {
	es.mu.RLock()
	defer es.mu.RUnlock()

	services := make([]string, 0, len(es.serviceSwitchers))
	for name := range es.serviceSwitchers {
		services = append(services, name)
	}
	return services
}

// shellMetaChars are characters that require a shell. Hooks are executed with
// direct exec (no sh -c), so these must be rejected rather than blocklisted.
var shellMetaChars = []string{
	"|", ">", "<", ";", "&", "$", "`", "\n", "\r",
	"&&", "||", "|&", "$(", ">>", "<<",
}

// ValidateHookCommand validates a hook command for direct exec (no shell).
// Shell metacharacters are rejected; allowed form is a single program plus args
// with optional simple quotes (see ParseHookCommand).
func ValidateHookCommand(command string) error {
	if command == "" {
		return errors.New("hook command cannot be empty")
	}

	if len(command) > 1000 {
		return errors.New("hook command too long (max 1000 characters)")
	}

	for _, meta := range shellMetaChars {
		if strings.Contains(command, meta) {
			return fmt.Errorf("hook command contains shell metacharacter %q — use a script instead of shell features", meta)
		}
	}

	// Deny-list high-risk program names even without shell (config may be shared).
	dangerousPrograms := []string{
		"rm", "curl", "wget", "sudo", "su", "eval", "exec", "sh", "bash", "zsh", "python", "perl", "ruby", "node",
	}
	args := ParseHookCommand(command)
	if len(args) == 0 {
		return errors.New("hook command has no executable after parse")
	}
	base := strings.ToLower(args[0])
	// basename of path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	for _, d := range dangerousPrograms {
		if base == d {
			return fmt.Errorf("hook command invokes disallowed program: %s", base)
		}
	}

	// Remaining characters after quote stripping must be conservative.
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9\s\-_./=:@\[\]{}()"']+$`)
	if !safePattern.MatchString(command) {
		return errors.New("hook command contains unsafe characters")
	}

	return nil
}

// ParseHookCommand splits a hook command into argv without invoking a shell.
// Supports simple single/double quotes; does not support pipes, redirects, or expansion.
func ParseHookCommand(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	inQuote := false
	var quoteChar rune

	for _, r := range cmd {
		switch {
		case inQuote:
			if r == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			inQuote = true
			quoteChar = r
		case r == ' ' || r == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
