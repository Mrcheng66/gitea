// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"fmt"

	"gitea.dev/services/orgproject/config"
)

// AST is the validated internal representation used by the query compiler.
type AST struct {
	fields      map[string]field
	columns     []field
	filters     map[string]filter
	defaultSort []sortRule
	metrics     map[string]metric
}

type field struct {
	key         string
	fieldType   config.FieldType
	valueColumn string
}

type filter struct {
	key      string
	field    field
	operator config.FilterOperator
	value    config.RawValue
}

type sortRule struct {
	field     field
	direction config.SortDirection
}

type metric struct {
	definition config.Metric
	field      *field
	groupBy    *field
}

// Parse validates a published schema and converts it to the compiler AST.
func Parse(schema config.Schema) (*AST, error) {
	if err := config.Validate(schema); err != nil {
		return nil, fmt.Errorf("invalid organization project query schema: %w", err)
	}

	ast := &AST{
		fields:  make(map[string]field, len(schema.Fields)),
		filters: make(map[string]filter, len(schema.Filters)),
		metrics: make(map[string]metric, len(schema.Metrics)),
	}
	for _, configured := range schema.Fields {
		if configured.Archived {
			continue
		}
		column, err := columnForType(configured.Type)
		if err != nil {
			return nil, err
		}
		ast.fields[configured.Key] = field{key: configured.Key, fieldType: configured.Type, valueColumn: column}
	}
	for _, key := range schema.ListView.Columns {
		configured, ok := ast.fields[key]
		if !ok {
			return nil, fmt.Errorf("list column references unknown field %q", key)
		}
		ast.columns = append(ast.columns, configured)
	}
	for _, configured := range schema.ListView.Sort {
		queryField, ok := ast.fields[configured.FieldKey]
		if !ok {
			return nil, fmt.Errorf("sort references unknown field %q", configured.FieldKey)
		}
		ast.defaultSort = append(ast.defaultSort, sortRule{field: queryField, direction: configured.Direction})
	}
	for _, configured := range schema.Filters {
		queryField, ok := ast.fields[configured.FieldKey]
		if !ok {
			return nil, fmt.Errorf("filter %q references unknown field %q", configured.Key, configured.FieldKey)
		}
		ast.filters[configured.Key] = filter{
			key: configured.Key, field: queryField, operator: configured.Operator, value: configured.Value,
		}
	}
	for _, configured := range schema.Metrics {
		queryMetric := metric{definition: configured}
		if configured.FieldKey != "" {
			queryField, ok := ast.fields[configured.FieldKey]
			if !ok {
				return nil, fmt.Errorf("metric %q references unknown field %q", configured.Key, configured.FieldKey)
			}
			queryMetric.field = &queryField
		}
		if configured.GroupBy != "" {
			queryField, ok := ast.fields[configured.GroupBy]
			if !ok {
				return nil, fmt.Errorf("metric %q groups by unknown field %q", configured.Key, configured.GroupBy)
			}
			queryMetric.groupBy = &queryField
		}
		ast.metrics[configured.Key] = queryMetric
	}
	return ast, nil
}

func columnForType(fieldType config.FieldType) (string, error) {
	switch fieldType {
	case config.FieldTypeShortText, config.FieldTypeLongText, config.FieldTypeSingle:
		return "value_text", nil
	case config.FieldTypeMulti, config.FieldTypeMemberArray:
		return "value_json", nil
	case config.FieldTypeInteger, config.FieldTypeDecimal, config.FieldTypePercent:
		return "value_number", nil
	case config.FieldTypeDate, config.FieldTypeDateTime:
		return "value_time", nil
	case config.FieldTypeBoolean:
		return "value_bool", nil
	case config.FieldTypeMember:
		return "value_user_id", nil
	default:
		return "", fmt.Errorf("unsupported organization project field type %q", fieldType)
	}
}
