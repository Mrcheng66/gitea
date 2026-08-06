// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"
	"errors"
	"fmt"

	"gitea.dev/models/db"
	"gitea.dev/modules/setting"
)

var sqliteSchemaStatements = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS UQE_org_project_repository_primary ON org_project_repository(project_id) WHERE role = 'primary'`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_field_value_text ON org_project_field_value(field_key, value_text)`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_field_value_number ON org_project_field_value(field_key, value_number)`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_field_value_time ON org_project_field_value(field_key, value_time)`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_field_value_bool ON org_project_field_value(field_key, value_bool)`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_field_value_user ON org_project_field_value(field_key, value_user_id)`,
	`CREATE INDEX IF NOT EXISTS IDX_org_project_change_log_project_created ON org_project_change_log(project_id, created_unix, id)`,
	`CREATE TRIGGER IF NOT EXISTS TRG_org_project_field_value_one_insert BEFORE INSERT ON org_project_field_value BEGIN SELECT CASE WHEN ((NEW.value_text IS NOT NULL) + (NEW.value_number IS NOT NULL) + (NEW.value_time IS NOT NULL) + (NEW.value_bool IS NOT NULL) + (NEW.value_user_id IS NOT NULL) + (NEW.value_json IS NOT NULL)) != 1 THEN RAISE(ABORT, 'org project field value must set exactly one typed value') END; END`,
	`CREATE TRIGGER IF NOT EXISTS TRG_org_project_field_value_one_update BEFORE UPDATE ON org_project_field_value BEGIN SELECT CASE WHEN ((NEW.value_text IS NOT NULL) + (NEW.value_number IS NOT NULL) + (NEW.value_time IS NOT NULL) + (NEW.value_bool IS NOT NULL) + (NEW.value_user_id IS NOT NULL) + (NEW.value_json IS NOT NULL)) != 1 THEN RAISE(ABORT, 'org project field value must set exactly one typed value') END; END`,
}

// EnsureSQLiteSchema creates SQLite-specific indexes and constraints missing from XORM sync.
func EnsureSQLiteSchema(ctx context.Context, engine db.Engine) error {
	if !setting.Database.Type.IsSQLite3() {
		return fmt.Errorf("organization project schema requires sqlite3, configured database is %q", setting.Database.Type)
	}
	var jsonValue int
	has, err := engine.Context(ctx).SQL(`SELECT value FROM json_each('[1]')`).Get(&jsonValue)
	if err != nil {
		return fmt.Errorf("organization project schema requires SQLite JSON1: %w", err)
	}
	if !has || jsonValue != 1 {
		return errors.New("organization project schema requires SQLite JSON1")
	}
	for _, statement := range sqliteSchemaStatements {
		if _, err := engine.Context(ctx).Exec(statement); err != nil {
			return fmt.Errorf("initialize organization project SQLite schema: %w", err)
		}
	}
	return nil
}
