// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package devenv

import (
	"github.com/gizzahub/gzh-cli-dev-env/pkg/aws"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/azure"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/docker"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/gcp"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/kubernetes"
	"github.com/gizzahub/gzh-cli-dev-env/pkg/ssh"
)

// registerDefaultSwitchers registers all default service switchers.
func registerDefaultSwitchers(switcher *environment.EnvironmentSwitcher) {
	// Register AWS switcher
	switcher.RegisterServiceSwitcher("aws", aws.NewSwitcher())

	// Register GCP switcher
	switcher.RegisterServiceSwitcher("gcp", gcp.NewSwitcher())

	// Register Azure switcher
	switcher.RegisterServiceSwitcher("azure", azure.NewSwitcher())

	// Register Docker switcher
	switcher.RegisterServiceSwitcher("docker", docker.NewSwitcher())

	// Register Kubernetes switcher
	switcher.RegisterServiceSwitcher("kubernetes", kubernetes.NewSwitcher())

	// Register SSH switcher
	switcher.RegisterServiceSwitcher("ssh", ssh.NewSwitcher())
}
