// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"

	"gitea.dev/modules/json"
)

// OrgProjectRepository links an organization project to a repository.
type OrgProjectRepository struct {
	RepositoryID int64  `json:"repository_id"`
	Role         string `json:"role"`
}

// OrgProject represents an organization-owned project.
// swagger:response OrgProject
type OrgProject struct {
	ID           int64                  `json:"id"`
	OwnerID      int64                  `json:"owner_id"`
	Slug         string                 `json:"slug"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Lifecycle    string                 `json:"lifecycle"`
	Version      int64                  `json:"version"`
	Values       map[string]json.Value  `json:"values"`
	Repositories []OrgProjectRepository `json:"repositories"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// OrgProjectList is a paginated organization project response.
// swagger:response OrgProjectList
type OrgProjectList struct {
	Projects []OrgProject `json:"projects"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

// CreateOrgProjectOption creates an organization project.
// swagger:model CreateOrgProjectOption
type CreateOrgProjectOption struct {
	Slug         string                 `json:"slug" binding:"Required"`
	Name         string                 `json:"name" binding:"Required"`
	Description  string                 `json:"description"`
	Values       map[string]json.Value  `json:"values"`
	Repositories []OrgProjectRepository `json:"repositories"`
	RequestID    string                 `json:"request_id" binding:"Required"`
}

// EditOrgProjectOption updates an organization project.
// swagger:model EditOrgProjectOption
type EditOrgProjectOption struct {
	Version     int64                  `json:"version" binding:"Required"`
	Slug        *string                `json:"slug,omitempty"`
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Lifecycle   *string                `json:"lifecycle,omitempty"`
	Values      *map[string]json.Value `json:"values,omitempty"`
	RequestID   string                 `json:"request_id" binding:"Required"`
}

// LinkOrgProjectRepositoryOption links a repository to an organization project.
// swagger:model LinkOrgProjectRepositoryOption
type LinkOrgProjectRepositoryOption struct {
	Role      string `json:"role" binding:"Required"`
	Version   int64  `json:"version" binding:"Required"`
	RequestID string `json:"request_id" binding:"Required"`
}

// OrgProjectChange represents an audited organization project mutation.
type OrgProjectChange struct {
	ID            int64      `json:"id"`
	ActorID       int64      `json:"actor_id"`
	RequestID     string     `json:"request_id"`
	ChangedFields json.Value `json:"changed_fields"`
	Before        json.Value `json:"before"`
	After         json.Value `json:"after"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
}

// OrgProjectChangeList is an organization project change history response.
// swagger:response OrgProjectChangeList
type OrgProjectChangeList []OrgProjectChange

// OrgProjectConfigVersion represents one configuration snapshot.
// swagger:response OrgProjectConfigVersion
type OrgProjectConfigVersion struct {
	ID             int64      `json:"id"`
	Version        int64      `json:"version"`
	State          string     `json:"state"`
	Schema         json.Value `json:"schema"`
	PointerVersion int64      `json:"pointer_version,omitempty"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	PublishedBy    int64      `json:"published_by,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
}

// OrgProjectConfigVersionList is a configuration history response.
// swagger:response OrgProjectConfigVersionList
type OrgProjectConfigVersionList []OrgProjectConfigVersion

// UpdateOrgProjectConfigOption saves a draft configuration.
// swagger:model UpdateOrgProjectConfigOption
type UpdateOrgProjectConfigOption struct {
	PointerVersion int64      `json:"pointer_version" binding:"Required"`
	Schema         json.Value `json:"schema" binding:"Required"`
}

// ValidateOrgProjectConfigOption validates a configuration without saving it.
// swagger:model ValidateOrgProjectConfigOption
type ValidateOrgProjectConfigOption struct {
	Schema json.Value `json:"schema" binding:"Required"`
}

// PublishOrgProjectConfigOption publishes the current draft.
// swagger:model PublishOrgProjectConfigOption
type PublishOrgProjectConfigOption struct {
	PointerVersion int64 `json:"pointer_version" binding:"Required"`
}

// RollbackOrgProjectConfigOption rolls back to a published configuration version.
// swagger:model RollbackOrgProjectConfigOption
type RollbackOrgProjectConfigOption struct {
	PointerVersion int64 `json:"pointer_version" binding:"Required"`
}
