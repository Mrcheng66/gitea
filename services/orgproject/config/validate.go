// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"gitea.dev/modules/json"
	"gitea.dev/modules/setting"
)

const MigrationRequireEmpty = "require_empty"

var stableKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func Validate(schema Schema) error {
	if schema.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported organization project schema version %d", schema.SchemaVersion)
	}
	maxFields := setting.OrgProject.MaxFields
	if maxFields <= 0 {
		maxFields = 64
	}
	if len(schema.Fields) > maxFields {
		return fmt.Errorf("organization project schema has %d fields, maximum is %d", len(schema.Fields), maxFields)
	}

	fields := make(map[string]Field, len(schema.Fields))
	for _, field := range schema.Fields {
		if !stableKeyPattern.MatchString(field.Key) {
			return fmt.Errorf("invalid field key %q", field.Key)
		}
		if _, exists := fields[field.Key]; exists {
			return fmt.Errorf("duplicate field key %q", field.Key)
		}
		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("field %q has an empty label", field.Key)
		}
		if !isFieldTypeSupported(field.Type) {
			return fmt.Errorf("field %q has unsupported type %q", field.Key, field.Type)
		}
		if err := validateOptions(field); err != nil {
			return err
		}
		if len(field.Default) > 0 {
			if err := validateValue(field, field.Default); err != nil {
				return fmt.Errorf("field %q default: %w", field.Key, err)
			}
		}
		if field.Required && len(field.Default) == 0 && field.MigrationStrategy != MigrationRequireEmpty {
			return fmt.Errorf("required field %q needs a default or %q migration strategy", field.Key, MigrationRequireEmpty)
		}
		if field.MigrationStrategy != "" && field.MigrationStrategy != MigrationRequireEmpty {
			return fmt.Errorf("field %q has unsupported migration strategy %q", field.Key, field.MigrationStrategy)
		}
		fields[field.Key] = field
	}

	if err := validateListView(schema.ListView, fields); err != nil {
		return err
	}
	if err := validateFilters(schema.Filters, fields); err != nil {
		return err
	}
	return validateMetrics(schema.Metrics, fields)
}

// ValidateTransition rejects deletion or incompatible type changes of published fields.
func ValidateTransition(previous, next Schema) error {
	nextFields := make(map[string]Field, len(next.Fields))
	for _, field := range next.Fields {
		nextFields[field.Key] = field
	}
	for _, field := range previous.Fields {
		nextField, ok := nextFields[field.Key]
		if !ok {
			return fmt.Errorf("published field %q must be archived instead of deleted", field.Key)
		}
		if field.Type != nextField.Type {
			return fmt.Errorf("published field %q cannot change type from %q to %q", field.Key, field.Type, nextField.Type)
		}
	}
	return nil
}

func isFieldTypeSupported(fieldType FieldType) bool {
	return slices.Contains([]FieldType{
		FieldTypeShortText, FieldTypeLongText, FieldTypeSingle, FieldTypeMulti,
		FieldTypeInteger, FieldTypeDecimal, FieldTypePercent, FieldTypeDate,
		FieldTypeDateTime, FieldTypeBoolean, FieldTypeMember, FieldTypeMemberArray,
	}, fieldType)
}

func validateOptions(field Field) error {
	isSelect := field.Type == FieldTypeSingle || field.Type == FieldTypeMulti
	if !isSelect && len(field.Options) > 0 {
		return fmt.Errorf("field %q type %q cannot define options", field.Key, field.Type)
	}
	if isSelect && len(field.Options) == 0 {
		return fmt.Errorf("field %q requires at least one option", field.Key)
	}
	maxOptions := setting.OrgProject.MaxEnumOptions
	if maxOptions <= 0 {
		maxOptions = 100
	}
	if len(field.Options) > maxOptions {
		return fmt.Errorf("field %q has %d options, maximum is %d", field.Key, len(field.Options), maxOptions)
	}
	seen := make(map[string]struct{}, len(field.Options))
	for _, option := range field.Options {
		if !stableKeyPattern.MatchString(option.Key) {
			return fmt.Errorf("field %q has invalid option key %q", field.Key, option.Key)
		}
		if _, exists := seen[option.Key]; exists {
			return fmt.Errorf("field %q has duplicate option key %q", field.Key, option.Key)
		}
		if strings.TrimSpace(option.Label) == "" {
			return fmt.Errorf("field %q option %q has an empty label", field.Key, option.Key)
		}
		seen[option.Key] = struct{}{}
	}
	return nil
}

func validateListView(view ListView, fields map[string]Field) error {
	seenColumns := make(map[string]struct{}, len(view.Columns))
	for _, key := range view.Columns {
		if _, exists := seenColumns[key]; exists {
			return fmt.Errorf("list view has duplicate column %q", key)
		}
		if err := validateActiveReference("list column", key, fields); err != nil {
			return err
		}
		seenColumns[key] = struct{}{}
	}
	seenSort := make(map[string]struct{}, len(view.Sort))
	for _, sortRule := range view.Sort {
		if _, exists := seenSort[sortRule.FieldKey]; exists {
			return fmt.Errorf("list view has duplicate sort field %q", sortRule.FieldKey)
		}
		if err := validateActiveReference("sort field", sortRule.FieldKey, fields); err != nil {
			return err
		}
		if sortRule.Direction != SortAscending && sortRule.Direction != SortDescending {
			return fmt.Errorf("sort field %q has unsupported direction %q", sortRule.FieldKey, sortRule.Direction)
		}
		seenSort[sortRule.FieldKey] = struct{}{}
	}
	return nil
}

