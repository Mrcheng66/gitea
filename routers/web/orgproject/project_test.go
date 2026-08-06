// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"
	"testing"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"
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
