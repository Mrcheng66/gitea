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

func TestConfigConstraints(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	_, err := db.GetEngine(t.Context()).Insert(&ConfigVersion{
		OwnerID: 3, Version: 1, State: ConfigStateDraft, Payload: "{}", CreatedBy: 2,
	})
	assert.Error(t, err)

	_, err = db.GetEngine(t.Context()).Insert(&EditorTeam{OwnerID: 3, TeamID: 1, CreatedBy: 2})
	assert.Error(t, err)

	_, err = db.GetEngine(t.Context()).Insert(&ChangeLog{
		ProjectID: 1, ActorID: 2, RequestID: "fixture-request-1", ChangedFields: "[]", BeforeValue: "{}", AfterValue: "{}", Source: ChangeSourceAPI,
	})
	assert.Error(t, err)
}

func TestFieldValueRequiresExactlyOneTypedValue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	_, err := db.GetEngine(t.Context()).Insert(&FieldValue{ProjectID: 1, FieldKey: "empty"})
	assert.ErrorContains(t, err, "exactly one typed value")

	text := "high"
	number := 1.0
	_, err = db.GetEngine(t.Context()).Insert(&FieldValue{ProjectID: 1, FieldKey: "multiple", ValueText: &text, ValueNumber: &number})
	assert.ErrorContains(t, err, "exactly one typed value")

	_, err = db.GetEngine(t.Context()).Insert(&FieldValue{ProjectID: 1, FieldKey: "risk", ValueText: &text})
	assert.NoError(t, err)
}
