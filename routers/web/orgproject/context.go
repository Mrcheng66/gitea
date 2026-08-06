// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	"gitea.dev/services/context"
	orgproject_service "gitea.dev/services/orgproject"
)

// RequireRead requires organization project read permission.
func RequireRead(ctx *context.Context) {
	requirePermission(ctx, orgproject_service.AccessRead)
}

// RequireEdit requires organization project edit permission.
func RequireEdit(ctx *context.Context) {
	requirePermission(ctx, orgproject_service.AccessEdit)
}

// RequireConfigure requires organization project configuration permission.
func RequireConfigure(ctx *context.Context) {
	requirePermission(ctx, orgproject_service.AccessConfigure)
}

func requirePermission(ctx *context.Context, level orgproject_service.AccessLevel) {
	if ctx.Org == nil || ctx.Org.Organization == nil {
		ctx.NotFound(nil)
		return
	}

	permission, err := orgproject_service.GetPermission(ctx, ctx.Org.Organization.ID, ctx.Doer)
	if err != nil {
		ctx.ServerError("GetOrgProjectPermission", err)
		return
	}
	if !permission.Read {
		ctx.NotFound(nil)
		return
	}

	ctx.Data["OrgProjectPermission"] = permission
	ctx.Data["CanEditOrgProjects"] = permission.Edit
	ctx.Data["CanConfigureOrgProjects"] = permission.Configure
	if !permission.Allows(level) {
		ctx.HTTPError(http.StatusForbidden)
	}
}
