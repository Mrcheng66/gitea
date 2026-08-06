// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"net/http"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/setting"
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

type metricDisplay struct {
	Key     string
	Label   string
	Buckets []query.MetricBucket
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

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}
	result, err := query.List(ctx, schema, query.ListOptions{
		OwnerID:         ctx.Org.Organization.ID,
		Page:            page,
		PageSize:        setting.OrgProject.DefaultPageSize,
		IncludeArchived: ctx.FormBool("include_archived"),
	})
	if err != nil {
		ctx.ServerError("ListOrgProjects", err)
		return
	}

	fieldByKey := make(map[string]config.Field, len(schema.Fields))
	for _, field := range schema.Fields {
		fieldByKey[field.Key] = field
	}
	rows := make([]projectListRow, 0, len(result.Items))
	for _, item := range result.Items {
		fields := make([]fieldDisplay, 0, len(schema.ListView.Columns))
		for _, key := range schema.ListView.Columns {
			field, ok := fieldByKey[key]
			if !ok {
				continue
			}
			value := "—"
			if stored := item.Values[key]; stored != nil {
				value = formatFieldValue(field, stored)
			}
			fields = append(fields, fieldDisplay{Key: key, Label: field.Label, Value: value})
		}
		rows = append(rows, projectListRow{Project: item.Project, Fields: fields})
	}

	ctx.Data["OrgProjectConfigReady"] = true
	ctx.Data["OrgProjectSchema"] = schema
	ctx.Data["OrgProjectRows"] = rows
	ctx.Data["OrgProjectTotal"] = result.Total
	ctx.Data["IncludeArchived"] = ctx.FormBool("include_archived")
	pager := context.NewPagination(result.Total, result.PageSize, result.Page, 5)
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	if dashboard {
		metrics := make([]metricDisplay, 0, len(schema.Metrics))
		for _, metric := range schema.Metrics {
			result, err := query.Metric(ctx, schema, query.MetricOptions{OwnerID: ctx.Org.Organization.ID, MetricKey: metric.Key})
			if err != nil {
				ctx.ServerError("OrgProjectMetric", err)
				return
			}
			metrics = append(metrics, metricDisplay{Key: metric.Key, Label: metric.Label, Buckets: result.Buckets})
		}
		ctx.Data["OrgProjectMetrics"] = metrics
	}
	ctx.HTML(http.StatusOK, templateName)
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

	ctx.Data["Title"] = detail.Project.Name
	ctx.Data["OrgProject"] = detail.Project
	ctx.Data["OrgProjectFields"] = buildFieldDisplays(schema, detail.Values)
	ctx.Data["OrgProjectRepositories"] = repositories
	if err := setProjectFormData(ctx, schema, values); err != nil {
		ctx.ServerError("LoadOrgProjectMembers", err)
		return
	}
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
	ctx.Data["Title"] = ctx.Tr("org_project.history.title", detail.Project.Name)
	ctx.Data["OrgProject"] = detail.Project
	ctx.Data["OrgProjectChanges"] = changes
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
