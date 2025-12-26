# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-26

### Added

- Initial release extracted from gzh-cli
- Core packages:
  - `pkg/environment` - Environment switching with dependency resolution
  - `pkg/status` - Service status collection and formatting
  - `pkg/config` - Configuration management
  - `pkg/tui` - Bubbletea TUI dashboard
- Service checkers and switchers:
  - AWS (profile, region, credentials)
  - GCP (project, account, region)
  - Azure (subscription, tenant)
  - Docker (context)
  - Kubernetes (context, namespace)
  - SSH (configuration)
- Status formatters: Table, JSON, YAML
- Parallel status collection with configurable timeout
- Rollback support for environment switching
