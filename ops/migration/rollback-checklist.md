# Organization Project Rollback Checklist

## Trigger and authorization

- [ ] Confirm a rollback trigger: authentication or Git failure, migration count mismatch, permission leak, configuration failure, inconsistent project writes, or failed backup recovery.
- [ ] Record the trigger, detection time, decision maker, operator, and last known safe backup.
- [ ] Freeze writes and announce rollback; do not attempt reverse synchronization to Workbench.

## Restore the old topology

- [ ] Stop the customized Gitea and preserve its logs, image metadata, database, and import reports for investigation.
- [ ] Restore the pre-cutover Gitea data and configuration from the verified backup.
- [ ] Restore the exact old Gitea and Workbench images, Compose file, Workbench env file, and Nginx routing configuration.
- [ ] Restore the read-only Workbench backup to its original runtime location and permissions.
- [ ] Start old Gitea and Workbench without allowing either to read the customized Gitea database.
- [ ] Validate both local health endpoints, then require `nginx -t` before restoring the old path split.

## Validate and reopen

- [ ] Verify login, Workbench OAuth, project profiles, repository visibility, HTTPS/SSH Git, repository, Issue, pull request, and Release flows.
- [ ] Verify backup and health timers use the restored topology and complete one manual backup.
- [ ] Record the rollback completion time and reopen writes only after the smoke matrix passes.
- [ ] Preserve the failed cutover artifacts until the incident review approves disposal.
