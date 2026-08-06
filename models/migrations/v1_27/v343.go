// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_27

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/models/orgproject"
	"gitea.dev/modules/timeutil"
)

func AddOrgProjectSchema(x db.EngineMigration) error {
	type OrgProject struct {
		ID          int64              `xorm:"pk autoincr"`
		OwnerID     int64              `xorm:"UNIQUE(owner_slug) INDEX(owner_lifecycle_updated) NOT NULL"`
		Slug        string             `xorm:"UNIQUE(owner_slug) VARCHAR(255) NOT NULL"`
		Name        string             `xorm:"VARCHAR(255) NOT NULL"`
		Description string             `xorm:"TEXT"`
		Lifecycle   string             `xorm:"INDEX(owner_lifecycle_updated) VARCHAR(16) NOT NULL"`
		Version     int64              `xorm:"NOT NULL DEFAULT 1"`
		CreatedBy   int64              `xorm:"NOT NULL"`
		CreatedUnix timeutil.TimeStamp `xorm:"created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX(owner_lifecycle_updated)"`
	}
	type OrgProjectRepository struct {
		ID           int64              `xorm:"pk autoincr"`
		ProjectID    int64              `xorm:"UNIQUE(project_repository) INDEX NOT NULL"`
		RepositoryID int64              `xorm:"UNIQUE(project_repository) INDEX NOT NULL"`
		Role         string             `xorm:"VARCHAR(16) NOT NULL"`
		CreatedBy    int64              `xorm:"NOT NULL"`
		CreatedUnix  timeutil.TimeStamp `xorm:"created"`
	}
	type OrgProjectFieldValue struct {
		ID          int64   `xorm:"pk autoincr"`
		ProjectID   int64   `xorm:"UNIQUE(project_field) INDEX NOT NULL"`
		FieldKey    string  `xorm:"UNIQUE(project_field) VARCHAR(64) NOT NULL"`
		ValueText   *string `xorm:"TEXT"`
		ValueNumber *float64
		ValueTime   *timeutil.TimeStamp
		ValueBool   *bool
		ValueUserID *int64
		ValueJSON   *string `xorm:"TEXT"`
	}
	type OrgProjectConfigVersion struct {
		ID            int64              `xorm:"pk autoincr"`
		OwnerID       int64              `xorm:"UNIQUE(owner_version) INDEX NOT NULL"`
		Version       int64              `xorm:"UNIQUE(owner_version) NOT NULL"`
		State         string             `xorm:"INDEX VARCHAR(16) NOT NULL"`
		Payload       string             `xorm:"TEXT NOT NULL"`
		CreatedBy     int64              `xorm:"NOT NULL"`
		CreatedUnix   timeutil.TimeStamp `xorm:"created"`
		PublishedBy   int64
		PublishedUnix timeutil.TimeStamp
	}
	type OrgProjectConfigPointer struct {
		OwnerID            int64 `xorm:"pk"`
		DraftVersionID     int64
		PublishedVersionID int64
		Version            int64 `xorm:"NOT NULL DEFAULT 1"`
	}
	type OrgProjectEditorTeam struct {
		ID          int64              `xorm:"pk autoincr"`
		OwnerID     int64              `xorm:"UNIQUE(owner_team) INDEX NOT NULL"`
		TeamID      int64              `xorm:"UNIQUE(owner_team) INDEX NOT NULL"`
		CreatedBy   int64              `xorm:"NOT NULL"`
		CreatedUnix timeutil.TimeStamp `xorm:"created"`
	}
	type OrgProjectChangeLog struct {
		ID            int64              `xorm:"pk autoincr"`
		ProjectID     int64              `xorm:"NOT NULL"`
		ActorID       int64              `xorm:"INDEX NOT NULL"`
		RequestID     string             `xorm:"UNIQUE VARCHAR(64) NOT NULL"`
		ChangedFields string             `xorm:"TEXT NOT NULL"`
		BeforeValue   string             `xorm:"TEXT NOT NULL"`
		AfterValue    string             `xorm:"TEXT NOT NULL"`
		Source        string             `xorm:"VARCHAR(32) NOT NULL"`
		CreatedUnix   timeutil.TimeStamp `xorm:"created"`
	}

	if err := x.Sync(
		new(OrgProject),
		new(OrgProjectRepository),
		new(OrgProjectFieldValue),
		new(OrgProjectConfigVersion),
		new(OrgProjectConfigPointer),
		new(OrgProjectEditorTeam),
		new(OrgProjectChangeLog),
	); err != nil {
		return err
	}
	return orgproject.EnsureSQLiteSchema(context.Background(), x)
}
