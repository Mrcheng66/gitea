// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkAndUnlinkRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	created := createProject(t, owner, "repository-mutation")

	linked, err := LinkRepository(t.Context(), LinkRepositoryOptions{
		OwnerID: 3, ProjectID: created.ID, RepositoryID: 32, Role: orgproject_model.RepositoryRoleRelated,
		ExpectedVersion: 1, Actor: owner, RequestID: "link-repository", Source: orgproject_model.ChangeSourceAPI,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, linked.Version)
	require.Len(t, loadRepositories(t, created.ID), 2)

	replayed, err := LinkRepository(t.Context(), LinkRepositoryOptions{
		OwnerID: 3, ProjectID: created.ID, RepositoryID: 32, Role: orgproject_model.RepositoryRoleRelated,
		ExpectedVersion: 1, Actor: owner, RequestID: "link-repository", Source: orgproject_model.ChangeSourceAPI,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, replayed.Version)

	unlinked, err := UnlinkRepository(t.Context(), UnlinkRepositoryOptions{
		OwnerID: 3, ProjectID: created.ID, RepositoryID: 32, ExpectedVersion: 2,
		Actor: owner, RequestID: "unlink-repository", Source: orgproject_model.ChangeSourceAPI,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, unlinked.Version)
	links := loadRepositories(t, created.ID)
	require.Len(t, links, 1)
	assert.Equal(t, int64(3), links[0].RepositoryID)
}
