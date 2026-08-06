// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "gitea.dev/modules/timeutil"

type RepositoryRole string

const (
	RepositoryRolePrimary RepositoryRole = "primary"
	RepositoryRoleRelated RepositoryRole = "related"
)

// Repository links a project to a Gitea repository.
type Repository struct {
	ID           int64              `xorm:"pk autoincr"`
	ProjectID    int64              `xorm:"UNIQUE(project_repository) INDEX NOT NULL"`
	RepositoryID int64              `xorm:"UNIQUE(project_repository) INDEX NOT NULL"`
	Role         RepositoryRole     `xorm:"VARCHAR(16) NOT NULL"`
	CreatedBy    int64              `xorm:"NOT NULL"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created"`
}

func (*Repository) TableName() string { return "org_project_repository" }
