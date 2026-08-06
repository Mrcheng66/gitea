// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	api "gitea.dev/modules/structs"
	"gitea.dev/services/orgproject/config"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgProjectAPI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	resetOrgProjectAPIData(t)
	_, pointer, err := config.InitializeDefaultDraft(t.Context(), 3, 2)
	require.NoError(t, err)
	_, err = config.PublishDraft(t.Context(), 3, 2, pointer.Version)
	require.NoError(t, err)

	writeToken := getUserToken(t, "user2", auth_model.AccessTokenScopeWriteProject)
	readToken := getUserToken(t, "user2", auth_model.AccessTokenScopeReadProject)

	create := api.CreateOrgProjectOption{
		Slug: "api-project", Name: "API Project", RequestID: "api-create-project",
		Repositories: []api.OrgProjectRepository{{RepositoryID: 3, Role: string(orgproject_model.RepositoryRolePrimary)}},
	}
	req := NewRequestWithJSON(t, http.MethodPost, "/api/v1/orgs/org3/projects", &create).AddTokenAuth(writeToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	created := DecodeJSON(t, resp, &api.OrgProject{})
	assert.Equal(t, "api-project", created.Slug)
	assert.EqualValues(t, 1, created.Version)
	assert.Contains(t, created.Values, "stage")
	require.Len(t, created.Repositories, 1)

	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/projects").AddTokenAuth(readToken)
	resp = MakeRequest(t, req, http.StatusOK)
	list := DecodeJSON(t, resp, &api.OrgProjectList{})
	assert.EqualValues(t, 1, list.Total)
	require.Len(t, list.Projects, 1)

	updatedName := "Updated API Project"
	edit := api.EditOrgProjectOption{Version: 1, Name: &updatedName, RequestID: "api-update-project"}
	req = NewRequestWithJSON(t, http.MethodPatch, "/api/v1/orgs/org3/projects/api-project", &edit).AddTokenAuth(writeToken)
	resp = MakeRequest(t, req, http.StatusOK)
	updated := DecodeJSON(t, resp, &api.OrgProject{})
	assert.Equal(t, "Updated API Project", updated.Name)
	assert.EqualValues(t, 2, updated.Version)

	link := api.LinkOrgProjectRepositoryOption{Role: string(orgproject_model.RepositoryRoleRelated), Version: 2, RequestID: "api-link-repository"}
	req = NewRequestWithJSON(t, http.MethodPost, "/api/v1/orgs/org3/projects/api-project/repositories/32", &link).AddTokenAuth(writeToken)
	resp = MakeRequest(t, req, http.StatusOK)
	linked := DecodeJSON(t, resp, &api.OrgProject{})
	assert.EqualValues(t, 3, linked.Version)
	require.Len(t, linked.Repositories, 2)

	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/projects/api-project/history").AddTokenAuth(readToken)
	resp = MakeRequest(t, req, http.StatusOK)
	history := DecodeJSON(t, resp, &api.OrgProjectChangeList{})
	require.Len(t, *history, 3)

	req = NewRequest(t, http.MethodDelete, "/api/v1/orgs/org3/projects/api-project/repositories/32?version=3&request_id=api-unlink-repository").AddTokenAuth(writeToken)
	MakeRequest(t, req, http.StatusNoContent)

	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/project-config/draft").AddTokenAuth(readToken)
	resp = MakeRequest(t, req, http.StatusOK)
	draft := DecodeJSON(t, resp, &api.OrgProjectConfigVersion{})

	updateDraft := api.UpdateOrgProjectConfigOption{PointerVersion: draft.PointerVersion, Schema: draft.Schema}
	req = NewRequestWithJSON(t, http.MethodPut, "/api/v1/orgs/org3/project-config/draft", &updateDraft).AddTokenAuth(writeToken)
	resp = MakeRequest(t, req, http.StatusOK)
	savedDraft := DecodeJSON(t, resp, &api.OrgProjectConfigVersion{})

	publish := api.PublishOrgProjectConfigOption{PointerVersion: savedDraft.PointerVersion}
	req = NewRequestWithJSON(t, http.MethodPost, "/api/v1/orgs/org3/project-config/publish", &publish).AddTokenAuth(writeToken)
	MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/project-config/versions").AddTokenAuth(readToken)
	resp = MakeRequest(t, req, http.StatusOK)
	versions := DecodeJSON(t, resp, &api.OrgProjectConfigVersionList{})
	require.Len(t, *versions, 4)

	publicOnlyToken := getUserToken(t, "user2", auth_model.AccessTokenScopeReadProject, auth_model.AccessTokenScopePublicOnly)
	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/projects").AddTokenAuth(publicOnlyToken)
	MakeRequest(t, req, http.StatusForbidden)

	memberToken := getUserToken(t, "user4", auth_model.AccessTokenScopeWriteProject)
	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/projects").AddTokenAuth(memberToken)
	MakeRequest(t, req, http.StatusOK)
	req = NewRequestWithJSON(t, http.MethodPost, "/api/v1/orgs/org3/projects", &api.CreateOrgProjectOption{
		Slug: "forbidden", Name: "Forbidden", RequestID: "api-forbidden-project",
	}).AddTokenAuth(memberToken)
	MakeRequest(t, req, http.StatusForbidden)

	wrongScopeToken := getUserToken(t, "user2", auth_model.AccessTokenScopeReadOrganization)
	req = NewRequest(t, http.MethodGet, "/api/v1/orgs/org3/projects").AddTokenAuth(wrongScopeToken)
	MakeRequest(t, req, http.StatusForbidden)
}

func resetOrgProjectAPIData(t *testing.T) {
	t.Helper()
	engine := db.GetEngine(t.Context())
	for _, bean := range []any{
		new(orgproject_model.ChangeLog), new(orgproject_model.Repository), new(orgproject_model.FieldValue), new(orgproject_model.EditorTeam),
		new(orgproject_model.Project), new(orgproject_model.ConfigPointer), new(orgproject_model.ConfigVersion),
	} {
		_, err := engine.Where("1 = 1").Delete(bean)
		require.NoError(t, err)
	}
}
