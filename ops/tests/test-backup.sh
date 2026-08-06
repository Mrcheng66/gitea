#!/usr/bin/env bash
set -euo pipefail

OPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf -- "$test_dir"' EXIT

stack_dir="$test_dir/stack"
compose_dir="$stack_dir/ops/compose"
data_dir="$test_dir/storage/data"
state_dir="$test_dir/state"
work_root="$test_dir/work"
lock_dir="$test_dir/backup.lock"
remote_root="$test_dir/remote"
docker_log="$test_dir/docker.log"
coscli_log="$test_dir/coscli.log"

mkdir -p "$compose_dir" "$data_dir/repositories/team/project.git" "$state_dir" "$work_root" "$remote_root"
cp "$OPS_DIR/compose/compose.yaml" "$compose_dir/compose.yaml"
cp "$OPS_DIR/compose/.env.example" "$compose_dir/.env"
printf 'repository data\n' >"$data_dir/repositories/team/project.git/HEAD"
printf '{}\n' >"$test_dir/cos.yaml"
: >"$docker_log"
: >"$coscli_log"

export STACK_DIR="$stack_dir"
export COMPOSE_FILE="$compose_dir/compose.yaml"
export COMPOSE_ENV_FILE="$compose_dir/.env"
export GITEA_DATA_DIR="$data_dir"
export BACKUP_WORK_ROOT="$work_root"
export BACKUP_STATE_DIR="$state_dir"
export BACKUP_LOCK_DIR="$lock_dir"
export DOCKER_BIN="$OPS_DIR/tests/stubs/docker"
export COSCLI_BIN="$OPS_DIR/tests/stubs/coscli"
export COS_CONFIG_PATH="$test_dir/cos.yaml"
export COS_BUCKET=test-bucket-1250000000
export COS_PREFIX=gitea
export COS_REMOTE_ROOT="$remote_root"
export DOCKER_LOG="$docker_log"
export COSCLI_LOG="$coscli_log"
export GITEA_DOMAIN=git.test.example

"$OPS_DIR/scripts/backup.sh"

current_backup="$remote_root/$COS_BUCKET/$COS_PREFIX/current/gitea-backup.tar.gz"
[[ -s $current_backup ]]
[[ -s "$state_dir/last-successful-backup.json" ]]
grep -q 'stop --timeout 60 gitea' "$docker_log"
grep -q 'start gitea' "$docker_log"
if grep -q 'workbench' "$docker_log"; then
  printf 'Backup must not manage the retired Workbench service\n' >&2
  exit 1
fi
tar -tzf "$current_backup" | grep -q 'project.git/HEAD'
tar -tzf "$current_backup" | grep -q 'ops/compose/compose.yaml'
tar -tzf "$current_backup" | grep -q 'ops/compose/.env'
if tar -tzf "$current_backup" | grep -q 'workbench'; then
  printf 'Backup must not archive legacy Workbench runtime data\n' >&2
  exit 1
fi
tar -xOzf "$current_backup" backup-metadata.txt | grep -q '^gitea_image=code-lab/gitea:1.27.1-internal.1$'
tar -xOzf "$current_backup" backup-metadata.txt | grep -q '^gitea_upstream_commit=a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f$'
tar -xOzf "$current_backup" backup-metadata.txt | grep -q '^gitea_internal_commit=replace-with-release-commit$'
tar -xOzf "$current_backup" backup-metadata.txt | grep -q '^gitea_build_date=2026-08-05T00:00:00Z$'
[[ ! -d $lock_dir ]]

real_tar=$(command -v tar)
: >"$docker_log"
export TAR_BIN="$OPS_DIR/tests/stubs/tar-fail"
archive_error="$test_dir/archive-error.log"

if "$OPS_DIR/scripts/backup.sh" 2>"$archive_error"; then
  printf 'Expected backup failure when local archiving fails\n' >&2
  exit 1
fi

grep -q 'stop --timeout 60 gitea' "$docker_log"
grep -q 'start gitea' "$docker_log"
grep -q 'creating the local archive' "$archive_error"
[[ ! -d $lock_dir ]]

checksum_before=$(openssl dgst -sha256 -r "$current_backup" | awk '{print $1}')
: >"$docker_log"
export TAR_BIN="$real_tar"
export COSCLI_FAIL_UPLOAD=1
upload_error="$test_dir/upload-error.log"

if "$OPS_DIR/scripts/backup.sh" 2>"$upload_error"; then
  printf 'Expected backup failure when COS upload fails\n' >&2
  exit 1
fi

checksum_after=$(openssl dgst -sha256 -r "$current_backup" | awk '{print $1}')
[[ $checksum_after == "$checksum_before" ]]
grep -q 'start gitea' "$docker_log"
grep -q 'uploading the temporary COS object' "$upload_error"
[[ ! -d $lock_dir ]]

printf 'backup tests passed\n'
