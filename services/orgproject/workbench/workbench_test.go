// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package workbench

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDueState(t *testing.T) {
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	overdue, dueSoon := dueState("2026-08-08", today)
	assert.True(t, overdue)
	assert.False(t, dueSoon)

	overdue, dueSoon = dueState("2026-08-12", today)
	assert.False(t, overdue)
	assert.True(t, dueSoon)

	overdue, dueSoon = dueState("2026-08-16", today)
	assert.False(t, overdue)
	assert.False(t, dueSoon)
}

func TestCompareProjectsPrioritizesAttentionThenTarget(t *testing.T) {
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	blocked := Project{RiskKey: "blocked", TargetDate: "2026-08-20", UpdatedAt: now}
	overdue := Project{Overdue: true, TargetDate: "2026-08-08", UpdatedAt: now}
	normalSoon := Project{TargetDate: "2026-08-10", UpdatedAt: now}
	normalLater := Project{TargetDate: "2026-08-18", UpdatedAt: now.Add(time.Hour)}

	assert.Negative(t, compareProjects(blocked, overdue))
	assert.Negative(t, compareProjects(overdue, normalSoon))
	assert.Negative(t, compareProjects(normalSoon, normalLater))
}
