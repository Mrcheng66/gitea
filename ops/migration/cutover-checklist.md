# Organization Project Cutover Checklist

## Before the maintenance window

- [ ] Record the approved internal version, upstream commit, internal commit, build time, operator, and maintenance window.
- [ ] Restore the latest Gitea and Workbench backups in isolation and complete two idempotent imports.
- [ ] Reconcile project, repository link, follower, field value, audit event, configuration, and editor-team counts.
- [ ] Pass Owner, editor-team, member, non-member, and site-admin permission tests.
- [ ] Pass HTTPS/SSH Git, repository, Issue, pull request, Release, backup, and recovery checks.
- [ ] Confirm the rollback owner, rollback deadline, old image, old Compose, Nginx config, and read-only Workbench database are available.

## Maintenance window

- [ ] Announce the write freeze and record its exact start time.
- [ ] Stop old Gitea and Workbench; verify no process is writing either SQLite database.
- [ ] Create and verify final backups of Gitea data, configuration, image metadata, and Workbench SQLite.
- [ ] Run Workbench preflight against the final read-only database; save the report and require zero blockers.
- [ ] Deploy the approved customized Gitea image and run native database migrations.
- [ ] Run the Workbench import twice; save both reports and require zero additions on the second run.
- [ ] Apply the single-upstream Nginx configuration and require `nginx -t` before reload.
- [ ] Start Gitea and pass API, SQLite, JSON1, configuration-pointer, TLS, disk, and backup readiness checks.
- [ ] Run the production smoke matrix for permissions, projects, configuration, repository links, activity, and Git protocols.
- [ ] Record the exact cutover completion time and deployed commit before reopening writes.

## After cutover

- [ ] Monitor authentication, Git failures, HTTP 5xx, health checks, project writes, and backup status through the rollback window.
- [ ] Keep the old topology and Workbench database offline and unchanged; do not reverse-sync new writes.
- [ ] Close the maintenance record with reports, counts, operators, timestamps, and follow-up issues.
