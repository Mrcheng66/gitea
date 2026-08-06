// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gitea.dev/models/db"
	org_model "gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
)

type Options struct {
	DatabasePath string
	Organization string
	Actor        string
	EditorTeams  []string
}

type profilePlan struct {
	Profile   legacyProfile
	ProjectID int64
	Slug      string
	Skip      bool
	Blocked   bool
}

type auditPlan struct {
	Audit   legacyAudit
	ActorID int64
	Skip    bool
	Blocked bool
}

type importPlan struct {
	Options  Options
	Report   *Report
	OwnerID  int64
	Actor    *user_model.User
	TeamIDs  []int64
	Profiles []profilePlan
	Audits   []auditPlan
}

var (
	validStages = map[string]struct{}{
		"planned": {}, "development": {}, "testing": {}, "released": {}, "paused": {},
	}
	validRisks = map[string]struct{}{
		"normal": {}, "attention": {}, "blocked": {},
	}
	reservedProjectSlugs = map[string]struct{}{
		"config": {}, "dashboard": {}, "history": {}, "new": {}, "settings": {},
	}
)

func Preflight(ctx context.Context, options Options) (*Report, error) {
	plan, err := buildPlan(ctx, options, "preflight")
	if err != nil {
		return nil, err
	}
	return plan.Report, nil
}

func buildPlan(ctx context.Context, options Options, mode string) (*importPlan, error) {
	options.DatabasePath = strings.TrimSpace(options.DatabasePath)
	options.Organization = strings.TrimSpace(options.Organization)
	options.Actor = strings.TrimSpace(options.Actor)
	for i := range options.EditorTeams {
		options.EditorTeams[i] = strings.TrimSpace(options.EditorTeams[i])
	}
	options.EditorTeams = slices.DeleteFunc(options.EditorTeams, func(value string) bool { return value == "" })
	if options.DatabasePath == "" || options.Organization == "" || options.Actor == "" {
		return nil, errors.New("database, organization, and actor are required")
	}

	organization, err := org_model.GetOrgByName(ctx, options.Organization)
	if err != nil {
		return nil, fmt.Errorf("resolve organization %q: %w", options.Organization, err)
	}
	actor, err := user_model.GetUserByName(ctx, options.Actor)
	if err != nil {
		return nil, fmt.Errorf("resolve migration actor %q: %w", options.Actor, err)
	}
	isOwner, err := org_model.IsOrganizationOwner(ctx, organization.ID, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("verify migration actor ownership: %w", err)
	}
	if !isOwner {
		return nil, fmt.Errorf("migration actor %q must be an owner of organization %q", actor.Name, organization.Name)
	}

	teamIDs, err := org_model.GetTeamIDsByNames(ctx, organization.ID, options.EditorTeams, false)
	if err != nil {
		return nil, fmt.Errorf("resolve editor teams: %w", err)
	}

	legacySource, err := openSource(ctx, options.DatabasePath)
	if err != nil {
		return nil, err
	}
	defer legacySource.close()
	data, err := legacySource.load(ctx)
	if err != nil {
		return nil, err
	}

	report := &Report{
		Mode: mode, Database: options.DatabasePath, Organization: organization.Name, Actor: actor.Name,
		GeneratedAt: time.Now().UTC(),
		Summary:     ReportSummary{Profiles: len(data.Profiles), Followers: data.Followers, Audits: len(data.Audits)},
		Items:       make([]ReportItem, 0, len(data.Profiles)+len(data.Audits)),
	}
	plan := &importPlan{
		Options: options, Report: report, OwnerID: organization.ID, Actor: actor, TeamIDs: teamIDs,
		Profiles: make([]profilePlan, 0, len(data.Profiles)), Audits: make([]auditPlan, 0, len(data.Audits)),
	}

	usedSlugs, err := loadUsedSlugs(ctx, organization.ID)
	if err != nil {
		return nil, err
	}
	profilesByRepository := make(map[int64]int, len(data.Profiles))
	for _, profile := range data.Profiles {
		profilePlan := inspectProfile(ctx, plan, profile, usedSlugs)
		profilesByRepository[profile.RepoID] = len(plan.Profiles)
		plan.Profiles = append(plan.Profiles, profilePlan)
	}

	requestIDs := make(map[string]int64, len(data.Audits))
	for _, audit := range data.Audits {
		auditPlan := inspectAudit(ctx, plan, audit, profilesByRepository, requestIDs)
		plan.Audits = append(plan.Audits, auditPlan)
	}
	return plan, nil
}

