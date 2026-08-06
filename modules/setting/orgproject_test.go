// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadOrgProjectTestConfig(t *testing.T, ini string) error {
	t.Helper()
	cfg, err := NewConfigProviderFromData(ini)
	require.NoError(t, err)
	loadDBSetting(cfg)
	return loadOrgProjectFrom(cfg)
}

func TestOrgProjectDefaults(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	require.NoError(t, loadOrgProjectTestConfig(t, "[database]\nDB_TYPE = sqlite3"))
	assert.True(t, OrgProject.Enabled)
	assert.Equal(t, 25, OrgProject.DefaultPageSize)
	assert.Equal(t, 100, OrgProject.MaxPageSize)
	assert.Equal(t, 64, OrgProject.MaxFields)
	assert.Equal(t, 100, OrgProject.MaxEnumOptions)
	assert.Equal(t, 20, OrgProject.MaxRepositoriesPerProject)
}

func TestOrgProjectBoundaries(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	require.NoError(t, loadOrgProjectTestConfig(t, `
[database]
DB_TYPE = sqlite3
[org_project]
DEFAULT_PAGE_SIZE = 1
MAX_PAGE_SIZE = 1
MAX_FIELDS = 1
MAX_ENUM_OPTIONS = 1
MAX_REPOSITORIES_PER_PROJECT = 1
`))
	assert.Equal(t, 1, OrgProject.DefaultPageSize)
	assert.Equal(t, 1, OrgProject.MaxPageSize)
	assert.Equal(t, 1, OrgProject.MaxFields)
	assert.Equal(t, 1, OrgProject.MaxEnumOptions)
	assert.Equal(t, 1, OrgProject.MaxRepositoriesPerProject)
}

func TestOrgProjectRejectsInvalidLimits(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	tests := []struct {
		name string
		cfg  string
	}{
		{name: "default page size", cfg: "DEFAULT_PAGE_SIZE = 0"},
		{name: "max page size", cfg: "MAX_PAGE_SIZE = 0"},
		{name: "page size order", cfg: "DEFAULT_PAGE_SIZE = 2\nMAX_PAGE_SIZE = 1"},
		{name: "max fields", cfg: "MAX_FIELDS = 0"},
		{name: "max enum options", cfg: "MAX_ENUM_OPTIONS = 0"},
		{name: "max repositories", cfg: "MAX_REPOSITORIES_PER_PROJECT = 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loadOrgProjectTestConfig(t, "[database]\nDB_TYPE = sqlite3\n[org_project]\n"+tt.cfg)
			assert.Error(t, err)
		})
	}
}

func TestOrgProjectRequiresSQLite(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	err := loadOrgProjectTestConfig(t, "[database]\nDB_TYPE = postgres")
	require.Error(t, err)
	assert.ErrorContains(t, err, "requires sqlite3")
}

func TestOrgProjectDisabledAllowsOtherDatabases(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	require.NoError(t, loadOrgProjectTestConfig(t, "[database]\nDB_TYPE = mysql\n[org_project]\nENABLED = false"))
	assert.False(t, OrgProject.Enabled)
}

func TestOrgProjectInstallDatabaseValidation(t *testing.T) {
	oldDatabase, oldOrgProject := Database, OrgProject
	t.Cleanup(func() {
		Database = oldDatabase
		OrgProject = oldOrgProject
	})

	cfg, err := NewConfigProviderFromData("[org_project]\nENABLED = true")
	require.NoError(t, err)
	loadDBSetting(cfg)
	require.NoError(t, loadOrgProjectFrom(cfg))

	Database.Type = DatabaseTypeSQLite3
	assert.NoError(t, ValidateOrgProjectDatabase())
	Database.Type = "mysql"
	assert.ErrorContains(t, ValidateOrgProjectDatabase(), "requires sqlite3")
}
