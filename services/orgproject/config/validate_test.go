// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSchema(t *testing.T) {
	schema := DefaultSchema()
	require.NoError(t, Validate(schema))
	assert.Equal(t, []string{"stage", "progress", "owner", "followers", "start_date", "target_date", "risk", "summary"}, fieldKeys(schema.Fields))

	stage := schema.Fields[0]
	assert.Equal(t, []string{"planned", "development", "testing", "released", "paused"}, optionKeys(stage.Options))
	risk := schema.Fields[6]
	assert.Equal(t, []string{"normal", "attention", "blocked"}, optionKeys(risk.Options))
}

func TestNormalizeProducesStableJSON(t *testing.T) {
	first := DefaultSchema()
	fields := append([]Field(nil), first.Fields...)
	first.Fields = append([]Field{fields[1], fields[0]}, fields[2:]...)
	first.Filters = append([]Filter{first.Filters[2]}, first.Filters[:2]...)
	first.Metrics = append([]Metric{first.Metrics[2]}, first.Metrics[:2]...)
	first.Fields[1].Options = append(first.Fields[1].Options, first.Fields[1].Options[0])

	second := DefaultSchema()
	firstJSON, err := CanonicalJSON(first)
	require.NoError(t, err)
	secondJSON, err := CanonicalJSON(second)
	require.NoError(t, err)
	assert.JSONEq(t, string(secondJSON), string(firstJSON))
	assert.True(t, bytes.Equal(secondJSON, firstJSON))
}

func TestValidateRejectsInvalidSchema(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Schema)
		want   string
	}{
		{name: "schema version", mutate: func(s *Schema) { s.SchemaVersion = 2 }, want: "unsupported"},
		{name: "field key", mutate: func(s *Schema) { s.Fields[0].Key = "Stage Value" }, want: "invalid field key"},
		{name: "duplicate field", mutate: func(s *Schema) { s.Fields = append(s.Fields, s.Fields[0]) }, want: "duplicate field"},
		{name: "empty label", mutate: func(s *Schema) { s.Fields[0].Label = " " }, want: "empty label"},
		{name: "unsupported type", mutate: func(s *Schema) { s.Fields[0].Type = "formula" }, want: "unsupported type"},
		{name: "options on text", mutate: func(s *Schema) { s.Fields[7].Options = []Option{{Key: "x", Label: "X"}} }, want: "cannot define options"},
		{name: "missing select options", mutate: func(s *Schema) { s.Fields[0].Options = nil }, want: "requires at least one option"},
		{name: "duplicate option", mutate: func(s *Schema) { s.Fields[0].Options = append(s.Fields[0].Options, s.Fields[0].Options[0]) }, want: "duplicate option"},
		{name: "invalid default", mutate: func(s *Schema) { s.Fields[1].Default = RawValue(`101`) }, want: "between 0 and 100"},
		{name: "required without strategy", mutate: func(s *Schema) { s.Fields[0].Default = nil }, want: "needs a default"},
		{name: "archived list column", mutate: func(s *Schema) { s.Fields[0].Archived = true }, want: "list column references unavailable"},
		{name: "duplicate list column", mutate: func(s *Schema) { s.ListView.Columns = append(s.ListView.Columns, s.ListView.Columns[0]) }, want: "duplicate column"},
		{name: "bad sort direction", mutate: func(s *Schema) { s.ListView.Sort[0].Direction = "sideways" }, want: "unsupported direction"},
		{name: "bad filter reference", mutate: func(s *Schema) { s.Filters[0].FieldKey = "missing" }, want: "references unavailable"},
		{name: "bad filter operator", mutate: func(s *Schema) { s.Filters[0].Operator = FilterGreaterEqual }, want: "not supported"},
		{name: "average nonnumeric", mutate: func(s *Schema) {
			s.Metrics[0] = Metric{Key: "avg", Label: "Average", Aggregation: MetricAverage, FieldKey: "stage"}
		}, want: "numeric field"},
		{name: "bad group type", mutate: func(s *Schema) { s.Metrics[0].GroupBy = "summary" }, want: "cannot group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := DefaultSchema()
			tt.mutate(&schema)
			assert.ErrorContains(t, Validate(schema), tt.want)
		})
	}
}

func TestValidateFieldDefaults(t *testing.T) {
	tests := []struct {
		fieldType FieldType
		value     string
		valid     bool
	}{
		{FieldTypeShortText, `"text"`, true},
		{FieldTypeInteger, `12`, true},
		{FieldTypeInteger, `12.5`, false},
		{FieldTypeDecimal, `12.5`, true},
		{FieldTypePercent, `0`, true},
		{FieldTypePercent, `-1`, false},
		{FieldTypeDate, `"2026-08-05"`, true},
		{FieldTypeDate, `"08/05/2026"`, false},
		{FieldTypeDateTime, `"2026-08-05T12:00:00Z"`, true},
		{FieldTypeBoolean, `true`, true},
		{FieldTypeMember, `2`, true},
		{FieldTypeMember, `0`, false},
		{FieldTypeMemberArray, `[2,3]`, true},
		{FieldTypeMemberArray, `[0]`, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.fieldType)+tt.value, func(t *testing.T) {
			field := Field{Key: "value", Label: "Value", Type: tt.fieldType, Default: RawValue(tt.value)}
			err := validateValue(field, field.Default)
			assert.Equal(t, tt.valid, err == nil, err)
		})
	}
}

func TestValidateTransition(t *testing.T) {
	previous := DefaultSchema()

	deleted := DefaultSchema()
	deleted.Fields = deleted.Fields[1:]
	assert.ErrorContains(t, ValidateTransition(previous, deleted), "archived instead of deleted")

	changedType := DefaultSchema()
	changedType.Fields[0].Type = FieldTypeShortText
	assert.ErrorContains(t, ValidateTransition(previous, changedType), "cannot change type")

	archived := DefaultSchema()
	archived.Fields[0].Archived = true
	assert.NoError(t, ValidateTransition(previous, archived))
}

func fieldKeys(fields []Field) []string {
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		result = append(result, field.Key)
	}
	return result
}

func optionKeys(options []Option) []string {
	result := make([]string, 0, len(options))
	for _, option := range options {
		result = append(result, option.Key)
	}
	return result
}
