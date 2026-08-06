// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"

	"gitea.dev/models/unittest"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		FixtureFiles: []string{
			"user.yml",
			"org_user.yml",
			"team.yml",
			"team_user.yml",
			"team_unit.yml",
			"team_repo.yml",
			"repository.yml",
			"repo_unit.yml",
			"access.yml",
			"collaboration.yml",
			"org_project_editor_team.yml",
		},
	})
}
