// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/models/orgproject"
	"gitea.dev/modules/timeutil"
)

func RollbackPublished(ctx context.Context, ownerID, actorID, targetVersionID, expectedPointerVersion int64) (*orgproject.ConfigVersion, error) {
	var rollback *orgproject.ConfigVersion
	err := db.WithTx(ctx, func(ctx context.Context) error {
		pointer, err := getPointer(ctx, ownerID)
		if err != nil {
			return err
		}
		if pointer.Version != expectedPointerVersion {
			return ErrConfigConflict{Expected: expectedPointerVersion, Actual: pointer.Version}
		}
		target, err := getConfigVersion(ctx, ownerID, targetVersionID, orgproject.ConfigStatePublished)
		if err != nil {
			return err
		}
		nextVersion, err := nextConfigVersion(ctx, ownerID)
		if err != nil {
			return err
		}
		rollback = &orgproject.ConfigVersion{
			OwnerID: ownerID, Version: nextVersion, State: orgproject.ConfigStatePublished, Payload: target.Payload,
			CreatedBy: actorID, PublishedBy: actorID, PublishedUnix: timeutil.TimeStampNow(),
		}
		if _, err := db.GetEngine(ctx).Insert(rollback); err != nil {
			return err
		}
		return updatePointer(ctx, ownerID, expectedPointerVersion, "published_version_id", rollback.ID)
	})
	return rollback, err
}
