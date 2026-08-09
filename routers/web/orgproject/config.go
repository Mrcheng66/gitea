// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	org_model "gitea.dev/models/organization"
	"gitea.dev/modules/json"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/forms"
	orgproject_service "gitea.dev/services/orgproject"
	"gitea.dev/services/orgproject/config"
)

type editorTeamDisplay struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Selected    bool   `json:"selected"`
	Deleted     bool   `json:"deleted"`
}

func configEditorLabels(ctx *context.Context) map[string]string {
	keys := []string{
		"fields", "fieldsDescription", "fieldCount", "addField", "newField", "untitledField", "emptyFields",
		"key", "emptyKey", "label", "type", "required", "archived", "options", "optionKey", "optionLabel",
		"newOption", "addOption", "removeOption", "remove", "moveUp", "moveDown", "restore", "archive",
		"listView", "listViewDescription", "columns", "columnsDescription", "defaultSorting", "addSort", "emptySorts",
		"field", "direction", "ascending", "descending", "removeSort", "filters", "filtersDescription", "addFilter",
		"newFilter", "emptyFilters", "operator", "equals", "notEqual", "contains", "isEmpty", "isNotEmpty",
		"atLeast", "atMost", "containsMember", "removeFilter", "metrics", "metricsDescription", "addMetric",
		"newMetric", "emptyMetrics", "aggregation", "count", "average", "valueField", "projects", "groupBy", "none",
		"removeMetric", "validationTitle", "invalidFieldKey", "duplicateFieldKey", "fieldNeedsLabel", "fieldNeedsOption",
		"type_short_text", "type_long_text", "type_single_select", "type_multi_select", "type_integer", "type_decimal",
		"type_percent", "type_date", "type_date_time", "type_boolean", "type_member", "type_member_array",
	}
	labels := make(map[string]string, len(keys))
	for _, key := range keys {
		labels[key] = string(ctx.Tr("org_project.settings.editor." + key))
	}
	return labels
}

// Settings renders the organization project configuration page.
func Settings(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org_project.settings.title")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsOrgProjectSettings"] = true

	pointer, err := config.GetPointer(ctx, ctx.Org.Organization.ID)
	if err != nil && isConfigUninitialized(err) {
		_, pointer, err = config.InitializeDefaultDraft(ctx, ctx.Org.Organization.ID, ctx.Doer.ID)
	}
	if err != nil {
		ctx.ServerError("InitializeOrgProjectConfig", err)
		return
	}
	draft, err := config.GetDraft(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("GetOrgProjectDraft", err)
		return
	}
	history, err := config.ListHistory(ctx, ctx.Org.Organization.ID, 50)
	if err != nil {
		ctx.ServerError("ListOrgProjectConfigHistory", err)
		return
	}

	ctx.Data["OrgProjectConfigPointer"] = pointer
	ctx.Data["OrgProjectConfigDraft"] = draft
	ctx.Data["OrgProjectConfigSchema"] = draft.Payload
	ctx.Data["OrgProjectConfigHistory"] = history
	ctx.Data["OrgProjectConfigLabels"] = configEditorLabels(ctx)
	if err := setEditorTeamData(ctx); err != nil {
		ctx.ServerError("LoadOrgProjectEditorTeams", err)
		return
	}
	ctx.HTML(http.StatusOK, tplOrgProjectSettings)
}

func setEditorTeamData(ctx *context.Context) error {
	teamIDs, err := orgproject_service.ListEditorTeamIDs(ctx, ctx.Org.Organization.ID)
	if err != nil {
		return err
	}
	selected := make(map[int64]bool, len(teamIDs))
	for _, teamID := range teamIDs {
		selected[teamID] = true
	}
	teams, err := org_model.FindOrgTeams(ctx, ctx.Org.Organization.ID)
	if err != nil {
		return err
	}
	displays := make([]editorTeamDisplay, 0, len(teams)+len(teamIDs))
	for _, team := range teams {
		displays = append(displays, editorTeamDisplay{
			ID: team.ID, Name: team.Name, Description: team.Description, Selected: selected[team.ID],
		})
		delete(selected, team.ID)
	}
	for _, teamID := range teamIDs {
		if selected[teamID] {
			displays = append(displays, editorTeamDisplay{
				ID: teamID, Name: string(ctx.Tr("org_project.settings.deleted_team", teamID)), Selected: true, Deleted: true,
			})
		}
	}
	ctx.Data["OrgProjectEditorTeams"] = displays
	return nil
}

// SettingsPost saves an organization project configuration draft.
func SettingsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectConfigForm)
	if ctx.HasError() {
		ctx.Flash.Error(ctx.Tr("form.invalid_data"))
		ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
		return
	}
	schema, err := decodeSchema(form.Schema)
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
		return
	}
	if _, err := config.SaveDraft(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.PointerVersion, schema); err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
		return
	}
	ctx.Flash.Success(ctx.Tr("org_project.settings.save_success"))
	ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
}

// ValidateConfigPost validates configuration JSON without saving it.
func ValidateConfigPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectConfigForm)
	schema, err := decodeSchema(form.Schema)
	if err == nil {
		err = config.Validate(schema)
	}
	if err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.settings.validate_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
}

// PublishConfigPost publishes the current organization project draft.
func PublishConfigPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectConfigVersionForm)
	if _, err := config.PublishDraft(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.PointerVersion); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.settings.publish_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
}

// RollbackConfigPost republishes an earlier configuration version.
func RollbackConfigPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectConfigVersionForm)
	if _, err := config.RollbackPublished(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.Version, form.PointerVersion); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.settings.rollback_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
}

// EditorTeamsPost updates the teams allowed to edit organization projects.
func EditorTeamsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectEditorTeamsForm)
	if err := orgproject_service.ReplaceEditorTeams(ctx, ctx.Org.Organization.ID, ctx.Doer.ID, form.TeamIDs); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.settings.editor_teams_save_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/projects")
}

func decodeSchema(input string) (config.Schema, error) {
	var schema config.Schema
	if err := json.Unmarshal([]byte(input), &schema); err != nil {
		return config.Schema{}, err
	}
	return schema, nil
}
