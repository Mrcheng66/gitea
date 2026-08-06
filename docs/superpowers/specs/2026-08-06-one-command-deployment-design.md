# One-Command Production Deployment Design

## Purpose

Provide one production command for the internal Gitea fork:

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
```

The command deploys the latest `origin/main`, creates a consistent local backup before changing the running version, retains the latest five release backups, verifies local and public health, and automatically restores the previous code, configuration, image, and data when the new release fails.

Frontend-only changes, Go business-logic changes, and schema-affecting changes use the same server command. Their differences remain development-time formatting, linting, and test requirements rather than production operator steps.

## Goals

- Make production deployment a single command with no version argument.
- Deploy the exact commit currently at `origin/main`.
- Keep the existing `/srv/gitea/data` data across successful releases.
- Create a consistent pre-release backup and abort if backup creation or verification fails.
- Keep the running service available while the new image builds.
- Automatically roll back failures after the new container is started.
- Retain the five newest release backup directories.
- Make deployment status, logs, release metadata, and rollback outcomes auditable.
- Prevent concurrent deployments and accidental deployment from a dirty tracked worktree.

## Non-goals

- Automatically deploy when GitHub receives a push.
- Deploy branches other than `origin/main`.
- Replace the existing Nginx, TLS, Docker, or Compose installation.
- Delete Docker images or unrelated Docker resources automatically.
- Back up to COS; the release backup is local and independent of the existing COS backup service.
- Provide zero-downtime database migration. Gitea is briefly stopped while its data is archived and again while the container is replaced.

## Operator Interface

The script supports four commands:

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
sudo /opt/gitea-stack/ops/scripts/deploy.sh deploy
sudo /opt/gitea-stack/ops/scripts/deploy.sh status
sudo /opt/gitea-stack/ops/scripts/deploy.sh rollback
sudo /opt/gitea-stack/ops/scripts/deploy.sh help
```

No argument is equivalent to `deploy`.

- `deploy` fetches and deploys the latest `origin/main`.
- `status` prints the deployed commit, image, last deployment result, container status, and local/public health results without changing state.
- `rollback` restores the newest valid pre-release backup that has not already been consumed by a successful rollback.
- `help` prints usage and the important filesystem paths.

All modifying commands require root. `status` may run as root as well so its behavior is consistent with protected state files.

## Filesystem and Runtime Layout

| Purpose | Path |
| --- | --- |
| Source checkout | `/opt/gitea-stack` |
| Compose environment | `/opt/gitea-stack/ops/compose/.env` |
| Compose definition | `/opt/gitea-stack/ops/compose/compose.yaml` |
| Persistent Gitea data | `/srv/gitea/data` |
| Release state root | `/var/lib/gitea-stack` |
| Release backups | `/var/lib/gitea-stack/releases` |
| Temporary script/runtime files | `/var/lib/gitea-stack/run` |
| Deployment lock | `/run/gitea-stack-deploy.lock` |
| Append-only deployment log | `/var/lib/gitea-stack/deployments.log` |
| Current release state | `/var/lib/gitea-stack/current-release.env` |

The script copies itself to the protected runtime directory and re-executes the copy before changing the Git checkout. This prevents `git checkout` from replacing the script file while it is running and allows a newly deployed script version to be used on the following invocation.

## Release Identity

The deployment target is resolved with:

```bash
git fetch --prune origin main
git rev-parse origin/main^{commit}
```

The internal image version is generated from the upstream baseline and target commit:

```text
1.27.1-internal.${SHORT_COMMIT}
```

For example:

```text
1.27.1-internal.817a183823ab
```

The script updates only these environment values for each release:

- `GITEA_VERSION`
- `GITEA_INTERNAL_COMMIT`
- `GITEA_BUILD_DATE`

It preserves the configured domain, image repository, upstream commit, and data directory. The build date uses UTC RFC3339 format.

If the target commit equals the recorded deployed commit, `deploy` performs health checks and exits successfully without creating another backup or rebuilding the image.

## Preconditions

Before modifying production, the script verifies:

