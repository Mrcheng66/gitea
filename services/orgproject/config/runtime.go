// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import "context"

// GetPublishedSchema loads and validates the configuration used by project runtime operations.
func GetPublishedSchema(ctx context.Context, ownerID int64) (Schema, error) {
	version, err := GetPublished(ctx, ownerID)
	if err != nil {
		return Schema{}, err
	}
	schema, err := decodeSchema(version.Payload)
	if err != nil {
		return Schema{}, err
	}
	if err := Validate(schema); err != nil {
		return Schema{}, err
	}
	return schema, nil
}

// ValidateFieldValue validates one JSON value against a configured field.
func ValidateFieldValue(field Field, value RawValue) error {
	return validateValue(field, value)
}
