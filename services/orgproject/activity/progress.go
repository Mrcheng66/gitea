// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activity

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"gitea.dev/models/db"
	repo_model "gitea.dev/models/repo"
)

// ProgressEvent is one user-visible delivery fact derived from repository activity.
type ProgressEvent struct {
	Kind               string    `json:"kind"`
	Title              string    `json:"title"`
	Link               string    `json:"link"`
	RepositoryID       int64     `json:"repository_id"`
	RepositoryFullName string    `json:"repository_full_name"`
	RepositoryLink     string    `json:"repository_link"`
	AuthorName         string    `json:"author_name"`
	OccurredAt         time.Time `json:"occurred_at"`
}

type progressEventRow struct {
	Index        int64  `xorm:"index"`
	Title        string `xorm:"title"`
	TagName      string `xorm:"tag_name"`
	AuthorName   string `xorm:"author_name"`
	OccurredUnix int64  `xorm:"occurred_unix"`
}

func (nativeReader) ProgressEvents(ctx context.Context, repository *repo_model.Repository, since time.Time, visible visibleRepository) ([]ProgressEvent, error) {
	events := make([]ProgressEvent, 0, 6)
	if visible.CanReadReleases {
		releases, err := recentReleaseEvents(ctx, repository, since)
		if err != nil {
			return nil, err
		}
		events = append(events, releases...)
	}
	if visible.CanReadPulls {
		pulls, err := recentMergedPullEvents(ctx, repository, since)
		if err != nil {
			return nil, err
		}
		events = append(events, pulls...)
	}
	if visible.CanReadIssues {
		issues, err := recentClosedIssueEvents(ctx, repository, since)
		if err != nil {
			return nil, err
		}
		events = append(events, issues...)
	}
	return events, nil
}

func recentReleaseEvents(ctx context.Context, repository *repo_model.Repository, since time.Time) ([]ProgressEvent, error) {
	rows := make([]progressEventRow, 0, 3)
	err := db.GetEngine(ctx).
		Table("release").
		Select("release.title AS title, release.tag_name AS tag_name, COALESCE(NULLIF(`user`.full_name, ''), `user`.name, release.original_author) AS author_name, release.created_unix AS occurred_unix").
		Join("LEFT", "`user`", "`user`.id = release.publisher_id").
		Where("release.repo_id = ? AND release.is_draft = ? AND release.is_tag = ? AND release.created_unix >= ?", repository.ID, false, false, since.Unix()).
		Desc("release.created_unix", "release.id").
		Limit(3).
		Find(&rows)
	if err != nil {
		return nil, err
	}
	events := make([]ProgressEvent, 0, len(rows))
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = row.TagName
		}
		events = append(events, newProgressEvent(repository, "release", title,
			repository.Link()+"/releases/tag/"+url.PathEscape(row.TagName), row.AuthorName, row.OccurredUnix))
	}
	return events, nil
}

func recentMergedPullEvents(ctx context.Context, repository *repo_model.Repository, since time.Time) ([]ProgressEvent, error) {
	rows := make([]progressEventRow, 0, 3)
	err := db.GetEngine(ctx).
		Table("pull_request").
		Select("issue.`index` AS `index`, issue.name AS title, COALESCE(NULLIF(`user`.full_name, ''), `user`.name, '') AS author_name, pull_request.merged_unix AS occurred_unix").
		Join("INNER", "issue", "issue.id = pull_request.issue_id").
		Join("LEFT", "`user`", "`user`.id = pull_request.merger_id").
		Where("pull_request.base_repo_id = ? AND pull_request.has_merged = ? AND pull_request.merged_unix >= ?", repository.ID, true, since.Unix()).
		Desc("pull_request.merged_unix", "pull_request.id").
		Limit(3).
		Find(&rows)
	if err != nil {
		return nil, err
	}
	events := make([]ProgressEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, newProgressEvent(repository, "pull_merged", row.Title,
			repository.Link()+"/pulls/"+strconv.FormatInt(row.Index, 10), row.AuthorName, row.OccurredUnix))
	}
	return events, nil
}

func recentClosedIssueEvents(ctx context.Context, repository *repo_model.Repository, since time.Time) ([]ProgressEvent, error) {
	rows := make([]progressEventRow, 0, 3)
	err := db.GetEngine(ctx).
		Table("issue").
		Select("issue.`index` AS `index`, issue.name AS title, COALESCE(NULLIF(`user`.full_name, ''), `user`.name, issue.original_author) AS author_name, issue.closed_unix AS occurred_unix").
		Join("LEFT", "`user`", "`user`.id = issue.poster_id").
		Where("issue.repo_id = ? AND issue.is_pull = ? AND issue.is_closed = ? AND issue.closed_unix >= ?", repository.ID, false, true, since.Unix()).
		Desc("issue.closed_unix", "issue.id").
		Limit(3).
		Find(&rows)
	if err != nil {
		return nil, err
	}
	events := make([]ProgressEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, newProgressEvent(repository, "issue_closed", row.Title,
			repository.Link()+"/issues/"+strconv.FormatInt(row.Index, 10), row.AuthorName, row.OccurredUnix))
	}
	return events, nil
}

func newProgressEvent(repository *repo_model.Repository, kind, title, link, author string, occurredUnix int64) ProgressEvent {
	return ProgressEvent{
		Kind: kind, Title: title, Link: link, RepositoryID: repository.ID,
		RepositoryFullName: repository.FullName(), RepositoryLink: repository.Link(),
		AuthorName: author, OccurredAt: time.Unix(occurredUnix, 0).UTC(),
	}
}
