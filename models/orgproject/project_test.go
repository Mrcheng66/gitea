// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project, err := GetProjectBySlug(t.Context(), 3, "platform")
	require.NoError(t, err)
	assert.EqualValues(t, 1, project.ID)
	assert.Equal(t, LifecycleActive, project.Lifecycle)

	_, err = GetProjectByID(t.Context(), unittest.NonexistentID)
	assert.True(t, IsErrProjectNotExist(err))
}

func TestProjectConstraints(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	_, err := db.GetEngine(t.Context()).Insert(&Project{
		OwnerID: 3, Slug: "platform", Name: "Duplicate", Lifecycle: LifecycleActive, Version: 1, CreatedBy: 2,
	})
	assert.Error(t, err)

	_, err = db.GetEngine(t.Context()).Insert(&Repository{
		ProjectID: 1, RepositoryID: 2, Role: RepositoryRolePrimary, CreatedBy: 2,
	})
	assert.Error(t, err)

	_, err = db.GetEngine(t.Context()).Insert(&Repository{
		ProjectID: 1, RepositoryID: 2, Role: RepositoryRoleRelated, CreatedBy: 2,
	})
	assert.NoError(t, err)
}
