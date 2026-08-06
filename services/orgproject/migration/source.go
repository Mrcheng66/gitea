// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
)

const sqliteDriverName = "sqlite3"

var requiredTables = []string{"project_profiles", "project_followers", "project_audit_events"}

type legacyProfile struct {
	RepoID      int64
	Stage       string
	Progress    int
	OwnerUserID sql.NullInt64
	StartDate   sql.NullString
	TargetDate  sql.NullString
	Risk        string
	Summary     string
	Version     int64
	CreatedAt   string
	UpdatedAt   string
	UpdatedBy   int64
	Followers   []int64
}

type legacyAudit struct {
	ID            int64
	RepoID        int64
	ActorUserID   int64
	RequestID     string
	ChangedFields string
	BeforeValue   sql.NullString
	AfterValue    string
	CreatedAt     string
}

type legacyData struct {
	Profiles  []legacyProfile
	Followers int
	Audits    []legacyAudit
}

type source struct {
	db *sql.DB
}

func openSource(ctx context.Context, path string) (*source, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve Workbench database path: %w", err)
	}
	databaseURL := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	query := databaseURL.Query()
	query.Set("mode", "ro")
	databaseURL.RawQuery = query.Encode()
	database, err := sql.Open(sqliteDriverName, databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("open Workbench database: %w", err)
	}
	result := &source{db: database}
	if err := result.validateSchema(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return result, nil
}

func (s *source) close() error {
	return s.db.Close()
}

func (s *source) validateSchema(ctx context.Context) error {
	for _, table := range requiredTables {
		var name string
		err := s.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Workbench database is missing required table %q", table)
			}
			return fmt.Errorf("inspect Workbench schema: %w", err)
		}
	}
	return nil
}

func (s *source) load(ctx context.Context) (*legacyData, error) {
	profiles, err := s.loadProfiles(ctx)
	if err != nil {
		return nil, err
	}
	followers, err := s.loadFollowers(ctx, profiles)
	if err != nil {
		return nil, err
	}
	audits, err := s.loadAudits(ctx)
	if err != nil {
		return nil, err
	}
	return &legacyData{Profiles: profiles, Followers: followers, Audits: audits}, nil
}

func (s *source) loadProfiles(ctx context.Context) ([]legacyProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT repo_id, stage, progress, owner_user_id, start_date, target_date, risk, summary,
       version, created_at, updated_at, updated_by
FROM project_profiles
ORDER BY repo_id`)
	if err != nil {
		return nil, fmt.Errorf("read Workbench project profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]legacyProfile, 0)
	for rows.Next() {
		var profile legacyProfile
		if err := rows.Scan(
			&profile.RepoID, &profile.Stage, &profile.Progress, &profile.OwnerUserID,
			&profile.StartDate, &profile.TargetDate, &profile.Risk, &profile.Summary,
			&profile.Version, &profile.CreatedAt, &profile.UpdatedAt, &profile.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan Workbench project profile: %w", err)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workbench project profiles: %w", err)
	}
	return profiles, nil
}

func (s *source) loadFollowers(ctx context.Context, profiles []legacyProfile) (int, error) {
	byRepository := make(map[int64]*legacyProfile, len(profiles))
	for i := range profiles {
		byRepository[profiles[i].RepoID] = &profiles[i]
	}

	rows, err := s.db.QueryContext(ctx, "SELECT repo_id, user_id FROM project_followers ORDER BY repo_id, user_id")
	if err != nil {
		return 0, fmt.Errorf("read Workbench project followers: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var repoID, userID int64
		if err := rows.Scan(&repoID, &userID); err != nil {
			return 0, fmt.Errorf("scan Workbench project follower: %w", err)
		}
		profile := byRepository[repoID]
		if profile == nil {
			return 0, fmt.Errorf("Workbench follower references missing profile for repository %d", repoID)
		}
		profile.Followers = append(profile.Followers, userID)
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate Workbench project followers: %w", err)
	}
	return count, nil
}

func (s *source) loadAudits(ctx context.Context) ([]legacyAudit, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, repo_id, actor_user_id, request_id, changed_fields, before_value, after_value, created_at
FROM project_audit_events
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("read Workbench audit events: %w", err)
	}
	defer rows.Close()

	audits := make([]legacyAudit, 0)
	for rows.Next() {
		var audit legacyAudit
		if err := rows.Scan(
			&audit.ID, &audit.RepoID, &audit.ActorUserID, &audit.RequestID,
			&audit.ChangedFields, &audit.BeforeValue, &audit.AfterValue, &audit.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan Workbench audit event: %w", err)
		}
		audits = append(audits, audit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workbench audit events: %w", err)
	}
	return audits, nil
}
