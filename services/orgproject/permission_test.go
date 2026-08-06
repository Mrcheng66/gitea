// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"

	"gitea.dev/models/db"
	org_model "gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermission(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	editor := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 28})
	nonMember := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := db.Insert(t.Context(), &orgproject_model.EditorTeam{
		OwnerID:   3,
		TeamID:    2,
		CreatedBy: owner.ID,
	})
	require.NoError(t, err)

	tests := []struct {
		name string
		user *user_model.User
		want Permission
	}{
		{name: "anonymous", user: nil, want: Permission{}},
		{name: "owner", user: owner, want: Permission{Read: true, Edit: true, Configure: true}},
		{name: "editor team member", user: editor, want: Permission{Read: true, Edit: true}},
		{name: "ordinary member", user: member, want: Permission{Read: true}},
		{name: "non-member", user: nonMember, want: Permission{}},
		{name: "site administrator", user: admin, want: Permission{Read: true, Edit: true, Configure: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permission, err := GetPermission(t.Context(), 3, tt.user)
			require.NoError(t, err)
			assert.Equal(t, tt.want, permission)
		})
	}
}

func TestGetPermissionIgnoresEditorTeamFromAnotherOrganization(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})

	err := db.Insert(t.Context(), &org_model.OrgUser{UID: user.ID, OrgID: 3})
	require.NoError(t, err)
	err = db.Insert(t.Context(), &orgproject_model.EditorTeam{
		OwnerID:   3,
		TeamID:    3,
		CreatedBy: 2,
	})
	require.NoError(t, err)

	permission, err := GetPermission(t.Context(), 3, user)
	require.NoError(t, err)
	assert.Equal(t, Permission{Read: true}, permission)
}

func TestRepositoryVisibility(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 28})
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	privateVisible := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	privateHidden := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
	publicVisible := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 32})
	crossOrganization := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	visible, err := CanReadRepository(t.Context(), member, privateVisible)
	require.NoError(t, err)
	assert.False(t, visible)

	visible, err = CanReadRepository(t.Context(), member, publicVisible)
	require.NoError(t, err)
	assert.True(t, visible)

	visible, err = CanReadRepository(t.Context(), admin, privateHidden)
	require.NoError(t, err)
	assert.True(t, visible)

	linkable, err := CanLinkRepository(t.Context(), 3, owner, privateVisible)
	require.NoError(t, err)
	assert.True(t, linkable)

	linkable, err = CanLinkRepository(t.Context(), 3, member, privateHidden)
	require.NoError(t, err)
	assert.False(t, linkable)

	linkable, err = CanLinkRepository(t.Context(), 3, member, publicVisible)
	require.NoError(t, err)
	assert.True(t, linkable)

	linkable, err = CanLinkRepository(t.Context(), 3, owner, crossOrganization)
	require.NoError(t, err)
	assert.False(t, linkable)
}

func TestFilterVisibleRepositories(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 28})
	privateHidden := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	publicVisible := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 32})

	visible, err := FilterVisibleRepositories(t.Context(), member, []*repo_model.Repository{privateHidden, nil, publicVisible})
	require.NoError(t, err)
	require.Len(t, visible, 1)
	assert.Equal(t, publicVisible.ID, visible[0].ID)
}
