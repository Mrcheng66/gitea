// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"testing"

	"gitea.dev/modules/json"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileListUsesBoundValuesAndGeneratedAliases(t *testing.T) {
	ast, err := Parse(queryTestSchema())
	require.NoError(t, err)
	injection := `delivery'); DROP TABLE org_project; --`

	compiled, err := CompileList(ast, ListOptions{
		OwnerID:      3,
		FilterValues: map[string]config.RawValue{"stage": rawQueryValue(t, injection)},
		Sort:         []config.Sort{{FieldKey: "progress", Direction: config.SortDescending}},
		Page:         2,
		PageSize:     10,
	})
	require.NoError(t, err)

	assert.Contains(t, compiled.Data.SQL, "org_project_field_value fv_0")
	assert.Contains(t, compiled.Data.SQL, "org_project_field_value fv_1")
	assert.Contains(t, compiled.Data.SQL, "fv_1.value_number DESC, org_project.id ASC")
	assert.NotContains(t, compiled.Data.SQL, "stage")
	assert.NotContains(t, compiled.Data.SQL, injection)
	assert.Equal(t, []any{"stage", "progress", int64(3), "active", injection, 10, 10}, compiled.Data.Args)
	assert.Equal(t, []any{"stage", "progress", int64(3), "active", injection}, compiled.Count.Args)
}

func TestCompileListRejectsUnknownSortAndFilter(t *testing.T) {
	ast, err := Parse(queryTestSchema())
	require.NoError(t, err)

	_, err = CompileList(ast, ListOptions{OwnerID: 3, Sort: []config.Sort{{FieldKey: `name; DROP TABLE org_project`, Direction: config.SortAscending}}})
	assert.ErrorContains(t, err, "unknown sort field")

	_, err = CompileList(ast, ListOptions{OwnerID: 3, FilterValues: map[string]config.RawValue{
		`stage) OR 1=1 --`: rawQueryValue(t, "delivery"),
	}})
	assert.ErrorContains(t, err, "unknown organization project filter")
}

func TestCompileListUsesJSONEachForArrayMembership(t *testing.T) {
	schema := queryTestSchema()
	schema.Fields = append(schema.Fields, config.Field{Key: "followers", Label: "Followers", Type: config.FieldTypeMemberArray})
	schema.Filters = append(schema.Filters, config.Filter{Key: "follower", Label: "Follower", FieldKey: "followers", Operator: config.FilterMember})
	ast, err := Parse(schema)
	require.NoError(t, err)

	compiled, err := CompileList(ast, ListOptions{OwnerID: 3, FilterValues: map[string]config.RawValue{
		"follower": rawQueryValue(t, int64(42)),
	}})
	require.NoError(t, err)
	assert.Contains(t, compiled.Data.SQL, "json_each(fv_0.value_json)")
	assert.NotContains(t, compiled.Data.SQL, "42")
	assert.Contains(t, compiled.Data.Args, int64(42))
}

func TestParseRejectsUnknownQueryConfiguration(t *testing.T) {
	schema := queryTestSchema()
	schema.Filters[0].Operator = config.FilterOperator("raw_sql")
	_, err := Parse(schema)
	assert.ErrorContains(t, err, "not supported")

	schema = queryTestSchema()
	schema.Metrics[0].Aggregation = config.MetricAggregation("sum")
	_, err = Parse(schema)
	assert.ErrorContains(t, err, "unsupported aggregation")
}

func queryTestSchema() config.Schema {
	return config.Schema{
		SchemaVersion: config.SchemaVersion,
		Fields: []config.Field{
			{Key: "stage", Label: "Stage", Type: config.FieldTypeSingle, Options: []config.Option{
				{Key: "planned", Label: "Planned"}, {Key: "delivery", Label: "Delivery"},
			}},
			{Key: "progress", Label: "Progress", Type: config.FieldTypePercent},
		},
		ListView: config.ListView{
			Columns: []string{"stage", "progress"},
			Sort:    []config.Sort{{FieldKey: "progress", Direction: config.SortDescending}},
		},
		Filters: []config.Filter{{Key: "stage", Label: "Stage", FieldKey: "stage", Operator: config.FilterEqual}},
		Metrics: []config.Metric{
			{Key: "total", Label: "Total", Aggregation: config.MetricCount},
			{Key: "by_stage", Label: "By stage", Aggregation: config.MetricCount, GroupBy: "stage"},
			{Key: "average_progress", Label: "Average progress", Aggregation: config.MetricAverage, FieldKey: "progress"},
		},
	}
}

func rawQueryValue(t *testing.T, value any) config.RawValue {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}
