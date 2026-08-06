// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	"gitea.dev/services/context"
	project_service "gitea.dev/services/orgproject/project"
)

// RepositoryLinks renders organization projects associated with the current repository.
func RepositoryLinks(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org_project.repository.title")
	ctx.Data["PageIsOrgProjectLinks"] = true
	ctx.Data["OrgProjectRepository"] = ctx.Repo.Repository

	if ctx.Repo.Owner == nil || !ctx.Repo.Owner.IsOrganization() {
		ctx.Data["OrgProjects"] = nil
		ctx.HTML(http.StatusOK, tplOrgProjectRepository)
		return
	}
	projects, err := project_service.ListByRepository(ctx, ctx.Repo.Owner.ID, ctx.Repo.Repository.ID, ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectRepository)
		return
	}
	ctx.Data["OrgProjects"] = projects
	ctx.Data["OrgProjectOrgLink"] = "/org/" + ctx.Repo.Owner.Name + "/projects"
	ctx.HTML(http.StatusOK, tplOrgProjectRepository)
}
