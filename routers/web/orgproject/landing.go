// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	org_model "gitea.dev/models/organization"
	"gitea.dev/modules/setting"
	"gitea.dev/services/context"
)

// Landing renders the organization project entry point.
func Landing(ctx *context.Context) {
	organizations, err := org_model.GetUserOrgsList(ctx, ctx.Doer)
	if err != nil {
		ctx.ServerError("GetUserOrgsList", err)
		return
	}
	if len(organizations) == 1 {
		ctx.Redirect(setting.AppSubURL + "/org/" + organizations[0].Name + "/projects")
		return
	}

	ctx.Data["Title"] = ctx.Tr("org_project.title")
	ctx.Data["Organizations"] = organizations
	ctx.Data["PageIsOrgProjects"] = true
	ctx.HTML(http.StatusOK, tplOrgProjectLanding)
}
