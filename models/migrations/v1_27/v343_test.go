// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"testing"

	"gitea.dev/models/migrations/migrationtest"
	"gitea.dev/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddOrgProjectSchema(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip("organization project migration only supports SQLite")
	}

	x, deferrable := migrationtest.PrepareTestEnv(t, 0)
	defer deferrable()
	if x == nil || t.Failed() {
		return
	}

	require.NoError(t, AddOrgProjectSchema(x))
	require.NoError(t, AddOrgProjectSchema(x))

	for _, table := range []string{
		"org_project",
		"org_project_repository",
		"org_project_field_value",
		"org_project_config_version",
		"org_project_config_pointer",
		"org_project_editor_team",
		"org_project_change_log",
	} {
		exists, err := x.IsTableExist(table)
		require.NoError(t, err)
		assert.True(t, exists, table)
	}

	_, err := x.Exec("INSERT INTO org_project (id, owner_id, slug, name, lifecycle, version, created_by) VALUES (1, 3, 'platform', 'Platform', 'active', 1, 2)")
	require.NoError(t, err)
	_, err = x.Exec("INSERT INTO org_project_repository (project_id, repository_id, role, created_by) VALUES (1, 1, 'primary', 2)")
	require.NoError(t, err)
	_, err = x.Exec("INSERT INTO org_project_repository (project_id, repository_id, role, created_by) VALUES (1, 2, 'primary', 2)")
	assert.Error(t, err)

	_, err = x.Exec("INSERT INTO org_project_field_value (project_id, field_key) VALUES (1, 'empty')")
	assert.ErrorContains(t, err, "exactly one typed value")
	_, err = x.Exec("INSERT INTO org_project_field_value (project_id, field_key, value_text, value_number) VALUES (1, 'multiple', 'high', 1)")
	assert.ErrorContains(t, err, "exactly one typed value")
	_, err = x.Exec("INSERT INTO org_project_field_value (project_id, field_key, value_text) VALUES (1, 'risk', 'high')")
	assert.NoError(t, err)

	var count int
	has, err := x.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", "UQE_org_project_repository_primary").Get(&count)
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, 1, count)
}
