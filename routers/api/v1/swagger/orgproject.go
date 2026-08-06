// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import api "gitea.dev/modules/structs"

// OrgProject
// swagger:response OrgProject
type swaggerResponseOrgProject struct {
	// in:body
	Body api.OrgProject `json:"body"`
}

// OrgProjectList
// swagger:response OrgProjectList
type swaggerResponseOrgProjectList struct {
	// in:body
	Body api.OrgProjectList `json:"body"`
}

// OrgProjectChangeList
// swagger:response OrgProjectChangeList
type swaggerResponseOrgProjectChangeList struct {
	// in:body
	Body api.OrgProjectChangeList `json:"body"`
}

// OrgProjectConfigVersion
// swagger:response OrgProjectConfigVersion
type swaggerResponseOrgProjectConfigVersion struct {
	// in:body
	Body api.OrgProjectConfigVersion `json:"body"`
}

// OrgProjectConfigVersionList
// swagger:response OrgProjectConfigVersionList
type swaggerResponseOrgProjectConfigVersionList struct {
	// in:body
	Body api.OrgProjectConfigVersionList `json:"body"`
}
