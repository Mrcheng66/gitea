// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"gitea.dev/models/db"
	org_model "gitea.dev/models/organization"
	orgproject_model "gitea.dev/models/orgproject"
)

// ListEditorTeamIDs returns the team IDs that may edit organization projects.
func ListEditorTeamIDs(ctx context.Context, ownerID int64) ([]int64, error) {
	links := make([]*orgproject_model.EditorTeam, 0)
	if err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Asc("team_id").Find(&links); err != nil {
		return nil, err
	}
	teamIDs := make([]int64, 0, len(links))
	for _, link := range links {
		teamIDs = append(teamIDs, link.TeamID)
	}
	return teamIDs, nil
}

// ReplaceEditorTeams replaces editor-team grants after validating organization ownership.
func ReplaceEditorTeams(ctx context.Context, ownerID, actorID int64, teamIDs []int64) error {
	if ownerID <= 0 || actorID <= 0 {
		return errors.New("organization and actor IDs must be positive")
	}
	teamIDs = slices.Clone(teamIDs)
	slices.Sort(teamIDs)
	teamIDs = slices.Compact(teamIDs)

	teams, err := org_model.GetTeamsByIDs(ctx, teamIDs)
	if err != nil {
		return err
	}
	for _, teamID := range teamIDs {
		team := teams[teamID]
		if team == nil || team.OrgID != ownerID {
			return fmt.Errorf("team %d does not belong to organization %d", teamID, ownerID)
		}
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Delete(new(orgproject_model.EditorTeam)); err != nil {
			return err
		}
		for _, teamID := range teamIDs {
			if err := db.Insert(ctx, &orgproject_model.EditorTeam{OwnerID: ownerID, TeamID: teamID, CreatedBy: actorID}); err != nil {
				return err
			}
		}
		return nil
	})
}