func validateFilters(filters []Filter, fields map[string]Field) error {
	seen := make(map[string]struct{}, len(filters))
	for _, filter := range filters {
		if !stableKeyPattern.MatchString(filter.Key) {
			return fmt.Errorf("invalid filter key %q", filter.Key)
		}
		if _, exists := seen[filter.Key]; exists {
			return fmt.Errorf("duplicate filter key %q", filter.Key)
		}
		field, ok := fields[filter.FieldKey]
		if !ok || field.Archived {
			return fmt.Errorf("filter %q references unavailable field %q", filter.Key, filter.FieldKey)
		}
		if !operatorAllowed(field.Type, filter.Operator) {
			return fmt.Errorf("filter %q operator %q is not supported for field type %q", filter.Key, filter.Operator, field.Type)
		}
		if len(filter.Value) > 0 {
			if err := validateValue(field, filter.Value); err != nil {
				return fmt.Errorf("filter %q value: %w", filter.Key, err)
			}
		}
		seen[filter.Key] = struct{}{}
	}
	return nil
}

func validateMetrics(metrics []Metric, fields map[string]Field) error {
	seen := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		if !stableKeyPattern.MatchString(metric.Key) {
			return fmt.Errorf("invalid metric key %q", metric.Key)
		}
		if _, exists := seen[metric.Key]; exists {
			return fmt.Errorf("duplicate metric key %q", metric.Key)
		}
		switch metric.Aggregation {
		case MetricCount:
			if metric.FieldKey != "" {
				if err := validateActiveReference("metric field", metric.FieldKey, fields); err != nil {
					return err
				}
			}
		case MetricAverage:
			field, ok := fields[metric.FieldKey]
			if !ok || field.Archived {
				return fmt.Errorf("metric %q references unavailable field %q", metric.Key, metric.FieldKey)
			}
			if !isNumeric(field.Type) {
				return fmt.Errorf("metric %q average requires a numeric field", metric.Key)
			}
		default:
			return fmt.Errorf("metric %q has unsupported aggregation %q", metric.Key, metric.Aggregation)
		}
		if metric.GroupBy != "" {
			field, ok := fields[metric.GroupBy]
			if !ok || field.Archived {
				return fmt.Errorf("metric %q groups by unavailable field %q", metric.Key, metric.GroupBy)
			}
			if field.Type != FieldTypeSingle && field.Type != FieldTypeMember && field.Type != FieldTypeBoolean {
				return fmt.Errorf("metric %q cannot group by field type %q", metric.Key, field.Type)
			}
		}
		seen[metric.Key] = struct{}{}
	}
	return nil
}

func validateActiveReference(kind, key string, fields map[string]Field) error {
	field, ok := fields[key]
	if !ok || field.Archived {
		return fmt.Errorf("%s references unavailable field %q", kind, key)
	}
	return nil
}

func operatorAllowed(fieldType FieldType, operator FilterOperator) bool {
	emptyOperators := []FilterOperator{FilterIsEmpty, FilterIsNotEmpty}
	if slices.Contains(emptyOperators, operator) {
		return true
	}
	switch fieldType {
	case FieldTypeShortText, FieldTypeLongText:
		return slices.Contains([]FilterOperator{FilterEqual, FilterNotEqual, FilterContains}, operator)
	case FieldTypeSingle, FieldTypeBoolean:
		return slices.Contains([]FilterOperator{FilterEqual, FilterNotEqual}, operator)
	case FieldTypeMulti, FieldTypeMemberArray:
		return slices.Contains([]FilterOperator{FilterContains, FilterMember}, operator)
	case FieldTypeInteger, FieldTypeDecimal, FieldTypePercent, FieldTypeDate, FieldTypeDateTime:
		return slices.Contains([]FilterOperator{FilterEqual, FilterNotEqual, FilterGreaterEqual, FilterLessEqual}, operator)
	case FieldTypeMember:
		return slices.Contains([]FilterOperator{FilterEqual, FilterNotEqual, FilterMember}, operator)
	default:
		return false
	}
}

func validateValue(field Field, raw RawValue) error {
	switch field.Type {
	case FieldTypeShortText, FieldTypeLongText:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a string")
		}
	case FieldTypeSingle:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || !hasOption(field, value) {
			return errors.New("must reference an existing option")
		}
	case FieldTypeMulti:
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return errors.New("must be an array of option keys")
		}
		for _, value := range values {
			if !hasOption(field, value) {
				return errors.New("must contain only existing option keys")
			}
		}
	case FieldTypeInteger:
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be an integer")
		}
	case FieldTypeDecimal:
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a number")
		}
	case FieldTypePercent:
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil || value < 0 || value > 100 {
			return errors.New("must be between 0 and 100")
		}
	case FieldTypeDate:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a date string")
		}
		if _, err := time.Parse(time.DateOnly, value); err != nil {
			return errors.New("must use YYYY-MM-DD")
		}
	case FieldTypeDateTime:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a date-time string")
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return errors.New("must use RFC3339")
		}
	case FieldTypeBoolean:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must be a boolean")
		}
	case FieldTypeMember:
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil || value <= 0 {
			return errors.New("must be a positive member ID")
		}
	case FieldTypeMemberArray:
		var values []int64
		if err := json.Unmarshal(raw, &values); err != nil {
			return errors.New("must be an array of member IDs")
		}
		for _, value := range values {
			if value <= 0 {
				return errors.New("must contain only positive member IDs")
			}
		}
	}
	return nil
}

func hasOption(field Field, key string) bool {
	return slices.ContainsFunc(field.Options, func(option Option) bool { return option.Key == key })
}

func isNumeric(fieldType FieldType) bool {
	return fieldType == FieldTypeInteger || fieldType == FieldTypeDecimal || fieldType == FieldTypePercent
}
