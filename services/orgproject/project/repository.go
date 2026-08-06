// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"fmt"
	"sort"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	user_model "gitea.dev/models/user"
	orgproject_service "gitea.dev/services/orgproject"
)

type preparedRepository struct {
	RepositoryID int64
	Role         orgproject_model.RepositoryRole
}

func prepareRepositories(ctx context.Context, ownerID int64, actor *user_model.User, input []RepositoryInput) ([]preparedRepository, error) {
	if len(input) > maxRepositoriesPerProject() {
		return nil, ValidationErrors{"repositories": fmt.Sprintf("must contain at most %d repositories", maxRepositoriesPerProject())}
	}

	seen := make(map[int64]struct{}, len(input))
	primaryCount := 0
	prepared := make([]preparedRepository, 0, len(input))
	for index, item := range input {
		key := fmt.Sprintf("repositories.%d", index)
		if item.RepositoryID <= 0 {
			return nil, ValidationErrors{key: "repository ID must be positive"}
		}
		if _, exists := seen[item.RepositoryID]; exists {
			return nil, ValidationErrors{key: "repository is duplicated"}
		}
		seen[item.RepositoryID] = struct{}{}
		switch item.Role {
		case orgproject_model.RepositoryRolePrimary:
			primaryCount++
		case orgproject_model.RepositoryRoleRelated:
		default:
			return nil, ValidationErrors{key: "repository role is invalid"}
		}
		if primaryCount > 1 {
			return nil, ValidationErrors{"repositories": "must contain at most one primary repository"}
		}

		repository, err := repo_model.GetRepositoryByID(ctx, item.RepositoryID)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				return nil, ErrRepositoryNotVisible{RepositoryID: item.RepositoryID}
			}
			return nil, err
		}
		canLink, err := orgproject_service.CanLinkRepository(ctx, ownerID, actor, repository)
		if err != nil {
			return nil, err
		}
		if !canLink {
			return nil, ErrRepositoryNotVisible{RepositoryID: item.RepositoryID}
		}
		prepared = append(prepared, preparedRepository(item))
	}
	sort.Slice(prepared, func(i, j int) bool { return prepared[i].RepositoryID < prepared[j].RepositoryID })
	return prepared, nil
}

func insertRepositories(ctx context.Context, projectID, actorID int64, repositories []preparedRepository) error {
	for _, repository := range repositories {
		link := &orgproject_model.Repository{
			ProjectID: projectID, RepositoryID: repository.RepositoryID, Role: repository.Role, CreatedBy: actorID,
		}
		if _, err := db.GetEngine(ctx).Insert(link); err != nil {
			return err
		}
	}
	return nil
}
