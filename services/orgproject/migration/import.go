// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/modules/json"
	"gitea.dev/modules/timeutil"
	orgproject_service "gitea.dev/services/orgproject"
	"gitea.dev/services/orgproject/config"
	project_service "gitea.dev/services/orgproject/project"
)

type ErrBlocked struct {
	Count int
}

func (err ErrBlocked) Error() string {
	return fmt.Sprintf("Workbench migration is blocked by %d preflight issue(s)", err.Count)
}

func Import(ctx context.Context, options Options) (*Report, error) {
	plan, err := buildPlan(ctx, options, "import")
	if err != nil {
		return nil, err
	}
	if plan.Report.HasBlockers() {
		return plan.Report, ErrBlocked{Count: plan.Report.Summary.Blocked}
	}
	if err := ensureDefaultConfiguration(ctx, plan.OwnerID, plan.Actor.ID); err != nil {
		return plan.Report, err
	}
	if err := orgproject_service.ReplaceEditorTeams(ctx, plan.OwnerID, plan.Actor.ID, plan.TeamIDs); err != nil {
		return plan.Report, fmt.Errorf("configure organization project editor teams: %w", err)
	}

	projectsByRepository := make(map[int64]int64, len(plan.Profiles))
	for i := range plan.Profiles {
		profile := &plan.Profiles[i]
		if profile.Skip {
			projectsByRepository[profile.Profile.RepoID] = profile.ProjectID
			continue
		}
		project, err := importProfile(ctx, plan, profile)
		if err != nil {
			return plan.Report, fmt.Errorf("import Workbench profile for repository %d: %w", profile.Profile.RepoID, err)
		}
		profile.ProjectID = project.ID
		projectsByRepository[profile.Profile.RepoID] = project.ID
	}

	for i := range plan.Audits {
		audit := &plan.Audits[i]
		if audit.Skip {
			continue
		}
		projectID := projectsByRepository[audit.Audit.RepoID]
		if err := importAudit(ctx, projectID, audit); err != nil {
			return plan.Report, fmt.Errorf("import Workbench audit %d: %w", audit.Audit.ID, err)
		}
	}
	return plan.Report, nil
}

func ensureDefaultConfiguration(ctx context.Context, ownerID, actorID int64) error {
	pointer, err := config.GetPointer(ctx, ownerID)
	if err != nil {
		var uninitialized config.ErrConfigUninitialized
		if !errors.As(err, &uninitialized) {
			return err
		}
		_, pointer, err = config.InitializeDefaultDraft(ctx, ownerID, actorID)
		if err != nil {
			return fmt.Errorf("initialize default organization project configuration: %w", err)
		}
	}
	if pointer.PublishedVersionID != 0 {
		return nil
	}
	if pointer.DraftVersionID == 0 {
		return errors.New("organization project configuration has no draft to publish")
	}
	if _, err := config.PublishDraft(ctx, ownerID, actorID, pointer.Version); err != nil {
		return fmt.Errorf("publish default organization project configuration: %w", err)
	}
	return nil
}

func importProfile(ctx context.Context, plan *importPlan, item *profilePlan) (*orgproject_model.Project, error) {
	profile := item.Profile
	repository, err := repo_model.GetRepositoryByID(ctx, profile.RepoID)
	if err != nil {
		return nil, err
	}
	values, err := profileValues(profile)
	if err != nil {
		return nil, err
	}

	var project *orgproject_model.Project
	err = db.WithTx(ctx, func(ctx context.Context) error {
		project, err = project_service.Create(ctx, project_service.CreateOptions{
			OwnerID: plan.OwnerID, Actor: plan.Actor, Slug: item.Slug, Name: repository.Name, Description: repository.Description,
			Values: values,
			Repositories: []project_service.RepositoryInput{{
				RepositoryID: repository.ID,
				Role:         orgproject_model.RepositoryRolePrimary,
			}},
			RequestID: profileRequestID(profile.RepoID), Source: orgproject_model.ChangeSourceLegacyImport,
		})
		if err != nil {
			return err
		}

		createdAt, _ := time.Parse(time.RFC3339Nano, profile.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339Nano, profile.UpdatedAt)
		version := max(profile.Version, 1)
		if _, err := db.GetEngine(ctx).Exec(
			"UPDATE org_project SET version = ?, created_unix = ?, updated_unix = ? WHERE id = ?",
			version, timeutil.TimeStamp(createdAt.Unix()), timeutil.TimeStamp(updatedAt.Unix()), project.ID,
		); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Exec(
			"UPDATE org_project_change_log SET created_unix = ? WHERE request_id = ?",
			timeutil.TimeStamp(createdAt.Unix()), profileRequestID(profile.RepoID),
		); err != nil {
			return err
		}
		project.Version = version
		project.CreatedUnix = timeutil.TimeStamp(createdAt.Unix())
		project.UpdatedUnix = timeutil.TimeStamp(updatedAt.Unix())
		return nil
	})
	return project, err
}

func profileValues(profile legacyProfile) (map[string]config.RawValue, error) {
	values := make(map[string]config.RawValue, 8)
	for key, value := range map[string]any{
		"stage": profile.Stage, "progress": profile.Progress, "followers": profile.Followers,
		"risk": profile.Risk, "summary": profile.Summary,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		values[key] = config.RawValue(encoded)
	}
	if profile.OwnerUserID.Valid {
		encoded, _ := json.Marshal(profile.OwnerUserID.Int64)
		values["owner"] = config.RawValue(encoded)
	}
	if profile.StartDate.Valid && profile.StartDate.String != "" {
		encoded, _ := json.Marshal(profile.StartDate.String)
		values["start_date"] = config.RawValue(encoded)
	}
	if profile.TargetDate.Valid && profile.TargetDate.String != "" {
		encoded, _ := json.Marshal(profile.TargetDate.String)
		values["target_date"] = config.RawValue(encoded)
	}
	return values, nil
}

type legacyAuditSnapshot struct {
	LegacyRequestID string          `json:"legacy_request_id"`
	Value           config.RawValue `json:"value"`
}

func importAudit(ctx context.Context, projectID int64, item *auditPlan) error {
	marker := auditRequestID(item.Audit.ID)
	_, exists, err := findChange(ctx, marker)
	if err != nil || exists {
		return err
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, item.Audit.CreatedAt)
	before := config.RawValue("null")
	if item.Audit.BeforeValue.Valid {
		before = config.RawValue(item.Audit.BeforeValue.String)
	}
	beforeJSON, err := json.Marshal(legacyAuditSnapshot{LegacyRequestID: item.Audit.RequestID, Value: before})
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(legacyAuditSnapshot{
		LegacyRequestID: item.Audit.RequestID,
		Value:           config.RawValue(item.Audit.AfterValue),
	})
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).Insert(&orgproject_model.ChangeLog{
		ProjectID: projectID, ActorID: item.ActorID, RequestID: marker,
		ChangedFields: item.Audit.ChangedFields, BeforeValue: string(beforeJSON), AfterValue: string(afterJSON),
		Source: orgproject_model.ChangeSourceLegacyImport, CreatedUnix: timeutil.TimeStamp(createdAt.Unix()),
	})
	return err
}
