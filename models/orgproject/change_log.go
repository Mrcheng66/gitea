// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "gitea.dev/modules/timeutil"

type ChangeSource string

const (
	ChangeSourceWeb          ChangeSource = "web"
	ChangeSourceAPI          ChangeSource = "api"
	ChangeSourceLegacyImport ChangeSource = "legacy-import"
)

// ChangeLog records one project mutation per request.
type ChangeLog struct {
	ID            int64              `xorm:"pk autoincr"`
	ProjectID     int64              `xorm:"NOT NULL"`
	ActorID       int64              `xorm:"INDEX NOT NULL"`
	RequestID     string             `xorm:"UNIQUE VARCHAR(64) NOT NULL"`
	ChangedFields string             `xorm:"TEXT NOT NULL"`
	BeforeValue   string             `xorm:"TEXT NOT NULL"`
	AfterValue    string             `xorm:"TEXT NOT NULL"`
	Source        ChangeSource       `xorm:"VARCHAR(32) NOT NULL"`
	CreatedUnix   timeutil.TimeStamp `xorm:"created"`
}

func (*ChangeLog) TableName() string { return "org_project_change_log" }
