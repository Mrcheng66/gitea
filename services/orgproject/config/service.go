// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"errors"
	"fmt"

	"gitea.dev/models/db"
	"gitea.dev/models/orgproject"
	"gitea.dev/modules/json"
)

type ErrConfigUninitialized struct{ OwnerID int64 }

func (err ErrConfigUninitialized) Error() string {
	return fmt.Sprintf("organization project configuration is not initialized [owner_id: %d]", err.OwnerID)
}

type ErrConfigConflict struct {
	Expected int64
	Actual   int64
}

// ErrConfigValidation reports a configuration rejected before persistence or publication.
type ErrConfigValidation struct{ Err error }

func (err ErrConfigValidation) Error() string { return err.Err.Error() }
func (err ErrConfigValidation) Unwrap() error { return err.Err }

// IsErrConfigValidation reports whether err is a configuration validation error.
func IsErrConfigValidation(err error) bool {
	var target ErrConfigValidation
	return errors.As(err, &target)
}

func (err ErrConfigConflict) Error() string {
	return fmt.Sprintf("organization project configuration conflict [expected: %d, actual: %d]", err.Expected, err.Actual)
}

func IsErrConfigConflict(err error) bool {
	var target ErrConfigConflict
	return errors.As(err, &target)
}

func InitializeDefaultDraft(ctx context.Context, ownerID, actorID int64) (*orgproject.ConfigVersion, *orgproject.ConfigPointer, error) {
	var version *orgproject.ConfigVersion
	var pointer *orgproject.ConfigPointer
	err := db.WithTx(ctx, func(ctx context.Context) error {
		exists, err := db.GetEngine(ctx).ID(ownerID).Exist(new(orgproject.ConfigPointer))
		if err != nil {
			return err
		}
		if exists {
			return ErrConfigConflict{Expected: 0, Actual: 1}
		}
		payload, err := CanonicalJSON(DefaultSchema())
		if err != nil {
			return err
		}
		version = &orgproject.ConfigVersion{
			OwnerID: ownerID, Version: 1, State: orgproject.ConfigStateDraft, Payload: string(payload), CreatedBy: actorID,
		}
		if _, err := db.GetEngine(ctx).Insert(version); err != nil {
			return err
		}
		pointer = &orgproject.ConfigPointer{OwnerID: ownerID, DraftVersionID: version.ID, Version: 1}
		_, err = db.GetEngine(ctx).Insert(pointer)
		return err
	})
	return version, pointer, err
}

func GetDraft(ctx context.Context, ownerID int64) (*orgproject.ConfigVersion, error) {
	pointer, err := getPointer(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if pointer.DraftVersionID == 0 {
		return nil, ErrConfigUninitialized{OwnerID: ownerID}
	}
	return getConfigVersion(ctx, ownerID, pointer.DraftVersionID, orgproject.ConfigStateDraft)
}

// GetPointer returns the current organization project configuration pointer.
func GetPointer(ctx context.Context, ownerID int64) (*orgproject.ConfigPointer, error) {
	return getPointer(ctx, ownerID)
}

func GetPublished(ctx context.Context, ownerID int64) (*orgproject.ConfigVersion, error) {
	pointer, err := getPointer(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if pointer.PublishedVersionID == 0 {
		return nil, ErrConfigUninitialized{OwnerID: ownerID}
	}
	return getConfigVersion(ctx, ownerID, pointer.PublishedVersionID, orgproject.ConfigStatePublished)
}

func SaveDraft(ctx context.Context, ownerID, actorID, expectedPointerVersion int64, schema Schema) (*orgproject.ConfigVersion, error) {
	payload, err := CanonicalJSON(schema)
	if err != nil {
		return nil, ErrConfigValidation{Err: err}
	}
	var saved *orgproject.ConfigVersion
	err = db.WithTx(ctx, func(ctx context.Context) error {
		pointer, err := getPointer(ctx, ownerID)
		if err != nil {
			return err
		}
		if pointer.Version != expectedPointerVersion {
			return ErrConfigConflict{Expected: expectedPointerVersion, Actual: pointer.Version}
		}
		nextVersion, err := nextConfigVersion(ctx, ownerID)
		if err != nil {
			return err
		}
		saved = &orgproject.ConfigVersion{
			OwnerID: ownerID, Version: nextVersion, State: orgproject.ConfigStateDraft, Payload: string(payload), CreatedBy: actorID,
		}
		if _, err := db.GetEngine(ctx).Insert(saved); err != nil {
			return err
		}
		return updatePointer(ctx, ownerID, expectedPointerVersion, "draft_version_id", saved.ID)
	})
	return saved, err
}

// GetPublishedVersion returns a published configuration by its logical version number.
func GetPublishedVersion(ctx context.Context, ownerID, versionNumber int64) (*orgproject.ConfigVersion, error) {
	version := &orgproject.ConfigVersion{OwnerID: ownerID, Version: versionNumber, State: orgproject.ConfigStatePublished}
	has, err := db.GetEngine(ctx).Get(version)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrConfigUninitialized{OwnerID: ownerID}
	}
	return version, nil
}

func ListHistory(ctx context.Context, ownerID int64, limit int) ([]*orgproject.ConfigVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	versions := make([]*orgproject.ConfigVersion, 0, limit)
	err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Desc("version").Limit(limit).Find(&versions)
	return versions, err
}

func getPointer(ctx context.Context, ownerID int64) (*orgproject.ConfigPointer, error) {
	pointer := &orgproject.ConfigPointer{OwnerID: ownerID}
	has, err := db.GetEngine(ctx).Get(pointer)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrConfigUninitialized{OwnerID: ownerID}
	}
	return pointer, nil
}

func getConfigVersion(ctx context.Context, ownerID, id int64, state orgproject.ConfigState) (*orgproject.ConfigVersion, error) {
	version := &orgproject.ConfigVersion{ID: id, OwnerID: ownerID, State: state}
	has, err := db.GetEngine(ctx).Get(version)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrConfigUninitialized{OwnerID: ownerID}
	}
	return version, nil
}

func nextConfigVersion(ctx context.Context, ownerID int64) (int64, error) {
	latest := new(orgproject.ConfigVersion)
	has, err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Desc("version").Get(latest)
	if err != nil {
		return 0, err
	}
	if !has {
		return 1, nil
	}
	return latest.Version + 1, nil
}

func updatePointer(ctx context.Context, ownerID, expectedVersion int64, column string, configVersionID int64) error {
	if column != "draft_version_id" && column != "published_version_id" {
		return fmt.Errorf("unsupported organization project pointer column %q", column)
	}
	result, err := db.GetEngine(ctx).Exec(
		"UPDATE org_project_config_pointer SET "+column+" = ?, version = version + 1 WHERE owner_id = ? AND version = ?",
		configVersionID, ownerID, expectedVersion,
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		pointer, getErr := getPointer(ctx, ownerID)
		if getErr != nil {
			return getErr
		}
		return ErrConfigConflict{Expected: expectedVersion, Actual: pointer.Version}
	}
	return nil
}

func decodeSchema(payload string) (Schema, error) {
	var schema Schema
	if err := json.Unmarshal([]byte(payload), &schema); err != nil {
		return Schema{}, fmt.Errorf("decode organization project configuration: %w", err)
	}
	return schema, nil
}
