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
	"gitea.dev/services/orgproject/config"
)

type UpdateOptions struct {
	OwnerID              int64
	ProjectID            int64
	ExpectedVersion      int64
	Actor                *user_model.User
	Slug                 string
	Name                 string
	Description          string
	Values               map[string]config.RawValue
	Repositories         []RepositoryInput
	PreserveRepositories bool
	RequestID            string
	Source               orgproject_model.ChangeSource
}

func Update(ctx context.Context, opts UpdateOptions) (*orgproject_model.Project, error) {
	normalizeUpdateOptions(&opts)
	if err := validateCommonInput(opts.Slug, opts.Name, opts.RequestID, opts.Source); err != nil {
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
		if current.Version != opts.ExpectedVersion {
			return ErrConflict{Expected: opts.ExpectedVersion, Actual: current.Version}
		}
		if current.Lifecycle == orgproject_model.LifecycleArchived {
			return ErrConflict{Field: "lifecycle"}
		}
		if err := ensureSlugAvailable(ctx, opts.OwnerID, opts.Slug, opts.ProjectID); err != nil {
			return err
		}

		schema, err := config.GetPublishedSchema(ctx, opts.OwnerID)
		if err != nil {
			return err
		}
		values, err := prepareFieldValues(ctx, opts.OwnerID, schema, opts.Values)
		if err != nil {
			return err
		}
		var repositories []preparedRepository
		if !opts.PreserveRepositories {
			repositories, err = prepareRepositories(ctx, opts.OwnerID, opts.Actor, opts.Repositories)
			if err != nil {
				return err
			}
		}
		before, err := loadSnapshot(ctx, current)
		if err != nil {
			return err
		}

		if err := updateProject(ctx, current, opts); err != nil {
			return err
		}
		if err := replaceActiveFieldValues(ctx, current.ID, schema, values); err != nil {
			return err
		}
		if !opts.PreserveRepositories {
			if err := replaceRepositories(ctx, current.ID, opts.Actor.ID, repositories); err != nil {
				return err
			}
		}
		updated, err = getProject(ctx, opts.OwnerID, opts.ProjectID)
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

func normalizeUpdateOptions(opts *UpdateOptions) {
	opts.Slug = strings.ToLower(strings.TrimSpace(opts.Slug))
	opts.Name = strings.TrimSpace(opts.Name)
	opts.RequestID = strings.TrimSpace(opts.RequestID)
}

func getProject(ctx context.Context, ownerID, projectID int64) (*orgproject_model.Project, error) {
	project := &orgproject_model.Project{ID: projectID}
	session := db.GetEngine(ctx).ID(projectID)
	if ownerID != 0 {
		session = session.And("owner_id = ?", ownerID)
	}
	has, err := session.Get(project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound{ProjectID: projectID, OwnerID: ownerID}
	}
	return project, nil
}

func updateProject(ctx context.Context, current *orgproject_model.Project, opts UpdateOptions) error {
	result, err := db.GetEngine(ctx).Exec(
		"UPDATE org_project SET slug = ?, name = ?, description = ?, version = version + 1, updated_unix = ? WHERE id = ? AND owner_id = ? AND version = ?",
		opts.Slug, opts.Name, opts.Description, timeutil.TimeStampNow(), current.ID, opts.OwnerID, opts.ExpectedVersion,
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
	return nil
}

func replaceActiveFieldValues(ctx context.Context, projectID int64, schema config.Schema, values []*orgproject_model.FieldValue) error {
	activeKeys := make([]string, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		if !field.Archived {
			activeKeys = append(activeKeys, field.Key)
		}
	}
	if len(activeKeys) > 0 {
		if _, err := db.GetEngine(ctx).Where("project_id = ?", projectID).In("field_key", activeKeys).Delete(new(orgproject_model.FieldValue)); err != nil {
			return err
		}
	}
	return insertFieldValues(ctx, projectID, values)
}

func replaceRepositories(ctx context.Context, projectID, actorID int64, repositories []preparedRepository) error {
	if _, err := db.GetEngine(ctx).Where("project_id = ?", projectID).Delete(new(orgproject_model.Repository)); err != nil {
		return err
	}
	return insertRepositories(ctx, projectID, actorID, repositories)
}
