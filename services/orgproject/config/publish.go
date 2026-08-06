// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/models/orgproject"
	"gitea.dev/modules/timeutil"
)

func PublishDraft(ctx context.Context, ownerID, actorID, expectedPointerVersion int64) (*orgproject.ConfigVersion, error) {
	var published *orgproject.ConfigVersion
	err := db.WithTx(ctx, func(ctx context.Context) error {
		pointer, err := getPointer(ctx, ownerID)
		if err != nil {
			return err
		}
		if pointer.Version != expectedPointerVersion {
			return ErrConfigConflict{Expected: expectedPointerVersion, Actual: pointer.Version}
		}
		draft, err := getConfigVersion(ctx, ownerID, pointer.DraftVersionID, orgproject.ConfigStateDraft)
		if err != nil {
			return err
		}
		schema, err := decodeSchema(draft.Payload)
		if err != nil {
			return err
		}
		if err := Validate(schema); err != nil {
			return ErrConfigValidation{Err: err}
		}
		if pointer.PublishedVersionID != 0 {
			current, err := getConfigVersion(ctx, ownerID, pointer.PublishedVersionID, orgproject.ConfigStatePublished)
			if err != nil {
				return err
			}
			currentSchema, err := decodeSchema(current.Payload)
			if err != nil {
				return err
			}
			if err := ValidateTransition(currentSchema, schema); err != nil {
				return ErrConfigValidation{Err: err}
			}
		}
		nextVersion, err := nextConfigVersion(ctx, ownerID)
		if err != nil {
			return err
		}
		published = &orgproject.ConfigVersion{
			OwnerID: ownerID, Version: nextVersion, State: orgproject.ConfigStatePublished, Payload: draft.Payload,
			CreatedBy: actorID, PublishedBy: actorID, PublishedUnix: timeutil.TimeStampNow(),
		}
		if _, err := db.GetEngine(ctx).Insert(published); err != nil {
			return err
		}
		return updatePointer(ctx, ownerID, expectedPointerVersion, "published_version_id", published.ID)
	})
	return published, err
}
