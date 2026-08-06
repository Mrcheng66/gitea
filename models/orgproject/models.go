// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"

	"gitea.dev/models/db"
	"gitea.dev/modules/setting"
)

func init() {
	db.RegisterModel(new(Project))
	db.RegisterModel(new(Repository))
	db.RegisterModel(new(FieldValue))
	db.RegisterModel(new(ConfigVersion))
	db.RegisterModel(new(ConfigPointer))
	db.RegisterModel(new(EditorTeam))
	db.RegisterModel(new(ChangeLog), func() error {
		if !setting.OrgProject.Enabled {
			return nil
		}
		return EnsureSQLiteSchema(context.Background(), db.GetEngine(context.Background()))
	})
}
