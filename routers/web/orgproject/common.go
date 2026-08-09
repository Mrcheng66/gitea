// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/modules/json"
	"gitea.dev/modules/templates"
	"gitea.dev/services/context"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
)

const (
	tplOrgProjectLanding    templates.TplName = "orgproject/landing"
	tplOrgProjectDashboard  templates.TplName = "orgproject/dashboard"
	tplOrgProjectList       templates.TplName = "orgproject/list"
	tplOrgProjectNew        templates.TplName = "orgproject/new"
	tplOrgProjectView       templates.TplName = "orgproject/view"
	tplOrgProjectSettings   templates.TplName = "orgproject/settings"
	tplOrgProjectHistory    templates.TplName = "orgproject/history"
	tplOrgProjectActivity   templates.TplName = "orgproject/activity"
	tplOrgProjectRepository templates.TplName = "orgproject/repository"
)

type fieldDisplay struct {
	Key     string
	Label   string
	Value   string
	Type    config.FieldType
	Raw     string
	Number  *float64
	Members []memberDisplay
}

type repositoryDisplay struct {
	Link       *orgproject_model.Repository
	Repository *repo_model.Repository
}

type memberDisplay struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Initial  string `json:"initial"`
}

func loadProjectMembers(ctx *context.Context) ([]memberDisplay, map[int64]memberDisplay, error) {
	members, _, err := ctx.Org.Organization.GetMembers(ctx, ctx.Doer)
	if err != nil {
		return nil, nil, err
	}
	displays := make([]memberDisplay, 0, len(members))
	byID := make(map[int64]memberDisplay, len(members))
	for _, member := range members {
		displayName := member.FullName
		if displayName == "" {
			displayName = member.Name
		}
		initial := "?"
		if runes := []rune(strings.TrimSpace(displayName)); len(runes) > 0 {
			initial = strings.ToUpper(string(runes[0]))
		}
		display := memberDisplay{ID: member.ID, Name: member.Name, FullName: member.FullName, Initial: initial}
		displays = append(displays, display)
		byID[member.ID] = display
	}
	return displays, byID, nil
}

func setPageData(ctx *context.Context, active string) {
	ctx.Data["Title"] = ctx.Tr("org_project.title")
	ctx.Data["PageIsOrgProjects"] = true
	ctx.Data["OrgProjectActive"] = active
	ctx.Data["OrgProjectBaseLink"] = ctx.Org.OrgLink + "/projects"
}

func setProjectFormData(ctx *context.Context, schema config.Schema, values string) error {
	members, _, err := loadProjectMembers(ctx)
	if err != nil {
		return err
	}
	ctx.Data["OrgProjectSchema"] = schema
	ctx.Data["OrgProjectValues"] = values
	ctx.Data["OrgProjectMembers"] = members
	return nil
}

func isConfigUninitialized(err error) bool {
	var target config.ErrConfigUninitialized
	return errors.As(err, &target)
}

func decodeValues(input string) (map[string]config.RawValue, error) {
	if input == "" {
		return map[string]config.RawValue{}, nil
	}
	values := map[string]config.RawValue{}
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil, fmt.Errorf("values: %w", err)
	}
	return values, nil
}

func encodeValues(schema config.Schema, values []*orgproject_model.FieldValue) (string, error) {
	fieldTypes := make(map[string]config.FieldType, len(schema.Fields))
	for _, field := range schema.Fields {
		fieldTypes[field.Key] = field.Type
	}
	encoded := make(map[string]json.Value, len(values))
	for _, value := range values {
		fieldType, ok := fieldTypes[value.FieldKey]
		if !ok {
			continue
		}
		raw, err := fieldValueToRaw(fieldType, value)
		if err != nil {
			return "", err
		}
		encoded[value.FieldKey] = raw
	}
	payload, err := json.Marshal(encoded)
	return string(payload), err
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
	default:
		return nil, fmt.Errorf("unsupported organization project field type %q", fieldType)
	}
	return json.Marshal(raw)
}

func buildFieldDisplays(schema config.Schema, values []*orgproject_model.FieldValue, members map[int64]memberDisplay) []fieldDisplay {
	valuesByKey := make(map[string]*orgproject_model.FieldValue, len(values))
	for _, value := range values {
		valuesByKey[value.FieldKey] = value
	}
	displays := make([]fieldDisplay, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		if field.Archived {
			continue
		}
		displays = append(displays, buildFieldDisplay(field, valuesByKey[field.Key], members))
	}
	return displays
}

