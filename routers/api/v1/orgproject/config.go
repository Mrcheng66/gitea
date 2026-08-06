// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	"gitea.dev/modules/json"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/orgproject/config"
)

// GetDraftConfig returns the current draft project configuration.
func GetDraftConfig(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/project-config/draft project orgProjectConfigDraft
	// ---
	// summary: Get the organization project draft configuration
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectConfigVersion"
	pointer, err := config.GetPointer(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	version, err := config.GetDraft(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, configVersionToAPI(version, pointer.Version))
}

// UpdateDraftConfig saves a new draft project configuration.
func UpdateDraftConfig(ctx *context.APIContext) {
	// swagger:operation PUT /orgs/{org}/project-config/draft project orgProjectConfigUpdateDraft
	// ---
	// summary: Save an organization project draft configuration
	// consumes:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/UpdateOrgProjectConfigOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectConfigVersion"
	form := web.GetForm(ctx).(*api.UpdateOrgProjectConfigOption)
	schema, err := decodeSchema(form.Schema)
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	version, err := config.SaveDraft(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.PointerVersion, schema)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	pointer, err := config.GetPointer(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, configVersionToAPI(version, pointer.Version))
}

// ValidateConfig validates a project configuration without saving it.
func ValidateConfig(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/project-config/validate project orgProjectConfigValidate
	// ---
	// summary: Validate an organization project configuration
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ValidateOrgProjectConfigOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	form := web.GetForm(ctx).(*api.ValidateOrgProjectConfigOption)
	schema, err := decodeSchema(form.Schema)
	if err == nil {
		err = config.Validate(schema)
	}
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

// PublishConfig publishes the current draft project configuration.
func PublishConfig(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/project-config/publish project orgProjectConfigPublish
	// ---
	// summary: Publish the organization project draft configuration
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/PublishOrgProjectConfigOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectConfigVersion"
	form := web.GetForm(ctx).(*api.PublishOrgProjectConfigOption)
	version, err := config.PublishDraft(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.PointerVersion)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	pointer, err := config.GetPointer(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, configVersionToAPI(version, pointer.Version))
}

// ListConfigVersions lists organization project configuration versions.
func ListConfigVersions(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/project-config/versions project orgProjectConfigVersions
	// ---
	// summary: List organization project configuration versions
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectConfigVersionList"
	versions, err := config.ListHistory(ctx, ctx.Org.Organization.ID, ctx.FormInt("limit"))
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	result := make(api.OrgProjectConfigVersionList, 0, len(versions))
	for _, version := range versions {
		result = append(result, configVersionToAPI(version, 0))
	}
	ctx.JSON(http.StatusOK, result)
}

// RollbackConfig rolls back to a published project configuration version.
func RollbackConfig(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/project-config/rollback/{version} project orgProjectConfigRollback
	// ---
	// summary: Roll back the organization project configuration
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// - name: version
	//   in: path
	//   description: published configuration version
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/RollbackOrgProjectConfigOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectConfigVersion"
	form := web.GetForm(ctx).(*api.RollbackOrgProjectConfigOption)
	versionNumber, err := parsePositiveInt64(ctx.PathParam("version"), "version")
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	target, err := config.GetPublishedVersion(ctx, ctx.Org.Organization.ID, versionNumber)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	version, err := config.RollbackPublished(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, target.ID, form.PointerVersion)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	pointer, err := config.GetPointer(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, configVersionToAPI(version, pointer.Version))
}

func decodeSchema(raw json.Value) (config.Schema, error) {
	var schema config.Schema
	err := json.Unmarshal(raw, &schema)
	return schema, err
}
