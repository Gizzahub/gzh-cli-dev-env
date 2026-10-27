// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	targetJournalVersion = "aws-target-v2"
	awsRecoveryType      = "AWSRecoveryState"
	nilRecoveryType      = "nil"
	awsServiceName       = "aws"
	presenceField        = "Present"
)

// MarshalJSON preserves the fail-closed wire discriminator even when a caller
// reserializes a loaded journal. A numeric version bump is unsafe: v0.1.1 never
// checked its value. A string fails its int unmarshal before any recovery call.
func (j RollbackJournal) MarshalJSON() ([]byte, error) {
	type fields RollbackJournal
	for _, entry := range j.Services {
		if entry.Type == awsRecoveryType && j.Version != 2 {
			return nil, fmt.Errorf("target-aware AWS state requires target journal version")
		}
	}
	var version any
	switch j.Version {
	case 1:
		version = 1
	case 2:
		version = targetJournalVersion
	default:
		return nil, fmt.Errorf("unsupported journal version %d", j.Version)
	}
	return json.Marshal(struct {
		Version any `json:"version"`
		*fields
	}{Version: version, fields: (*fields)(&j)})
}

// UnmarshalJSON accepts legacy non-target journals and the target-aware format.
// Unknown discriminators never become zero-value recovery snapshots.
func (j *RollbackJournal) UnmarshalJSON(data []byte) error {
	type fields RollbackJournal
	var wire struct {
		Version json.RawMessage `json:"version"`
		*fields
	}
	var decoded fields
	wire.fields = &decoded
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	switch string(wire.Version) {
	case "1":
		decoded.Version = 1
	case `"` + targetJournalVersion + `"`:
		decoded.Version = 2
	default:
		return fmt.Errorf("unsupported journal version %s", wire.Version)
	}
	*j = RollbackJournal(decoded)
	return nil
}

func validateRecoveryEntry(version int, name string, entry JournalEntry) error {
	expected := map[string]string{
		awsServiceName: awsRecoveryType, "gcp": "GCPConfig", "azure": "AzureConfig",
		"docker": "DockerConfig", "kubernetes": "KubernetesConfig", "ssh": "SSHConfig",
	}
	if name != awsServiceName && (entry.Type == "AWSConfig" || entry.Type == awsRecoveryType) {
		return fmt.Errorf("AWS recovery type belongs to aws service")
	}
	if entry.Type == nilRecoveryType && name != awsServiceName {
		return nil
	}
	if expected[name] != "" && entry.Type != expected[name] {
		return fmt.Errorf("unsupported recovery type %q for %s; legacy AWS state lacks target metadata", entry.Type, name)
	}
	if name == awsServiceName && version != 2 {
		return fmt.Errorf("target-aware AWS state requires target journal version")
	}
	if len(entry.Data) == 0 || (expected[name] != "" && bytes.Equal(bytes.TrimSpace(entry.Data), []byte("null"))) {
		return fmt.Errorf("missing recovery data")
	}
	return nil
}

// Require every presence bit and target field. A missing false/empty field must
// not silently decode as an instruction to remove a resource.
func requireAWSRecoveryFields(data []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	for _, key := range []string{"Version", "Request", "ProfileBefore", "ProfileExpected", "ConfigPath", "ConfigExisted", "ConfigMode", "TargetProfile", "RegionBefore", "RegionExpected"} {
		if !presentJSON(object[key]) {
			return fmt.Errorf("AWS recovery metadata missing %s", key)
		}
	}
	for key, required := range map[string][]string{
		"Request":         {"Profile", "Region", "AccountID"},
		"ProfileBefore":   {"Path", presenceField, "Raw", "Mode"},
		"ProfileExpected": {"Path", presenceField, "Raw", "Mode"},
		"RegionBefore":    {presenceField, "Value"}, "RegionExpected": {presenceField, "Value"},
	} {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(object[key], &nested); err != nil {
			return err
		}
		for _, field := range required {
			if !presentJSON(nested[field]) {
				return fmt.Errorf("AWS recovery metadata missing %s.%s", key, field)
			}
		}
	}
	return nil
}

func presentJSON(raw json.RawMessage) bool {
	return len(raw) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
