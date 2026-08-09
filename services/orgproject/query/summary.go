// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package query

import (
	"context"
	"errors"
	"time"

	"gitea.dev/models/db"
	"gitea.dev/services/orgproject/config"
)

// ProjectSummary contains organization-wide health indicators for active projects.
type ProjectSummary struct {
	Active          int64   `json:"active"`
	Blocked         int64   `json:"blocked"`
	Overdue         int64   `json:"overdue"`
	DueSoon         int64   `json:"due_soon"`
	AverageProgress float64 `json:"average_progress"`
}

// Summary calculates risk indicators across all active projects in an organization.
func Summary(ctx context.Context, schema config.Schema, ownerID int64, now time.Time) (*ProjectSummary, error) {
	ast, err := Parse(schema)
	if err != nil {
		return nil, err
	}
	if ownerID <= 0 {
		return nil, errors.New("organization owner ID must be positive")
	}

	start := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	joins := ""
	joinArgs := make([]any, 0, 3)
	riskValue := "NULL"
	targetValue := "NULL"
	progressValue := "NULL"
	if _, ok := ast.fields["risk"]; ok {
		joins += " LEFT JOIN org_project_field_value summary_risk ON summary_risk.project_id = org_project.id AND summary_risk.field_key = ?"
		joinArgs = append(joinArgs, "risk")
		riskValue = "summary_risk.value_text"
	}
	if _, ok := ast.fields["target_date"]; ok {
		joins += " LEFT JOIN org_project_field_value summary_target ON summary_target.project_id = org_project.id AND summary_target.field_key = ?"
		joinArgs = append(joinArgs, "target_date")
		targetValue = "summary_target.value_time"
	}
	if _, ok := ast.fields["progress"]; ok {
		joins += " LEFT JOIN org_project_field_value summary_progress ON summary_progress.project_id = org_project.id AND summary_progress.field_key = ?"
		joinArgs = append(joinArgs, "progress")
		progressValue = "summary_progress.value_number"
	}
	args := []any{"blocked", start.Unix(), start.Unix(), end.Unix()}
	args = append(args, joinArgs...)
	args = append(args, ownerID, "active")
	sql := "SELECT COUNT(DISTINCT org_project.id) AS active, " +
		"COUNT(DISTINCT CASE WHEN " + riskValue + " = ? THEN org_project.id END) AS blocked, " +
		"COUNT(DISTINCT CASE WHEN " + targetValue + " < ? THEN org_project.id END) AS overdue, " +
		"COUNT(DISTINCT CASE WHEN " + targetValue + " >= ? AND " + targetValue + " < ? THEN org_project.id END) AS due_soon, " +
		"COALESCE(AVG(" + progressValue + "), 0) AS average_progress " +
		"FROM org_project" + joins + " WHERE org_project.owner_id = ? AND org_project.lifecycle = ?"

	rows, err := db.GetEngine(ctx).QueryInterface(append([]any{sql}, args...)...)
	if err != nil {
		return nil, err
	}
	result := &ProjectSummary{}
	if len(rows) == 0 {
		return result, nil
	}
	row := rows[0]
	if result.Active, err = integerValue(row["active"]); err != nil {
		return nil, err
	}
	if result.Blocked, err = integerValue(row["blocked"]); err != nil {
		return nil, err
	}
	if result.Overdue, err = integerValue(row["overdue"]); err != nil {
		return nil, err
	}
	if result.DueSoon, err = integerValue(row["due_soon"]); err != nil {
		return nil, err
	}
	result.AverageProgress, err = numericValue(row["average_progress"])
	return result, err
}

func integerValue(value any) (int64, error) {
	number, err := numericValue(value)
	return int64(number), err
}
