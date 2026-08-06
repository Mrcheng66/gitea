// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import "fmt"

type ErrProjectNotExist struct {
	ID      int64
	OwnerID int64
	Slug    string
}

func (err ErrProjectNotExist) Error() string {
	return fmt.Sprintf("organization project does not exist [id: %d, owner_id: %d, slug: %s]", err.ID, err.OwnerID, err.Slug)
}

func IsErrProjectNotExist(err error) bool {
	_, ok := err.(ErrProjectNotExist)
	return ok
}

type ErrProjectAlreadyExists struct {
	OwnerID int64
	Slug    string
}

func (err ErrProjectAlreadyExists) Error() string {
	return fmt.Sprintf("organization project already exists [owner_id: %d, slug: %s]", err.OwnerID, err.Slug)
}

type ErrVersionConflict struct {
	Expected int64
	Actual   int64
}

func (err ErrVersionConflict) Error() string {
	return fmt.Sprintf("organization project version conflict [expected: %d, actual: %d]", err.Expected, err.Actual)
}
