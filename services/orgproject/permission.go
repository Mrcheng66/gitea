// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"

	"gitea.dev/models/db"
	org_model "gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
	access_model "gitea.dev/models/perm/access"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	user_model "gitea.dev/models/user"
)

// AccessLevel identifies an organization project capability required by a route.
type AccessLevel uint8

const (
	AccessRead AccessLevel = iota
	AccessEdit
	AccessConfigure
)

// Permission contains the organization project capabilities granted to a user.
type Permission struct {
	Read      bool
	Edit      bool
	Configure bool
}

// Allows reports whether the permission grants the requested access level.
func (permission Permission) Allows(level AccessLevel) bool {
	switch level {
	case AccessRead:
		return permission.Read
	case AccessEdit:
		return permission.Edit
	case AccessConfigure:
		return permission.Configure
	default:
		return false
	}
}

// GetPermission returns the organization project capabilities for a user.
func GetPermission(ctx context.Context, ownerID int64, user *user_model.User) (Permission, error) {
	if user == nil {
		return Permission{}, nil
	}
	if user.IsAdmin {
		return Permission{Read: true, Edit: true, Configure: true}, nil
	}

	isMember, err := org_model.IsOrganizationMember(ctx, ownerID, user.ID)
	if err != nil || !isMember {
		return Permission{}, err
	}

	permission := Permission{Read: true}
	isOwner, err := org_model.IsOrganizationOwner(ctx, ownerID, user.ID)
	if err != nil {
		return Permission{}, err
	}
	if isOwner {
		permission.Edit = true
		permission.Configure = true
		return permission, nil
	}

	permission.Edit, err = isEditorTeamMember(ctx, ownerID, user.ID)
	if err != nil {
		return Permission{}, err
	}
	return permission, nil
}

func isEditorTeamMember(ctx context.Context, ownerID, userID int64) (bool, error) {
	var editorTeam orgproject_model.EditorTeam
	return db.GetEngine(ctx).
		Table(editorTeam.TableName()).
		Join("INNER", "team", "team.id = org_project_editor_team.team_id AND team.org_id = org_project_editor_team.owner_id").
		Join("INNER", "team_user", "team_user.team_id = org_project_editor_team.team_id AND team_user.org_id = org_project_editor_team.owner_id").
		Where("org_project_editor_team.owner_id = ?", ownerID).
		And("team_user.uid = ?", userID).
		Exist(&editorTeam)
}

// CanReadRepository reports whether the user can read the repository's code unit.
func CanReadRepository(ctx context.Context, user *user_model.User, repo *repo_model.Repository) (bool, error) {
	if user == nil || repo == nil {
		return false, nil
	}
	permission, err := access_model.GetIndividualUserRepoPermission(ctx, repo, user)
	if err != nil {
		return false, err
	}
	return permission.CanRead(unit.TypeCode), nil
}

// CanLinkRepository reports whether a same-organization repository can be linked by the user.
func CanLinkRepository(ctx context.Context, ownerID int64, user *user_model.User, repo *repo_model.Repository) (bool, error) {
	if repo == nil || repo.OwnerID != ownerID {
		return false, nil
	}
	return CanReadRepository(ctx, user, repo)
}

// FilterVisibleRepositories removes repositories whose code unit the user cannot read.
func FilterVisibleRepositories(ctx context.Context, user *user_model.User, repos []*repo_model.Repository) ([]*repo_model.Repository, error) {
	visible := make([]*repo_model.Repository, 0, len(repos))
	for _, repo := range repos {
		canRead, err := CanReadRepository(ctx, user, repo)
		if err != nil {
			return nil, err
		}
		if canRead {
			visible = append(visible, repo)
		}
	}
	return visible, nil
}

// GetVisibleRepository returns a repository when actor may link it to an organization project.
func GetVisibleRepository(ctx context.Context, ownerID int64, actor *user_model.User, repositoryID int64) (*repo_model.Repository, bool, error) {
	repository, err := repo_model.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	visible, err := CanLinkRepository(ctx, ownerID, actor, repository)
	if err != nil || !visible {
		return nil, false, err
	}
	return repository, true, nil
}
