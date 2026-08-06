// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"context"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/services/orgproject/config"
)

// ListItem contains a project and the configured list-view field values.
type ListItem struct {
	Project *orgproject_model.Project
	Values  map[string]*orgproject_model.FieldValue
}

// ListResult contains one stable page of projects.
type ListResult struct {
	Items    []ListItem
	Total    int64
	Page     int
	PageSize int
}

// List executes a typed project list query against the published schema.
func List(ctx context.Context, schema config.Schema, opts ListOptions) (*ListResult, error) {
	ast, err := Parse(schema)
	if err != nil {
		return nil, err
	}
	compiled, err := CompileList(ast, opts)
	if err != nil {
		return nil, err
	}

	projects := make([]*orgproject_model.Project, 0, compiled.PageSize)
	if err := db.GetEngine(ctx).SQL(compiled.Data.SQL, compiled.Data.Args...).Find(&projects); err != nil {
		return nil, err
	}
	var total int64
	if _, err := db.GetEngine(ctx).SQL(compiled.Count.SQL, compiled.Count.Args...).Get(&total); err != nil {
		return nil, err
	}

	values, err := loadListValues(ctx, ast, projects)
	if err != nil {
		return nil, err
	}
	items := make([]ListItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, ListItem{Project: project, Values: values[project.ID]})
	}
	return &ListResult{Items: items, Total: total, Page: compiled.Page, PageSize: compiled.PageSize}, nil
}

func loadListValues(ctx context.Context, ast *AST, projects []*orgproject_model.Project) (map[int64]map[string]*orgproject_model.FieldValue, error) {
	result := make(map[int64]map[string]*orgproject_model.FieldValue, len(projects))
	if len(projects) == 0 || len(ast.columns) == 0 {
		return result, nil
	}
	projectIDs := make([]int64, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
		result[project.ID] = make(map[string]*orgproject_model.FieldValue, len(ast.columns))
	}
	fieldKeys := make([]string, 0, len(ast.columns))
	for _, column := range ast.columns {
		fieldKeys = append(fieldKeys, column.key)
	}
	values := make([]*orgproject_model.FieldValue, 0, len(projects)*len(ast.columns))
	if err := db.GetEngine(ctx).
		In("project_id", projectIDs).
		In("field_key", fieldKeys).
		Asc("project_id", "field_key").
		Find(&values); err != nil {
		return nil, err
	}
	for _, value := range values {
		if projectValues, ok := result[value.ProjectID]; ok {
			projectValues[value.FieldKey] = value
		}
	}
	return result, nil
}