- It is running as root.
- `git`, `docker`, Docker Compose v2, `curl`, `tar`, `sha256sum`, `flock`, and required core utilities are available.
- `/opt/gitea-stack` is the expected Git repository.
- The configured remote is available and `origin/main` resolves to a commit.
- Tracked files in the production checkout have no local modifications.
- The Compose environment and definition files exist.
- `GITEA_DATA_DIR` resolves to `/srv/gitea/data`; arbitrary data paths are rejected by the initial implementation.
- The data directory exists and is not a symbolic link.
- The filesystem containing the backup root has sufficient free space for a data archive plus a safety margin.
- No other deployment holds the `flock` lock.

A failed precondition exits without stopping or replacing the running Gitea container.

## Backup Design

Each deployment creates a directory named with the UTC timestamp and previous commit:

```text
/var/lib/gitea-stack/releases/20260806T143000Z-817a183823ab/
```

It contains:

- `gitea-data.tar.gz`: complete `/srv/gitea/data` archive.
- `gitea-data.tar.gz.sha256`: archive checksum.
- `compose.env`: exact pre-release `ops/compose/.env`.
- `compose.yaml`: exact pre-release Compose definition.
- `release.env`: previous commit, image, version, build date, timestamp, and backup status.
- `deploy.log`: log for the deployment associated with this backup.

For consistency, the script stops Gitea, creates the archive, verifies the gzip/tar structure and checksum, then restarts the old container before building the new image. A trap attempts to restart the old container if backup creation exits unexpectedly after the stop.

Backup failure is a hard failure: the target commit is not checked out and no new image is activated.

After a successful deployment or completed automatic rollback, backup directories are sorted by creation timestamp and only the newest five are retained. Pruning is restricted to directories underneath the canonical release backup root and rejects unexpected paths.

## Deployment State Machine

### 1. Initialize

- Re-execute a protected temporary copy of the script.
- Acquire the exclusive deployment lock.
- Create a per-run log.
- Validate prerequisites and current state.

### 2. Resolve target

- Fetch `origin/main`.
- Resolve and record the immutable target commit.
- Exit successfully after health checks if it is already deployed.

### 3. Capture previous release

- Record the current Git commit.
- Copy the current Compose environment and definition.
- Resolve the current image reference from the environment.
- Confirm the current container can be managed by the expected Compose project.

### 4. Back up data

- Stop Gitea with a bounded timeout.
- Archive and verify `/srv/gitea/data`.
- Restart the previous container.
- Verify the local health endpoint before continuing.

### 5. Prepare target source and metadata

- Check out the resolved target commit in detached-HEAD mode.
- Generate the commit-derived image version and UTC build date.
- Update only the three release metadata values in `.env`.
- Verify `GITEA_INTERNAL_COMMIT` exactly equals `git rev-parse HEAD`.
- Render and validate the Compose configuration.

### 6. Build

- Build the new `gitea` image with `--pull` while the previous container remains running.
- Verify the resulting image exists and its revision label equals the target commit.

A source, configuration, or build failure restores the previous checkout and `.env`. Because the new container has not started and the old data has not been migrated, the data archive is retained but does not need to be restored.

### 7. Switch

- Run `docker compose up -d gitea` using the new image and metadata.
- Wait for the Compose health status and local endpoint.
- Verify `http://127.0.0.1:3000/api/healthz`.
- Verify `https://git.ghgmall.cn/api/healthz`.
- Retry health checks for up to 120 seconds with bounded request timeouts.

### 8. Commit release state

- Write `current-release.env` atomically.
- Append a successful deployment record.
- Mark the backup as available for manual rollback.
- Retain the current and previous images; do not run image pruning.
- Prune release backup directories beyond the newest five.

## Automatic Rollback

A failure after the new container starts triggers automatic rollback:

