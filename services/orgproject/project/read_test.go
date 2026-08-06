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

func TestListByRepositoryChecksMembership(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	publishDefaultSchema(t)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	created, err := Create(t.Context(), CreateOptions{
		OwnerID: 3, Actor: owner, Slug: "linked-project", Name: "Linked Project",
		Repositories: []RepositoryInput{{RepositoryID: 3, Role: orgproject_model.RepositoryRolePrimary}},
		RequestID:    "linked-project", Source: orgproject_model.ChangeSourceWeb,
	})
	require.NoError(t, err)

	projects, err := ListByRepository(t.Context(), 3, 3, owner)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, created.ID, projects[0].ID)

	nonMember := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	_, err = ListByRepository(t.Context(), 3, 3, nonMember)
	assert.True(t, IsErrNotFound(err))
}
