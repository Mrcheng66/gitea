#!/usr/bin/env bash
set -Eeuo pipefail

STACK_DIR=${STACK_DIR:-/opt/gitea-stack}
COMPOSE_FILE=${COMPOSE_FILE:-$STACK_DIR/ops/compose/compose.yaml}
COMPOSE_ENV_FILE=${COMPOSE_ENV_FILE:-$STACK_DIR/ops/compose/.env}
GITEA_DATA_DIR=${GITEA_DATA_DIR:-/srv/gitea/data}
BACKUP_WORK_ROOT=${BACKUP_WORK_ROOT:-/var/lib/gitea-stack/work}
BACKUP_STATE_DIR=${BACKUP_STATE_DIR:-/var/lib/gitea-stack}
BACKUP_LOCK_DIR=${BACKUP_LOCK_DIR:-/run/gitea-stack-backup.lock}
DOCKER_BIN=${DOCKER_BIN:-docker}
COSCLI_BIN=${COSCLI_BIN:-coscli}
JQ_BIN=${JQ_BIN:-jq}
OPENSSL_BIN=${OPENSSL_BIN:-openssl}
TAR_BIN=${TAR_BIN:-tar}
COS_PREFIX=${COS_PREFIX:-gitea}

required_vars=(COS_BUCKET COS_CONFIG_PATH)
for var_name in "${required_vars[@]}"; do
  if [[ -z ${!var_name:-} ]]; then
    printf 'Missing required variable: %s\n' "$var_name" >&2
    exit 1
  fi
done

for required_file in "$COMPOSE_FILE" "$COMPOSE_ENV_FILE" "$COS_CONFIG_PATH"; do
  if [[ ! -r "$required_file" ]]; then
    printf 'Required file is not readable: %s\n' "$required_file" >&2
    exit 1
  fi
done
if [[ ! -d "$GITEA_DATA_DIR" ]]; then
  printf 'Gitea data directory does not exist: %s\n' "$GITEA_DATA_DIR" >&2
  exit 1
fi

