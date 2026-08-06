// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterModelInitFunc(t *testing.T) {
	oldModels, oldInitFuncs := registeredModels, registeredInitFuncs
	t.Cleanup(func() {
		registeredModels = oldModels
		registeredInitFuncs = oldInitFuncs
	})
	registeredModels = nil
	registeredInitFuncs = nil

	initFunc := func() error { return nil }
	RegisterModel(new(struct{}), initFunc)

	assert.Len(t, registeredModels, 1)
	assert.Len(t, registeredInitFuncs, 1)
}
