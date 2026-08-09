// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"testing"
	"time"

	"gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryCountsOrganizationProjectRisk(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	resetQueryProjects(t)

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	insertSummaryProject(t, "blocked", orgproject.LifecycleActive, "blocked", 40, now.AddDate(0, 0, 5))
	insertSummaryProject(t, "overdue", orgproject.LifecycleActive, "normal", 60, now.AddDate(0, 0, -2))
	insertSummaryProject(t, "due-soon", orgproject.LifecycleActive, "attention", 80, now.AddDate(0, 0, 3))
	insertSummaryProject(t, "later", orgproject.LifecycleActive, "normal", 20, now.AddDate(0, 0, 20))
	insertSummaryProject(t, "archived", orgproject.LifecycleArchived, "blocked", 100, now.AddDate(0, 0, -4))

	summary, err := Summary(t.Context(), projectLedgerTestSchema(), 3, now)
	require.NoError(t, err)

	assert.EqualValues(t, 1, summary.Blocked)
	assert.EqualValues(t, 1, summary.Overdue)
	assert.EqualValues(t, 2, summary.DueSoon)
	assert.InDelta(t, 50, summary.AverageProgress, 0.01)
	assert.EqualValues(t, 4, summary.Active)
}

func insertSummaryProject(t *testing.T, slug string, lifecycle orgproject.Lifecycle, risk string, progress float64, target time.Time) {
	t.Helper()
	project := insertQueryProject(t, slug, lifecycle, "delivery", progress)
	targetUnix := timeutil.TimeStamp(target.Unix())
	_, err := unittest.GetXORMEngine().Insert(
		&orgproject.FieldValue{ProjectID: project.ID, FieldKey: "risk", ValueText: &risk},
		&orgproject.FieldValue{ProjectID: project.ID, FieldKey: "target_date", ValueTime: &targetUnix},
	)
	require.NoError(t, err)
}
