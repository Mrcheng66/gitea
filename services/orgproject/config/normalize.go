// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"sort"
	"strings"

	"gitea.dev/modules/json"
)

func Normalize(input Schema) Schema {
	result := input
	if result.SchemaVersion == 0 {
		result.SchemaVersion = SchemaVersion
	}
	result.Fields = dedupeFields(result.Fields)
	for i := range result.Fields {
		field := &result.Fields[i]
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		field.MigrationStrategy = strings.TrimSpace(field.MigrationStrategy)
		field.Options = dedupeOptions(field.Options)
		field.Default = normalizeJSON(field.Default)
	}
	sort.SliceStable(result.Fields, func(i, j int) bool {
		if result.Fields[i].Order != result.Fields[j].Order {
			return result.Fields[i].Order < result.Fields[j].Order
		}
		return result.Fields[i].Key < result.Fields[j].Key
	})
	result.ListView.Columns = dedupeStrings(result.ListView.Columns)
	result.ListView.Sort = dedupeSort(result.ListView.Sort)
	result.Filters = dedupeFilters(result.Filters)
	result.Metrics = dedupeMetrics(result.Metrics)
	return result
}

func CanonicalJSON(input Schema) ([]byte, error) {
	return json.Marshal(Normalize(input))
}

func normalizeJSON(value RawValue) RawValue {
	if len(value) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return bytes.Clone(value)
	}
	result, err := json.Marshal(decoded)
	if err != nil {
		return bytes.Clone(value)
	}
	return result
}

func dedupeFields(values []Field) []Field {
	seen := make(map[string]struct{}, len(values))
	result := make([]Field, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value.Key)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dedupeOptions(values []Option) []Option {
	seen := make(map[string]struct{}, len(values))
	result := make([]Option, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		if _, ok := seen[value.Key]; ok {
			continue
		}
		seen[value.Key] = struct{}{}
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Order != result[j].Order {
			return result[i].Order < result[j].Order
		}
		return result[i].Key < result[j].Key
	})
	return result
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dedupeSort(values []Sort) []Sort {
	seen := make(map[string]struct{}, len(values))
	result := make([]Sort, 0, len(values))
	for _, value := range values {
		value.FieldKey = strings.TrimSpace(value.FieldKey)
		if _, ok := seen[value.FieldKey]; ok {
			continue
		}
		seen[value.FieldKey] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dedupeFilters(values []Filter) []Filter {
	seen := make(map[string]struct{}, len(values))
	result := make([]Filter, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.FieldKey = strings.TrimSpace(value.FieldKey)
		value.Value = normalizeJSON(value.Value)
		if _, ok := seen[value.Key]; ok {
			continue
		}
		seen[value.Key] = struct{}{}
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func dedupeMetrics(values []Metric) []Metric {
	seen := make(map[string]struct{}, len(values))
	result := make([]Metric, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.FieldKey = strings.TrimSpace(value.FieldKey)
		value.GroupBy = strings.TrimSpace(value.GroupBy)
		if _, ok := seen[value.Key]; ok {
			continue
		}
		seen[value.Key] = struct{}{}
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}
