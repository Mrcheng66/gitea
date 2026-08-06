// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "gitea.dev/modules/timeutil"

// FieldValue stores exactly one typed dynamic value for a project field.
type FieldValue struct {
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

func (*FieldValue) TableName() string { return "org_project_field_value" }
