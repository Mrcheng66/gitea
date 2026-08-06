// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "gitea.dev/modules/timeutil"

type ConfigState string

const (
	ConfigStateDraft     ConfigState = "draft"
	ConfigStatePublished ConfigState = "published"
)

// ConfigVersion is an immutable organization project configuration snapshot once published.
type ConfigVersion struct {
	ID            int64              `xorm:"pk autoincr"`
	OwnerID       int64              `xorm:"UNIQUE(owner_version) INDEX NOT NULL"`
	Version       int64              `xorm:"UNIQUE(owner_version) NOT NULL"`
	State         ConfigState        `xorm:"INDEX VARCHAR(16) NOT NULL"`
	Payload       string             `xorm:"TEXT NOT NULL"`
	CreatedBy     int64              `xorm:"NOT NULL"`
	CreatedUnix   timeutil.TimeStamp `xorm:"created"`
	PublishedBy   int64
	PublishedUnix timeutil.TimeStamp
}

func (*ConfigVersion) TableName() string { return "org_project_config_version" }

// ConfigPointer selects the mutable draft and current published configuration for an organization.
type ConfigPointer struct {
	OwnerID            int64 `xorm:"pk"`
	DraftVersionID     int64
	PublishedVersionID int64
	Version            int64 `xorm:"NOT NULL DEFAULT 1"`
}

func (*ConfigPointer) TableName() string { return "org_project_config_pointer" }
