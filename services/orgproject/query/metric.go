// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gitea.dev/models/db"
	"gitea.dev/services/orgproject/config"
)

// MetricBucket contains one aggregate result. Bucket is nil for ungrouped metrics.
type MetricBucket struct {
	Bucket any
	Value  float64
}

// MetricResult contains the configured metric output.
type MetricResult struct {
	Key     string
	Buckets []MetricBucket
}

// CompileMetric builds a parameterized SQLite metric query.
func CompileMetric(ast *AST, opts MetricOptions) (CompiledQuery, error) {
	if ast == nil {
		return CompiledQuery{}, errors.New("organization project query AST is required")
	}
	if opts.OwnerID <= 0 {
		return CompiledQuery{}, errors.New("organization owner ID must be positive")
	}
	configured, ok := ast.metrics[opts.MetricKey]
	if !ok {
		return CompiledQuery{}, fmt.Errorf("unknown organization project metric %q", opts.MetricKey)
	}

	queryCompiler := newCompiler(ast)
	where := []string{"org_project.owner_id = ?", "org_project.lifecycle = ?"}
	whereArgs := []any{opts.OwnerID, "active"}
	filterSQL, filterArgs, err := queryCompiler.compileFilters(opts.FilterValues)
	if err != nil {
		return CompiledQuery{}, err
	}
	where = append(where, filterSQL...)
	whereArgs = append(whereArgs, filterArgs...)

	aggregate, err := compileAggregate(queryCompiler, configured)
	if err != nil {
		return CompiledQuery{}, err
	}
	selectSQL := aggregate + " AS value"
	groupSQL := ""
	orderSQL := ""
	limitSQL := ""
	if configured.groupBy != nil {
		alias := queryCompiler.join(*configured.groupBy)
		column := alias + "." + configured.groupBy.valueColumn
		selectSQL = column + " AS bucket, " + selectSQL
		where = append(where, alias+".id IS NOT NULL")
		groupSQL = " GROUP BY " + column
		orderSQL = " ORDER BY " + column + " ASC"
		limitSQL = " LIMIT ?"
		whereArgs = append(whereArgs, maxMetricBuckets+1)
	}

	args := append(slices.Clone(queryCompiler.joinArgs), whereArgs...)
	sql := "SELECT " + selectSQL + " FROM org_project " + strings.Join(queryCompiler.joins, " ") +
		" WHERE " + strings.Join(where, " AND ") + groupSQL + orderSQL + limitSQL
	return CompiledQuery{SQL: sql, Args: args}, nil
}

func compileAggregate(queryCompiler *compiler, configured metric) (string, error) {
	switch configured.definition.Aggregation {
	case config.MetricCount:
		if configured.field == nil {
			return "COUNT(DISTINCT org_project.id)", nil
		}
		alias := queryCompiler.join(*configured.field)
		return "COUNT(DISTINCT CASE WHEN " + alias + ".id IS NOT NULL THEN org_project.id END)", nil
	case config.MetricAverage:
		if configured.field == nil || configured.field.valueColumn != "value_number" {
			return "", fmt.Errorf("metric %q average requires a numeric field", configured.definition.Key)
		}
		alias := queryCompiler.join(*configured.field)
		return "AVG(" + alias + ".value_number)", nil
	default:
		return "", fmt.Errorf("unsupported metric aggregation %q", configured.definition.Aggregation)
	}
}

// Metric executes one configured metric.
func Metric(ctx context.Context, schema config.Schema, opts MetricOptions) (*MetricResult, error) {
	ast, err := Parse(schema)
	if err != nil {
		return nil, err
	}
	compiled, err := CompileMetric(ast, opts)
	if err != nil {
		return nil, err
	}
	queryArgs := append([]any{compiled.SQL}, compiled.Args...)
	rows, err := db.GetEngine(ctx).QueryInterface(queryArgs...)
	if err != nil {
		return nil, err
	}
	if len(rows) > maxMetricBuckets {
		return nil, fmt.Errorf("metric %q exceeds the %d bucket limit", opts.MetricKey, maxMetricBuckets)
	}
	configured := ast.metrics[opts.MetricKey]
	buckets := make([]MetricBucket, 0, len(rows))
	for _, row := range rows {
		value, err := numericValue(row["value"])
		if err != nil {
			return nil, fmt.Errorf("metric %q: %w", opts.MetricKey, err)
		}
		var bucket any
		if configured.groupBy != nil {
			bucket, err = normalizeBucket(row["bucket"], configured.groupBy.fieldType)
			if err != nil {
				return nil, fmt.Errorf("metric %q: %w", opts.MetricKey, err)
			}
		}
		buckets = append(buckets, MetricBucket{Bucket: bucket, Value: value})
	}
	return &MetricResult{Key: opts.MetricKey, Buckets: buckets}, nil
}

func numericValue(value any) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		number, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, err
		}
		return number, nil
	case []byte:
		number, err := strconv.ParseFloat(string(typed), 64)
		if err != nil {
			return 0, err
		}
		return number, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unexpected aggregate value type %T", value)
	}
}

func normalizeBucket(value any, fieldType config.FieldType) (any, error) {
	if value == nil {
		return nil, errors.New("metric bucket must not be null")
	}
	switch fieldType {
	case config.FieldTypeSingle:
		switch typed := value.(type) {
		case string:
			return typed, nil
		case []byte:
			return string(typed), nil
		}
	case config.FieldTypeMember:
		switch typed := value.(type) {
		case int64:
			return typed, nil
		case int:
			return int64(typed), nil
		case string:
			return strconv.ParseInt(typed, 10, 64)
		case []byte:
			return strconv.ParseInt(string(typed), 10, 64)
		}
	case config.FieldTypeBoolean:
		switch typed := value.(type) {
		case bool:
			return typed, nil
		case int64:
			return typed != 0, nil
		case int:
			return typed != 0, nil
		case string:
			return typed != "0", nil
		case []byte:
			return string(typed) != "0", nil
		}
	}
	return nil, fmt.Errorf("unexpected %q metric bucket type %T", fieldType, value)
}
