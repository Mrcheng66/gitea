// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"

	"gitea.dev/modules/json"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeSchema(t *testing.T) {
	payload, err := json.Marshal(config.DefaultSchema())
	require.NoError(t, err)

	schema, err := decodeSchema(string(payload))
	require.NoError(t, err)
	assert.Equal(t, config.SchemaVersion, schema.SchemaVersion)
	assert.NotEmpty(t, schema.Fields)

	_, err = decodeSchema(`{"schema_version":`)
	assert.Error(t, err)
}