func buildFieldDisplay(field config.Field, value *orgproject_model.FieldValue, members map[int64]memberDisplay) fieldDisplay {
	display := fieldDisplay{Key: field.Key, Label: field.Label, Type: field.Type, Value: formatFieldValue(field, value, members)}
	if value == nil {
		return display
	}
	if value.ValueText != nil {
		display.Raw = *value.ValueText
	}
	if value.ValueNumber != nil {
		display.Number = value.ValueNumber
	}
	switch field.Type {
	case config.FieldTypeMember:
		if value.ValueUserID != nil {
			if member, ok := members[*value.ValueUserID]; ok {
				display.Members = []memberDisplay{member}
			}
		}
	case config.FieldTypeMemberArray:
		if value.ValueJSON != nil {
			var memberIDs []int64
			if json.Unmarshal([]byte(*value.ValueJSON), &memberIDs) == nil {
				for _, memberID := range memberIDs {
					if member, ok := members[memberID]; ok {
						display.Members = append(display.Members, member)
					}
				}
			}
		}
	}
	return display
}

func formatFieldValue(field config.Field, value *orgproject_model.FieldValue, members map[int64]memberDisplay) string {
	if value == nil {
		return "—"
	}
	switch field.Type {
	case config.FieldTypeShortText, config.FieldTypeLongText:
		if value.ValueText != nil {
			return *value.ValueText
		}
	case config.FieldTypeSingle:
		if value.ValueText != nil {
			for _, option := range field.Options {
				if option.Key == *value.ValueText {
					return option.Label
				}
			}
			return *value.ValueText
		}
	case config.FieldTypeInteger:
		if value.ValueNumber != nil {
			return strconv.FormatInt(int64(*value.ValueNumber), 10)
		}
	case config.FieldTypeDecimal:
		if value.ValueNumber != nil {
			return strconv.FormatFloat(*value.ValueNumber, 'f', -1, 64)
		}
	case config.FieldTypePercent:
		if value.ValueNumber != nil {
			return strconv.FormatFloat(*value.ValueNumber, 'f', -1, 64) + "%"
		}
	case config.FieldTypeDate:
		if value.ValueTime != nil {
			return value.ValueTime.AsTime().Format(time.DateOnly)
		}
	case config.FieldTypeDateTime:
		if value.ValueTime != nil {
			return value.ValueTime.AsTime().Format(time.RFC3339)
		}
	case config.FieldTypeBoolean:
		if value.ValueBool != nil {
			return strconv.FormatBool(*value.ValueBool)
		}
	case config.FieldTypeMember:
		if value.ValueUserID != nil {
			if member, ok := members[*value.ValueUserID]; ok {
				if member.FullName != "" {
					return member.FullName
				}
				return member.Name
			}
			return fmt.Sprintf("@%d", *value.ValueUserID)
		}
	case config.FieldTypeMemberArray:
		if value.ValueJSON != nil {
			var memberIDs []int64
			if json.Unmarshal([]byte(*value.ValueJSON), &memberIDs) == nil {
				names := make([]string, 0, len(memberIDs))
				for _, memberID := range memberIDs {
					member, ok := members[memberID]
					if !ok {
						continue
					}
					if member.FullName != "" {
						names = append(names, member.FullName)
					} else {
						names = append(names, member.Name)
					}
				}
				if len(names) > 0 {
					return strings.Join(names, "、")
				}
			}
		}
	case config.FieldTypeMulti:
		if value.ValueJSON != nil {
			return *value.ValueJSON
		}
	}
	return "—"
}

func loadRepositoryDisplays(ctx *context.Context, links []*orgproject_model.Repository) ([]repositoryDisplay, error) {
	displays := make([]repositoryDisplay, 0, len(links))
	for _, link := range links {
		repository, err := repo_model.GetRepositoryByID(ctx, link.RepositoryID)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				continue
			}
			return nil, err
		}
		displays = append(displays, repositoryDisplay{Link: link, Repository: repository})
	}
	return displays, nil
}

func writeProjectError(ctx *context.Context, err error, templateName templates.TplName) {
	switch {
	case project_service.IsErrNotFound(err):
		ctx.NotFound(err)
	case project_service.IsErrForbidden(err):
		ctx.HTTPError(http.StatusForbidden)
	case project_service.IsErrConflict(err):
		ctx.Flash.Error(ctx.Tr("org_project.error.conflict"))
		ctx.Redirect(ctx.Req.URL.Path)
	case project_service.IsValidationErrors(err), project_service.IsErrRepositoryNotVisible(err):
		ctx.Data["OrgProjectError"] = err.Error()
		ctx.HTML(http.StatusUnprocessableEntity, templateName)
	default:
		ctx.ServerError("OrgProject", err)
	}
}
