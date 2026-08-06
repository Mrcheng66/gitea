// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"reflect"
	"sort"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/json"
	"gitea.dev/modules/timeutil"
)

type fieldValueSnapshot struct {
	Key    string              `json:"key"`
	Text   *string             `json:"text,omitempty"`
	Number *float64            `json:"number,omitempty"`
	Time   *timeutil.TimeStamp `json:"time,omitempty"`
	Bool   *bool               `json:"bool,omitempty"`
	UserID *int64              `json:"user_id,omitempty"`
	JSON   *string             `json:"json,omitempty"`
}

type repositorySnapshot struct {
	RepositoryID int64                           `json:"repository_id"`
	Role         orgproject_model.RepositoryRole `json:"role"`
}

type projectSnapshot struct {
	Slug         string                     `json:"slug,omitempty"`
	Name         string                     `json:"name,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Lifecycle    orgproject_model.Lifecycle `json:"lifecycle,omitempty"`
	Version      int64                      `json:"version,omitempty"`
	Values       []fieldValueSnapshot       `json:"values,omitempty"`
	Repositories []repositorySnapshot       `json:"repositories,omitempty"`
}

func loadSnapshot(ctx context.Context, project *orgproject_model.Project) (projectSnapshot, error) {
	values := make([]*orgproject_model.FieldValue, 0)
	if err := db.GetEngine(ctx).Where("project_id = ?", project.ID).Asc("field_key").Find(&values); err != nil {
		return projectSnapshot{}, err
	}
	valueSnapshots := make([]fieldValueSnapshot, 0, len(values))
	for _, value := range values {
		valueSnapshots = append(valueSnapshots, fieldValueSnapshot{
			Key: value.FieldKey, Text: value.ValueText, Number: value.ValueNumber, Time: value.ValueTime,
			Bool: value.ValueBool, UserID: value.ValueUserID, JSON: value.ValueJSON,
		})
	}

	repositories := make([]*orgproject_model.Repository, 0)
	if err := db.GetEngine(ctx).Where("project_id = ?", project.ID).Asc("repository_id").Find(&repositories); err != nil {
		return projectSnapshot{}, err
	}
	repositorySnapshots := make([]repositorySnapshot, 0, len(repositories))
	for _, repository := range repositories {
		repositorySnapshots = append(repositorySnapshots, repositorySnapshot{
			RepositoryID: repository.RepositoryID,
			Role:         repository.Role,
		})
	}

	return projectSnapshot{
		Slug: project.Slug, Name: project.Name, Description: project.Description,
		Lifecycle: project.Lifecycle, Version: project.Version,
		Values: valueSnapshots, Repositories: repositorySnapshots,
	}, nil
}

func insertChangeLog(ctx context.Context, projectID, actorID int64, requestID string, source orgproject_model.ChangeSource, before, after projectSnapshot) error {
	changedFields := changedFields(before, after)
	changedJSON, err := json.Marshal(changedFields)
	if err != nil {
		return err
	}
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).Insert(&orgproject_model.ChangeLog{
		ProjectID: projectID, ActorID: actorID, RequestID: requestID,
		ChangedFields: string(changedJSON), BeforeValue: string(beforeJSON), AfterValue: string(afterJSON), Source: source,
	})
	return err
}

func changedFields(before, after projectSnapshot) []string {
	changed := make([]string, 0)
	if before.Slug != after.Slug {
		changed = append(changed, "slug")
	}
	if before.Name != after.Name {
		changed = append(changed, "name")
	}
	if before.Description != after.Description {
		changed = append(changed, "description")
	}
	if before.Lifecycle != after.Lifecycle {
		changed = append(changed, "lifecycle")
	}

	beforeValues := make(map[string]fieldValueSnapshot, len(before.Values))
	for _, value := range before.Values {
		beforeValues[value.Key] = value
	}
	afterValues := make(map[string]fieldValueSnapshot, len(after.Values))
	for _, value := range after.Values {
		afterValues[value.Key] = value
	}
	keys := make(map[string]struct{}, len(beforeValues)+len(afterValues))
	for key := range beforeValues {
		keys[key] = struct{}{}
	}
	for key := range afterValues {
		keys[key] = struct{}{}
	}
	for key := range keys {
		if !reflect.DeepEqual(beforeValues[key], afterValues[key]) {
			changed = append(changed, "values."+key)
		}
	}
	if !reflect.DeepEqual(before.Repositories, after.Repositories) {
		changed = append(changed, "repositories")
	}
	sort.Strings(changed)
	return changed
}

func replayProject(ctx context.Context, requestID string, expectedOwnerID, expectedProjectID, actorID int64, source orgproject_model.ChangeSource) (*orgproject_model.Project, error) {
	change := &orgproject_model.ChangeLog{RequestID: requestID}
	has, err := db.GetEngine(ctx).Get(change)
	if err != nil || !has {
		return nil, err
	}
	if expectedProjectID != 0 && change.ProjectID != expectedProjectID || change.ActorID != actorID || change.Source != source {
		return nil, ValidationErrors{"request_id": "was already used for another operation"}
	}
	project, err := getProject(ctx, expectedOwnerID, change.ProjectID)
	if IsErrNotFound(err) {
		return nil, ValidationErrors{"request_id": "was already used for another organization"}
	}
	return project, err
}
