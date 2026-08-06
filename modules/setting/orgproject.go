// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"fmt"
)

var OrgProject = struct {
	Enabled                   bool
	DefaultPageSize           int
	MaxPageSize               int
	MaxFields                 int
	MaxEnumOptions            int
	MaxRepositoriesPerProject int
}{}

func loadOrgProjectFrom(rootCfg ConfigProvider) error {
	sec := rootCfg.Section("org_project")
	OrgProject.Enabled = sec.Key("ENABLED").MustBool(true)
	OrgProject.DefaultPageSize = sec.Key("DEFAULT_PAGE_SIZE").MustInt(25)
	OrgProject.MaxPageSize = sec.Key("MAX_PAGE_SIZE").MustInt(100)
	OrgProject.MaxFields = sec.Key("MAX_FIELDS").MustInt(64)
	OrgProject.MaxEnumOptions = sec.Key("MAX_ENUM_OPTIONS").MustInt(100)
	OrgProject.MaxRepositoriesPerProject = sec.Key("MAX_REPOSITORIES_PER_PROJECT").MustInt(20)

	if !OrgProject.Enabled {
		return nil
	}
	if OrgProject.DefaultPageSize < 1 {
		return errors.New("org project DEFAULT_PAGE_SIZE must be at least 1")
	}
	if OrgProject.MaxPageSize < 1 {
		return errors.New("org project MAX_PAGE_SIZE must be at least 1")
	}
	if OrgProject.DefaultPageSize > OrgProject.MaxPageSize {
		return errors.New("org project DEFAULT_PAGE_SIZE must not exceed MAX_PAGE_SIZE")
	}
	if OrgProject.MaxFields < 1 {
		return errors.New("org project MAX_FIELDS must be at least 1")
	}
	if OrgProject.MaxEnumOptions < 1 {
		return errors.New("org project MAX_ENUM_OPTIONS must be at least 1")
	}
	if OrgProject.MaxRepositoriesPerProject < 1 {
		return errors.New("org project MAX_REPOSITORIES_PER_PROJECT must be at least 1")
	}
	return ValidateOrgProjectDatabase()
}

// LoadOrgProjectSettings loads and validates the organization project settings.
func LoadOrgProjectSettings() error {
	return loadOrgProjectFrom(CfgProvider)
}

// ValidateOrgProjectDatabase rejects databases unsupported by the organization project module.
func ValidateOrgProjectDatabase() error {
	if !OrgProject.Enabled || Database.Type == "" || Database.Type.IsSQLite3() {
		return nil
	}
	return fmt.Errorf("org project module requires sqlite3, configured database is %q", Database.Type)
}
