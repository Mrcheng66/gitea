// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	orgproject_model "gitea.dev/models/orgproject"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	project_service "gitea.dev/services/orgproject/project"
)

// LinkRepository links a repository to an organization project.
func LinkRepository(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{slug}/repositories/{repo_id} project orgProjectLinkRepository
	// ---
	// summary: Link a repository to an organization project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: slug
	//   in: path
	//   description: project slug
	//   type: string
	//   required: true
	// - name: repo_id
	//   in: path
	//   description: repository ID
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/LinkOrgProjectRepositoryOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProject"
	form := web.GetForm(ctx).(*api.LinkOrgProjectRepositoryOption)
	repositoryID, err := parsePositiveInt64(ctx.PathParam("repo_id"), "repository ID")
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	updated, err := project_service.LinkRepository(ctx, project_service.LinkRepositoryOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, RepositoryID: repositoryID,
		Role: orgproject_model.RepositoryRole(form.Role), ExpectedVersion: form.Version, Actor: ctx.Doer,
		RequestID: form.RequestID, Source: orgproject_model.ChangeSourceAPI,
	})
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	respondProject(ctx, updated.ID, http.StatusOK)
}

// UnlinkRepository removes a repository link from an organization project.
func UnlinkRepository(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{slug}/repositories/{repo_id} project orgProjectUnlinkRepository
	// ---
	// summary: Unlink a repository from an organization project
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: slug
	//   in: path
	//   description: project slug
	//   type: string
	//   required: true
	// - name: repo_id
	//   in: path
	//   description: repository ID
	//   type: integer
	//   format: int64
	//   required: true
	// - name: version
	//   in: query
	//   description: expected project version
	//   type: integer
	//   format: int64
	//   required: true
	// - name: request_id
	//   in: query
	//   description: idempotency request ID
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	repositoryID, err := parsePositiveInt64(ctx.PathParam("repo_id"), "repository ID")
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	version, err := parsePositiveInt64(ctx.FormString("version"), "version")
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	requestID := ctx.FormString("request_id")
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	_, err = project_service.UnlinkRepository(ctx, project_service.UnlinkRepositoryOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, RepositoryID: repositoryID,
		ExpectedVersion: version, Actor: ctx.Doer, RequestID: requestID, Source: orgproject_model.ChangeSourceAPI,
	})
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
