// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgProjectWorkbenchImportIsIdempotent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	databasePath := createWorkbenchDatabase(t)
	options := Options{
		DatabasePath: databasePath,
		Organization: "org3",
		Actor:        "user2",
		EditorTeams:  []string{"Owners"},
	}

	preflight, err := Preflight(t.Context(), options)
	require.NoError(t, err)
	assert.Equal(t, ReportSummary{
		Profiles: 1, Followers: 2, Audits: 2, ProjectsImport: 1, AuditsImport: 2, Warnings: 1,
	}, preflight.Summary)

	first, err := Import(t.Context(), options)
	require.NoError(t, err)
	assert.Equal(t, 1, first.Summary.ProjectsImport)
	assert.Equal(t, 2, first.Summary.AuditsImport)
	assert.Equal(t, 0, first.Summary.Blocked)

	project := unittest.AssertExistsAndLoadBean(t, &orgproject_model.Project{OwnerID: 3, Slug: "repo3"})
	assert.Equal(t, int64(3), project.Version)
	unittest.AssertExistsAndLoadBean(t, &orgproject_model.Repository{
		ProjectID: project.ID, RepositoryID: 3, Role: orgproject_model.RepositoryRolePrimary,
	})
	assert.Equal(t, 8, unittest.GetCount(t, &orgproject_model.FieldValue{ProjectID: project.ID}))
	assert.Equal(t, 3, unittest.GetCount(t, &orgproject_model.ChangeLog{ProjectID: project.ID}))
	pointer := &orgproject_model.ConfigPointer{OwnerID: 3}
	has, err := db.GetEngine(t.Context()).Get(pointer)
	require.NoError(t, err)
	require.True(t, has)
	assert.NotZero(t, pointer.PublishedVersionID)
	assert.Equal(t, 1, unittest.GetCount(t, &orgproject_model.EditorTeam{OwnerID: 3}))

	fallbackAudit := unittest.AssertExistsAndLoadBean(t, &orgproject_model.ChangeLog{RequestID: auditRequestID(2)})
	assert.Equal(t, int64(2), fallbackAudit.ActorID)
	var snapshot legacyAuditSnapshot
	require.NoError(t, json.Unmarshal([]byte(fallbackAudit.AfterValue), &snapshot))
	assert.Equal(t, "legacy-request-2", snapshot.LegacyRequestID)

	second, err := Import(t.Context(), options)
	require.NoError(t, err)
	assert.Equal(t, 0, second.Summary.ProjectsImport)
	assert.Equal(t, 1, second.Summary.ProjectsSkip)
	assert.Equal(t, 0, second.Summary.AuditsImport)
	assert.Equal(t, 2, second.Summary.AuditsSkip)
	assert.Equal(t, 1, unittest.GetCount(t, &orgproject_model.Project{OwnerID: 3}))
	assert.Equal(t, 3, unittest.GetCount(t, &orgproject_model.ChangeLog{ProjectID: project.ID}))
}

func TestOrgProjectWorkbenchPreflightBlocksInvalidSource(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	databasePath := createWorkbenchDatabase(t)
	sourceDB, err := sql.Open(sqliteDriverName, databasePath)
	require.NoError(t, err)
	_, err = sourceDB.Exec("DELETE FROM project_followers")
	require.NoError(t, err)
	_, err = sourceDB.Exec("UPDATE project_profiles SET repo_id = 999 WHERE repo_id = 3")
	require.NoError(t, err)
	_, err = sourceDB.Exec("UPDATE project_audit_events SET after_value = 'broken', repo_id = 999")
	require.NoError(t, err)
	require.NoError(t, sourceDB.Close())

	report, err := Preflight(t.Context(), Options{
		DatabasePath: databasePath, Organization: "org3", Actor: "user2", EditorTeams: []string{"Owners"},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, report.Summary.Blocked, 2)

	_, err = Import(t.Context(), Options{
		DatabasePath: databasePath, Organization: "org3", Actor: "user2", EditorTeams: []string{"Owners"},
	})
	var blocked ErrBlocked
	assert.ErrorAs(t, err, &blocked)
	assert.Equal(t, 0, unittest.GetCount(t, &orgproject_model.Project{OwnerID: 3}))
}

func TestOrgProjectWorkbenchPreflightResolvesSlugConflict(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, db.Insert(t.Context(), &orgproject_model.Project{
		OwnerID: 3, Slug: "repo3", Name: "Existing", Lifecycle: orgproject_model.LifecycleActive, Version: 1, CreatedBy: 2,
	}))

	report, err := Preflight(t.Context(), Options{
		DatabasePath: createWorkbenchDatabase(t), Organization: "org3", Actor: "user2", EditorTeams: []string{"Owners"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, report.Summary.Blocked)
	assert.Equal(t, 2, report.Summary.Warnings)
	assert.Contains(t, report.Items, ReportItem{
		Kind: "profile", LegacyID: 3, RepoID: 3, Action: "import", Slug: "repo3-3",
	})
}

func TestOrgProjectWorkbenchImportRequiresOrganizationOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, err := Preflight(t.Context(), Options{
		DatabasePath: createWorkbenchDatabase(t), Organization: "org3", Actor: "user4", EditorTeams: []string{"Owners"},
	})
	assert.ErrorContains(t, err, "must be an owner")
}

func createWorkbenchDatabase(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workbench.db")
	database, err := sql.Open(sqliteDriverName, path)
	require.NoError(t, err)
	script, err := os.ReadFile(filepath.Join("..", "..", "..", "ops", "migration", "testdata", "workbench-v1.sql"))
	require.NoError(t, err)
	_, err = database.Exec(string(script))
	require.NoError(t, err)
	require.NoError(t, database.Close())
	return path
}

func TestWriteReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reports", "preflight.json")
	report := &Report{Mode: "preflight", Summary: ReportSummary{Profiles: 1}}
	require.NoError(t, WriteReport(path, report))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"profiles": 1`)
	assert.FileExists(t, path)
}
