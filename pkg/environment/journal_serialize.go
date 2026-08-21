// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// secretFieldHints mark field/key names that must not be persisted in the journal.
var secretFieldHints = []string{
	"password", "secret", "token", "credential", "privatekey", "private_key",
	"accesskey", "access_key", "secretkey", "secret_key", "apikey", "api_key",
}

// serializeState converts a GetCurrentState value into a journal entry.
// Known service configs are marshaled as-is (safe fields only).
// Opaque values store the type name plus JSON of non-secret exported fields.
func serializeState(state any) (JournalEntry, error) {
	if state == nil {
		return JournalEntry{Type: "nil", Data: json.RawMessage("null")}, nil
	}

	switch v := state.(type) {
	case *AWSConfig:
		return marshalTyped("AWSConfig", v)
	case AWSConfig:
		return marshalTyped("AWSConfig", v)
	case *GCPConfig:
		return marshalTyped("GCPConfig", v)
	case GCPConfig:
		return marshalTyped("GCPConfig", v)
	case *AzureConfig:
		return marshalTyped("AzureConfig", v)
	case AzureConfig:
		return marshalTyped("AzureConfig", v)
	case *DockerConfig:
		return marshalTyped("DockerConfig", v)
	case DockerConfig:
		return marshalTyped("DockerConfig", v)
	case *KubernetesConfig:
		return marshalTyped("KubernetesConfig", v)
	case KubernetesConfig:
		return marshalTyped("KubernetesConfig", v)
	case *SSHConfig:
		return marshalTyped("SSHConfig", v)
	case SSHConfig:
		return marshalTyped("SSHConfig", v)
	}

	return serializeOpaque(state)
}

func marshalTyped(typeName string, v any) (JournalEntry, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return JournalEntry{}, err
	}
	return JournalEntry{Type: typeName, Data: data}, nil
}

func serializeOpaque(state any) (JournalEntry, error) {
	rv := reflect.ValueOf(state)
	rt := reflect.TypeOf(state)
	typeName := rt.String()

	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return JournalEntry{Type: typeName, Data: json.RawMessage("null")}, nil
		}
		rv = rv.Elem()
		rt = rt.Elem()
		typeName = rt.Name()
		if typeName == "" {
			typeName = rt.String()
		}
	}

	switch rv.Kind() {
	case reflect.Struct:
		safe := filterStructFields(rv, rt)
		data, err := json.Marshal(safe)
		if err != nil {
			return JournalEntry{}, err
		}
		return JournalEntry{Type: typeName, Data: data}, nil
	case reflect.Map:
		safe := filterMapKeys(rv)
		data, err := json.Marshal(safe)
		if err != nil {
			return JournalEntry{}, err
		}
		return JournalEntry{Type: typeName, Data: data}, nil
	default:
		data, err := json.Marshal(state)
		if err != nil {
			return JournalEntry{}, fmt.Errorf("opaque state %s not serializable: %w", typeName, err)
		}
		return JournalEntry{Type: typeName, Data: data}, nil
	}
}

func filterStructFields(rv reflect.Value, rt reflect.Type) map[string]any {
	out := make(map[string]any, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if isSecretName(f.Name) {
			continue
		}
		fv := rv.Field(i)
		if !fv.CanInterface() {
			continue
		}
		out[f.Name] = fv.Interface()
	}
	return out
}

func filterMapKeys(rv reflect.Value) map[string]any {
	out := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		k := fmt.Sprint(key.Interface())
		if isSecretName(k) {
			continue
		}
		val := rv.MapIndex(key)
		if !val.CanInterface() {
			continue
		}
		out[k] = val.Interface()
	}
	return out
}

func isSecretName(name string) bool {
	lower := strings.ToLower(name)
	for _, hint := range secretFieldHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// deserializeState rebuilds a previous-state value from a journal entry.
func deserializeState(entry JournalEntry) (any, error) {
	switch entry.Type {
	case "AWSConfig":
		var v AWSConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "GCPConfig":
		var v GCPConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "AzureConfig":
		var v AzureConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "DockerConfig":
		var v DockerConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "KubernetesConfig":
		var v KubernetesConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "SSHConfig":
		var v SSHConfig
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "nil":
		return nil, nil //nolint:nilnil // serialized nil state
	default:
		var v any
		if err := json.Unmarshal(entry.Data, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
}
