// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"testing"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryMetricCountsGroupsAndAverages(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	resetQueryProjects(t)
	insertQueryProject(t, "alpha", orgproject_model.LifecycleActive, "delivery", 40)
	insertQueryProject(t, "beta", orgproject_model.LifecycleActive, "delivery", 80)
	insertQueryProject(t, "gamma", orgproject_model.LifecycleActive, "planned", 100)
	insertQueryProject(t, "archived", orgproject_model.LifecycleArchived, "delivery", 100)

	total, err := Metric(t.Context(), queryTestSchema(), MetricOptions{OwnerID: 3, MetricKey: "total"})
	require.NoError(t, err)
	require.Len(t, total.Buckets, 1)
	assert.Nil(t, total.Buckets[0].Bucket)
	assert.InDelta(t, 3, total.Buckets[0].Value, 0)

	grouped, err := Metric(t.Context(), queryTestSchema(), MetricOptions{OwnerID: 3, MetricKey: "by_stage"})
	require.NoError(t, err)
	require.Len(t, grouped.Buckets, 2)
	assert.Equal(t, "delivery", grouped.Buckets[0].Bucket)
	assert.InDelta(t, 2, grouped.Buckets[0].Value, 0)
	assert.Equal(t, "planned", grouped.Buckets[1].Bucket)
	assert.InDelta(t, 1, grouped.Buckets[1].Value, 0)

	average, err := Metric(t.Context(), queryTestSchema(), MetricOptions{OwnerID: 3, MetricKey: "average_progress"})
	require.NoError(t, err)
	require.Len(t, average.Buckets, 1)
	assert.InDelta(t, 220.0/3.0, average.Buckets[0].Value, 0.001)
}

func TestMetricFiltersAreBoundParameters(t *testing.T) {
	ast, err := Parse(queryTestSchema())
	require.NoError(t, err)
	injection := `delivery' UNION SELECT name FROM sqlite_master --`

	compiled, err := CompileMetric(ast, MetricOptions{
		OwnerID: 3, MetricKey: "total", FilterValues: map[string]config.RawValue{"stage": rawQueryValue(t, injection)},
	})
	require.NoError(t, err)
	assert.NotContains(t, compiled.SQL, injection)
	assert.Contains(t, compiled.Args, injection)

	_, err = CompileMetric(ast, MetricOptions{OwnerID: 3, MetricKey: `total; DROP TABLE org_project`})
	assert.ErrorContains(t, err, "unknown organization project metric")
}
