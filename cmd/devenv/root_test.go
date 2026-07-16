package devenv

import (
	"strings"
	"testing"
)

func TestRootHelpMatchesRegisteredCommands(t *testing.T) {
	cmd := NewRootCmd()
	// Registered command names
	registered := map[string]bool{}
	for _, c := range cmd.Commands() {
		registered[c.Name()] = true
	}
	for _, name := range []string{"status", "tui", "switch-all"} {
		if !registered[name] {
			t.Errorf("expected registered command %q", name)
		}
	}

	// Help/examples must not advertise unregistered commands
	help := cmd.Long
	for _, bad := range []string{"kubeconfig save", "aws-profile list", "aws-profile switch"} {
		if strings.Contains(help, bad) {
			t.Errorf("root help advertises unregistered command fragment %q", bad)
		}
	}
	// Examples section should mention the three real commands
	for _, good := range []string{"dev-env status", "dev-env tui", "dev-env switch-all"} {
		if !strings.Contains(help, good) {
			t.Errorf("root help missing example for %q", good)
		}
	}
}
