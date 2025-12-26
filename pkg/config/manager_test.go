// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("test-service", "config.yaml", ".test/config")

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.serviceName != "test-service" {
		t.Errorf("serviceName = %v, want test-service", manager.serviceName)
	}
	if manager.configFileName != "config.yaml" {
		t.Errorf("configFileName = %v, want config.yaml", manager.configFileName)
	}
}

func TestManager_ServiceName(t *testing.T) {
	manager := NewManager("my-service", "config.yaml", "default")
	if manager.ServiceName() != "my-service" {
		t.Errorf("ServiceName() = %v, want my-service", manager.ServiceName())
	}
}

func TestManager_StorePath(t *testing.T) {
	manager := NewManager("test-service", "config.yaml", "default")
	storePath := manager.StorePath()
	if storePath == "" {
		t.Error("StorePath returned empty string")
	}
}

func TestManager_DefaultOptions(t *testing.T) {
	manager := NewManager("test-service", "config.yaml", ".test/config")
	opts := manager.DefaultOptions()

	if opts == nil {
		t.Fatal("DefaultOptions returned nil")
	}
	if opts.StorePath == "" {
		t.Error("StorePath should not be empty")
	}
}

func TestManager_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source config file
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "config.yaml")
	testContent := []byte("key: value\nname: test")
	if err := os.WriteFile(sourceFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create manager with custom store path
	manager := NewManager("test-service", "config.yaml", ".test/config")
	storePath := filepath.Join(tmpDir, "store")

	// Save config
	saveOpts := &Options{
		Name:        "test-config",
		Description: "Test configuration",
		ConfigPath:  sourceFile,
		StorePath:   storePath,
	}

	if err := manager.Save(saveOpts); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify saved file exists
	if !manager.Exists("test-config", storePath) {
		t.Error("Config should exist after save")
	}

	// Load config
	loadDir := filepath.Join(tmpDir, "loaded")
	if err := os.MkdirAll(loadDir, 0o755); err != nil {
		t.Fatalf("Failed to create load dir: %v", err)
	}

	loadFile := filepath.Join(loadDir, "config.yaml")
	loadOpts := &Options{
		Name:       "test-config",
		ConfigPath: loadFile,
		StorePath:  storePath,
		Force:      true,
	}

	metadata, err := manager.Load(loadOpts)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if metadata.Description != "Test configuration" {
		t.Errorf("Loaded description = %v, want 'Test configuration'", metadata.Description)
	}

	// Verify loaded file content
	loadedContent, err := os.ReadFile(loadFile)
	if err != nil {
		t.Fatalf("Failed to read loaded file: %v", err)
	}

	if string(loadedContent) != string(testContent) {
		t.Errorf("Loaded content mismatch")
	}
}

func TestManager_List(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "config-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager("test-service", "config.yaml", "default")

	// Create test config files directly
	for _, name := range []string{"config1", "config2", "config3"} {
		configFile := filepath.Join(tmpDir, name+".config.yaml")
		if err := os.WriteFile(configFile, []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
	}

	// List configs
	configs, err := manager.List(tmpDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(configs) != 3 {
		t.Errorf("List returned %d configs, want 3", len(configs))
	}
}

func TestManager_Delete(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "config-delete-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager("test-service", "config.yaml", "default")

	// Create a config to delete
	configFile := filepath.Join(tmpDir, "to-delete.config.yaml")
	if err := os.WriteFile(configFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify exists
	if !manager.Exists("to-delete", tmpDir) {
		t.Error("Config should exist before delete")
	}

	// Delete
	if err := manager.Delete("to-delete", tmpDir); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	if manager.Exists("to-delete", tmpDir) {
		t.Error("Config should not exist after delete")
	}
}

func TestManager_Exists_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-exists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager := NewManager("test-service", "config.yaml", "default")

	if manager.Exists("nonexistent", tmpDir) {
		t.Error("Nonexistent config should return false")
	}
}

func TestOptions_Fields(t *testing.T) {
	opts := Options{
		Name:        "my-config",
		Description: "My configuration",
		ConfigPath:  "/path/to/config",
		StorePath:   "/path/to/store",
		Force:       true,
	}

	if opts.Name != "my-config" {
		t.Error("Name mismatch")
	}
	if opts.Description != "My configuration" {
		t.Error("Description mismatch")
	}
	if opts.ConfigPath != "/path/to/config" {
		t.Error("ConfigPath mismatch")
	}
	if !opts.Force {
		t.Error("Force should be true")
	}
}

func TestConfigInfo_Fields(t *testing.T) {
	info := ConfigInfo{
		Name:        "test-config",
		Description: "Test configuration",
		Size:        1024,
	}

	if info.Name != "test-config" {
		t.Error("Name mismatch")
	}
	if info.Size != 1024 {
		t.Errorf("Size = %d, want 1024", info.Size)
	}
}
