// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

// Package config provides configuration management for development environments.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles saving, loading, and listing configuration files.
type Manager struct {
	serviceName    string
	configFileName string
	defaultConfig  string
	storePath      string
}

// Options represents options for configuration operations.
type Options struct {
	Name        string
	Description string
	ConfigPath  string
	StorePath   string
	Force       bool
}

// ConfigMetadata represents metadata for saved configurations.
type ConfigMetadata struct {
	Description string    `json:"description"`
	SavedAt     time.Time `json:"saved_at"`
	SourcePath  string    `json:"source_path"`
}

// ConfigInfo represents information about a saved configuration.
type ConfigInfo struct {
	Name        string
	Description string
	SavedAt     time.Time
	SourcePath  string
	Size        int64
}

// NewManager creates a new configuration manager.
func NewManager(serviceName, configFileName, defaultConfig string) *Manager {
	homeDir, _ := os.UserHomeDir()
	return &Manager{
		serviceName:    serviceName,
		configFileName: configFileName,
		defaultConfig:  defaultConfig,
		storePath:      filepath.Join(homeDir, ".gz", serviceName+"-configs"),
	}
}

// DefaultOptions returns default options for the service.
func (m *Manager) DefaultOptions() *Options {
	homeDir, _ := os.UserHomeDir()

	return &Options{
		ConfigPath: filepath.Join(homeDir, m.defaultConfig),
		StorePath:  m.storePath,
	}
}

// ServiceName returns the service name.
func (m *Manager) ServiceName() string {
	return m.serviceName
}

// StorePath returns the default store path.
func (m *Manager) StorePath() string {
	return m.storePath
}

// Save saves the current configuration to the store.
func (m *Manager) Save(opts *Options) error {
	if opts.Name == "" {
		return fmt.Errorf("configuration name is required")
	}

	// Check if source config exists
	if _, err := os.Stat(opts.ConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("%s config file not found at %s", m.serviceName, opts.ConfigPath)
	}

	storePath := opts.StorePath
	if storePath == "" {
		storePath = m.storePath
	}

	// Create store directory if it doesn't exist
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// Check if config already exists
	configFile := filepath.Join(storePath, opts.Name+"."+m.configFileName)
	if _, err := os.Stat(configFile); err == nil && !opts.Force {
		return fmt.Errorf("configuration '%s' already exists (use force to overwrite)", opts.Name)
	}

	// Copy config file
	if err := copyFile(opts.ConfigPath, configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Save metadata
	metadata := ConfigMetadata{
		Description: opts.Description,
		SavedAt:     time.Now(),
		SourcePath:  opts.ConfigPath,
	}

	metadataFile := filepath.Join(storePath, opts.Name+".metadata.json")
	if err := saveMetadata(metadataFile, metadata); err != nil {
		// Don't fail if metadata save fails
		return nil
	}

	return nil
}

// Load loads a saved configuration to the specified path.
func (m *Manager) Load(opts *Options) (*ConfigMetadata, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("configuration name is required")
	}

	storePath := opts.StorePath
	if storePath == "" {
		storePath = m.storePath
	}

	// Check if saved config exists
	configFile := filepath.Join(storePath, opts.Name+"."+m.configFileName)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration '%s' not found", opts.Name)
	}

	// Check if target config already exists
	if _, err := os.Stat(opts.ConfigPath); err == nil && !opts.Force {
		return nil, fmt.Errorf("config file already exists at %s (use force to overwrite)", opts.ConfigPath)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(opts.ConfigPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Copy config file
	if err := copyFile(configFile, opts.ConfigPath); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load metadata if available
	metadataFile := filepath.Join(storePath, opts.Name+".metadata.json")
	metadata, _ := loadMetadata(metadataFile)

	return metadata, nil
}

// List lists all saved configurations.
func (m *Manager) List(storePath string) ([]ConfigInfo, error) {
	if storePath == "" {
		storePath = m.storePath
	}

	// Check if store directory exists
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		return nil, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	// Filter for config files
	configExtension := "." + m.configFileName
	var configs []ConfigInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), configExtension) {
			continue
		}

		configName := strings.TrimSuffix(entry.Name(), configExtension)
		info := ConfigInfo{Name: configName}

		// Load metadata if available
		metadataFile := filepath.Join(storePath, configName+".metadata.json")
		if metadata, err := loadMetadata(metadataFile); err == nil {
			info.Description = metadata.Description
			info.SavedAt = metadata.SavedAt
			info.SourcePath = metadata.SourcePath
		}

		// Get file size
		configFile := filepath.Join(storePath, entry.Name())
		if stat, err := os.Stat(configFile); err == nil {
			info.Size = stat.Size()
		}

		configs = append(configs, info)
	}

	return configs, nil
}

// Delete deletes a saved configuration.
func (m *Manager) Delete(name, storePath string) error {
	if name == "" {
		return fmt.Errorf("configuration name is required")
	}

	if storePath == "" {
		storePath = m.storePath
	}

	configFile := filepath.Join(storePath, name+"."+m.configFileName)
	metadataFile := filepath.Join(storePath, name+".metadata.json")

	// Check if config exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration '%s' not found", name)
	}

	// Delete config file
	if err := os.Remove(configFile); err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	// Delete metadata file (ignore errors)
	_ = os.Remove(metadataFile)

	return nil
}

// Exists checks if a configuration with the given name exists.
func (m *Manager) Exists(name, storePath string) bool {
	if storePath == "" {
		storePath = m.storePath
	}

	configFile := filepath.Join(storePath, name+"."+m.configFileName)
	_, err := os.Stat(configFile)
	return err == nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// saveMetadata saves metadata to a JSON file.
func saveMetadata(filename string, metadata ConfigMetadata) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metadata)
}

// loadMetadata loads metadata from a JSON file.
func loadMetadata(filename string) (*ConfigMetadata, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metadata ConfigMetadata
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&metadata)
	return &metadata, err
}
