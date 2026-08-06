// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"sync"
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/orgproject"
	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeAndPublishDefaultConfiguration(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	draft, pointer, err := InitializeDefaultDraft(t.Context(), 101, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 1, draft.Version)
	assert.EqualValues(t, 1, pointer.Version)

	_, err = GetPublished(t.Context(), 101)
	assert.ErrorAs(t, err, new(ErrConfigUninitialized))

	published, err := PublishDraft(t.Context(), 101, 2, pointer.Version)
	require.NoError(t, err)
	assert.Equal(t, orgproject.ConfigStatePublished, published.State)
	assert.Equal(t, draft.Payload, published.Payload)

	loaded, err := GetPublished(t.Context(), 101)
	require.NoError(t, err)
	assert.Equal(t, published.ID, loaded.ID)
}

func TestSaveDraftOptimisticLock(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, pointer, err := InitializeDefaultDraft(t.Context(), 102, 2)
	require.NoError(t, err)

	schemaA := DefaultSchema()
	schemaA.Fields[0].Label = "Stage A"
	schemaB := DefaultSchema()
	schemaB.Fields[0].Label = "Stage B"

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wait sync.WaitGroup
	for _, schema := range []Schema{schemaA, schemaB} {
		wait.Go(func() {
			<-start
			_, saveErr := SaveDraft(t.Context(), 102, 2, pointer.Version, schema)
			errs <- saveErr
		})
	}
	close(start)
	wait.Wait()
	close(errs)

	successes, conflicts := 0, 0
	for saveErr := range errs {
		if saveErr == nil {
			successes++
		} else if IsErrConfigConflict(saveErr) {
			conflicts++
		} else {
			t.Fatalf("unexpected save error: %v", saveErr)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)

	history, err := ListHistory(t.Context(), 102, 100)
	require.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestPublishValidationFailureKeepsCurrentVersion(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, pointer, err := InitializeDefaultDraft(t.Context(), 103, 2)
	require.NoError(t, err)
	current, err := PublishDraft(t.Context(), 103, 2, pointer.Version)
	require.NoError(t, err)

	pointer = loadPointer(t, 103)
	invalid := DefaultSchema()
	invalid.Fields[0].Type = "formula"
	_, err = SaveDraft(t.Context(), 103, 2, pointer.Version, invalid)
	require.NoError(t, err)
	pointer = loadPointer(t, 103)

	_, err = PublishDraft(t.Context(), 103, 2, pointer.Version)
	assert.ErrorContains(t, err, "unsupported type")
	loaded, err := GetPublished(t.Context(), 103)
	require.NoError(t, err)
	assert.Equal(t, current.ID, loaded.ID)
}

func TestRollbackCreatesNewPublishedVersion(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, pointer, err := InitializeDefaultDraft(t.Context(), 104, 2)
	require.NoError(t, err)
	first, err := PublishDraft(t.Context(), 104, 2, pointer.Version)
	require.NoError(t, err)

	pointer = loadPointer(t, 104)
	changed := DefaultSchema()
	changed.Fields[0].Label = "Delivery Stage"
	_, err = SaveDraft(t.Context(), 104, 2, pointer.Version, changed)
	require.NoError(t, err)
	pointer = loadPointer(t, 104)
	second, err := PublishDraft(t.Context(), 104, 2, pointer.Version)
	require.NoError(t, err)
	assert.NotEqual(t, first.Payload, second.Payload)

	pointer = loadPointer(t, 104)
	rollback, err := RollbackPublished(t.Context(), 104, 2, first.ID, pointer.Version)
	require.NoError(t, err)
	assert.Greater(t, rollback.Version, second.Version)
	assert.Equal(t, first.Payload, rollback.Payload)

	firstReloaded := new(orgproject.ConfigVersion)
	has, err := db.GetEngine(t.Context()).ID(first.ID).Get(firstReloaded)
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, first.Payload, firstReloaded.Payload)
}

func loadPointer(t *testing.T, ownerID int64) *orgproject.ConfigPointer {
	t.Helper()
	pointer := &orgproject.ConfigPointer{OwnerID: ownerID}
	has, err := db.GetEngine(t.Context()).Get(pointer)
	require.NoError(t, err)
	require.True(t, has)
	return pointer
}
