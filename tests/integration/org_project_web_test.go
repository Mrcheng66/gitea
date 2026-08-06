// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/util"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgProjectWeb(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	resetOrgProjectAPIData(t)

	_, pointer, err := config.InitializeDefaultDraft(t.Context(), 3, 2)
	require.NoError(t, err)
	_, err = config.PublishDraft(t.Context(), 3, 2, pointer.Version)
	require.NoError(t, err)
	require.NoError(t, db.Insert(t.Context(), &orgproject_model.EditorTeam{OwnerID: 3, TeamID: 2, CreatedBy: 2}))

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	_, err = project_service.Create(t.Context(), project_service.CreateOptions{
		OwnerID: 3, Actor: owner, Slug: "web-project", Name: "Web Project", Description: "Native project acceptance",
		Values: map[string]config.RawValue{}, RequestID: "web-project-create", Source: orgproject_model.ChangeSourceWeb,
	})
	require.NoError(t, err)

	ownerSession := loginUser(t, "user2")
	editorSession := loginUser(t, "user4")
	memberSession := loginUser(t, "user28")
	nonMemberSession := loginUser(t, "user5")

	for _, session := range []*TestSession{ownerSession, editorSession, memberSession} {
		resp := session.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/list"), http.StatusOK)
		assert.Contains(t, util.UnsafeBytesToString(resp.Body.Bytes()), "Web Project")
	}

	editorSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/new"), http.StatusOK)
	memberSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/new"), http.StatusForbidden)
	nonMemberSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/list"), http.StatusNotFound)

	resp := memberSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/web-project"), http.StatusOK)
	body := util.UnsafeBytesToString(resp.Body.Bytes())
	assert.Contains(t, body, "Web Project")
	assert.NotContains(t, body, `action="/org/org3/projects/web-project/archive"`)

	resp = editorSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/web-project"), http.StatusOK)
	assert.Contains(t, util.UnsafeBytesToString(resp.Body.Bytes()), `action="/org/org3/projects/web-project/archive"`)

	resp = ownerSession.MakeRequest(t, NewRequest(t, http.MethodGet, "/org/org3/projects/web-project/activity"), http.StatusOK)
	assert.Contains(t, util.UnsafeBytesToString(resp.Body.Bytes()), "Web Project")
}
