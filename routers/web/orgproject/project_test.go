// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"
	"testing"
	"time"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
	"gitea.dev/services/contexttest"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLandingRedirectsSingleOrganizationMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx, recorder := contexttest.MockContext(t, "projects")
	contexttest.LoadUser(t, ctx, 4)

	Landing(ctx)

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/org/org3/projects", recorder.Header().Get("Location"))
}

func TestEncodeAndDecodeValues(t *testing.T) {
	text := "development"
	number := 25.0
	values := []*orgproject_model.FieldValue{
		{FieldKey: "stage", ValueText: &text},
		{FieldKey: "progress", ValueNumber: &number},
	}

	encoded, err := encodeValues(config.DefaultSchema(), values)
	require.NoError(t, err)
	assert.JSONEq(t, `{"progress":25,"stage":"development"}`, encoded)

	decoded, err := decodeValues(encoded)
	require.NoError(t, err)
	assert.JSONEq(t, `"development"`, string(decoded["stage"]))
	assert.JSONEq(t, `25`, string(decoded["progress"]))

	_, err = decodeValues(`{"stage":`)
	assert.Error(t, err)
}

func TestBuildProjectHistoryEntries(t *testing.T) {
	changes := []*orgproject_model.ChangeLog{{
		ID: 11, ActorID: 2, RequestID: "request-11", ChangedFields: `["name","values.stage"]`,
		BeforeValue: `{"name":"Old","values":[{"key":"stage","text":"planning"}]}`,
		AfterValue:  `{"name":"New","values":[{"key":"stage","text":"development"}]}`,
		Source:      orgproject_model.ChangeSourceWeb, CreatedUnix: 1,
	}}
	actors := map[int64]*user_model.User{2: {ID: 2, Name: "user2", FullName: "User Two"}}

	entries := buildProjectHistoryEntries(changes, actors)
	require.Len(t, entries, 1)
	assert.Equal(t, "User Two", entries[0].ActorName)
	assert.Equal(t, "/user2", entries[0].ActorLink)

	payload, err := json.Marshal(entries)
	require.NoError(t, err)
	expected := `[{"id":11,"actor_id":2,"actor_name":"User Two","actor_link":"/user2","request_id":"request-11","changed_fields":["name","values.stage"],"before":{"name":"Old","values":[{"key":"stage","text":"planning"}]},"after":{"name":"New","values":[{"key":"stage","text":"development"}]},"source":"web","created_at":"` + changes[0].CreatedUnix.AsTime().Format(time.RFC3339) + `"}]`
	assert.JSONEq(t, expected, string(payload))
}
