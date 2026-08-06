// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package orgproject

import (
	"testing"
	"time"

	orgproject_model "gitea.dev/models/orgproject"
	"gitea.dev/modules/timeutil"
	"gitea.dev/services/orgproject/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldValueToRaw(t *testing.T) {
	number := 42.0
	integer, err := fieldValueToRaw(config.FieldTypeInteger, &orgproject_model.FieldValue{ValueNumber: &number})
	require.NoError(t, err)
	assert.JSONEq(t, `42`, string(integer))

	date := timeutil.TimeStamp(time.Date(2026, time.August, 5, 0, 0, 0, 0, time.Local).Unix())
	dateValue, err := fieldValueToRaw(config.FieldTypeDate, &orgproject_model.FieldValue{ValueTime: &date})
	require.NoError(t, err)
	assert.JSONEq(t, `"2026-08-05"`, string(dateValue))

	members := `[2,4]`
	memberValue, err := fieldValueToRaw(config.FieldTypeMemberArray, &orgproject_model.FieldValue{ValueJSON: &members})
	require.NoError(t, err)
	assert.JSONEq(t, `[2,4]`, string(memberValue))
}