func inspectProfile(ctx context.Context, plan *importPlan, profile legacyProfile, usedSlugs map[string]struct{}) profilePlan {
	item := ReportItem{Kind: "profile", LegacyID: profile.RepoID, RepoID: profile.RepoID}
	result := profilePlan{Profile: profile}
	marker := profileRequestID(profile.RepoID)
	change, exists, err := findChange(ctx, marker)
	if err != nil {
		block(plan.Report, &result.Blocked, item, err.Error())
		return result
	}
	if exists {
		project, projectErr := orgproject_model.GetProjectByID(ctx, change.ProjectID)
		if projectErr != nil || change.Source != orgproject_model.ChangeSourceLegacyImport || project == nil || project.OwnerID != plan.OwnerID {
			block(plan.Report, &result.Blocked, item, "deterministic request ID is already used by another operation")
			return result
		}
		result.ProjectID = project.ID
		result.Slug = project.Slug
		result.Skip = true
		plan.Report.Summary.ProjectsSkip++
		item.Action, item.Reason, item.ProjectID, item.Slug = "skip", "already imported", project.ID, project.Slug
		plan.Report.add(item)
		return result
	}

	repository, err := repo_model.GetRepositoryByID(ctx, profile.RepoID)
	if err != nil {
		block(plan.Report, &result.Blocked, item, "repository does not exist")
	} else if repository.OwnerID != plan.OwnerID {
		block(plan.Report, &result.Blocked, item, "repository does not belong to the target organization")
	} else {
		result.Slug = availableSlug(strings.ToLower(repository.Name), profile.RepoID, usedSlugs)
		if !strings.EqualFold(result.Slug, repository.Name) {
			plan.Report.add(ReportItem{
				Kind: "profile", LegacyID: profile.RepoID, RepoID: profile.RepoID, Action: "warn",
				Reason: "repository name conflicts with an existing or reserved project slug; a deterministic suffix will be used", Slug: result.Slug,
			})
		}
	}

	validateProfile(ctx, plan, &result)
	if result.Blocked {
		return result
	}
	usedSlugs[result.Slug] = struct{}{}
	plan.Report.Summary.ProjectsImport++
	item.Action, item.Slug = "import", result.Slug
	plan.Report.add(item)
	return result
}

func validateProfile(ctx context.Context, plan *importPlan, result *profilePlan) {
	profile := result.Profile
	item := ReportItem{Kind: "profile", LegacyID: profile.RepoID, RepoID: profile.RepoID}
	if _, ok := validStages[profile.Stage]; !ok {
		block(plan.Report, &result.Blocked, item, fmt.Sprintf("invalid stage %q", profile.Stage))
	}
	if profile.Progress < 0 || profile.Progress > 100 {
		block(plan.Report, &result.Blocked, item, "progress must be between 0 and 100")
	}
	if _, ok := validRisks[profile.Risk]; !ok {
		block(plan.Report, &result.Blocked, item, fmt.Sprintf("invalid risk %q", profile.Risk))
	}
	if len([]rune(profile.Summary)) > 500 {
		block(plan.Report, &result.Blocked, item, "summary exceeds 500 characters")
	}
	for label, value := range map[string]sqlDate{"created_at": {profile.CreatedAt, time.RFC3339Nano}, "updated_at": {profile.UpdatedAt, time.RFC3339Nano}} {
		if !validDate(value.Value, value.Layout) {
			block(plan.Report, &result.Blocked, item, fmt.Sprintf("invalid %s %q", label, value.Value))
		}
	}
	for label, value := range map[string]sqlDate{"start_date": {profile.StartDate.String, time.DateOnly}, "target_date": {profile.TargetDate.String, time.DateOnly}} {
		if value.Value != "" && !validDate(value.Value, value.Layout) {
			block(plan.Report, &result.Blocked, item, fmt.Sprintf("invalid %s %q", label, value.Value))
		}
	}
	if profile.OwnerUserID.Valid {
		validateCriticalMember(ctx, plan, result, profile.OwnerUserID.Int64, "owner")
	}
	for _, userID := range profile.Followers {
		validateCriticalMember(ctx, plan, result, userID, "follower")
	}
	validateCriticalMember(ctx, plan, result, profile.UpdatedBy, "updated_by")
}

type sqlDate struct {
	Value  string
	Layout string
}

func validateCriticalMember(ctx context.Context, plan *importPlan, result *profilePlan, userID int64, role string) {
	valid, err := isOrganizationMember(ctx, plan.OwnerID, userID)
	if err != nil {
		block(plan.Report, &result.Blocked, ReportItem{Kind: "profile", LegacyID: result.Profile.RepoID, RepoID: result.Profile.RepoID}, err.Error())
	} else if !valid {
		block(plan.Report, &result.Blocked, ReportItem{Kind: "profile", LegacyID: result.Profile.RepoID, RepoID: result.Profile.RepoID}, fmt.Sprintf("%s user %d does not exist or is not a target organization member", role, userID))
	}
}