compose_file_relative=${COMPOSE_FILE#"$STACK_DIR"/}
compose_env_relative=${COMPOSE_ENV_FILE#"$STACK_DIR"/}
if [[ $compose_file_relative == "$COMPOSE_FILE" || $compose_env_relative == "$COMPOSE_ENV_FILE" ]]; then
  printf 'Compose files must be located below STACK_DIR: %s\n' "$STACK_DIR" >&2
  exit 1
fi

env_value() {
  local key=$1
  awk -F= -v key="$key" '$1 == key {print substr($0, index($0, "=") + 1); exit}' "$COMPOSE_ENV_FILE"
}

gitea_image=$(env_value GITEA_IMAGE)
gitea_version=$(env_value GITEA_VERSION)
gitea_upstream_commit=$(env_value GITEA_UPSTREAM_COMMIT)
gitea_internal_commit=$(env_value GITEA_INTERNAL_COMMIT)
gitea_build_date=$(env_value GITEA_BUILD_DATE)
for metadata_value in "$gitea_image" "$gitea_version" "$gitea_upstream_commit" "$gitea_internal_commit" "$gitea_build_date"; do
  if [[ -z $metadata_value ]]; then
    printf 'Gitea image metadata is incomplete in %s\n' "$COMPOSE_ENV_FILE" >&2
    exit 1
  fi
done
gitea_image="${gitea_image}:${gitea_version}"

mkdir -p "$BACKUP_WORK_ROOT" "$BACKUP_STATE_DIR"
if ! mkdir "$BACKUP_LOCK_DIR" 2>/dev/null; then
  printf 'Another backup is already running: %s\n' "$BACKUP_LOCK_DIR" >&2
  exit 75
fi
date +%s >"$BACKUP_LOCK_DIR/started_at"

work_dir=$(mktemp -d "${BACKUP_WORK_ROOT%/}/backup.XXXXXX")
gitea_stopped=0
remote_temp_created=0
remote_temp_uri=
failure_context='backup initialization'

compose() {
  "$DOCKER_BIN" compose \
    --project-directory "$STACK_DIR" \
    --env-file "$COMPOSE_ENV_FILE" \
    -f "$COMPOSE_FILE" \
    "$@"
}

coscli() {
  "$COSCLI_BIN" --config-path "$COS_CONFIG_PATH" --disable-log "$@"
}

cleanup() {
  local status=$?
  trap - EXIT
  set +e

  if [[ $gitea_stopped -eq 1 ]]; then
    if ! compose start gitea; then
      status=1
      printf '[CRITICAL] Platform backup could not restart Gitea on %s.\n' "${GITEA_DOMAIN:-unknown-host}" >&2
    fi
  fi

  if [[ $status -ne 0 ]]; then
    if [[ $remote_temp_created -eq 1 && -n $remote_temp_uri ]]; then
      coscli rm --force --fail-output=false "$remote_temp_uri" >/dev/null 2>&1
    fi
    printf '[ERROR] Platform backup failed on %s: %s.\n' \
      "${GITEA_DOMAIN:-unknown-host}" "$failure_context" >&2
  fi

  case "$work_dir" in
    "${BACKUP_WORK_ROOT%/}"/backup.*) rm -rf -- "$work_dir" ;;
  esac
  rm -f -- "$BACKUP_LOCK_DIR/started_at"
  rmdir "$BACKUP_LOCK_DIR" 2>/dev/null
  exit "$status"
}
trap cleanup EXIT

disk_percent=$(df -P "$GITEA_DATA_DIR" | awk 'NR == 2 {gsub(/%/, "", $5); print $5}')
if [[ ! $disk_percent =~ ^[0-9]+$ || $disk_percent -ge 80 ]]; then
  failure_context="disk usage for ${GITEA_DATA_DIR} is ${disk_percent:-unknown}%"
  exit 1
fi

timestamp=$(date -u +'%Y%m%dT%H%M%SZ')
archive="$work_dir/gitea-backup-${timestamp}.tar.gz"
verify_temp="$work_dir/verify-temp.tar.gz"
verify_current="$work_dir/verify-current.tar.gz"
metadata="$work_dir/backup-metadata.txt"
remote_temp_uri="cos://${COS_BUCKET}/${COS_PREFIX}/tmp/gitea-backup-${timestamp}.tar.gz"
remote_current_uri="cos://${COS_BUCKET}/${COS_PREFIX}/current/gitea-backup.tar.gz"

printf 'created_at=%s\ngitea_image=%s\ngitea_upstream_commit=%s\ngitea_internal_commit=%s\ngitea_build_date=%s\n' \
  "$timestamp" "$gitea_image" "$gitea_upstream_commit" "$gitea_internal_commit" "$gitea_build_date" >"$metadata"

failure_context='stopping Gitea for a consistent archive'
compose stop --timeout 60 gitea
gitea_stopped=1

failure_context='creating the local archive'
"$TAR_BIN" -czf "$archive" \
  -C "$(dirname "$GITEA_DATA_DIR")" "$(basename "$GITEA_DATA_DIR")" \
  -C "$STACK_DIR" "$compose_file_relative" "$compose_env_relative" \
  -C "$work_dir" backup-metadata.txt

failure_context='restarting Gitea after archiving'
compose start gitea
gitea_stopped=0

failure_context='calculating the local archive checksum'
checksum=$("$OPENSSL_BIN" dgst -sha256 -r "$archive" | awk '{print $1}')
archive_size=$(wc -c <"$archive" | tr -d ' ')
if [[ ! $checksum =~ ^[0-9a-fA-F]{64}$ || ! $archive_size =~ ^[0-9]+$ ]]; then
  exit 1
fi

object_metadata="x-cos-meta-sha256:${checksum}#x-cos-meta-gitea-image:${gitea_image}#x-cos-meta-gitea-revision:${gitea_internal_commit}"

failure_context='uploading the temporary COS object'
remote_temp_created=1
coscli cp "$archive" "$remote_temp_uri" \
  --encryption-type SSE-COS \
  --server-side-encryption AES256 \
  --meta "$object_metadata" \
  --process-log=false \
  --fail-output=false

failure_context='verifying the temporary COS object'
coscli cp "$remote_temp_uri" "$verify_temp" --process-log=false --fail-output=false
verify_checksum=$("$OPENSSL_BIN" dgst -sha256 -r "$verify_temp" | awk '{print $1}')
[[ $verify_checksum == "$checksum" ]]

failure_context='promoting the verified COS object'
coscli cp "$remote_temp_uri" "$remote_current_uri" \
  --encryption-type SSE-COS \
  --server-side-encryption AES256 \
  --meta "$object_metadata" \
  --process-log=false \
  --fail-output=false

failure_context='verifying the current COS object'
coscli cp "$remote_current_uri" "$verify_current" --process-log=false --fail-output=false
current_checksum=$("$OPENSSL_BIN" dgst -sha256 -r "$verify_current" | awk '{print $1}')
[[ $current_checksum == "$checksum" ]]

failure_context='removing the temporary COS object'
coscli rm --force --fail-output=false "$remote_temp_uri"
remote_temp_created=0

failure_context='recording backup success state'
state_temp="$BACKUP_STATE_DIR/last-successful-backup.json.tmp"
"$JQ_BIN" -n \
  --arg completed_at "$timestamp" \
  --argjson completed_at_epoch "$(date +%s)" \
  --arg object "$remote_current_uri" \
  --arg sha256 "$checksum" \
  --argjson size_bytes "$archive_size" \
  '{
    completed_at: $completed_at,
    completed_at_epoch: $completed_at_epoch,
    object: $object,
    sha256: $sha256,
    size_bytes: $size_bytes
  }' >"$state_temp"
mv "$state_temp" "$BACKUP_STATE_DIR/last-successful-backup.json"

printf 'Backup completed: %s (%s bytes, sha256 %s)\n' "$remote_current_uri" "$archive_size" "$checksum"
