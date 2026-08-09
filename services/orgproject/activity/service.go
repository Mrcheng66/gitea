// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activity

import (
	"context"
	"errors"
	"slices"
	"time"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	access_model "gitea.dev/models/perm/access"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	user_model "gitea.dev/models/user"
	"gitea.dev/modules/git"
	"gitea.dev/modules/log"
	"gitea.dev/modules/optional"
	project_service "gitea.dev/services/orgproject/project"

	"xorm.io/builder"
)

const (
	defaultWindow                   = 30 * 24 * time.Hour
	maximumWindow                   = 90 * 24 * time.Hour
	defaultPerRepositoryCommitLimit = 5
	maximumPerRepositoryCommitLimit = 20
	defaultTotalCommitLimit         = 20
	maximumTotalCommitLimit         = 100
	defaultTotalProgressLimit       = 10
	maximumTotalProgressLimit       = 50
)

// Options bounds one organization project activity query.
type Options struct {
	Since                    time.Time
	PerRepositoryCommitLimit int
	TotalCommitLimit         int
	TotalProgressLimit       int
}

// Commit represents one recent commit from a visible linked repository.
type Commit struct {
	RepositoryID       int64     `json:"repository_id"`
	RepositoryFullName string    `json:"repository_full_name"`
	RepositoryLink     string    `json:"repository_link"`
	SHA                string    `json:"sha"`
	ShortSHA           string    `json:"short_sha"`
	Link               string    `json:"link"`
	Message            string    `json:"message"`
	AuthorName         string    `json:"author_name"`
	CommittedAt        time.Time `json:"committed_at"`
}

// PullRequestCounts contains bounded pull-request activity for one repository.
type PullRequestCounts struct {
	Open   int64 `json:"open"`
	Merged int64 `json:"merged"`
}

// ReleaseSummary contains bounded release activity for one repository.
type ReleaseSummary struct {
	Count    int64      `json:"count"`
	LatestAt *time.Time `json:"latest_at,omitempty"`
}

// RepositorySummary contains activity counters for one visible linked repository.
type RepositorySummary struct {
	ID              int64      `json:"id"`
	FullName        string     `json:"full_name"`
	Link            string     `json:"link"`
	OpenPulls       int64      `json:"open_pulls"`
	MergedPulls     int64      `json:"merged_pulls"`
	ReleaseCount    int64      `json:"release_count"`
	LatestReleaseAt *time.Time `json:"latest_release_at,omitempty"`
}

// Summary contains project activity without revealing inaccessible repositories.
type Summary struct {
	Since           time.Time           `json:"since"`
	Repositories    []RepositorySummary `json:"repositories"`
	Commits         []Commit            `json:"commits"`
	OpenPulls       int64               `json:"open_pulls"`
	MergedPulls     int64               `json:"merged_pulls"`
	ReleaseCount    int64               `json:"release_count"`
	LatestReleaseAt *time.Time          `json:"latest_release_at,omitempty"`
	Progress        []ProgressEvent     `json:"progress"`
	Partial         bool                `json:"partial"`
}

type reader interface {
	RecentCommits(context.Context, *repo_model.Repository, time.Time, int) ([]Commit, error)
	PullRequestCounts(context.Context, *repo_model.Repository, time.Time) (PullRequestCounts, error)
	ReleaseSummary(context.Context, *repo_model.Repository, time.Time) (ReleaseSummary, error)
	ProgressEvents(context.Context, *repo_model.Repository, time.Time, visibleRepository) ([]ProgressEvent, error)
}

type nativeReader struct{}

type visibleRepository struct {
	Repository      *repo_model.Repository
	CanReadIssues   bool
	CanReadPulls    bool
	CanReadReleases bool
}

// Get returns bounded activity for repositories visible to actor.
func Get(ctx context.Context, ownerID, projectID int64, actor *user_model.User, opts Options) (*Summary, error) {
	repositories, err := loadVisibleRepositories(ctx, ownerID, projectID, actor)
	if err != nil {
		return nil, err
	}
	return summarize(ctx, nativeReader{}, repositories, opts, time.Now().UTC())
}

