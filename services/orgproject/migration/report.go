// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitea.dev/modules/json"
)

type ReportSummary struct {
	Profiles       int `json:"profiles"`
	Followers      int `json:"followers"`
	Audits         int `json:"audits"`
	ProjectsImport int `json:"projects_import"`
	ProjectsSkip   int `json:"projects_skip"`
	AuditsImport   int `json:"audits_import"`
	AuditsSkip     int `json:"audits_skip"`
	Blocked        int `json:"blocked"`
	Warnings       int `json:"warnings"`
}

type ReportItem struct {
	Kind      string `json:"kind"`
	LegacyID  int64  `json:"legacy_id,omitempty"`
	RepoID    int64  `json:"repo_id,omitempty"`
	Action    string `json:"action"`
	Reason    string `json:"reason,omitempty"`
	ProjectID int64  `json:"project_id,omitempty"`
	Slug      string `json:"slug,omitempty"`
}

type Report struct {
	Mode         string        `json:"mode"`
	Database     string        `json:"database"`
	Organization string        `json:"organization"`
	Actor        string        `json:"actor"`
	GeneratedAt  time.Time     `json:"generated_at"`
	Summary      ReportSummary `json:"summary"`
	Items        []ReportItem  `json:"items"`
}

// HasBlockers reports whether preflight found any issues that prevent import.
func (r *Report) HasBlockers() bool {
	return r.Summary.Blocked > 0
}

func (r *Report) add(item ReportItem) {
	r.Items = append(r.Items, item)
	switch item.Action {
	case "block":
		r.Summary.Blocked++
	case "warn":
		r.Summary.Warnings++
	}
}

func WriteReport(path string, report *Report) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode migration report: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create migration report directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write migration report: %w", err)
	}
	return nil
}
