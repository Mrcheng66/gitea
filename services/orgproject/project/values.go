// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"gitea.dev/models/db"
	org_model "gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/json"
	"gitea.dev/modules/timeutil"
	"gitea.dev/services/orgproject/config"
)

func prepareFieldValues(ctx context.Context, ownerID int64, schema config.Schema, input map[string]config.RawValue) ([]*orgproject_model.FieldValue, error) {
	fields := make(map[string]config.Field, len(schema.Fields))
	for _, field := range schema.Fields {
		fields[field.Key] = field
	}

	errs := ValidationErrors{}
	for key := range input {
		field, exists := fields[key]
		if !exists {
			errs["values."+key] = "is not configured"
		} else if field.Archived {
			errs["values."+key] = "is archived"
		}
	}

	values := make([]*orgproject_model.FieldValue, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		if field.Archived {
			continue
		}
		raw, exists := input[field.Key]
		if !exists && len(field.Default) > 0 {
			raw = field.Default
			exists = true
		}
		if !exists {
			if field.Required {
				errs["values."+field.Key] = "is required"
			}
			continue
		}
		if err := config.ValidateFieldValue(field, raw); err != nil {
			errs["values."+field.Key] = err.Error()
			continue
		}
		value, err := fieldValueFromRaw(ctx, ownerID, field, raw)
		if err != nil {
			errs["values."+field.Key] = err.Error()
			continue
		}
		values = append(values, value)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	sort.Slice(values, func(i, j int) bool { return values[i].FieldKey < values[j].FieldKey })
	return values, nil
}

func fieldValueFromRaw(ctx context.Context, ownerID int64, field config.Field, raw config.RawValue) (*orgproject_model.FieldValue, error) {
	value := &orgproject_model.FieldValue{FieldKey: field.Key}
	switch field.Type {
	case config.FieldTypeShortText, config.FieldTypeLongText, config.FieldTypeSingle:
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		value.ValueText = &decoded
	case config.FieldTypeMulti:
		var decoded []string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		slices.Sort(decoded)
		decoded = slices.Compact(decoded)
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return nil, err
		}
		text := string(encoded)
		value.ValueJSON = &text
	case config.FieldTypeInteger:
		var decoded int64
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		number := float64(decoded)
		value.ValueNumber = &number
	case config.FieldTypeDecimal, config.FieldTypePercent:
		var decoded float64
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		value.ValueNumber = &decoded
	case config.FieldTypeDate, config.FieldTypeDateTime:
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		layout := time.RFC3339
		if field.Type == config.FieldTypeDate {
			layout = time.DateOnly
		}
		parsed, err := time.Parse(layout, decoded)
		if err != nil {
			return nil, err
		}
		timestamp := timeutil.TimeStamp(parsed.Unix())
		value.ValueTime = &timestamp
	case config.FieldTypeBoolean:
		var decoded bool
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		value.ValueBool = &decoded
	case config.FieldTypeMember:
		var decoded int64
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		if err := validateMember(ctx, ownerID, decoded); err != nil {
			return nil, err
		}
		value.ValueUserID = &decoded
	case config.FieldTypeMemberArray:
		var decoded []int64
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		slices.Sort(decoded)
		decoded = slices.Compact(decoded)
		for _, userID := range decoded {
			if err := validateMember(ctx, ownerID, userID); err != nil {
				return nil, err
			}
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return nil, err
		}
		text := string(encoded)
		value.ValueJSON = &text
	default:
		return nil, fmt.Errorf("unsupported field type %q", field.Type)
	}
	return value, nil
}

func validateMember(ctx context.Context, ownerID, userID int64) error {
	isMember, err := org_model.IsOrganizationMember(ctx, ownerID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("user %d is not an organization member", userID)
	}
	return nil
}

func insertFieldValues(ctx context.Context, projectID int64, values []*orgproject_model.FieldValue) error {
	for _, value := range values {
		value.ProjectID = projectID
		if _, err := db.GetEngine(ctx).Insert(value); err != nil {
			return err
		}
	}
	return nil
}
