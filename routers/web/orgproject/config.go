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
