// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"

	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceEditorTeams(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, ReplaceEditorTeams(t.Context(), 3, 2, []int64{7, 2, 7}))
	teamIDs, err := ListEditorTeamIDs(t.Context(), 3)
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 7}, teamIDs)

	require.NoError(t, ReplaceEditorTeams(t.Context(), 3, 2, nil))
	teamIDs, err = ListEditorTeamIDs(t.Context(), 3)
	require.NoError(t, err)
	assert.Empty(t, teamIDs)
}

func TestReplaceEditorTeamsRejectsAnotherOrganization(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := ReplaceEditorTeams(t.Context(), 3, 2, []int64{3})
	require.ErrorContains(t, err, "does not belong")

	teamIDs, listErr := ListEditorTeamIDs(t.Context(), 3)
	require.NoError(t, listErr)
	assert.Equal(t, []int64{1}, teamIDs)
}
