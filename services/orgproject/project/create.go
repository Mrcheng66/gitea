// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"strings"
	"unicode/utf8"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/setting"
	orgproject_service "gitea.dev/services/orgproject"
	"gitea.dev/services/orgproject/config"
)

var reservedSlugs = map[string]struct{}{
	"config": {}, "dashboard": {}, "history": {}, "new": {}, "settings": {},
}

type RepositoryInput struct {
	RepositoryID int64
	Role         orgproject_model.RepositoryRole
}

type CreateOptions struct {
	OwnerID      int64
	Actor        *user_model.User
	Slug         string
	Name         string
	Description  string
	Values       map[string]config.RawValue
	Repositories []RepositoryInput
	RequestID    string
	Source       orgproject_model.ChangeSource
}

func Create(ctx context.Context, opts CreateOptions) (*orgproject_model.Project, error) {
	normalizeCreateOptions(&opts)
	if err := validateCommonInput(opts.Slug, opts.Name, opts.RequestID, opts.Source); err != nil {
		return nil, err
	}

	var created *orgproject_model.Project
	err := db.WithTx(ctx, func(ctx context.Context) error {
		if err := requireEdit(ctx, opts.OwnerID, opts.Actor, 0); err != nil {
			return err
		}
		if replay, err := replayProject(ctx, opts.RequestID, opts.OwnerID, 0, opts.Actor.ID, opts.Source); err != nil || replay != nil {
			created = replay
			return err
		}
		if err := ensureSlugAvailable(ctx, opts.OwnerID, opts.Slug, 0); err != nil {
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
		repositories, err := prepareRepositories(ctx, opts.OwnerID, opts.Actor, opts.Repositories)
		if err != nil {
			return err
		}

		created = &orgproject_model.Project{
			OwnerID: opts.OwnerID, Slug: opts.Slug, Name: opts.Name, Description: opts.Description,
			Lifecycle: orgproject_model.LifecycleActive, Version: 1, CreatedBy: opts.Actor.ID,
		}
		if _, err := db.GetEngine(ctx).Insert(created); err != nil {
			return err
		}
		if err := insertFieldValues(ctx, created.ID, values); err != nil {
			return err
		}
		if err := insertRepositories(ctx, created.ID, opts.Actor.ID, repositories); err != nil {
			return err
		}
		after, err := loadSnapshot(ctx, created)
		if err != nil {
			return err
		}
		return insertChangeLog(ctx, created.ID, opts.Actor.ID, opts.RequestID, opts.Source, projectSnapshot{}, after)
	})
	return created, err
}

func normalizeCreateOptions(opts *CreateOptions) {
	opts.Slug = strings.ToLower(strings.TrimSpace(opts.Slug))
	opts.Name = strings.TrimSpace(opts.Name)
	opts.RequestID = strings.TrimSpace(opts.RequestID)
}

func validateCommonInput(slug, name, requestID string, source orgproject_model.ChangeSource) error {
	errs := validateRequestFields(requestID, source)
	if err := repo_model.IsUsableRepoName(slug); err != nil || utf8.RuneCountInString(slug) > 255 {
		errs["slug"] = "must be a valid Gitea name of at most 255 characters"
	} else if _, reserved := reservedSlugs[slug]; reserved {
		errs["slug"] = "is reserved"
	}
	if name == "" {
		errs["name"] = "is required"
	} else if utf8.RuneCountInString(name) > 255 {
		errs["name"] = "must be at most 255 characters"
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func requireEdit(ctx context.Context, ownerID int64, actor *user_model.User, projectID int64) error {
	permission, err := orgproject_service.GetPermission(ctx, ownerID, actor)
	if err != nil {
		return err
	}
	if !permission.Read {
		return ErrNotFound{ProjectID: projectID, OwnerID: ownerID}
	}
	if !permission.Edit {
		return ErrForbidden{}
	}
	return nil
}

func ensureSlugAvailable(ctx context.Context, ownerID int64, slug string, excludeProjectID int64) error {
	session := db.GetEngine(ctx).Where("owner_id = ? AND slug = ?", ownerID, slug)
	if excludeProjectID != 0 {
		session = session.And("id != ?", excludeProjectID)
	}
	has, err := session.Exist(new(orgproject_model.Project))
	if err != nil {
		return err
	}
	if has {
		return ErrConflict{Field: "slug"}
	}
	return nil
}

func maxRepositoriesPerProject() int {
	if setting.OrgProject.MaxRepositoriesPerProject > 0 {
		return setting.OrgProject.MaxRepositoriesPerProject
	}
	return 20
}
