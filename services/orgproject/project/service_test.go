// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"slices"
	"testing"
	"time"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProjectWritesValuesRepositoriesAndAudit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	opts := CreateOptions{
		OwnerID: 3, Actor: owner, Slug: " Delivery ", Name: " Delivery Project ", Description: "Ships the platform",
		Values: map[string]config.RawValue{
			"stage":     rawValue(t, "development"),
			"followers": rawValue(t, []int64{4, 2, 4}),
			"owner":     rawValue(t, int64(4)),
		},
		Repositories: []RepositoryInput{
			{RepositoryID: 32, Role: orgproject_model.RepositoryRoleRelated},
			{RepositoryID: 3, Role: orgproject_model.RepositoryRolePrimary},
		},
		RequestID: "create-delivery", Source: orgproject_model.ChangeSourceWeb,
	}
	created, err := Create(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, "delivery", created.Slug)
	assert.Equal(t, "Delivery Project", created.Name)
	assert.EqualValues(t, 1, created.Version)

	values := loadValues(t, created.ID)
	assert.Equal(t, "development", *values["stage"].ValueText)
	assert.InDelta(t, 0, *values["progress"].ValueNumber, 0)
	assert.Equal(t, int64(4), *values["owner"].ValueUserID)
	assert.Equal(t, "[2,4]", *values["followers"].ValueJSON)
	assert.Equal(t, "normal", *values["risk"].ValueText)

	links := loadRepositories(t, created.ID)
	require.Len(t, links, 2)
	assert.Equal(t, int64(3), links[0].RepositoryID)
	assert.Equal(t, orgproject_model.RepositoryRolePrimary, links[0].Role)
	assert.Equal(t, int64(32), links[1].RepositoryID)

	change := unittest.AssertExistsAndLoadBean(t, &orgproject_model.ChangeLog{RequestID: opts.RequestID})
	assert.Equal(t, created.ID, change.ProjectID)
	assert.Equal(t, owner.ID, change.ActorID)
	assert.Contains(t, change.ChangedFields, "values.stage")
	assert.Contains(t, change.ChangedFields, "repositories")

	replayed, err := Create(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, created.ID, replayed.ID)
	assert.EqualValues(t, 1, countBeans(t, &orgproject_model.Project{}))
	assert.EqualValues(t, 1, countBeans(t, &orgproject_model.ChangeLog{}))
}

func TestCreateProjectStoresEveryConfiguredFieldType(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	schema := config.Schema{
		SchemaVersion: config.SchemaVersion,
		Fields: []config.Field{
			{Key: "short", Label: "Short", Type: config.FieldTypeShortText},
			{Key: "long", Label: "Long", Type: config.FieldTypeLongText},
			{Key: "single", Label: "Single", Type: config.FieldTypeSingle, Options: []config.Option{{Key: "one", Label: "One"}}},
			{Key: "multi", Label: "Multi", Type: config.FieldTypeMulti, Options: []config.Option{{Key: "one", Label: "One"}, {Key: "two", Label: "Two"}}},
			{Key: "integer", Label: "Integer", Type: config.FieldTypeInteger},
			{Key: "decimal", Label: "Decimal", Type: config.FieldTypeDecimal},
			{Key: "percent", Label: "Percent", Type: config.FieldTypePercent},
			{Key: "date", Label: "Date", Type: config.FieldTypeDate},
			{Key: "date_time", Label: "Date Time", Type: config.FieldTypeDateTime},
			{Key: "boolean", Label: "Boolean", Type: config.FieldTypeBoolean},
			{Key: "member", Label: "Member", Type: config.FieldTypeMember},
			{Key: "members", Label: "Members", Type: config.FieldTypeMemberArray},
		},
	}
	_, pointer, err := config.InitializeDefaultDraft(t.Context(), 3, 2)
	require.NoError(t, err)
	_, err = config.SaveDraft(t.Context(), 3, 2, pointer.Version, schema)
	require.NoError(t, err)
	_, err = config.PublishDraft(t.Context(), 3, 2, pointer.Version+1)
	require.NoError(t, err)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	created, err := Create(t.Context(), CreateOptions{
		OwnerID: 3, Actor: owner, Slug: "typed-values", Name: "Typed Values",
		Values: map[string]config.RawValue{
			"short":     rawValue(t, "short"),
			"long":      rawValue(t, "long"),
			"single":    rawValue(t, "one"),
			"multi":     rawValue(t, []string{"two", "one", "two"}),
			"integer":   rawValue(t, int64(42)),
			"decimal":   rawValue(t, 12.5),
			"percent":   rawValue(t, 75.5),
			"date":      rawValue(t, "2026-08-05"),
			"date_time": rawValue(t, "2026-08-05T08:30:00+08:00"),
			"boolean":   rawValue(t, true),
			"member":    rawValue(t, int64(4)),
			"members":   rawValue(t, []int64{4, 2, 4}),
		},
		RequestID: "typed-values", Source: orgproject_model.ChangeSourceAPI,
	})
	require.NoError(t, err)

	values := loadValues(t, created.ID)
	assert.Equal(t, "short", *values["short"].ValueText)
	assert.Equal(t, "long", *values["long"].ValueText)
	assert.Equal(t, "one", *values["single"].ValueText)
	assert.Equal(t, "[\"one\",\"two\"]", *values["multi"].ValueJSON)
	assert.InDelta(t, 42, *values["integer"].ValueNumber, 0)
	assert.InDelta(t, 12.5, *values["decimal"].ValueNumber, 0)
	assert.InDelta(t, 75.5, *values["percent"].ValueNumber, 0)
	assert.EqualValues(t, time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC).Unix(), *values["date"].ValueTime)
	assert.EqualValues(t, time.Date(2026, time.August, 5, 0, 30, 0, 0, time.UTC).Unix(), *values["date_time"].ValueTime)
	assert.True(t, *values["boolean"].ValueBool)
	assert.Equal(t, int64(4), *values["member"].ValueUserID)
	assert.Equal(t, "[2,4]", *values["members"].ValueJSON)
}

