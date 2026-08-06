// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOrgProjectCommand(t *testing.T) {
	command := newAdminOrgProjectCommand()
	require.Len(t, command.Commands, 2)
	assert.Equal(t, "preflight-workbench", command.Commands[0].Name)
	assert.Equal(t, "import-workbench", command.Commands[1].Name)
	for _, subcommand := range command.Commands {
		flags := make(map[string]bool)
		for _, flag := range subcommand.Flags {
			flags[flag.Names()[0]] = true
		}
		assert.True(t, flags["database"])
		assert.True(t, flags["organization"])
		assert.True(t, flags["actor"])
		assert.True(t, flags["editor-teams"])
		assert.True(t, flags["report"])
	}
}
