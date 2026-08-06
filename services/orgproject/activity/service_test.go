// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activity

import (
	"context"
	"testing"
	"time"

	"gitea.dev/models/db"
	orgproject_model "gitea.dev/models/orgproject"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReader struct {
	commits    map[int64][]Commit
	pulls      map[int64]PullRequestCounts
	releases   map[int64]ReleaseSummary
	limits     []int
	pullIDs    []int64
	releaseIDs []int64
}

func (reader *fakeReader) RecentCommits(_ context.Context, repository *repo_model.Repository, _ time.Time, limit int) ([]Commit, error) {
	reader.limits = append(reader.limits, limit)
	commits := reader.commits[repository.ID]
	if len(commits) > limit {
		commits = commits[:limit]
	}
	return commits, nil
}

func (reader *fakeReader) PullRequestCounts(_ context.Context, repository *repo_model.Repository, _ time.Time) (PullRequestCounts, error) {
	reader.pullIDs = append(reader.pullIDs, repository.ID)
	return reader.pulls[repository.ID], nil
}

func (reader *fakeReader) ReleaseSummary(_ context.Context, repository *repo_model.Repository, _ time.Time) (ReleaseSummary, error) {
	reader.releaseIDs = append(reader.releaseIDs, repository.ID)
	return reader.releases[repository.ID], nil
}

func TestLoadVisibleRepositories(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	actor := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 28})
	project := &orgproject_model.Project{
		ID: 99, OwnerID: 3, Slug: "activity", Name: "Activity", Lifecycle: orgproject_model.LifecycleActive, Version: 1, CreatedBy: 2,
	}
	require.NoError(t, db.Insert(t.Context(), project))
	require.NoError(t, db.Insert(t.Context(),
		&orgproject_model.Repository{ID: 99, ProjectID: project.ID, RepositoryID: 3, Role: orgproject_model.RepositoryRoleRelated, CreatedBy: 2},
		&orgproject_model.Repository{ID: 100, ProjectID: project.ID, RepositoryID: 32, Role: orgproject_model.RepositoryRoleRelated, CreatedBy: 2},
	))

	repositories, err := loadVisibleRepositories(t.Context(), 3, project.ID, actor)
	require.NoError(t, err)
	require.Len(t, repositories, 1)
	assert.EqualValues(t, 32, repositories[0].Repository.ID)
}

func TestSummarizeBoundsAndAggregatesVisibleRepositories(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	latest := now.Add(-time.Hour)
	repositories := []visibleRepository{
		{Repository: &repo_model.Repository{ID: 1, OwnerName: "org", Name: "one"}, CanReadPulls: true, CanReadReleases: true},
		{Repository: &repo_model.Repository{ID: 2, OwnerName: "org", Name: "two"}, CanReadPulls: true},
	}
	reader := &fakeReader{
		commits: map[int64][]Commit{
			1: {{RepositoryID: 1, SHA: "1", CommittedAt: now.Add(-3 * time.Hour)}, {RepositoryID: 1, SHA: "2", CommittedAt: now.Add(-time.Hour)}},
			2: {{RepositoryID: 2, SHA: "3", CommittedAt: now.Add(-2 * time.Hour)}, {RepositoryID: 2, SHA: "4", CommittedAt: now.Add(-30 * time.Minute)}},
		},
		pulls:    map[int64]PullRequestCounts{1: {Open: 2, Merged: 1}, 2: {Open: 3, Merged: 4}},
		releases: map[int64]ReleaseSummary{1: {Count: 1, LatestAt: &latest}, 2: {Count: 2}},
	}

	summary, err := summarize(t.Context(), reader, repositories, Options{
		Since: now.Add(-7 * 24 * time.Hour), PerRepositoryCommitLimit: 2, TotalCommitLimit: 3,
	}, now)
	require.NoError(t, err)
	assert.Equal(t, []int{2, 2}, reader.limits)
	assert.Equal(t, []int64{1, 2}, reader.pullIDs)
	assert.Equal(t, []int64{1}, reader.releaseIDs)
	assert.EqualValues(t, 5, summary.OpenPulls)
	assert.EqualValues(t, 5, summary.MergedPulls)
	assert.EqualValues(t, 1, summary.ReleaseCount)
	require.Len(t, summary.Commits, 3)
	assert.Equal(t, []string{"4", "2", "3"}, []string{summary.Commits[0].SHA, summary.Commits[1].SHA, summary.Commits[2].SHA})
	require.NotNil(t, summary.LatestReleaseAt)
	assert.Equal(t, latest, *summary.LatestReleaseAt)
}

func TestNormalizeOptions(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	normalized, err := normalizeOptions(Options{
		Since: now.Add(-365 * 24 * time.Hour), PerRepositoryCommitLimit: 100, TotalCommitLimit: 1000,
	}, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-maximumWindow), normalized.Since)
	assert.Equal(t, maximumPerRepositoryCommitLimit, normalized.PerRepositoryCommitLimit)
	assert.Equal(t, maximumTotalCommitLimit, normalized.TotalCommitLimit)

	_, err = normalizeOptions(Options{Since: now.Add(time.Minute)}, now)
	require.ErrorContains(t, err, "future")
}

func TestNativeReaderPullRequestsAndReleases(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repository := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	since := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	reader := nativeReader{}

	pulls, err := reader.PullRequestCounts(t.Context(), repository, since)
	require.NoError(t, err)
	assert.EqualValues(t, 3, pulls.Open)
	assert.Zero(t, pulls.Merged)

	releases, err := reader.ReleaseSummary(t.Context(), repository, since)
	require.NoError(t, err)
	assert.EqualValues(t, 2, releases.Count)
	require.NotNil(t, releases.LatestAt)
	assert.True(t, since.Equal(*releases.LatestAt))
}
