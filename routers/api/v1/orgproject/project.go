// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/json"
	api "gitea.dev/modules/structs"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
	"gitea.dev/services/orgproject/query"
)

// ListProjects lists organization projects.
func ListProjects(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects project orgProjectList
	// ---
	// summary: List organization projects
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: organization name
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectList"
	//   "404":
	//     "$ref": "#/responses/notFound"
	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	filters, err := queryFilters(ctx, schema)
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	sorts, err := querySorts(ctx)
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	result, err := query.List(ctx, schema, query.ListOptions{
		OwnerID: ctx.Org.Organization.ID, FilterValues: filters, Sort: sorts,
		Page: ctx.FormInt("page"), PageSize: ctx.FormInt("limit"), IncludeArchived: ctx.FormBool("include_archived"),
	})
	if err != nil {
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
		return
	}
	projects := make([]api.OrgProject, 0, len(result.Items))
	for _, item := range result.Items {
		values := make([]*orgproject_model.FieldValue, 0, len(item.Values))
		for _, value := range item.Values {
			values = append(values, value)
		}
		converted, err := fieldValuesToAPI(schema, values)
		if err != nil {
			ctx.APIErrorInternal(err)
			return
		}
		projects = append(projects, *projectToAPI(item.Project, converted, nil))
	}
	ctx.SetTotalCountHeader(result.Total)
	ctx.JSON(http.StatusOK, api.OrgProjectList{Projects: projects, Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// CreateProject creates an organization project.
func CreateProject(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects project orgProjectCreate
	// ---
	// summary: Create an organization project
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
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateOrgProjectOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/OrgProject"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*api.CreateOrgProjectOption)
	created, err := project_service.Create(ctx, project_service.CreateOptions{
		OwnerID: ctx.Org.Organization.ID, Actor: ctx.Doer, Slug: form.Slug, Name: form.Name, Description: form.Description,
		Values: rawValuesToConfig(form.Values), Repositories: repositoriesToService(form.Repositories),
		RequestID: form.RequestID, Source: orgproject_model.ChangeSourceAPI,
	})
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	respondProject(ctx, created.ID, http.StatusCreated)
}

// GetProject returns one organization project.
func GetProject(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{slug} project orgProjectGet
	// ---
	// summary: Get an organization project
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
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProject"
	//   "404":
	//     "$ref": "#/responses/notFound"
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	respondProjectDetail(ctx, detail, http.StatusOK)
}

// EditProject updates one organization project.
func EditProject(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{slug} project orgProjectEdit
	// ---
	// summary: Update an organization project
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
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/EditOrgProjectOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProject"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*api.EditOrgProjectOption)
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	if form.Lifecycle != nil {
		if *form.Lifecycle != string(orgproject_model.LifecycleArchived) || form.Slug != nil || form.Name != nil || form.Description != nil || form.Values != nil {
			ctx.APIError(http.StatusUnprocessableEntity, "lifecycle can only be changed to archived in a standalone request")
			return
		}
		archived, err := project_service.Archive(ctx, project_service.ArchiveOptions{
			OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, ExpectedVersion: form.Version,
			Actor: ctx.Doer, RequestID: form.RequestID, Source: orgproject_model.ChangeSourceAPI,
		})
		if err != nil {
			writeProjectError(ctx, err)
			return
		}
		respondProject(ctx, archived.ID, http.StatusOK)
		return
	}

	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	values, err := fieldValuesToAPI(schema, detail.Values)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	if form.Values != nil {
		values = *form.Values
	}
	slug, name, description := detail.Project.Slug, detail.Project.Name, detail.Project.Description
	if form.Slug != nil {
		slug = *form.Slug
	}
	if form.Name != nil {
		name = *form.Name
	}
	if form.Description != nil {
		description = *form.Description
	}
	updated, err := project_service.Update(ctx, project_service.UpdateOptions{
		OwnerID: ctx.Org.Organization.ID, ProjectID: detail.Project.ID, ExpectedVersion: form.Version, Actor: ctx.Doer,
		Slug: slug, Name: name, Description: description, Values: rawValuesToConfig(values), PreserveRepositories: true,
		RequestID: form.RequestID, Source: orgproject_model.ChangeSourceAPI,
	})
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	respondProject(ctx, updated.ID, http.StatusOK)
}

// ListProjectHistory lists one project's audit history.
func ListProjectHistory(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{slug}/history project orgProjectHistory
	// ---
	// summary: List organization project history
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
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrgProjectChangeList"
	detail, err := project_service.GetBySlug(ctx, ctx.Org.Organization.ID, ctx.PathParam("slug"), ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	changes, err := project_service.ListChanges(ctx, ctx.Org.Organization.ID, detail.Project.ID, ctx.Doer, ctx.FormInt("limit"))
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	result := make(api.OrgProjectChangeList, 0, len(changes))
	for _, change := range changes {
		result = append(result, api.OrgProjectChange{
			ID: change.ID, ActorID: change.ActorID, RequestID: change.RequestID,
			ChangedFields: json.Value(change.ChangedFields), Before: json.Value(change.BeforeValue), After: json.Value(change.AfterValue),
			Source: string(change.Source), CreatedAt: change.CreatedUnix.AsTime(),
		})
	}
	ctx.JSON(http.StatusOK, result)
}

func respondProject(ctx *context.APIContext, projectID int64, status int) {
	detail, err := project_service.GetByID(ctx, ctx.Org.Organization.ID, projectID, ctx.Doer)
	if err != nil {
		writeProjectError(ctx, err)
		return
	}
	respondProjectDetail(ctx, detail, status)
}

func respondProjectDetail(ctx *context.APIContext, detail *project_service.Detail, status int) {
	schema, err := config.GetPublishedSchema(ctx, ctx.Org.Organization.ID)
	if err != nil {
		writeConfigError(ctx, err)
		return
	}
	converted, err := detailToAPI(schema, detail)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(status, converted)
}

func queryFilters(ctx *context.APIContext, schema config.Schema) (map[string]config.RawValue, error) {
	configured := make(map[string]struct{}, len(schema.Filters))
	for _, filter := range schema.Filters {
		configured[filter.Key] = struct{}{}
	}
	filters := map[string]config.RawValue{}
	for key, values := range ctx.Req.URL.Query() {
		if !strings.HasPrefix(key, "filter.") {
			continue
		}
		filterKey := strings.TrimPrefix(key, "filter.")
		if _, ok := configured[filterKey]; !ok {
			return nil, fmt.Errorf("unknown organization project filter %q", filterKey)
		}
		if len(values) == 0 {
			continue
		}
		raw := []byte(values[len(values)-1])
		if !json.Valid(raw) {
			encoded, err := json.Marshal(values[len(values)-1])
			if err != nil {
				return nil, err
			}
			raw = encoded
		}
		filters[filterKey] = config.RawValue(raw)
	}
	return filters, nil
}

func querySorts(ctx *context.APIContext) ([]config.Sort, error) {
	values := ctx.Req.URL.Query()["sort"]
	sorts := make([]config.Sort, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid sort %q", value)
		}
		direction := config.SortDirection(parts[1])
		if direction != config.SortAscending && direction != config.SortDescending {
			return nil, fmt.Errorf("invalid sort direction %q", parts[1])
		}
		sorts = append(sorts, config.Sort{FieldKey: parts[0], Direction: direction})
	}
	return sorts, nil
}

func parsePositiveInt64(value, name string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}