func TestCreateProjectRejectsUnauthorizedAndInvalidInput(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	nonMember := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})

	base := CreateOptions{
		OwnerID: 3, Actor: owner, Slug: "valid-project", Name: "Valid Project",
		RequestID: "validation", Source: orgproject_model.ChangeSourceAPI,
	}

	unauthorized := base
	unauthorized.Actor = member
	unauthorized.RequestID = "member"
	_, err := Create(t.Context(), unauthorized)
	assert.True(t, IsErrForbidden(err))

	invisible := base
	invisible.Actor = nonMember
	invisible.RequestID = "non-member"
	_, err = Create(t.Context(), invisible)
	assert.True(t, IsErrNotFound(err))

	reserved := base
	reserved.Slug = "settings"
	reserved.RequestID = "reserved"
	_, err = Create(t.Context(), reserved)
	assert.True(t, IsValidationErrors(err))

	unknownField := base
	unknownField.Values = map[string]config.RawValue{"formula": rawValue(t, "drop table")}
	unknownField.RequestID = "unknown-field"
	_, err = Create(t.Context(), unknownField)
	assert.True(t, IsValidationErrors(err))

	invalidMember := base
	invalidMember.Values = map[string]config.RawValue{"owner": rawValue(t, int64(5))}
	invalidMember.RequestID = "invalid-member"
	_, err = Create(t.Context(), invalidMember)
	assert.True(t, IsValidationErrors(err))

	crossOrganization := base
	crossOrganization.Repositories = []RepositoryInput{{RepositoryID: 1, Role: orgproject_model.RepositoryRolePrimary}}
	crossOrganization.RequestID = "cross-org"
	_, err = Create(t.Context(), crossOrganization)
	assert.True(t, IsErrRepositoryNotVisible(err))

	assert.EqualValues(t, 0, countBeans(t, &orgproject_model.Project{}))
	assert.EqualValues(t, 0, countBeans(t, &orgproject_model.ChangeLog{}))
}

func TestUpdateProjectUsesOptimisticLockAndRequestIdempotency(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	created := createProject(t, owner, "update-project")

	opts := UpdateOptions{
		OwnerID: 3, ProjectID: created.ID, ExpectedVersion: 1, Actor: owner,
		Slug: "updated-project", Name: "Updated Project", Description: "Updated",
		Values: map[string]config.RawValue{
			"stage":     rawValue(t, "released"),
			"progress":  rawValue(t, 100),
			"followers": rawValue(t, []int64{4, 2}),
		},
		Repositories: []RepositoryInput{{RepositoryID: 32, Role: orgproject_model.RepositoryRolePrimary}},
		RequestID:    "update-project", Source: orgproject_model.ChangeSourceAPI,
	}
	updated, err := Update(t.Context(), opts)
	require.NoError(t, err)
	assert.EqualValues(t, 2, updated.Version)
	assert.Equal(t, "updated-project", updated.Slug)
	assert.Equal(t, "released", *loadValues(t, updated.ID)["stage"].ValueText)

	replayed, err := Update(t.Context(), opts)
	require.NoError(t, err)
	assert.EqualValues(t, 2, replayed.Version)

	stale := opts
	stale.RequestID = "stale-update"
	_, err = Update(t.Context(), stale)
	var conflict ErrConflict
	require.ErrorAs(t, err, &conflict)
	assert.EqualValues(t, 1, conflict.Expected)
	assert.EqualValues(t, 2, conflict.Actual)
	assert.EqualValues(t, 2, countBeans(t, &orgproject_model.ChangeLog{}))
}

