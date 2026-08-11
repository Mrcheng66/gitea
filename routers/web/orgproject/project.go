// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"
	"strconv"
	"time"

	orgproject_model "gitea.dev/models/orgproject"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/forms"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
	"gitea.dev/services/orgproject/query"

	"github.com/google/uuid"
)

type projectListRow struct {
	Project *orgproject_model.Project
	Fields  []fieldDisplay
}

type projectFilterDisplay struct {
	Key     string
	Label   string
	Value   string
	Field   config.Field
	Members []memberDisplay
}

type projectHistoryEntry struct {
	ID            int64      `json:"id"`
	ActorID       int64      `json:"actor_id"`
	ActorName     string     `json:"actor_name,omitempty"`
	ActorLink     string     `json:"actor_link,omitempty"`
	RequestID     string     `json:"request_id"`
	ChangedFields json.Value `json:"changed_fields"`
	Before        json.Value `json:"before"`
	After         json.Value `json:"after"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
}

func buildProjectHistoryEntries(changes []*orgproject_model.ChangeLog, actors map[int64]*user_model.User) []projectHistoryEntry {
	entries := make([]projectHistoryEntry, 0, len(changes))
	for _, change := range changes {
		entry := projectHistoryEntry{
			ID: change.ID, ActorID: change.ActorID, RequestID: change.RequestID,
			ChangedFields: json.Value(change.ChangedFields), Before: json.Value(change.BeforeValue), After: json.Value(change.AfterValue),
			Source: string(change.Source), CreatedAt: change.CreatedUnix.AsTime(),
		}
		if actor := actors[change.ActorID]; actor != nil {
			entry.ActorName = actor.DisplayName()
			entry.ActorLink = actor.HomeLink()
		}
		entries = append(entries, entry)
	}
	return entries
}

// Dashboard renders configured metrics and recent organization projects.
func Dashboard(ctx *context.Context) {
	renderProjectList(ctx, true)
}

// List renders the organization project list.
func List(ctx *context.Context) {
	renderProjectList(ctx, false)
}

func renderProjectList(ctx *context.Context, dashboard bool) {
	active := "list"
	templateName := tplOrgProjectList
	if dashboard {
		active = "dashboard"
		templateName = tplOrgProjectDashboard
	}
	setPageData(ctx, active)

	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		if isConfigUninitialized(err) {
			ctx.Data["OrgProjectConfigReady"] = false
			ctx.HTML(http.StatusOK, templateName)
			return
		}
		ctx.ServerError("GetPublishedSchema", err)
		return
	}
	members, membersByID, err := loadProjectMembers(ctx)
	if err != nil {
		ctx.ServerError("LoadOrgProjectMembers", err)
		return
	}
	filterValues, filters := projectListFilters(ctx, schema, members)
	listSchema := schema
	listSchema.ListView.Columns = projectLedgerColumns(schema)
	now := timeutil.TimeStampNow().AsTime()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	onlyUserID := int64(0)
	if ctx.FormBool("mine") {
		onlyUserID = ctx.Doer.ID
	}

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}
	result, err := query.List(ctx, listSchema, query.ListOptions{
		OwnerID:         ctx.Org.Organization.ID,
		FilterValues:    filterValues,
		Search:          ctx.FormString("q"),
		OnlyUserID:      onlyUserID,
		Due:             ctx.FormString("due"),
		RiskFirst:       true,
		Now:             timeutil.TimeStamp(today.Unix()),
		Page:            page,
		PageSize:        setting.OrgProject.DefaultPageSize,
		IncludeArchived: ctx.FormBool("include_archived"),
	})
	if err != nil {
		ctx.ServerError("ListOrgProjects", err)
		return
	}

	fieldByKey := make(map[string]config.Field, len(listSchema.Fields))
	for _, field := range schema.Fields {
		fieldByKey[field.Key] = field
	}
	rows := make([]projectListRow, 0, len(result.Items))
	for _, item := range result.Items {
		fields := make([]fieldDisplay, 0, len(listSchema.ListView.Columns))
		for _, key := range listSchema.ListView.Columns {
			field, ok := fieldByKey[key]
			if !ok {
				continue
			}
			fields = append(fields, buildFieldDisplay(field, item.Values[key], membersByID))
		}
		rows = append(rows, projectListRow{Project: item.Project, Fields: fields})
	}

	ctx.Data["OrgProjectConfigReady"] = true
	ctx.Data["OrgProjectSchema"] = schema
	ctx.Data["OrgProjectRows"] = rows
	ctx.Data["OrgProjectTotal"] = result.Total
	ctx.Data["IncludeArchived"] = ctx.FormBool("include_archived")
	ctx.Data["OrgProjectFilters"] = filters
	ctx.Data["OrgProjectSearch"] = ctx.FormString("q")
	ctx.Data["OrgProjectOnlyMine"] = ctx.FormBool("mine")
	ctx.Data["OrgProjectDue"] = ctx.FormString("due")
	ctx.Data["OrgProjectHasFilters"] = ctx.FormString("q") != "" || ctx.FormBool("mine") || ctx.FormString("due") != "" || len(filterValues) > 0 || ctx.FormBool("include_archived")
	pager := context.NewPagination(result.Total, result.PageSize, result.Page, 5)
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	summary, err := query.Summary(ctx, schema, ctx.Org.Organization.ID, now)
	if err != nil {
		ctx.ServerError("OrgProjectSummary", err)
		return
	}
	ctx.Data["OrgProjectSummary"] = summary
	ctx.HTML(http.StatusOK, templateName)
}

func projectLedgerColumns(schema config.Schema) []string {
	active := make(map[string]struct{}, len(schema.Fields))
	for _, field := range schema.Fields {
		if !field.Archived {
			active[field.Key] = struct{}{}
		}
	}
	columns := make([]string, 0, len(schema.ListView.Columns)+6)
	seen := make(map[string]struct{}, len(schema.ListView.Columns)+6)
	for _, key := range []string{"stage", "owner", "followers", "progress", "risk", "target_date"} {
		if _, ok := active[key]; ok {
			columns = append(columns, key)
			seen[key] = struct{}{}
		}
	}
	for _, key := range schema.ListView.Columns {
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := active[key]; ok {
			columns = append(columns, key)
			seen[key] = struct{}{}
		}
	}
	return columns
}

func projectListFilters(ctx *context.Context, schema config.Schema, members []memberDisplay) (map[string]config.RawValue, []projectFilterDisplay) {
	fieldByKey := make(map[string]config.Field, len(schema.Fields))
	for _, field := range schema.Fields {
		fieldByKey[field.Key] = field
	}
	values := make(map[string]config.RawValue)
	displays := make([]projectFilterDisplay, 0, len(schema.Filters))
	for _, filter := range schema.Filters {
		field, ok := fieldByKey[filter.FieldKey]
		if !ok || field.Archived {
			continue
		}
		value := ctx.FormString("filter_" + filter.Key)
		display := projectFilterDisplay{Key: filter.Key, Label: filter.Label, Value: value, Field: field}
		if field.Type == config.FieldTypeMember || field.Type == config.FieldTypeMemberArray {
			display.Members = members
		}
		displays = append(displays, display)
		if value == "" {
			continue
		}
		var raw any = value
		switch field.Type {
		case config.FieldTypeMember, config.FieldTypeMemberArray, config.FieldTypeInteger:
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				continue
			}
			raw = parsed
		case config.FieldTypeDecimal, config.FieldTypePercent:
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			raw = parsed
		case config.FieldTypeBoolean:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				continue
			}
			raw = parsed
		}
		encoded, err := json.Marshal(raw)
		if err == nil {
			values[filter.Key] = encoded
		}
	}
	return values, displays
}

// New renders the organization project creation form.
func New(ctx *context.Context) {
	setPageData(ctx, "new")
	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		if isConfigUninitialized(err) {
			ctx.Flash.Error(ctx.Tr("org_project.error.config_required"))
			ctx.Redirect(ctx.Org.OrgLink + "/projects")
			return
		}
		ctx.ServerError("GetPublishedSchema", err)
		return
	}
	if err := setProjectFormData(ctx, schema, "{}"); err != nil {
		ctx.ServerError("LoadOrgProjectMembers", err)
		return
	}
	ctx.HTML(http.StatusOK, tplOrgProjectNew)
}

// NewPost creates an organization project.
func NewPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateOrgProjectForm)
	setPageData(ctx, "new")
	ctx.Data["Form"] = form
	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("GetPublishedSchema", err)
		return
	}
	if err := setProjectFormData(ctx, schema, form.Values); err != nil {
		ctx.ServerError("LoadOrgProjectMembers", err)
		return
	}
	if ctx.HasError() {
		ctx.HTML(http.StatusUnprocessableEntity, tplOrgProjectNew)
		return
	}
	values, err := decodeValues(form.Values)
	if err != nil {
		ctx.Data["OrgProjectError"] = err.Error()
		ctx.HTML(http.StatusUnprocessableEntity, tplOrgProjectNew)
		return
	}
	created, err := project_service.Create(ctx, project_service.CreateOptions{
		OwnerID: ctx.Org.Organization.ID, Actor: ctx.Doer, Slug: form.Slug, Name: form.Name, Description: form.Description,
		Values: values, RequestID: uuid.New().String(), Source: orgproject_model.ChangeSourceWeb,
	})
	if err != nil {
		ctx.Data["OrgProjectError"] = err.Error()
		writeProjectError(ctx, err, tplOrgProjectNew)
		return
	}
	ctx.Flash.Success(ctx.Tr("org_project.create_success", created.Name))
	ctx.Redirect(ctx.Org.OrgLink + "/projects/" + created.Slug)
}

// View renders one organization project.
func View(ctx *context.Context) {
	setPageData(ctx, "view")
	ctx.Data["OrgProjectDetailActive"] = "overview"
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectView)
		return
	}
	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("GetPublishedSchema", err)
		return
	}
	values, err := encodeValues(schema, detail.Values)
	if err != nil {
		ctx.ServerError("EncodeOrgProjectValues", err)
		return
	}
	repositories, err := loadRepositoryDisplays(ctx, detail.Repositories)
	if err != nil {
		ctx.ServerError("LoadOrgProjectRepositories", err)
		return
	}

	members, membersByID, err := loadProjectMembers(ctx)
	if err != nil {
		ctx.ServerError("LoadOrgProjectMembers", err)
		return
	}
	fields := buildFieldDisplays(schema, detail.Values, membersByID)
	fieldByKey := make(map[string]*fieldDisplay, len(fields))
	otherFields := make([]fieldDisplay, 0, len(fields))
	coreFields := map[string]struct{}{
		"stage": {}, "owner": {}, "followers": {}, "progress": {}, "risk": {}, "start_date": {}, "target_date": {}, "summary": {},
	}
	for index := range fields {
		field := &fields[index]
		fieldByKey[field.Key] = field
		if _, ok := coreFields[field.Key]; !ok {
			otherFields = append(otherFields, *field)
		}
	}
	ctx.Data["Title"] = detail.Project.Name
	ctx.Data["OrgProject"] = detail.Project
	ctx.Data["OrgProjectFields"] = fields
	ctx.Data["OrgProjectFieldByKey"] = fieldByKey
	ctx.Data["OrgProjectOtherFields"] = otherFields
	ctx.Data["OrgProjectRepositories"] = repositories
	ctx.Data["OrgProjectEditing"] = ctx.FormBool("edit")
	ctx.Data["OrgProjectSchema"] = schema
	ctx.Data["OrgProjectValues"] = values
	ctx.Data["OrgProjectMembers"] = members
	ctx.HTML(http.StatusOK, tplOrgProjectView)
}

// EditPost updates one organization project.
func EditPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateOrgProjectForm)
	if ctx.HasError() {
		ctx.Flash.Error(ctx.Tr("form.invalid_data"))
		ctx.Redirect(ctx.Req.URL.Path)
		return
	}
	values, err := decodeValues(form.Values)
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(ctx.Req.URL.Path)
		return
	}
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectView)
		return
	}
	updated, err := project_service.Update(ctx, project_service.UpdateOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, ExpectedVersion: form.Version, Actor: ctx.Doer,
		Slug: form.Slug, Name: form.Name, Description: form.Description, Values: values, PreserveRepositories: true,
		RequestID: uuid.New().String(), Source: orgproject_model.ChangeSourceWeb,
	})
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(ctx.Req.URL.Path)
		return
	}
	ctx.Flash.Success(ctx.Tr("org_project.update_success", updated.Name))
	ctx.Redirect(ctx.Org.OrgLink + "/projects/" + updated.Slug)
}

// ArchivePost archives one organization project.
func ArchivePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectVersionForm)
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectView)
		return
	}
	_, err = project_service.Archive(ctx, project_service.ArchiveOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, ExpectedVersion: form.Version, Actor: ctx.Doer,
		RequestID: uuid.New().String(), Source: orgproject_model.ChangeSourceWeb,
	})
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(ctx.Org.OrgLink + "/projects/" + detail.Project.Slug)
		return
	}
	ctx.Flash.Success(ctx.Tr("org_project.archive_success", detail.Project.Name))
	ctx.Redirect(ctx.Org.OrgLink + "/projects/list?include_archived=true")
}

// History renders the audited history for one organization project.
func History(ctx *context.Context) {
	setPageData(ctx, "history")
	ctx.Data["OrgProjectDetailActive"] = "history"
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectHistory)
		return
	}
	changes, err := project_service.ListChanges(ctx, ctx.Org.Organization.ID, detail.Project.ID, ctx.Doer, 100)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectHistory)
		return
	}
	actorIDs := make([]int64, 0, len(changes))
	seenActorIDs := make(map[int64]struct{}, len(changes))
	for _, change := range changes {
		if _, seen := seenActorIDs[change.ActorID]; seen {
			continue
		}
		seenActorIDs[change.ActorID] = struct{}{}
		actorIDs = append(actorIDs, change.ActorID)
	}
	actors, err := user_model.GetUsersMapByIDs(ctx, actorIDs)
	if err != nil {
		ctx.ServerError("GetUsersMapByIDs", err)
		return
	}
	fieldLabels := map[string]string{
		"slug": ctx.Locale.TrString("org_project.slug"), "name": ctx.Locale.TrString("org_project.name"),
		"description": ctx.Locale.TrString("org_project.description"), "lifecycle": ctx.Locale.TrString("org_project.status"),
		"repositories": ctx.Locale.TrString("org_project.repositories"),
	}
	if schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID); err == nil {
		for _, field := range schema.Fields {
			fieldLabels["values."+field.Key] = field.Label
		}
	}
	ctx.Data["Title"] = ctx.Tr("org_project.history.title", detail.Project.Name)
	ctx.Data["OrgProject"] = detail.Project
	ctx.Data["OrgProjectChanges"] = buildProjectHistoryEntries(changes, actors)
	ctx.Data["OrgProjectHistoryFieldLabels"] = fieldLabels
	ctx.HTML(http.StatusOK, tplOrgProjectHistory)
}

// LinkRepositoryPost links a repository to an organization project.
func LinkRepositoryPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectRepositoryForm)
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectView)
		return
	}
	_, err = project_service.LinkRepository(ctx, project_service.LinkRepositoryOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, RepositoryID: form.RepositoryID,
		Role: orgproject_model.RepositoryRole(form.Role), ExpectedVersion: form.Version, Actor: ctx.Doer,
		RequestID: uuid.New().String(), Source: orgproject_model.ChangeSourceWeb,
	})
	if err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.repository.link_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/projects/" + detail.Project.Slug)
}

// UnlinkRepositoryPost removes a repository from an organization project.
func UnlinkRepositoryPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.OrgProjectVersionForm)
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err, tplOrgProjectView)
		return
	}
	_, err = project_service.UnlinkRepository(ctx, project_service.UnlinkRepositoryOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, RepositoryID: ctx.PathParamInt64("repositoryID"),
		ExpectedVersion: form.Version, Actor: ctx.Doer, RequestID: uuid.New().String(), Source: orgproject_model.ChangeSourceWeb,
	})
	if err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org_project.repository.unlink_success"))
	}
	ctx.Redirect(ctx.Org.OrgLink + "/projects/" + detail.Project.Slug)
}
