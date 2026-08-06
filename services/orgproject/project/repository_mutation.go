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

// LinkRepositoryOptions links one repository to a project.
type LinkRepositoryOptions struct {
	OwnerID         int64
	ProjectID       int64
	RepositoryID    int64
	Role            orgproject_model.RepositoryRole
	ExpectedVersion int64
	Actor           *user_model.User
	RequestID       string
	Source          orgproject_model.ChangeSource
}

// LinkRepository links one visible repository and records the mutation.
func LinkRepository(ctx context.Context, opts LinkRepositoryOptions) (*orgproject_model.Project, error) {
	opts.RequestID = strings.TrimSpace(opts.RequestID)
	if err := validateRequest(opts.RequestID, opts.Source); err != nil {
		return nil, err
	}

	var updated *orgproject_model.Project
	err := db.WithTx(ctx, func(ctx context.Context) error {
		current, err := getProject(ctx, opts.OwnerID, opts.ProjectID)
		if err != nil {
			return err
		}
		if err := requireEdit(ctx, opts.OwnerID, opts.Actor, opts.ProjectID); err != nil {
			return err
		}
		if replay, err := replayProject(ctx, opts.RequestID, opts.OwnerID, opts.ProjectID, opts.Actor.ID, opts.Source); err != nil || replay != nil {
			updated = replay
			return err
		}
		if err := validateMutableProject(current, opts.ExpectedVersion); err != nil {
			return err
		}
		links := make([]*orgproject_model.Repository, 0)
		if err := db.GetEngine(ctx).Where("project_id = ?", current.ID).Find(&links); err != nil {
			return err
		}
		if len(links) >= maxRepositoriesPerProject() {
			return ValidationErrors{"repositories": "has reached the configured limit"}
		}
		for _, link := range links {
			if link.RepositoryID == opts.RepositoryID {
				return ErrConflict{Field: "repositories"}
			}
			if opts.Role == orgproject_model.RepositoryRolePrimary && link.Role == orgproject_model.RepositoryRolePrimary {
				return ValidationErrors{"role": "a primary repository already exists"}
			}
		}
		prepared, err := prepareRepositories(ctx, opts.OwnerID, opts.Actor, []RepositoryInput{{RepositoryID: opts.RepositoryID, Role: opts.Role}})
		if err != nil {
			return err
		}
		before, err := loadSnapshot(ctx, current)
		if err != nil {
			return err
		}
		if err := incrementVersion(ctx, current, opts.ExpectedVersion); err != nil {
			return err
		}
		if err := insertRepositories(ctx, current.ID, opts.Actor.ID, prepared); err != nil {
			return err
		}
		updated, err = getProject(ctx, opts.OwnerID, current.ID)
		if err != nil {
			return err
		}
		after, err := loadSnapshot(ctx, updated)
		if err != nil {
			return err
		}
		return insertChangeLog(ctx, updated.ID, opts.Actor.ID, opts.RequestID, opts.Source, before, after)
	})
	return updated, err
}

// UnlinkRepositoryOptions removes one repository link from a project.
type UnlinkRepositoryOptions struct {
	OwnerID         int64
	ProjectID       int64
	RepositoryID    int64
	ExpectedVersion int64
	Actor           *user_model.User
	RequestID       string
	Source          orgproject_model.ChangeSource
}

// UnlinkRepository removes one repository link and records the mutation.
func UnlinkRepository(ctx context.Context, opts UnlinkRepositoryOptions) (*orgproject_model.Project, error) {
	opts.RequestID = strings.TrimSpace(opts.RequestID)
	if err := validateRequest(opts.RequestID, opts.Source); err != nil {
		return nil, err
	}

	var updated *orgproject_model.Project
	err := db.WithTx(ctx, func(ctx context.Context) error {
		current, err := getProject(ctx, opts.OwnerID, opts.ProjectID)
		if err != nil {
			return err
		}
		if err := requireEdit(ctx, opts.OwnerID, opts.Actor, opts.ProjectID); err != nil {
			return err
		}
		if replay, err := replayProject(ctx, opts.RequestID, opts.OwnerID, opts.ProjectID, opts.Actor.ID, opts.Source); err != nil || replay != nil {
			updated = replay
			return err
		}
		if err := validateMutableProject(current, opts.ExpectedVersion); err != nil {
			return err
		}
		link := &orgproject_model.Repository{ProjectID: current.ID, RepositoryID: opts.RepositoryID}
		has, err := db.GetEngine(ctx).Get(link)
		if err != nil {
			return err
		}
		if !has {
			return ErrNotFound{ProjectID: current.ID, OwnerID: opts.OwnerID}
		}
		before, err := loadSnapshot(ctx, current)
		if err != nil {
			return err
		}
		if err := incrementVersion(ctx, current, opts.ExpectedVersion); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).ID(link.ID).Delete(new(orgproject_model.Repository)); err != nil {
			return err
		}
		updated, err = getProject(ctx, opts.OwnerID, current.ID)
		if err != nil {
			return err
		}
		after, err := loadSnapshot(ctx, updated)
		if err != nil {
			return err
		}
		return insertChangeLog(ctx, updated.ID, opts.Actor.ID, opts.RequestID, opts.Source, before, after)
	})
	return updated, err
}

func validateMutableProject(project *orgproject_model.Project, expectedVersion int64) error {
	if project.Version != expectedVersion {
		return ErrConflict{Expected: expectedVersion, Actual: project.Version}
	}
	if project.Lifecycle == orgproject_model.LifecycleArchived {
		return ErrConflict{Field: "lifecycle"}
	}
	return nil
}

func incrementVersion(ctx context.Context, project *orgproject_model.Project, expectedVersion int64) error {
	result, err := db.GetEngine(ctx).Exec(
		"UPDATE org_project SET version = version + 1, updated_unix = ? WHERE id = ? AND owner_id = ? AND version = ?",
		timeutil.TimeStampNow(), project.ID, project.OwnerID, expectedVersion,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		latest, getErr := getProject(ctx, project.OwnerID, project.ID)
		if getErr != nil {
			return getErr
		}
		return ErrConflict{Expected: expectedVersion, Actual: latest.Version}
	}
	return nil
}