func inspectAudit(ctx context.Context, plan *importPlan, audit legacyAudit, profiles map[int64]int, requestIDs map[string]int64) auditPlan {
	item := ReportItem{Kind: "audit", LegacyID: audit.ID, RepoID: audit.RepoID}
	result := auditPlan{Audit: audit, ActorID: audit.ActorUserID}
	profileIndex, hasProfile := profiles[audit.RepoID]
	if !hasProfile {
		block(plan.Report, &result.Blocked, item, "audit references a missing project profile")
	} else if plan.Profiles[profileIndex].Blocked {
		block(plan.Report, &result.Blocked, item, "project profile is blocked")
	}

	if previous, duplicate := requestIDs[audit.RequestID]; duplicate {
		block(plan.Report, &result.Blocked, item, fmt.Sprintf("legacy request ID duplicates audit %d", previous))
	} else {
		requestIDs[audit.RequestID] = audit.ID
	}

	marker := auditRequestID(audit.ID)
	change, exists, err := findChange(ctx, marker)
	if err != nil {
		block(plan.Report, &result.Blocked, item, err.Error())
	} else if exists {
		expectedProjectID := int64(0)
		if hasProfile {
			expectedProjectID = plan.Profiles[profileIndex].ProjectID
		}
		if change.Source != orgproject_model.ChangeSourceLegacyImport || expectedProjectID == 0 || change.ProjectID != expectedProjectID {
			block(plan.Report, &result.Blocked, item, "deterministic request ID is already used by another operation")
		} else if !result.Blocked {
			result.Skip = true
			plan.Report.Summary.AuditsSkip++
			item.Action, item.Reason, item.ProjectID = "skip", "already imported", change.ProjectID
			plan.Report.add(item)
			return result
		}
	}
	var changedFields []string
	if err := json.Unmarshal([]byte(audit.ChangedFields), &changedFields); err != nil {
		block(plan.Report, &result.Blocked, item, "changed_fields contains invalid JSON")
	}
	if audit.BeforeValue.Valid && !json.Valid([]byte(audit.BeforeValue.String)) {
		block(plan.Report, &result.Blocked, item, "before_value contains invalid JSON")
	}
	if !json.Valid([]byte(audit.AfterValue)) {
		block(plan.Report, &result.Blocked, item, "after_value contains invalid JSON")
	}
	if !validDate(audit.CreatedAt, time.RFC3339Nano) {
		block(plan.Report, &result.Blocked, item, fmt.Sprintf("invalid created_at %q", audit.CreatedAt))
	}
	validActor, memberErr := isOrganizationMember(ctx, plan.OwnerID, audit.ActorUserID)
	if memberErr != nil {
		block(plan.Report, &result.Blocked, item, memberErr.Error())
	} else if !validActor {
		result.ActorID = plan.Actor.ID
		plan.Report.add(ReportItem{
			Kind: "audit", LegacyID: audit.ID, RepoID: audit.RepoID, Action: "warn",
			Reason: fmt.Sprintf("legacy actor %d is unavailable; migration actor %d will be recorded", audit.ActorUserID, plan.Actor.ID),
		})
	}
	if result.Blocked {
		return result
	}
	plan.Report.Summary.AuditsImport++
	item.Action = "import"
	plan.Report.add(item)
	return result
}

func loadUsedSlugs(ctx context.Context, ownerID int64) (map[string]struct{}, error) {
	projects := make([]*orgproject_model.Project, 0)
	if err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Find(&projects); err != nil {
		return nil, err
	}
	used := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		used[project.Slug] = struct{}{}
	}
	return used, nil
}

func availableSlug(base string, repoID int64, used map[string]struct{}) string {
	candidate := base
	_, reserved := reservedProjectSlugs[candidate]
	_, exists := used[candidate]
	if !reserved && !exists {
		return candidate
	}
	candidate = base + "-" + strconv.FormatInt(repoID, 10)
	for suffix := int64(2); ; suffix++ {
		_, reserved = reservedProjectSlugs[candidate]
		_, exists = used[candidate]
		if !reserved && !exists {
			return candidate
		}
		candidate = base + "-" + strconv.FormatInt(repoID, 10) + "-" + strconv.FormatInt(suffix, 10)
	}
}

func findChange(ctx context.Context, requestID string) (*orgproject_model.ChangeLog, bool, error) {
	change := &orgproject_model.ChangeLog{RequestID: requestID}
	has, err := db.GetEngine(ctx).Get(change)
	return change, has, err
}

func isOrganizationMember(ctx context.Context, ownerID, userID int64) (bool, error) {
	user := &user_model.User{ID: userID}
	has, err := db.GetEngine(ctx).ID(userID).Get(user)
	if err != nil || !has {
		return false, err
	}
	return org_model.IsOrganizationMember(ctx, ownerID, userID)
}

func validDate(value, layout string) bool {
	_, err := time.Parse(layout, value)
	return err == nil
}

func block(report *Report, blocked *bool, item ReportItem, reason string) {
	*blocked = true
	item.Action = "block"
	item.Reason = reason
	report.add(item)
}

func profileRequestID(repoID int64) string {
	return "legacy-import:profile:" + strconv.FormatInt(repoID, 10)
}

func auditRequestID(auditID int64) string {
	return "legacy-import:audit:" + strconv.FormatInt(auditID, 10)
}
