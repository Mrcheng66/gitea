// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"fmt"
	"sort"
	"strings"
)

type ErrNotFound struct {
	ProjectID int64
	OwnerID   int64
}

func (err ErrNotFound) Error() string {
	return fmt.Sprintf("organization project not found [id: %d, owner_id: %d]", err.ProjectID, err.OwnerID)
}

func IsErrNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}

type ErrForbidden struct{}

func (ErrForbidden) Error() string { return "organization project operation is forbidden" }

func IsErrForbidden(err error) bool {
	_, ok := err.(ErrForbidden)
	return ok
}

type ErrConflict struct {
	Expected int64
	Actual   int64
	Field    string
}

func (err ErrConflict) Error() string {
	if err.Field != "" {
		return fmt.Sprintf("organization project conflict [field: %s]", err.Field)
	}
	return fmt.Sprintf("organization project version conflict [expected: %d, actual: %d]", err.Expected, err.Actual)
}

func IsErrConflict(err error) bool {
	_, ok := err.(ErrConflict)
	return ok
}

type ErrRepositoryNotVisible struct {
	RepositoryID int64
}

func (err ErrRepositoryNotVisible) Error() string {
	return fmt.Sprintf("repository is not visible for organization project [id: %d]", err.RepositoryID)
}

func IsErrRepositoryNotVisible(err error) bool {
	_, ok := err.(ErrRepositoryNotVisible)
	return ok
}

type ValidationErrors map[string]string

func (errs ValidationErrors) Error() string {
	keys := make([]string, 0, len(errs))
	for key := range errs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	messages := make([]string, 0, len(keys))
	for _, key := range keys {
		messages = append(messages, key+": "+errs[key])
	}
	return strings.Join(messages, "; ")
}

func IsValidationErrors(err error) bool {
	_, ok := err.(ValidationErrors)
	return ok
}