func loadVisibleRepositories(ctx context.Context, ownerID, projectID int64, actor *user_model.User) ([]visibleRepository, error) {
	detail, err := project_service.GetByID(ctx, ownerID, projectID, actor)
	if err != nil {
		return nil, err
	}
	repositories := make([]visibleRepository, 0, len(detail.Repositories))
	for _, link := range detail.Repositories {
		repository, err := repo_model.GetRepositoryByID(ctx, link.RepositoryID)
		if err != nil {
			return nil, err
		}
		permission, err := access_model.GetIndividualUserRepoPermission(ctx, repository, actor)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, visibleRepository{
			Repository: repository, CanReadIssues: permission.CanRead(unit.TypeIssues),
			CanReadPulls: permission.CanRead(unit.TypePullRequests), CanReadReleases: permission.CanRead(unit.TypeReleases),
		})
	}
	return repositories, nil
}

func summarize(ctx context.Context, source reader, repositories []visibleRepository, opts Options, now time.Time) (*Summary, error) {
	normalized, err := normalizeOptions(opts, now)
	if err != nil {
		return nil, err
	}
	summary := &Summary{
		Since: normalized.Since, Repositories: make([]RepositorySummary, 0, len(repositories)),
		Commits: make([]Commit, 0), Progress: make([]ProgressEvent, 0),
	}
	for _, visible := range repositories {
		repository := visible.Repository
		if repository == nil {
			continue
		}
		commits, err := source.RecentCommits(ctx, repository, normalized.Since, normalized.PerRepositoryCommitLimit)
		if err != nil {
			log.Warn("Unable to load recent commits for %s: %v", repository.FullName(), err)
			summary.Partial = true
			commits = nil
		}
		progress, err := source.ProgressEvents(ctx, repository, normalized.Since, visible)
		if err != nil {
			log.Warn("Unable to load progress events for %s: %v", repository.FullName(), err)
			summary.Partial = true
			progress = nil
		}
		pulls := PullRequestCounts{}
		if visible.CanReadPulls {
			pulls, err = source.PullRequestCounts(ctx, repository, normalized.Since)
			if err != nil {
				log.Warn("Unable to load pull request counts for %s: %v", repository.FullName(), err)
				summary.Partial = true
				pulls = PullRequestCounts{}
			}
		}
		releases := ReleaseSummary{}
		if visible.CanReadReleases {
			releases, err = source.ReleaseSummary(ctx, repository, normalized.Since)
			if err != nil {
				log.Warn("Unable to load release summary for %s: %v", repository.FullName(), err)
				summary.Partial = true
				releases = ReleaseSummary{}
			}
		}
		summary.Repositories = append(summary.Repositories, RepositorySummary{
			ID: repository.ID, FullName: repository.FullName(), Link: repository.Link(), OpenPulls: pulls.Open,
			MergedPulls: pulls.Merged, ReleaseCount: releases.Count, LatestReleaseAt: releases.LatestAt,
		})
		summary.Commits = append(summary.Commits, commits...)
		summary.Progress = append(summary.Progress, progress...)
		summary.OpenPulls += pulls.Open
		summary.MergedPulls += pulls.Merged
		summary.ReleaseCount += releases.Count
		if releases.LatestAt != nil && (summary.LatestReleaseAt == nil || releases.LatestAt.After(*summary.LatestReleaseAt)) {
			latest := *releases.LatestAt
			summary.LatestReleaseAt = &latest
		}
	}
	slices.SortStableFunc(summary.Commits, func(left, right Commit) int {
		return right.CommittedAt.Compare(left.CommittedAt)
	})
	if len(summary.Commits) > normalized.TotalCommitLimit {
		summary.Commits = summary.Commits[:normalized.TotalCommitLimit]
	}
	appendCommitProgress(summary)
	slices.SortStableFunc(summary.Progress, func(left, right ProgressEvent) int {
		return right.OccurredAt.Compare(left.OccurredAt)
	})
	if len(summary.Progress) > normalized.TotalProgressLimit {
		summary.Progress = summary.Progress[:normalized.TotalProgressLimit]
	}
	return summary, nil
}

