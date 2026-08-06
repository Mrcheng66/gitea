// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"testing"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFiltersSortsAndPaginatesProjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	resetQueryProjects(t)

	alpha := insertQueryProject(t, "alpha", orgproject_model.LifecycleActive, "delivery", 40)
	beta := insertQueryProject(t, "beta", orgproject_model.LifecycleActive, "delivery", 80)
	insertQueryProject(t, "gamma", orgproject_model.LifecycleActive, "planned", 100)
	insertQueryProject(t, "archived", orgproject_model.LifecycleArchived, "delivery", 90)

	first, err := List(t.Context(), queryTestSchema(), ListOptions{
		OwnerID: 3, FilterValues: map[string]config.RawValue{"stage": rawQueryValue(t, "delivery")}, PageSize: 1,
	})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	assert.Equal(t, beta.ID, first.Items[0].Project.ID)
	assert.EqualValues(t, 2, first.Total)
	assert.Equal(t, "delivery", *first.Items[0].Values["stage"].ValueText)
	assert.InDelta(t, 80, *first.Items[0].Values["progress"].ValueNumber, 0)

	second, err := List(t.Context(), queryTestSchema(), ListOptions{
		OwnerID: 3, FilterValues: map[string]config.RawValue{"stage": rawQueryValue(t, "delivery")}, Page: 2, PageSize: 1,
	})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.Equal(t, alpha.ID, second.Items[0].Project.ID)
}

func TestListUsesStableIDSortForEqualValues(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	resetQueryProjects(t)

	firstProject := insertQueryProject(t, "first", orgproject_model.LifecycleActive, "delivery", 50)
	secondProject := insertQueryProject(t, "second", orgproject_model.LifecycleActive, "delivery", 50)

	firstPage, err := List(t.Context(), queryTestSchema(), ListOptions{OwnerID: 3, PageSize: 1})
	require.NoError(t, err)
	secondPage, err := List(t.Context(), queryTestSchema(), ListOptions{OwnerID: 3, Page: 2, PageSize: 1})
	require.NoError(t, err)

	assert.Equal(t, firstProject.ID, firstPage.Items[0].Project.ID)
	assert.Equal(t, secondProject.ID, secondPage.Items[0].Project.ID)
}

func TestSQLiteJSON1IsAvailableForProjectQueries(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	var value int64
	has, err := db.GetEngine(t.Context()).SQL(`SELECT value FROM json_each('[42]')`).Get(&value)
	require.NoError(t, err)
	assert.True(t, has)
	assert.EqualValues(t, 42, value)
}

func resetQueryProjects(t *testing.T) {
	t.Helper()
	_, err := db.GetEngine(t.Context()).Exec("DELETE FROM org_project_field_value")
	require.NoError(t, err)
	_, err = db.GetEngine(t.Context()).Exec("DELETE FROM org_project")
	require.NoError(t, err)
}

func insertQueryProject(t *testing.T, slug string, lifecycle orgproject_model.Lifecycle, stage string, progress float64) *orgproject_model.Project {
	t.Helper()
	project := &orgproject_model.Project{
		OwnerID: 3, Slug: slug, Name: slug, Lifecycle: lifecycle, Version: 1, CreatedBy: 2,
	}
	_, err := db.GetEngine(t.Context()).Insert(project)
	require.NoError(t, err)
	_, err = db.GetEngine(t.Context()).Insert(
		&orgproject_model.FieldValue{ProjectID: project.ID, FieldKey: "stage", ValueText: &stage},
		&orgproject_model.FieldValue{ProjectID: project.ID, FieldKey: "progress", ValueNumber: &progress},
	)
	require.NoError(t, err)
	return project
}

func TestListSQLFragmentsCannotChangeDatabaseStructure(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	resetQueryProjects(t)
	insertQueryProject(t, "safe", orgproject_model.LifecycleActive, "delivery", 50)
	injection := `delivery'); DROP TABLE org_project; --`

	result, err := List(t.Context(), queryTestSchema(), ListOptions{
		OwnerID: 3, FilterValues: map[string]config.RawValue{"stage": rawQueryValue(t, injection)},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Items)

	exists, err := db.GetEngine(t.Context()).IsTableExist(new(orgproject_model.Project))
	require.NoError(t, err)
	assert.True(t, exists)
}
