// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"errors"
	"net/http"
	"time"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/json"
	api "gitea.dev/modules/structs"
	"gitea.dev/services/context"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
)

func writeProjectError(ctx *context.APIContext, err error) {
	var validation project_service.ValidationErrors
	var notFound project_service.ErrNotFound
	var forbidden project_service.ErrForbidden
	var conflict project_service.ErrConflict
	var hiddenRepository project_service.ErrRepositoryNotVisible
	switch {
	case errors.As(err, &validation):
		ctx.APIError(http.StatusUnprocessableEntity, validation.Error())
	case errors.As(err, &notFound), errors.As(err, &hiddenRepository):
		ctx.APIErrorNotFound()
	case errors.As(err, &forbidden):
		ctx.APIError(http.StatusForbidden, err.Error())
	case errors.As(err, &conflict):
		ctx.APIError(http.StatusConflict, err.Error())
	default:
		ctx.APIErrorInternal(err)
	}
}

func writeConfigError(ctx *context.APIContext, err error) {
	var uninitialized config.ErrConfigUninitialized
	switch {
	case config.IsErrConfigConflict(err):
		ctx.APIError(http.StatusConflict, err.Error())
	case errors.As(err, &uninitialized):
		ctx.APIErrorNotFound()
	case config.IsErrConfigValidation(err):
		ctx.APIError(http.StatusUnprocessableEntity, err.Error())
	default:
		ctx.APIErrorInternal(err)
	}
}

func detailToAPI(schema config.Schema, detail *project_service.Detail) (*api.OrgProject, error) {
	values, err := fieldValuesToAPI(schema, detail.Values)
	if err != nil {
		return nil, err
	}
	repositories := make([]api.OrgProjectRepository, 0, len(detail.Repositories))
	for _, repository := range detail.Repositories {
		repositories = append(repositories, api.OrgProjectRepository{
			RepositoryID: repository.RepositoryID,
			Role:         string(repository.Role),
		})
	}
	return projectToAPI(detail.Project, values, repositories), nil
}

func projectToAPI(project *orgproject_model.Project, values map[string]json.Value, repositories []api.OrgProjectRepository) *api.OrgProject {
	if values == nil {
		values = map[string]json.Value{}
	}
	if repositories == nil {
		repositories = []api.OrgProjectRepository{}
	}
	return &api.OrgProject{
		ID: project.ID, OwnerID: project.OwnerID, Slug: project.Slug, Name: project.Name,
		Description: project.Description, Lifecycle: string(project.Lifecycle), Version: project.Version,
		Values: values, Repositories: repositories,
		CreatedAt: project.CreatedUnix.AsTime(), UpdatedAt: project.UpdatedUnix.AsTime(),
	}
}

func fieldValuesToAPI(schema config.Schema, values []*orgproject_model.FieldValue) (map[string]json.Value, error) {
	fields := make(map[string]config.FieldType, len(schema.Fields))
	for _, field := range schema.Fields {
		fields[field.Key] = field.Type
	}
	result := make(map[string]json.Value, len(values))
	for _, value := range values {
		fieldType, ok := fields[value.FieldKey]
		if !ok {
			continue
		}
		raw, err := fieldValueToRaw(fieldType, value)
		if err != nil {
			return nil, err
		}
		result[value.FieldKey] = raw
	}
	return result, nil
}

func fieldValueToRaw(fieldType config.FieldType, value *orgproject_model.FieldValue) (json.Value, error) {
	var raw any
	switch fieldType {
	case config.FieldTypeShortText, config.FieldTypeLongText, config.FieldTypeSingle:
		raw = value.ValueText
	case config.FieldTypeInteger:
		if value.ValueNumber != nil {
			raw = int64(*value.ValueNumber)
		}
	case config.FieldTypeDecimal, config.FieldTypePercent:
		raw = value.ValueNumber
	case config.FieldTypeDate:
		if value.ValueTime != nil {
			raw = value.ValueTime.AsTime().Format(time.DateOnly)
		}
	case config.FieldTypeDateTime:
		if value.ValueTime != nil {
			raw = value.ValueTime.AsTime().Format(time.RFC3339)
		}
	case config.FieldTypeBoolean:
		raw = value.ValueBool
	case config.FieldTypeMember:
		raw = value.ValueUserID
	case config.FieldTypeMulti, config.FieldTypeMemberArray:
		if value.ValueJSON == nil {
			return json.Value("null"), nil
		}
		return json.Value(*value.ValueJSON), nil
	}
	return json.Marshal(raw)
}

func rawValuesToConfig(values map[string]json.Value) map[string]config.RawValue {
	result := make(map[string]config.RawValue, len(values))
	for key, value := range values {
		result[key] = config.RawValue(value)
	}
	return result
}

func repositoriesToService(repositories []api.OrgProjectRepository) []project_service.RepositoryInput {
	result := make([]project_service.RepositoryInput, 0, len(repositories))
	for _, repository := range repositories {
		result = append(result, project_service.RepositoryInput{
			RepositoryID: repository.RepositoryID,
			Role:         orgproject_model.RepositoryRole(repository.Role),
		})
	}
	return result
}

func configVersionToAPI(version *orgproject_model.ConfigVersion, pointerVersion int64) api.OrgProjectConfigVersion {
	result := api.OrgProjectConfigVersion{
		ID: version.ID, Version: version.Version, State: string(version.State), Schema: json.Value(version.Payload),
		PointerVersion: pointerVersion, CreatedBy: version.CreatedBy, CreatedAt: version.CreatedUnix.AsTime(), PublishedBy: version.PublishedBy,
	}
	if !version.PublishedUnix.IsZero() {
		publishedAt := version.PublishedUnix.AsTime()
		result.PublishedAt = &publishedAt
	}
	return result
}
