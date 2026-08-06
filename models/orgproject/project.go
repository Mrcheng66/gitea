// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/modules/timeutil"
)

type Lifecycle string

const (
	LifecycleActive   Lifecycle = "active"
	LifecycleArchived Lifecycle = "archived"
)

// Project is an organization-owned project independent of repositories.
type Project struct {
	ID          int64              `xorm:"pk autoincr"`
	OwnerID     int64              `xorm:"UNIQUE(owner_slug) INDEX(owner_lifecycle_updated) NOT NULL"`
	Slug        string             `xorm:"UNIQUE(owner_slug) VARCHAR(255) NOT NULL"`
	Name        string             `xorm:"VARCHAR(255) NOT NULL"`
	Description string             `xorm:"TEXT"`
	Lifecycle   Lifecycle          `xorm:"INDEX(owner_lifecycle_updated) VARCHAR(16) NOT NULL"`
	Version     int64              `xorm:"NOT NULL DEFAULT 1"`
	CreatedBy   int64              `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX(owner_lifecycle_updated)"`
}

func (*Project) TableName() string { return "org_project" }

func GetProjectByID(ctx context.Context, id int64) (*Project, error) {
	project := &Project{ID: id}
	has, err := db.GetEngine(ctx).Get(project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectNotExist{ID: id}
	}
	return project, nil
}

func GetProjectBySlug(ctx context.Context, ownerID int64, slug string) (*Project, error) {
	project := &Project{OwnerID: ownerID, Slug: slug}
	has, err := db.GetEngine(ctx).Get(project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectNotExist{OwnerID: ownerID, Slug: slug}
	}
	return project, nil
}
