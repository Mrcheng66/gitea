// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	user_model "gitea.dev/models/user"
	orgproject_service "gitea.dev/services/orgproject"
)

// Detail contains an organization project and its readable related data.
type Detail struct {
	Project      *orgproject_model.Project
	Values       []*orgproject_model.FieldValue
	Repositories []*orgproject_model.Repository
}

// GetBySlug returns a project detail visible to actor.
func GetBySlug(ctx context.Context, ownerID int64, slug string, actor *user_model.User) (*Detail, error) {
	permission, err := orgproject_service.GetPermission(ctx, ownerID, actor)
	if err != nil {
		return nil, err
	}
	if !permission.Read {
		return nil, ErrNotFound{OwnerID: ownerID}
	}
	project, err := orgproject_model.GetProjectBySlug(ctx, ownerID, slug)
	if err != nil {
		if orgproject_model.IsErrProjectNotExist(err) {
			return nil, ErrNotFound{OwnerID: ownerID}
		}
		return nil, err
	}
	return getDetail(ctx, project, actor)
}

// GetByID returns a project detail visible to actor.
func GetByID(ctx context.Context, ownerID, projectID int64, actor *user_model.User) (*Detail, error) {
	permission, err := orgproject_service.GetPermission(ctx, ownerID, actor)
	if err != nil {
		return nil, err
	}
	if !permission.Read {
		return nil, ErrNotFound{ProjectID: projectID, OwnerID: ownerID}
	}
	project, err := getProject(ctx, ownerID, projectID)
	if err != nil {
		return nil, err
	}
	return getDetail(ctx, project, actor)
}

func getDetail(ctx context.Context, project *orgproject_model.Project, actor *user_model.User) (*Detail, error) {
	values := make([]*orgproject_model.FieldValue, 0)
	if err := db.GetEngine(ctx).Where("project_id = ?", project.ID).Asc("field_key").Find(&values); err != nil {
		return nil, err
	}

	links := make([]*orgproject_model.Repository, 0)
	if err := db.GetEngine(ctx).Where("project_id = ?", project.ID).Asc("repository_id").Find(&links); err != nil {
		return nil, err
	}
	visible := make([]*orgproject_model.Repository, 0, len(links))
	for _, link := range links {
		_, isVisible, err := orgproject_service.GetVisibleRepository(ctx, project.OwnerID, actor, link.RepositoryID)
		if err != nil {
			return nil, err
		}
		if isVisible {
			visible = append(visible, link)
		}
	}
	return &Detail{Project: project, Values: values, Repositories: visible}, nil
}

// ListChanges returns the newest project change records first.
func ListChanges(ctx context.Context, ownerID, projectID int64, actor *user_model.User, limit int) ([]*orgproject_model.ChangeLog, error) {
	if _, err := GetByID(ctx, ownerID, projectID, actor); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	changes := make([]*orgproject_model.ChangeLog, 0, limit)
	err := db.GetEngine(ctx).Where("project_id = ?", projectID).Desc("id").Limit(limit).Find(&changes)
	return changes, err
}

// ListByRepository returns active organization projects linked to a visible repository.
func ListByRepository(ctx context.Context, ownerID, repositoryID int64, actor *user_model.User) ([]*orgproject_model.Project, error) {
	permission, err := orgproject_service.GetPermission(ctx, ownerID, actor)
	if err != nil {
		return nil, err
	}
	if !permission.Read {
		return nil, ErrNotFound{OwnerID: ownerID}
	}
	projects := make([]*orgproject_model.Project, 0)
	err = db.GetEngine(ctx).
		Table("org_project").
		Join("INNER", "org_project_repository", "org_project_repository.project_id = org_project.id").
		Where("org_project.owner_id = ?", ownerID).
		And("org_project_repository.repository_id = ?", repositoryID).
		And("org_project.lifecycle = ?", orgproject_model.LifecycleActive).
		Asc("org_project.name", "org_project.id").
		Find(&projects)
	return projects, err
}
