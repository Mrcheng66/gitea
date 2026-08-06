// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"strings"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/timeutil"
)

type ArchiveOptions struct {
	OwnerID         int64
	ProjectID       int64
	ExpectedVersion int64
	Actor           *user_model.User
	RequestID       string
	Source          orgproject_model.ChangeSource
}

func Archive(ctx context.Context, opts ArchiveOptions) (*orgproject_model.Project, error) {
	opts.RequestID = strings.TrimSpace(opts.RequestID)
	if err := validateRequest(opts.RequestID, opts.Source); err != nil {
		return nil, err
	}

	var archived *orgproject_model.Project
	err := db.WithTx(ctx, func(ctx context.Context) error {
		current, err := getProject(ctx, opts.OwnerID, opts.ProjectID)
		if err != nil {
			return err
		}
		if err := requireEdit(ctx, opts.OwnerID, opts.Actor, opts.ProjectID); err != nil {
			return err
		}
		if replay, err := replayProject(ctx, opts.RequestID, opts.OwnerID, opts.ProjectID, opts.Actor.ID, opts.Source); err != nil || replay != nil {
			archived = replay
			return err
		}
		if current.Version != opts.ExpectedVersion {
			return ErrConflict{Expected: opts.ExpectedVersion, Actual: current.Version}
		}
		if current.Lifecycle == orgproject_model.LifecycleArchived {
			return ErrConflict{Field: "lifecycle"}
		}
		before, err := loadSnapshot(ctx, current)
		if err != nil {
			return err
		}

		result, err := db.GetEngine(ctx).Exec(
			"UPDATE org_project SET lifecycle = ?, version = version + 1, updated_unix = ? WHERE id = ? AND owner_id = ? AND version = ?",
			orgproject_model.LifecycleArchived, timeutil.TimeStampNow(), current.ID, opts.OwnerID, opts.ExpectedVersion,
		)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			latest, getErr := getProject(ctx, opts.OwnerID, current.ID)
			if getErr != nil {
				return getErr
			}
			return ErrConflict{Expected: opts.ExpectedVersion, Actual: latest.Version}
		}

		archived, err = getProject(ctx, opts.OwnerID, opts.ProjectID)
		if err != nil {
			return err
		}
		after, err := loadSnapshot(ctx, archived)
		if err != nil {
			return err
		}
		return insertChangeLog(ctx, archived.ID, opts.Actor.ID, opts.RequestID, opts.Source, before, after)
	})
	return archived, err
}

func validateRequest(requestID string, source orgproject_model.ChangeSource) error {
	errs := validateRequestFields(requestID, source)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateRequestFields(requestID string, source orgproject_model.ChangeSource) ValidationErrors {
	errs := ValidationErrors{}
	if requestID == "" {
		errs["request_id"] = "is required"
	} else if len(requestID) > 64 {
		errs["request_id"] = "must be at most 64 characters"
	}
	switch source {
	case orgproject_model.ChangeSourceWeb, orgproject_model.ChangeSourceAPI, orgproject_model.ChangeSourceLegacyImport:
	default:
		errs["source"] = "is invalid"
	}
	return errs
}
