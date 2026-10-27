// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package aws

import (
	"fmt"
	"strings"

	"github.com/gizzahub/gzh-cli-dev-env/pkg/environment"
)

type regionDocument struct {
	lines        []string
	index        int
	sectionStart int
	sectionEnd   int
	state        environment.AWSRegionState
}

func regionSection(profile string) string {
	if profile == DefaultProfile {
		return DefaultProfile
	}
	return "profile " + profile
}

// parseRegion은 모호한 INI 문법을 추측하지 않고 복구 전에 명시적으로 거절한다.
func parseRegion(data []byte, profile string) (regionDocument, error) {
	doc := regionDocument{lines: strings.SplitAfter(string(data), "\n"), index: -1, sectionStart: -1, sectionEnd: -1}
	section := ""
	seen := map[string]bool{}
	target := regionSection(profile)
	for i, line := range doc.lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			if !strings.HasSuffix(trimmed, "]") || strings.Count(trimmed, "[") != 1 || strings.Count(trimmed, "]") != 1 {
				return doc, fmt.Errorf("unsupported AWS config section syntax")
			}
			if section == target {
				doc.sectionEnd = i
			}
			var err error
			section, err = canonicalSection(trimmed[1 : len(trimmed)-1])
			if err != nil {
				return doc, err
			}
			if section == "" || seen[section] {
				return doc, fmt.Errorf("duplicate/empty AWS config section")
			}
			seen[section] = true
			if section == target {
				doc.sectionStart = i
				doc.sectionEnd = len(doc.lines)
			}
			continue
		}
		if err := doc.readKey(line, trimmed, section, target, i); err != nil {
			return doc, err
		}
	}
	return doc, nil
}

func (doc *regionDocument) readKey(line, trimmed, section, target string, i int) error {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return fmt.Errorf("unsupported indented AWS config key/continuation")
	}
	delimiter := strings.IndexAny(trimmed, "=:")
	if delimiter < 1 || section == "" {
		return fmt.Errorf("unsupported AWS config continuation/key syntax")
	}
	key := strings.TrimSpace(trimmed[:delimiter])
	if section == target && strings.EqualFold(key, awsCLIRegion) && key != awsCLIRegion {
		return fmt.Errorf("unsupported AWS region key case")
	}
	if section == target && strings.TrimSpace(trimmed[:delimiter]) == awsCLIRegion {
		if doc.index >= 0 {
			return fmt.Errorf("duplicate AWS region key")
		}
		if err := validRegionValue(strings.TrimSpace(trimmed[delimiter+1:])); err != nil {
			return err
		}
		doc.index = i
		doc.state = environment.AWSRegionState{Present: true, Value: strings.TrimSpace(trimmed[delimiter+1:])}
	}
	return nil
}

func readRegion(data []byte, profile string) (environment.AWSRegionState, error) {
	doc, err := parseRegion(data, profile)
	return doc.state, err
}

func editRegion(data []byte, profile string, value environment.AWSRegionState) ([]byte, error) {
	doc, err := parseRegion(data, profile)
	if err != nil {
		return nil, err
	}
	if doc.index >= 0 {
		if value.Present {
			doc.lines[doc.index] = "region = " + value.Value + "\n"
		} else {
			doc.lines[doc.index] = ""
		}
		return []byte(strings.Join(doc.lines, "")), nil
	}
	if !value.Present {
		return data, nil
	}
	text := string(data)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if doc.sectionStart < 0 {
		return []byte(text + "[" + regionSection(profile) + "]\nregion = " + value.Value + "\n"), nil
	}
	position := doc.sectionEnd
	prefix := strings.Join(doc.lines[:position], "")
	if !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	return []byte(prefix + "region = " + value.Value + "\n" + strings.Join(doc.lines[position:], "")), nil
}

func hasConfigValues(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(line)
		if s != "" && !strings.HasPrefix(s, "#") && !strings.HasPrefix(s, ";") && !strings.HasPrefix(s, "[") {
			return true
		}
	}
	return false
}

// canonicalSection은 AWS profile 구분자의 공백을 임의로 정규화하지 않는다.
func canonicalSection(section string) (string, error) {
	if section != strings.TrimSpace(section) || strings.ContainsAny(section, "\"'\\") {
		return "", fmt.Errorf("unsupported AWS section whitespace/quoting")
	}
	if strings.HasPrefix(section, "profile") {
		if !strings.HasPrefix(section, "profile ") {
			return "", fmt.Errorf("unsupported AWS profile section separator")
		}
		profile := strings.TrimPrefix(section, "profile ")
		if profile == "" {
			return "", fmt.Errorf("empty AWS profile section")
		}
		if err := validProfile(profile); err != nil {
			return "", err
		}
	}
	return section, nil
}
