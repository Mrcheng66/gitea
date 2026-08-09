// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package workbench builds the project-management view shown on the signed-in dashboard.
package workbench

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/json"
	"gitea.dev/modules/log"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/timeutil"
	"gitea.dev/services/orgproject/activity"
	"gitea.dev/services/orgproject/config"
	"gitea.dev/services/orgproject/query"
)

const projectLimit = 10

type Options struct {
	OnlyMine bool
}

type Result struct {
	Projects                []Project `json:"projects"`
	People                  []Person  `json:"people"`
	Attention               Attention `json:"attention"`
	ConfiguredOrganizations int       `json:"configured_organizations"`
}

type Attention struct {
	Blocked int `json:"blocked"`
	Overdue int `json:"overdue"`
	DueSoon int `json:"due_soon"`
	Stale   int `json:"stale"`
	Unowned int `json:"unowned"`
}

type Person struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Link          string   `json:"link"`
	Owned         int      `json:"owned"`
	Participating int      `json:"participating"`
	Projects      []string `json:"projects"`
}

type Project struct {
	ID              int64                   `json:"id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Link            string                  `json:"link"`
	Organization    string                  `json:"organization"`
	OrganizationURL string                  `json:"organization_url"`
	StageKey        string                  `json:"stage_key"`
	Stage           string                  `json:"stage"`
	RiskKey         string                  `json:"risk_key"`
	Risk            string                  `json:"risk"`
	Progress        float64                 `json:"progress"`
	Owner           *Person                 `json:"owner,omitempty"`
	Participants    []Person                `json:"participants"`
	CurrentProblem  string                  `json:"current_problem"`
	NextAction      string                  `json:"next_action"`
	NextActionOwner *Person                 `json:"next_action_owner,omitempty"`
	NextActionDue   string                  `json:"next_action_due"`
	TargetDate      string                  `json:"target_date"`
	Overdue         bool                    `json:"overdue"`
	DueSoon         bool                    `json:"due_soon"`
	Stale           bool                    `json:"stale"`
	Expanded        bool                    `json:"expanded"`
	Activity        activity.Summary        `json:"activity"`
	ActivityError   bool                    `json:"activity_error"`
	UpdatedAt       time.Time               `json:"updated_at"`
	LatestEventAt   *time.Time              `json:"latest_event_at,omitempty"`
	Fields          map[string]projectField `json:"-"`
}

type projectField struct {
	Value *orgproject_model.FieldValue
	Field config.Field
}

type pendingProject struct {
	project           Project
	organizationID    int64
	ownerID           int64
	participantIDs    []int64
	nextActionOwnerID int64
}

// Build returns visible organization projects and their repository-derived progress.
func Build(ctx context.Context, actor *user_model.User, organizations []*organization.Organization, opts Options) (*Result, error) {
	result := &Result{Projects: make([]Project, 0), People: make([]Person, 0)}
	pending := make([]pendingProject, 0)
	userIDs := make(map[int64]struct{})
	for _, org := range organizations {
		schema, err := config.GetPublishedSchema(ctx, org.ID)
		if err != nil {
			var uninitialized config.ErrConfigUninitialized
			if errors.As(err, &uninitialized) {
				continue
			}
			return nil, err
		}
		result.ConfiguredOrganizations++
		listSchema := schema
		listSchema.ListView.Columns = workbenchColumns(schema)
		onlyUserID := int64(0)
		if opts.OnlyMine {
			onlyUserID = actor.ID
		}
		listed, err := query.List(ctx, listSchema, query.ListOptions{
			OwnerID: org.ID, OnlyUserID: onlyUserID, RiskFirst: true,
			Now: timeutil.TimeStampNow(), Page: 1, PageSize: min(projectLimit, setting.OrgProject.MaxPageSize),
		})
		if err != nil {
			return nil, err
		}
		fields := fieldsByKey(schema)
		for _, item := range listed.Items {
			entry := pendingProject{organizationID: org.ID, project: Project{
				ID: item.Project.ID, Name: item.Project.Name, Description: item.Project.Description,
				Link:         org.OrganisationLink() + "/projects/" + url.PathEscape(item.Project.Slug),
				Organization: org.DisplayName(), OrganizationURL: org.OrganisationLink(),
				Participants: make([]Person, 0), Fields: make(map[string]projectField),
				UpdatedAt: item.Project.UpdatedUnix.AsTime(),
			}}
			for key, value := range item.Values {
				entry.project.Fields[key] = projectField{Value: value, Field: fields[key]}
			}
			entry.project.StageKey, entry.project.Stage = selectValue(entry.project.Fields["stage"])
			entry.project.RiskKey, entry.project.Risk = selectValue(entry.project.Fields["risk"])
			entry.project.Progress = numberValue(entry.project.Fields["progress"])
			entry.project.CurrentProblem = textValue(entry.project.Fields["current_problem"])
			if entry.project.CurrentProblem == "" {
				entry.project.CurrentProblem = textValue(entry.project.Fields["summary"])
			}
			entry.project.NextAction = textValue(entry.project.Fields["next_action"])
			entry.project.TargetDate = dateValue(entry.project.Fields["target_date"])
			entry.project.NextActionDue = dateValue(entry.project.Fields["next_action_due"])
			entry.ownerID = memberValue(entry.project.Fields["owner"])
			entry.nextActionOwnerID = memberValue(entry.project.Fields["next_action_owner"])
			entry.participantIDs = memberArrayValue(entry.project.Fields["followers"])
			for _, id := range append(slices.Clone(entry.participantIDs), entry.ownerID, entry.nextActionOwnerID) {
				if id > 0 {
					userIDs[id] = struct{}{}
				}
			}
			pending = append(pending, entry)
		}
	}

	ids := make([]int64, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}
	users, err := user_model.GetUsersMapByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	people := make(map[int64]*Person, len(users))
	for id, user := range users {
		people[id] = &Person{ID: id, Name: user.Name, FullName: user.DisplayName(), Link: user.HomeLink(), Projects: make([]string, 0)}
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, entry := range pending {
		project := entry.project
		if person := people[entry.ownerID]; person != nil {
			person.Owned++
			person.Projects = appendUnique(person.Projects, project.Name)
			owner := *person
			project.Owner = &owner
		}
		if person := people[entry.nextActionOwnerID]; person != nil {
			nextOwner := *person
			project.NextActionOwner = &nextOwner
		}
		for _, id := range entry.participantIDs {
			if person := people[id]; person != nil {
				person.Participating++
				person.Projects = appendUnique(person.Projects, project.Name)
				project.Participants = append(project.Participants, *person)
			}
		}
		project.Overdue, project.DueSoon = dueState(project.TargetDate, today)
		summary, activityErr := activity.Get(ctx, entry.organizationID, project.ID, actor, activity.Options{
			Since: now.Add(-7 * 24 * time.Hour), PerRepositoryCommitLimit: 3, TotalCommitLimit: 6, TotalProgressLimit: 6,
		})
		if activityErr != nil {
			log.Warn("Unable to load organization project activity [project_id: %d]: %v", project.ID, activityErr)
			project.ActivityError = true
			project.Activity = activity.Summary{Since: now.Add(-7 * 24 * time.Hour), Repositories: []activity.RepositorySummary{}, Commits: []activity.Commit{}, Progress: []activity.ProgressEvent{}}
		} else {
			project.Activity = *summary
			if len(summary.Progress) > 0 {
				latest := summary.Progress[0].OccurredAt
				project.LatestEventAt = &latest
			}
			project.Stale = len(summary.Repositories) > 0 && len(summary.Progress) == 0
		}
		if project.RiskKey == "blocked" {
			result.Attention.Blocked++
		}
		if project.Overdue {
			result.Attention.Overdue++
		}
		if project.DueSoon {
			result.Attention.DueSoon++
		}
		if project.Stale {
			result.Attention.Stale++
		}
		if project.Owner == nil {
			result.Attention.Unowned++
		}
		project.Expanded = project.RiskKey == "blocked" || project.Overdue || project.Stale || project.Owner == nil
		project.Fields = nil
		result.Projects = append(result.Projects, project)
	}
	for _, person := range people {
		if person.Owned+person.Participating > 0 {
			result.People = append(result.People, *person)
		}
	}
	slices.SortStableFunc(result.Projects, compareProjects)
	slices.SortStableFunc(result.People, func(left, right Person) int {
		return strings.Compare(left.FullName, right.FullName)
	})
	if len(result.Projects) > projectLimit {
		result.Projects = result.Projects[:projectLimit]
	}
	return result, nil
}

func workbenchColumns(schema config.Schema) []string {
	keys := []string{"stage", "risk", "progress", "owner", "followers", "target_date", "summary", "current_problem", "next_action", "next_action_owner", "next_action_due"}
	active := make(map[string]bool, len(schema.Fields))
	for _, field := range schema.Fields {
		active[field.Key] = !field.Archived
	}
	return slices.DeleteFunc(keys, func(key string) bool { return !active[key] })
}

func fieldsByKey(schema config.Schema) map[string]config.Field {
	result := make(map[string]config.Field, len(schema.Fields))
	for _, field := range schema.Fields {
		result[field.Key] = field
	}
	return result
}

func textValue(field projectField) string {
	if field.Value != nil && field.Value.ValueText != nil {
		return strings.TrimSpace(*field.Value.ValueText)
	}
	return ""
}

func numberValue(field projectField) float64 {
	if field.Value != nil && field.Value.ValueNumber != nil {
		return *field.Value.ValueNumber
	}
	return 0
}

func memberValue(field projectField) int64 {
	if field.Value != nil && field.Value.ValueUserID != nil {
		return *field.Value.ValueUserID
	}
	return 0
}

func memberArrayValue(field projectField) []int64 {
	if field.Value == nil || field.Value.ValueJSON == nil {
		return nil
	}
	var ids []int64
	_ = json.Unmarshal([]byte(*field.Value.ValueJSON), &ids)
	return ids
}

func dateValue(field projectField) string {
	if field.Value != nil && field.Value.ValueTime != nil {
		return field.Value.ValueTime.AsTime().UTC().Format(time.DateOnly)
	}
	return ""
}

func selectValue(field projectField) (string, string) {
	key := textValue(field)
	for _, option := range field.Field.Options {
		if option.Key == key {
			return key, option.Label
		}
	}
	return key, key
}

func dueState(value string, today time.Time) (bool, bool) {
	if value == "" {
		return false, false
	}
	target, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return false, false
	}
	return target.Before(today), !target.Before(today) && target.Before(today.Add(7*24*time.Hour))
}

func compareProjects(left, right Project) int {
	leftScore := attentionScore(left)
	rightScore := attentionScore(right)
	if leftScore != rightScore {
		return rightScore - leftScore
	}
	if left.TargetDate != right.TargetDate {
		if left.TargetDate == "" {
			return 1
		}
		if right.TargetDate == "" {
			return -1
		}
		return strings.Compare(left.TargetDate, right.TargetDate)
	}
	return right.UpdatedAt.Compare(left.UpdatedAt)
}

func attentionScore(project Project) int {
	score := 0
	if project.RiskKey == "blocked" {
		score += 8
	}
	if project.Overdue {
		score += 4
	}
	if project.Stale {
		score += 2
	}
	if project.Owner == nil {
		score++
	}
	return score
}

func appendUnique(values []string, value string) []string {
	if !slices.Contains(values, value) {
		return append(values, value)
	}
	return values
}
