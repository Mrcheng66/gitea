// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	"gitea.dev/services/context"
	"gitea.dev/services/orgproject/activity"
	project_service "gitea.dev/services/orgproject/project"
)

// Activity renders bounded native repository activity for one organization project.
func Activity(ctx *context.Context) {
	setPageData(ctx, "view")
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectActivity)
		return
	}
	summary, err := activity.Get(ctx, ctx.Org.Organization.ID, detail.Project.ID, ctx.Doer, activity.Options{})
	if err != nil {
		ctx.ServerError("GetOrgProjectActivity", err)
		return
	}
	ctx.Data["Title"] = ctx.Tr("org_project.activity.title", detail.Project.Name)
	ctx.Data["OrgProject"] = detail.Project
	ctx.Data["OrgProjectActivity"] = summary
	ctx.HTML(http.StatusOK, tplOrgProjectActivity)
}
