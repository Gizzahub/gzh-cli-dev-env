// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package environment

import (
	"errors"
	"fmt"
	"strings"
)

// ErrOptionLikeValue reports a configuration value that an external CLI would
// parse as an option because it starts with '-'.
var ErrOptionLikeValue = errors.New("value must not start with '-'")

// RejectOptionLikeValue returns an error when value starts with '-'.
// Switchers call it before running any command, so a value from an environment
// file, a rollback snapshot or a recovery journal never reaches an option slot.
func RejectOptionLikeValue(field, value string) error {
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("%s %q: %w", field, value, ErrOptionLikeValue)
	}
	return nil
}
