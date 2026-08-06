// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitea.dev/modules/json"
	orgproject_migration "gitea.dev/services/orgproject/migration"

	"github.com/urfave/cli/v3"
)

func newAdminOrgProjectCommand() *cli.Command {
	return &cli.Command{
		Name:  "org-project",
		Usage: "Manage organization project data",
		Commands: []*cli.Command{
			newAdminOrgProjectMigrationCommand("preflight-workbench", "Validate a Workbench database without changing Gitea", runAdminOrgProjectPreflight),
			newAdminOrgProjectMigrationCommand("import-workbench", "Import a validated Workbench database", runAdminOrgProjectImport),
		},
	}
}

func newAdminOrgProjectMigrationCommand(name, usage string, action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name: name, Usage: usage, Action: action,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database", Usage: "path to the read-only Workbench SQLite database", Required: true},
			&cli.StringFlag{Name: "organization", Usage: "target Gitea organization", Required: true},
			&cli.StringFlag{Name: "actor", Usage: "target organization owner used for migration audit", Required: true},
			&cli.StringFlag{Name: "editor-teams", Usage: "comma-separated target organization editor teams", Value: "Owners"},
			&cli.StringFlag{Name: "report", Usage: "path for the JSON migration report"},
		},
	}
}

func runAdminOrgProjectPreflight(ctx context.Context, command *cli.Command) error {
	if err := initDB(ctx); err != nil {
		return err
	}
	report, err := orgproject_migration.Preflight(ctx, orgProjectMigrationOptions(command))
	if report != nil {
		if writeErr := writeOrgProjectMigrationReport(command, report); writeErr != nil {
			return writeErr
		}
		if report.HasBlockers() {
			return orgproject_migration.ErrBlocked{Count: report.Summary.Blocked}
		}
	}
	return err
}

func runAdminOrgProjectImport(ctx context.Context, command *cli.Command) error {
	if err := initDB(ctx); err != nil {
		return err
	}
	report, err := orgproject_migration.Import(ctx, orgProjectMigrationOptions(command))
	if report != nil {
		if writeErr := writeOrgProjectMigrationReport(command, report); writeErr != nil {
			return writeErr
		}
	}
	var blocked orgproject_migration.ErrBlocked
	if errors.As(err, &blocked) {
		return fmt.Errorf("%w; review the JSON report before retrying", err)
	}
	return err
}

func orgProjectMigrationOptions(command *cli.Command) orgproject_migration.Options {
	teams := strings.Split(command.String("editor-teams"), ",")
	return orgproject_migration.Options{
		DatabasePath: command.String("database"), Organization: command.String("organization"),
		Actor: command.String("actor"), EditorTeams: teams,
	}
}

func writeOrgProjectMigrationReport(command *cli.Command, report *orgproject_migration.Report) error {
	if err := orgproject_migration.WriteReport(command.String("report"), report); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = command.Writer.Write(data)
	return err
}