1. Stop the new container.
2. Verify the release archive and checksum again.
3. Move the failed data directory to a temporary path under the current release backup directory.
4. Restore the archived data to `/srv/gitea/data`.
5. Restore ownership to `1000:1000`.
6. Restore the previous Git commit and `compose.env`.
7. Start the previous image with the previous Compose configuration.
8. Retry local and public health checks.
9. If recovery succeeds, remove the temporary failed data copy, mark the deployment failed and rolled back, and retain the backup metadata.
10. If recovery fails, preserve both the restored data and failed data copy, emit a critical error, and stop further automated mutation.

The script must never start an older image against data that has been modified by the failed new release without first restoring the pre-release archive.

A build failure before the switch restores source and configuration only; it does not unnecessarily overwrite the still-valid production data.

## Manual Rollback

`deploy.sh rollback` selects the newest valid rollback backup, asks no interactive questions, and performs the same verified data/code/config restoration used by automatic rollback. It refuses to run when:

- another deployment holds the lock;
- no valid backup is available;
- the archive or checksum is invalid;
- required previous-release metadata is missing.

A successfully consumed rollback is marked in `release.env` so repeated `rollback` commands do not oscillate between releases.

## Logging and Exit Behavior

Console output uses short stage-oriented messages so the normal operator view remains readable:

```text
[1/6] Checking production state
[2/6] Backing up current release
[3/6] Fetching origin/main
[4/6] Building 1.27.1-internal.817a183823ab
[5/6] Switching container
[6/6] Verifying health
Deployment succeeded: 817a183823ab
```

Detailed command output is written to the per-release `deploy.log` and summarized in `/var/lib/gitea-stack/deployments.log`.

Exit codes distinguish broad failure classes:

- `0`: success or already deployed and healthy.
- `1`: validation, fetch, source, configuration, or build failure before switch.
- `2`: deployment failed but automatic rollback succeeded.
- `3`: deployment and automatic rollback both failed; operator intervention is required.
- `64`: invalid command-line usage.
- `75`: deployment lock is already held.

## Security and Safety

- Modifying commands require root because they stop containers and replace protected data.
- Environment files and backup metadata are created with restrictive permissions.
- The script never evaluates values from `.env` as shell code; it reads required keys explicitly.
- Filesystem deletion is allowed only after canonical-path validation beneath `/var/lib/gitea-stack/releases` or for the exact `/srv/gitea/data` rollback target.
- The script does not run `docker system prune`, `docker volume prune`, or `docker compose down -v`.
- Nginx, TLS certificates, firewall rules, COS objects, and unrelated Docker resources are never modified.

## Testing Strategy

Add `ops/tests/test-deploy.sh` using command stubs and temporary directories, following the existing operations test style. Tests cover:

- no-argument `deploy` and explicit subcommand parsing;
- root and prerequisite validation;
- deployment lock contention;
- clean and dirty Git worktrees;
- unchanged target commit short-circuit;
- successful backup, checksum verification, and restart;
- backup failure restarting the previous container and aborting;
- generated version, internal commit, and build date metadata;
- configuration or image build failure before switch;
- successful container switch and health verification;
- failed health check with successful automatic rollback;
- failed automatic rollback preserving recovery artifacts;
- retention of only the newest five release backups;
- `status`, `rollback`, and `help` behavior;
- refusal to delete or restore through unsafe paths.

Update `ops/tests/check.sh` so the deployment tests run with the other operations checks. Update `ops/README.md` and add `ops/docs/release.md` with the short operator workflow and local checks for frontend-only, Go logic, and schema-affecting changes.

## Acceptance Criteria

- A clean server release requires only `sudo /opt/gitea-stack/ops/scripts/deploy.sh`.
- The command deploys the exact commit fetched from `origin/main`.
- An unchanged healthy commit exits without backup or rebuild.
- Every changed release has a verified pre-release local backup.
- The previous service runs during target image construction.
- Successful deployment preserves existing users, repositories, and configuration.
- Failed post-switch health checks restore the previous data, code, environment, and image automatically.
- The newest five release backup directories remain after cleanup.
- `status` and logs identify the deployed commit and last outcome.
- Tests exercise success, pre-switch failure, rollback success, and rollback failure without requiring real Docker, GitHub, or production data.
