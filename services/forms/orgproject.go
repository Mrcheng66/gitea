// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forms

import (
	"net/http"

	"gitea.dev/modules/web/middleware"
	"gitea.dev/services/context"

	"gitea.com/go-chi/binding"
)

// CreateOrgProjectForm contains the server-rendered organization project fields.
type CreateOrgProjectForm struct {
	Slug        string `binding:"Required;MaxSize(255)"`
	Name        string `binding:"Required;MaxSize(255)"`
	Description string
	Values      string
}

// Validate validates the fields.
func (f *CreateOrgProjectForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// UpdateOrgProjectForm contains editable organization project fields.
type UpdateOrgProjectForm struct {
	Version     int64  `binding:"Required"`
	Slug        string `binding:"Required;MaxSize(255)"`
	Name        string `binding:"Required;MaxSize(255)"`
	Description string
	Values      string
}

// Validate validates the fields.
func (f *UpdateOrgProjectForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// OrgProjectVersionForm identifies the project version for an archive request.
type OrgProjectVersionForm struct {
	Version int64 `binding:"Required"`
}

// OrgProjectRepositoryForm contains one repository link mutation.
type OrgProjectRepositoryForm struct {
	RepositoryID int64  `binding:"Required"`
	Role         string `binding:"Required"`
	Version      int64  `binding:"Required"`
}

// OrgProjectConfigForm contains the JSON configuration editor payload.
type OrgProjectConfigForm struct {
	PointerVersion int64  `binding:"Required"`
	Schema         string `binding:"Required"`
}

// OrgProjectConfigVersionForm identifies a configuration mutation and target version.
type OrgProjectConfigVersionForm struct {
	PointerVersion int64 `binding:"Required"`
	Version        int64
}

// OrgProjectEditorTeamsForm contains the teams allowed to edit projects.
type OrgProjectEditorTeamsForm struct {
	TeamIDs []int64
}
