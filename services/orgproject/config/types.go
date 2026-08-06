// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"errors"

	"gitea.dev/modules/json"
)

const SchemaVersion = 1

type RawValue []byte

func (value RawValue) MarshalJSON() ([]byte, error) {
	if len(value) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(value) {
		return nil, errors.New("invalid raw JSON value")
	}
	return bytes.Clone(value), nil
}

func (value *RawValue) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return errors.New("invalid raw JSON value")
	}
	*value = bytes.Clone(data)
	return nil
}

type FieldType string

const (
	FieldTypeShortText   FieldType = "short_text"
	FieldTypeLongText    FieldType = "long_text"
	FieldTypeSingle      FieldType = "single_select"
	FieldTypeMulti       FieldType = "multi_select"
	FieldTypeInteger     FieldType = "integer"
	FieldTypeDecimal     FieldType = "decimal"
	FieldTypePercent     FieldType = "percent"
	FieldTypeDate        FieldType = "date"
	FieldTypeDateTime    FieldType = "date_time"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeMember      FieldType = "member"
	FieldTypeMemberArray FieldType = "member_array"
)

type Option struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type Field struct {
	Key               string    `json:"key"`
	Label             string    `json:"label"`
	Type              FieldType `json:"type"`
	Order             int       `json:"order"`
	Required          bool      `json:"required,omitempty"`
	Archived          bool      `json:"archived,omitempty"`
	Default           RawValue  `json:"default,omitempty"`
	MigrationStrategy string    `json:"migration_strategy,omitempty"`
	Options           []Option  `json:"options,omitempty"`
}

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type Sort struct {
	FieldKey  string        `json:"field_key"`
	Direction SortDirection `json:"direction"`
}

type ListView struct {
	Columns []string `json:"columns"`
	Sort    []Sort   `json:"sort,omitempty"`
}

type FilterOperator string

const (
	FilterEqual        FilterOperator = "eq"
	FilterNotEqual     FilterOperator = "ne"
	FilterContains     FilterOperator = "contains"
	FilterIsEmpty      FilterOperator = "is_empty"
	FilterIsNotEmpty   FilterOperator = "is_not_empty"
	FilterGreaterEqual FilterOperator = "gte"
	FilterLessEqual    FilterOperator = "lte"
	FilterMember       FilterOperator = "member"
)

type Filter struct {
	Key      string         `json:"key"`
	Label    string         `json:"label"`
	FieldKey string         `json:"field_key"`
	Operator FilterOperator `json:"operator"`
	Value    RawValue       `json:"value,omitempty"`
}

type MetricAggregation string

const (
	MetricCount   MetricAggregation = "count"
	MetricAverage MetricAggregation = "average"
)

type Metric struct {
	Key         string            `json:"key"`
	Label       string            `json:"label"`
	Aggregation MetricAggregation `json:"aggregation"`
	FieldKey    string            `json:"field_key,omitempty"`
	GroupBy     string            `json:"group_by,omitempty"`
}

type Schema struct {
	SchemaVersion int      `json:"schema_version"`
	Fields        []Field  `json:"fields"`
	ListView      ListView `json:"list_view"`
	Filters       []Filter `json:"filters"`
	Metrics       []Metric `json:"metrics"`
}