func TestUpdateProjectPreservesArchivedValues(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, pointerVersion := publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	created := createProject(t, owner, "archive-field")

	schema := config.DefaultSchema()
	for index := range schema.Fields {
		if schema.Fields[index].Key == "stage" {
			schema.Fields[index].Archived = true
		}
	}
	schema.ListView.Columns = slices.DeleteFunc(schema.ListView.Columns, func(key string) bool { return key == "stage" })
	schema.Filters = slices.DeleteFunc(schema.Filters, func(filter config.Filter) bool { return filter.FieldKey == "stage" })
	schema.Metrics = slices.DeleteFunc(schema.Metrics, func(metric config.Metric) bool { return metric.GroupBy == "stage" })
	_, err := config.SaveDraft(t.Context(), 3, owner.ID, pointerVersion, schema)
	require.NoError(t, err)
	_, err = config.PublishDraft(t.Context(), 3, owner.ID, pointerVersion+1)
	require.NoError(t, err)

	updated, err := Update(t.Context(), UpdateOptions{
		OwnerID: 3, ProjectID: created.ID, ExpectedVersion: 1, Actor: owner,
		Slug: created.Slug, Name: created.Name, Description: created.Description,
		Values:    map[string]config.RawValue{"progress": rawValue(t, 50)},
		RequestID: "preserve-archived", Source: orgproject_model.ChangeSourceWeb,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, updated.Version)
	assert.Equal(t, "planned", *loadValues(t, created.ID)["stage"].ValueText)

	_, err = Update(t.Context(), UpdateOptions{
		OwnerID: 3, ProjectID: created.ID, ExpectedVersion: 2, Actor: owner,
		Slug: created.Slug, Name: created.Name, Values: map[string]config.RawValue{"stage": rawValue(t, "released")},
		RequestID: "write-archived", Source: orgproject_model.ChangeSourceWeb,
	})
	assert.True(t, IsValidationErrors(err))
	assert.EqualValues(t, 2, unittest.AssertExistsAndLoadBean(t, &orgproject_model.Project{ID: created.ID}).Version)
}

func TestArchiveProjectPreservesValuesAndRepositories(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	created := createProject(t, owner, "archive-project")
	valueCount := countProjectBeans(t, &orgproject_model.FieldValue{}, created.ID)
	repositoryCount := countProjectBeans(t, &orgproject_model.Repository{}, created.ID)

	opts := ArchiveOptions{
		OwnerID: 3, ProjectID: created.ID, ExpectedVersion: 1, Actor: owner,
		RequestID: "archive-project", Source: orgproject_model.ChangeSourceWeb,
	}
	archived, err := Archive(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, orgproject_model.LifecycleArchived, archived.Lifecycle)
	assert.EqualValues(t, 2, archived.Version)
	assert.Equal(t, valueCount, countProjectBeans(t, &orgproject_model.FieldValue{}, created.ID))
	assert.Equal(t, repositoryCount, countProjectBeans(t, &orgproject_model.Repository{}, created.ID))

	replayed, err := Archive(t.Context(), opts)
	require.NoError(t, err)
	assert.EqualValues(t, 2, replayed.Version)
	change := unittest.AssertExistsAndLoadBean(t, &orgproject_model.ChangeLog{RequestID: opts.RequestID})
	assert.Equal(t, "[\"lifecycle\"]", change.ChangedFields)
}

func publishDefaultSchema(t *testing.T) (*orgproject_model.ConfigVersion, int64) {
	t.Helper()
	_, pointer, err := config.InitializeDefaultDraft(t.Context(), 3, 2)
	require.NoError(t, err)
	published, err := config.PublishDraft(t.Context(), 3, 2, pointer.Version)
	require.NoError(t, err)
	return published, pointer.Version + 1
}

func createProject(t *testing.T, owner *user_model.User, requestID string) *orgproject_model.Project {
	t.Helper()
	created, err := Create(t.Context(), CreateOptions{
		OwnerID: 3, Actor: owner, Slug: requestID, Name: requestID,
		Repositories: []RepositoryInput{{RepositoryID: 3, Role: orgproject_model.RepositoryRolePrimary}},
		RequestID:    "create-" + requestID, Source: orgproject_model.ChangeSourceWeb,
	})
	require.NoError(t, err)
	return created
}

func rawValue(t *testing.T, value any) config.RawValue {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}

func loadValues(t *testing.T, projectID int64) map[string]*orgproject_model.FieldValue {
	t.Helper()
	values := make([]*orgproject_model.FieldValue, 0)
	require.NoError(t, db.GetEngine(t.Context()).Where("project_id = ?", projectID).Find(&values))
	result := make(map[string]*orgproject_model.FieldValue, len(values))
	for _, value := range values {
		result[value.FieldKey] = value
	}
	return result
}

func loadRepositories(t *testing.T, projectID int64) []*orgproject_model.Repository {
	t.Helper()
	repositories := make([]*orgproject_model.Repository, 0)
	require.NoError(t, db.GetEngine(t.Context()).Where("project_id = ?", projectID).Asc("repository_id").Find(&repositories))
	return repositories
}

func countBeans(t *testing.T, bean any) int64 {
	t.Helper()
	count, err := db.GetEngine(t.Context()).Count(bean)
	require.NoError(t, err)
	return count
}

func countProjectBeans(t *testing.T, bean any, projectID int64) int64 {
	t.Helper()
	count, err := db.GetEngine(t.Context()).Where("project_id = ?", projectID).Count(bean)
	require.NoError(t, err)
	return count
}
