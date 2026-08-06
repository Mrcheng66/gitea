// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "gitea.dev/modules/timeutil"

// EditorTeam grants a same-organization team permission to edit projects.
type EditorTeam struct {
	ID          int64              `xorm:"pk autoincr"`
	OwnerID     int64              `xorm:"UNIQUE(owner_team) INDEX NOT NULL"`
	TeamID      int64              `xorm:"UNIQUE(owner_team) INDEX NOT NULL"`
	CreatedBy   int64              `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func (*EditorTeam) TableName() string { return "org_project_editor_team" }