func normalizeOptions(opts Options, now time.Time) (Options, error) {
	now = now.UTC()
	if opts.Since.IsZero() {
		opts.Since = now.Add(-defaultWindow)
	} else {
		opts.Since = opts.Since.UTC()
	}
	if opts.Since.After(now) {
		return Options{}, errors.New("activity start time must not be in the future")
	}
	if earliest := now.Add(-maximumWindow); opts.Since.Before(earliest) {
		opts.Since = earliest
	}
	if opts.PerRepositoryCommitLimit <= 0 {
		opts.PerRepositoryCommitLimit = defaultPerRepositoryCommitLimit
	} else if opts.PerRepositoryCommitLimit > maximumPerRepositoryCommitLimit {
		opts.PerRepositoryCommitLimit = maximumPerRepositoryCommitLimit
	}
	if opts.TotalCommitLimit <= 0 {
		opts.TotalCommitLimit = defaultTotalCommitLimit
	} else if opts.TotalCommitLimit > maximumTotalCommitLimit {
		opts.TotalCommitLimit = maximumTotalCommitLimit
	}
	if opts.TotalProgressLimit <= 0 {
		opts.TotalProgressLimit = defaultTotalProgressLimit
	} else if opts.TotalProgressLimit > maximumTotalProgressLimit {
		opts.TotalProgressLimit = maximumTotalProgressLimit
	}
	return opts, nil
}

func appendCommitProgress(summary *Summary) {
	for _, commit := range summary.Commits {
		summary.Progress = append(summary.Progress, ProgressEvent{
			Kind: "commit", Title: commit.Message, Link: commit.Link, RepositoryID: commit.RepositoryID,
			RepositoryFullName: commit.RepositoryFullName, RepositoryLink: commit.RepositoryLink,
			AuthorName: commit.AuthorName, OccurredAt: commit.CommittedAt,
		})
	}
}

func (nativeReader) RecentCommits(ctx context.Context, repository *repo_model.Repository, since time.Time, limit int) ([]Commit, error) {
	if repository.IsEmpty || repository.DefaultBranch == "" {
		return nil, nil
	}
	gitRepository, err := git.OpenRepository(ctx, repository.RepoPath())
	if err != nil {
		return nil, err
	}
	defer gitRepository.Close()

	head, err := gitRepository.GetBranchCommit(repository.DefaultBranch)
	if err != nil {
		if git.IsErrNotExist(err) || git.IsErrBranchNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	commits, err := head.CommitsByRange(1, limit, "", since.UTC().Format(time.RFC3339), "")
	if err != nil {
		return nil, err
	}
	result := make([]Commit, 0, len(commits))
	for _, commit := range commits {
		if commit.Committer.When.Before(since) {
			continue
		}
		sha := commit.ID.String()
		shortSHA := sha
		if len(shortSHA) > 10 {
			shortSHA = shortSHA[:10]
		}
		result = append(result, Commit{
			RepositoryID: repository.ID, RepositoryFullName: repository.FullName(), RepositoryLink: repository.Link(),
			SHA: sha, ShortSHA: shortSHA, Link: repository.Link() + "/commit/" + sha, Message: commit.MessageTitle(),
			AuthorName: commit.Author.Name, CommittedAt: commit.Committer.When,
		})
	}
	return result, nil
}

func (nativeReader) PullRequestCounts(ctx context.Context, repository *repo_model.Repository, since time.Time) (PullRequestCounts, error) {
	open, err := issues_model.CountIssues(ctx, &issues_model.IssuesOptions{
		RepoIDs: []int64{repository.ID}, IsPull: optional.Some(true), IsClosed: optional.Some(false), UpdatedAfterUnix: since.Unix(),
	})
	if err != nil {
		return PullRequestCounts{}, err
	}
	merged, err := db.GetEngine(ctx).
		Where(builder.Eq{"base_repo_id": repository.ID, "has_merged": true}).
		And(builder.Gte{"merged_unix": since.Unix()}).
		Count(new(issues_model.PullRequest))
	return PullRequestCounts{Open: open, Merged: merged}, err
}

func (nativeReader) ReleaseSummary(ctx context.Context, repository *repo_model.Repository, since time.Time) (ReleaseSummary, error) {
	conditions := repo_model.FindReleasesOptions{RepoID: repository.ID}.ToConds().And(builder.Gte{"created_unix": since.Unix()})
	count, err := db.GetEngine(ctx).Where(conditions).Count(new(repo_model.Release))
	if err != nil {
		return ReleaseSummary{}, err
	}
	latest := new(repo_model.Release)
	has, err := db.GetEngine(ctx).Where(conditions).Desc("created_unix", "id").Get(latest)
	if err != nil {
		return ReleaseSummary{}, err
	}
	result := ReleaseSummary{Count: count}
	if has {
		latestAt := latest.CreatedUnix.AsTime()
		result.LatestAt = &latestAt
	}
	return result, nil
}
