// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/unittest"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		FixtureFiles: []string{
			"org_project.yml",
			"org_project_repository.yml",
			"org_project_field_value.yml",
			"org_project_config_version.yml",
			"org_project_config_pointer.yml",
			"org_project_editor_team.yml",
			"org_project_change_log.yml",
		},
		SetUp: func() error {
			return EnsureSQLiteSchema(context.Background(), db.GetEngine(context.Background()))
		},
	})
}
