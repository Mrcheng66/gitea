#!/usr/bin/env bash
set -Eeuo pipefail

STACK_DIR=${STACK_DIR:-/opt/gitea-stack}
COMPOSE_FILE=${COMPOSE_FILE:-$STACK_DIR/ops/compose/compose.yaml}
COMPOSE_ENV_FILE=${COMPOSE_ENV_FILE:-$STACK_DIR/ops/compose/.env}
GITEA_DATA_DIR=${GITEA_DATA_DIR:-/srv/gitea/data}
EXPECTED_GITEA_DATA_DIR=${EXPECTED_GITEA_DATA_DIR:-/srv/gitea/data}
STATE_ROOT=${STATE_ROOT:-/var/lib/gitea-stack}
RELEASES_DIR=${RELEASES_DIR:-$STATE_ROOT/releases}
RUN_DIR=${RUN_DIR:-$STATE_ROOT/run}
CURRENT_RELEASE_FILE=${CURRENT_RELEASE_FILE:-$STATE_ROOT/current-release.env}
DEPLOYMENT_LOG=${DEPLOYMENT_LOG:-$STATE_ROOT/deployments.log}
LOCK_FILE=${LOCK_FILE:-/run/gitea-stack-deploy.lock}
REMOTE_NAME=${REMOTE_NAME:-origin}
BRANCH_NAME=${BRANCH_NAME:-main}
GITEA_BASE_VERSION=${GITEA_BASE_VERSION:-1.27.1}
BACKUP_RETENTION=${BACKUP_RETENTION:-5}
BACKUP_SAFETY_KB=${BACKUP_SAFETY_KB:-102400}
HEALTH_ATTEMPTS=${HEALTH_ATTEMPTS:-24}
HEALTH_SLEEP_SECONDS=${HEALTH_SLEEP_SECONDS:-5}
LOCAL_HEALTH_URL=${LOCAL_HEALTH_URL:-http://127.0.0.1:3000/api/healthz}
PUBLIC_HEALTH_URL=${PUBLIC_HEALTH_URL:-}
DOCKER_BIN=${DOCKER_BIN:-docker}
GIT_BIN=${GIT_BIN:-git}
CURL_BIN=${CURL_BIN:-curl}
TAR_BIN=${TAR_BIN:-tar}
SHA256_BIN=${SHA256_BIN:-sha256sum}
FLOCK_BIN=${FLOCK_BIN:-flock}
SLEEP_BIN=${SLEEP_BIN:-sleep}
GITEA_UID=${GITEA_UID:-1000}
GITEA_GID=${GITEA_GID:-1000}

COMMAND=${1:-deploy}
RUN_TIMESTAMP=${DEPLOY_TIMESTAMP:-$(date -u +'%Y%m%dT%H%M%SZ')}
RUN_LOG=
CURRENT_BACKUP_DIR=
PREVIOUS_COMMIT=
PREVIOUS_IMAGE=
PREVIOUS_VERSION=
PREVIOUS_BUILD_DATE=
TARGET_COMMIT=
TARGET_VERSION=
TARGET_IMAGE=

usage() {
  cat <<'USAGE'
Usage:
  deploy.sh [deploy]
  deploy.sh status
  deploy.sh rollback
  deploy.sh help

Commands:
  deploy    Deploy the latest origin/main commit. This is the default.
  status    Show the recorded release and current health.
  rollback  Restore the newest unused pre-release backup.
  help      Show this help text.

Production paths:
  source:   /opt/gitea-stack
  data:     /srv/gitea/data
  backups:  /var/lib/gitea-stack/releases
USAGE
}

case "$COMMAND" in
  deploy | status | rollback) ;;
  help | -h | --help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    exit 64
    ;;
esac

if [[ ${DEPLOY_SKIP_ROOT_CHECK:-0} != 1 && $EUID -ne 0 ]]; then
  printf 'This command must run as root.\n' >&2
  exit 1
fi

mkdir -p "$STATE_ROOT" "$RELEASES_DIR" "$RUN_DIR"
for protected_dir in "$STATE_ROOT" "$RELEASES_DIR" "$RUN_DIR"; do
  if [[ -L $protected_dir ]]; then
    printf 'Protected deployment directory must not be a symbolic link: %s\n' "$protected_dir" >&2
    exit 1
  fi
done
chmod 0700 "$STATE_ROOT" "$RELEASES_DIR" "$RUN_DIR"

if [[ ${DEPLOY_SKIP_REEXEC:-0} != 1 && ${GITEA_DEPLOY_REEXECUTED:-0} != 1 ]]; then
  runtime_script="$RUN_DIR/deploy-${RUN_TIMESTAMP}-$$.sh"
  cp "$0" "$runtime_script"
  chmod 0700 "$runtime_script"
  export GITEA_DEPLOY_REEXECUTED=1
  export GITEA_DEPLOY_RUNTIME_SCRIPT="$runtime_script"
  exec "$runtime_script" "$@"
fi

cleanup_runtime() {
  if [[ -n ${GITEA_DEPLOY_RUNTIME_SCRIPT:-} ]]; then
    rm -f -- "$GITEA_DEPLOY_RUNTIME_SCRIPT"
  fi
}
trap cleanup_runtime EXIT

if ! command -v "$FLOCK_BIN" >/dev/null 2>&1; then
  printf 'Required command is unavailable: %s\n' "$FLOCK_BIN" >&2
  exit 1
fi
exec 9>"$LOCK_FILE"
if ! "$FLOCK_BIN" -n 9; then
  printf 'Another deployment is already running.\n' >&2
  exit 75
fi

RUN_LOG="$RUN_DIR/${COMMAND}-${RUN_TIMESTAMP}-$$.log"
: >"$RUN_LOG"
chmod 0600 "$RUN_LOG"

log() {
  printf '%s\n' "$*" | tee -a "$RUN_LOG"
}

log_error() {
  printf '%s\n' "$*" | tee -a "$RUN_LOG" >&2
}

run_logged() {
  "$@" >>"$RUN_LOG" 2>&1
}

read_key() {
  local file=$1
  local key=$2
  [[ -r $file ]] || return 1
  awk -F= -v key="$key" '$1 == key {print substr($0, index($0, "=") + 1); exit}' "$file"
}

set_key() {
  local file=$1
  local key=$2
  local value=$3
  local temp_file="${file}.tmp.$$"
  awk -F= -v key="$key" -v value="$value" '
    BEGIN {updated = 0}
    $1 == key {print key "=" value; updated = 1; next}
    {print}
    END {if (!updated) print key "=" value}
  ' "$file" >"$temp_file"
  chmod --reference="$file" "$temp_file" 2>/dev/null || chmod 0600 "$temp_file"
  mv "$temp_file" "$file"
}

write_metadata() {
  local file=$1
  shift
  : >"$file"
  chmod 0600 "$file"
  while [[ $# -gt 0 ]]; do
    printf '%s=%s\n' "$1" "$2" >>"$file"
    shift 2
  done
}

append_summary() {
  local result=$1
  local commit=$2
  local detail=$3
  printf '%s result=%s commit=%s detail=%s\n' \
    "$RUN_TIMESTAMP" "$result" "$commit" "$detail" >>"$DEPLOYMENT_LOG"
  chmod 0600 "$DEPLOYMENT_LOG"
}

compose() {
  "$DOCKER_BIN" compose \
    --project-directory "$STACK_DIR" \
    --env-file "$COMPOSE_ENV_FILE" \
    -f "$COMPOSE_FILE" \
    "$@"
}

require_command() {
  local command_name=$1
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log_error "Required command is unavailable: $command_name"
    return 1
  fi
}

load_configuration() {
  local configured_data canonical_configured_data canonical_runtime_data domain
  for required_file in "$COMPOSE_FILE" "$COMPOSE_ENV_FILE"; do
    if [[ ! -r $required_file ]]; then
      log_error "Required file is not readable: $required_file"
      return 1
    fi
  done

  domain=$(read_key "$COMPOSE_ENV_FILE" GITEA_DOMAIN || true)
  configured_data=$(read_key "$COMPOSE_ENV_FILE" GITEA_DATA_DIR || true)
  if [[ -z $domain ]]; then
    log_error "GITEA_DOMAIN is missing from $COMPOSE_ENV_FILE"
    return 1
  fi
  if [[ ! -d $configured_data ]]; then
    log_error "Configured GITEA_DATA_DIR does not exist: $configured_data"
    return 1
  fi
  canonical_configured_data=$(cd "$configured_data" && pwd -P)
  canonical_runtime_data=$(cd "$GITEA_DATA_DIR" && pwd -P)
  if [[ $canonical_configured_data != "$canonical_runtime_data" ]]; then
    log_error 'Compose GITEA_DATA_DIR does not match the release backup directory'
    return 1
  fi
  if [[ -z $PUBLIC_HEALTH_URL ]]; then
    PUBLIC_HEALTH_URL="https://${domain}/api/healthz"
  fi
}

validate_environment() {
  local mode=${1:-modify}
  local tracked_changes canonical_data canonical_expected canonical_releases canonical_state data_kb available_kb required_kb

  for command_name in "$DOCKER_BIN" "$GIT_BIN" "$CURL_BIN" "$TAR_BIN" "$SHA256_BIN" "$FLOCK_BIN" "$SLEEP_BIN" awk cp date df du find grep mv rm sort; do
    require_command "$command_name" || return 1
  done

  if [[ ! -d $STACK_DIR/.git ]]; then
    log_error "STACK_DIR is not a Git checkout: $STACK_DIR"
    return 1
  fi
  if [[ ! -d $GITEA_DATA_DIR || -L $GITEA_DATA_DIR ]]; then
    log_error "Gitea data directory is missing or unsafe: $GITEA_DATA_DIR"
    return 1
  fi

  canonical_data=$(cd "$GITEA_DATA_DIR" && pwd -P)
  canonical_expected=$(cd "$EXPECTED_GITEA_DATA_DIR" && pwd -P)
  canonical_state=$(cd "$STATE_ROOT" && pwd -P)
  canonical_releases=$(cd "$RELEASES_DIR" && pwd -P)
  if [[ $canonical_data != "$canonical_expected" ]]; then
    log_error "Unexpected Gitea data directory: $canonical_data"
    return 1
  fi
  case "$canonical_releases" in
    "$canonical_state"/*) ;;
    *)
      log_error 'Release backup directory must be below STATE_ROOT'
      return 1
      ;;
  esac

  if [[ $mode != status ]]; then
    tracked_changes=$("$GIT_BIN" -C "$STACK_DIR" status --porcelain --untracked-files=no)
    if [[ -n $tracked_changes ]]; then
      log_error "Tracked files in $STACK_DIR contain local changes"
      return 1
    fi
  fi

  if ! run_logged "$DOCKER_BIN" compose version; then
    log_error 'Docker Compose v2 is unavailable'
    return 1
  fi

  if [[ $mode != status ]]; then
    data_kb=$(du -sk "$GITEA_DATA_DIR" | awk '{print $1}')
    available_kb=$(df -Pk "$STATE_ROOT" | awk 'NR == 2 {print $4}')
    if [[ ! $data_kb =~ ^[0-9]+$ || ! $available_kb =~ ^[0-9]+$ ]]; then
      log_error 'Unable to determine backup disk requirements'
      return 1
    fi
    required_kb=$((data_kb + data_kb / 10 + BACKUP_SAFETY_KB))
    if ((available_kb < required_kb)); then
      log_error "Insufficient backup space: need ${required_kb} KiB, have ${available_kb} KiB"
      return 1
    fi
  fi

  load_configuration
}

container_health() {
  local container_id health_status
  container_id=$(compose ps -q gitea 2>>"$RUN_LOG" || true)
  [[ -n $container_id ]] || return 1
  health_status=$("$DOCKER_BIN" inspect \
    --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' \
    "$container_id" 2>>"$RUN_LOG" || true)
  [[ $health_status == healthy || $health_status == running ]] || return 1
  "$CURL_BIN" -fsS --max-time 10 "$LOCAL_HEALTH_URL" >>"$RUN_LOG" 2>&1 || return 1
  "$CURL_BIN" -fsS --max-time 10 "$PUBLIC_HEALTH_URL" >>"$RUN_LOG" 2>&1 || return 1
}

wait_for_health() {
  local attempt
  for ((attempt = 1; attempt <= HEALTH_ATTEMPTS; attempt++)); do
    if container_health; then
      return 0
    fi
    if ((attempt < HEALTH_ATTEMPTS)); then
      "$SLEEP_BIN" "$HEALTH_SLEEP_SECONDS"
    fi
  done
  return 1
}

capture_previous_release() {
  local checkout_commit image_repo metadata_commit
  checkout_commit=$("$GIT_BIN" -C "$STACK_DIR" rev-parse HEAD)
  metadata_commit=$(read_key "$COMPOSE_ENV_FILE" GITEA_INTERNAL_COMMIT || true)
  PREVIOUS_VERSION=$(read_key "$COMPOSE_ENV_FILE" GITEA_VERSION || true)
  PREVIOUS_BUILD_DATE=$(read_key "$COMPOSE_ENV_FILE" GITEA_BUILD_DATE || true)
  image_repo=$(read_key "$COMPOSE_ENV_FILE" GITEA_IMAGE || true)
  if [[ ! $metadata_commit =~ ^[0-9a-f]{40}$ || -z $PREVIOUS_VERSION || -z $PREVIOUS_BUILD_DATE || -z $image_repo ]]; then
    log_error 'Current release metadata is incomplete'
    return 1
  fi
  if [[ $checkout_commit != "$metadata_commit" ]]; then
    log_error 'The production checkout does not match GITEA_INTERNAL_COMMIT'
    return 1
  fi
  PREVIOUS_COMMIT=$metadata_commit
  PREVIOUS_IMAGE="${image_repo}:${PREVIOUS_VERSION}"
}

fetch_target() {
  if ! run_logged "$GIT_BIN" -C "$STACK_DIR" fetch --prune "$REMOTE_NAME" "$BRANCH_NAME"; then
    log_error "Unable to fetch ${REMOTE_NAME}/${BRANCH_NAME}"
    return 1
  fi
  TARGET_COMMIT=$("$GIT_BIN" -C "$STACK_DIR" rev-parse "${REMOTE_NAME}/${BRANCH_NAME}^{commit}" 2>>"$RUN_LOG" || true)
  if [[ ! $TARGET_COMMIT =~ ^[0-9a-f]{40}$ ]]; then
    log_error "Unable to resolve ${REMOTE_NAME}/${BRANCH_NAME}"
    return 1
  fi
  TARGET_VERSION="${GITEA_BASE_VERSION}-internal.${TARGET_COMMIT:0:12}"
  image_repo=$(read_key "$COMPOSE_ENV_FILE" GITEA_IMAGE || true)
  TARGET_IMAGE="${image_repo}:${TARGET_VERSION}"
}

record_backup_metadata() {
  local metadata_file=$1
  local status=$2
  local result=$3
  write_metadata "$metadata_file" \
    CREATED_AT "$RUN_TIMESTAMP" \
    BACKUP_STATUS "$status" \
    PREVIOUS_COMMIT "$PREVIOUS_COMMIT" \
    PREVIOUS_IMAGE "$PREVIOUS_IMAGE" \
    PREVIOUS_VERSION "$PREVIOUS_VERSION" \
    PREVIOUS_BUILD_DATE "$PREVIOUS_BUILD_DATE" \
    TARGET_COMMIT "$TARGET_COMMIT" \
    DEPLOY_RESULT "$result" \
    ROLLBACK_AVAILABLE true \
    ROLLBACK_CONSUMED false
}

create_release_backup() {
  local backup_name archive checksum_file metadata_file data_parent data_name
  backup_name="${RUN_TIMESTAMP}-${PREVIOUS_COMMIT:0:12}"
  CURRENT_BACKUP_DIR="$RELEASES_DIR/$backup_name"
  archive="$CURRENT_BACKUP_DIR/gitea-data.tar.gz"
  checksum_file="$CURRENT_BACKUP_DIR/gitea-data.tar.gz.sha256"
  metadata_file="$CURRENT_BACKUP_DIR/release.env"
  data_parent=$(dirname "$GITEA_DATA_DIR")
  data_name=$(basename "$GITEA_DATA_DIR")

  if [[ -e $CURRENT_BACKUP_DIR ]]; then
    log_error "Release backup already exists: $CURRENT_BACKUP_DIR"
    return 1
  fi
  mkdir -p "$CURRENT_BACKUP_DIR"
  chmod 0700 "$CURRENT_BACKUP_DIR"
  cp "$COMPOSE_ENV_FILE" "$CURRENT_BACKUP_DIR/compose.env"
  cp "$COMPOSE_FILE" "$CURRENT_BACKUP_DIR/compose.yaml"
  chmod 0600 "$CURRENT_BACKUP_DIR/compose.env" "$CURRENT_BACKUP_DIR/compose.yaml"
  record_backup_metadata "$metadata_file" creating pending

  if ! run_logged compose stop --timeout 60 gitea; then
    log_error 'Unable to stop Gitea for the release backup'
    return 1
  fi

  if ! run_logged "$TAR_BIN" -czf "$archive" -C "$data_parent" "$data_name"; then
    run_logged compose start gitea || true
    log_error 'Unable to create the release backup; the previous container was restarted'
    return 1
  fi

  if ! run_logged compose start gitea; then
    log_error 'Release backup completed, but the previous container could not restart'
    return 1
  fi

  if ! run_logged "$TAR_BIN" -tzf "$archive"; then
    log_error 'Release backup archive verification failed'
    return 1
  fi
  "$SHA256_BIN" "$archive" >"$checksum_file"
  if ! run_logged "$SHA256_BIN" -c "$checksum_file"; then
    log_error 'Release backup checksum verification failed'
    return 1
  fi

  set_key "$metadata_file" BACKUP_STATUS complete
  cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
  chmod 0600 "$CURRENT_BACKUP_DIR/deploy.log" "$checksum_file"

  if ! wait_for_health; then
    log_error 'Previous release did not become healthy after backup'
    return 1
  fi
}

restore_source_and_environment() {
  local backup_dir=$1
  local previous_commit
  previous_commit=$(read_key "$backup_dir/release.env" PREVIOUS_COMMIT || true)
  [[ -n $previous_commit ]] || return 1
  run_logged "$GIT_BIN" -C "$STACK_DIR" checkout --detach "$previous_commit" || return 1
  cp "$backup_dir/compose.env" "$COMPOSE_ENV_FILE"
  chmod 0600 "$COMPOSE_ENV_FILE"
}

prepare_target() {
  local build_date
  if ! run_logged "$GIT_BIN" -C "$STACK_DIR" checkout --detach "$TARGET_COMMIT"; then
    log_error "Unable to check out target commit: $TARGET_COMMIT"
    return 1
  fi

  build_date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
  set_key "$COMPOSE_ENV_FILE" GITEA_VERSION "$TARGET_VERSION"
  set_key "$COMPOSE_ENV_FILE" GITEA_INTERNAL_COMMIT "$TARGET_COMMIT"
  set_key "$COMPOSE_ENV_FILE" GITEA_BUILD_DATE "$build_date"

  if [[ $(read_key "$COMPOSE_ENV_FILE" GITEA_INTERNAL_COMMIT || true) != $("$GIT_BIN" -C "$STACK_DIR" rev-parse HEAD) ]]; then
    log_error 'Release metadata does not match the checked-out commit'
    return 1
  fi

  if ! run_logged compose config; then
    log_error 'Rendered Compose configuration is invalid'
    return 1
  fi
}

build_target() {
  local image_revision
  if ! run_logged compose build --pull gitea; then
    log_error 'Build failed; previous release remains active'
    return 1
  fi
  image_revision=$("$DOCKER_BIN" image inspect "$TARGET_IMAGE" \
    --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' \
    2>>"$RUN_LOG" || true)
  if [[ $image_revision != "$TARGET_COMMIT" ]]; then
    log_error 'Built image revision label does not match the target commit'
    return 1
  fi
}

write_current_release() {
  local commit=$1
  local image=$2
  local version=$3
  local result=$4
  local temp_file="${CURRENT_RELEASE_FILE}.tmp.$$"
  write_metadata "$temp_file" \
    DEPLOYED_COMMIT "$commit" \
    GITEA_IMAGE "$image" \
    GITEA_VERSION "$version" \
    DEPLOYED_AT "$RUN_TIMESTAMP" \
    RESULT "$result"
  mv "$temp_file" "$CURRENT_RELEASE_FILE"
  chmod 0600 "$CURRENT_RELEASE_FILE"
}

safe_remove_release_dir() {
  local candidate=$1
  local canonical_root canonical_candidate
  [[ -d $candidate && ! -L $candidate ]] || return 1
  canonical_root=$(cd "$RELEASES_DIR" && pwd -P)
  canonical_candidate=$(cd "$candidate" && pwd -P)
  case "$canonical_candidate" in
    "$canonical_root"/*) rm -rf -- "$canonical_candidate" ;;
    *) return 1 ;;
  esac
}

prune_release_backups() {
  local index
  local -a backups=()
  while IFS= read -r backup; do
    backups+=("$backup")
  done < <(find "$RELEASES_DIR" -mindepth 1 -maxdepth 1 -type d -print | sort -r)
  for ((index = BACKUP_RETENTION; index < ${#backups[@]}; index++)); do
    if ! safe_remove_release_dir "${backups[$index]}"; then
      log_error "Refused to remove unsafe release path: ${backups[$index]}"
      return 1
    fi
  done
}

verify_backup() {
  local backup_dir=$1
  local archive="$backup_dir/gitea-data.tar.gz"
  local checksum_file="$backup_dir/gitea-data.tar.gz.sha256"
  [[ -r $archive && -r $checksum_file && -r $backup_dir/compose.env && -r $backup_dir/release.env ]] || return 1
  run_logged "$TAR_BIN" -tzf "$archive" || return 1
  run_logged "$SHA256_BIN" -c "$checksum_file" || return 1
}

restore_backup() {
  local backup_dir=$1
  local mode=$2
  local archive="$backup_dir/gitea-data.tar.gz"
  local metadata_file="$backup_dir/release.env"
  local failed_data="$backup_dir/failed-data-${RUN_TIMESTAMP}"
  local previous_commit previous_image previous_version data_parent

  previous_commit=$(read_key "$metadata_file" PREVIOUS_COMMIT || true)
  previous_image=$(read_key "$metadata_file" PREVIOUS_IMAGE || true)
  previous_version=$(read_key "$metadata_file" PREVIOUS_VERSION || true)
  if [[ -z $previous_commit || -z $previous_image || -z $previous_version ]]; then
    log_error 'Rollback metadata is incomplete'
    return 1
  fi
  if ! verify_backup "$backup_dir"; then
    log_error 'Rollback backup verification failed'
    return 1
  fi

  run_logged compose stop --timeout 60 gitea || true
  if [[ -e $failed_data ]]; then
    log_error "Rollback recovery path already exists: $failed_data"
    return 1
  fi
  if ! mv "$GITEA_DATA_DIR" "$failed_data"; then
    log_error 'Unable to preserve the failed release data'
    return 1
  fi

  data_parent=$(dirname "$GITEA_DATA_DIR")
  if ! run_logged "$TAR_BIN" -xzf "$archive" -C "$data_parent"; then
    log_error 'Unable to restore the release backup'
    return 1
  fi
  if [[ ${DEPLOY_SKIP_CHOWN:-0} != 1 ]]; then
    if ! run_logged chown -R "${GITEA_UID}:${GITEA_GID}" "$GITEA_DATA_DIR"; then
      log_error 'Unable to restore Gitea data ownership'
      return 1
    fi
  fi
  if ! restore_source_and_environment "$backup_dir"; then
    log_error 'Unable to restore the previous source and environment'
    return 1
  fi
  if ! run_logged compose up -d --no-build gitea; then
    log_error 'Unable to restart the previous image'
    return 1
  fi
  if ! wait_for_health; then
    log_error 'Previous release is unhealthy after rollback'
    return 1
  fi

  if ! safe_remove_release_dir "$failed_data"; then
    log_error "Unable to remove restored failed data: $failed_data"
    return 1
  fi
  set_key "$metadata_file" DEPLOY_RESULT "$mode"
  set_key "$metadata_file" ROLLBACK_CONSUMED true
  write_current_release "$previous_commit" "$previous_image" "$previous_version" "$mode"
  cp "$RUN_LOG" "$backup_dir/rollback-${RUN_TIMESTAMP}.log"
  chmod 0600 "$backup_dir/rollback-${RUN_TIMESTAMP}.log"
  prune_release_backups
}

latest_rollback_backup() {
  local backup metadata available consumed status
  while IFS= read -r backup; do
    metadata="$backup/release.env"
    [[ -r $metadata ]] || continue
    available=$(read_key "$metadata" ROLLBACK_AVAILABLE || true)
    consumed=$(read_key "$metadata" ROLLBACK_CONSUMED || true)
    status=$(read_key "$metadata" BACKUP_STATUS || true)
    if [[ $available == true && $consumed == false && $status == complete ]]; then
      printf '%s\n' "$backup"
      return 0
    fi
  done < <(find "$RELEASES_DIR" -mindepth 1 -maxdepth 1 -type d -print | sort -r)
  return 1
}

run_status() {
  local deployed_commit deployed_image result health
  validate_environment status || return 1
  deployed_commit=$(read_key "$CURRENT_RELEASE_FILE" DEPLOYED_COMMIT || true)
  deployed_image=$(read_key "$CURRENT_RELEASE_FILE" GITEA_IMAGE || true)
  result=$(read_key "$CURRENT_RELEASE_FILE" RESULT || true)
  if container_health; then
    health=healthy
  else
    health=unhealthy
  fi
  printf 'Commit: %s\nImage: %s\nLast result: %s\nHealth: %s\n' \
    "${deployed_commit:-unknown}" "${deployed_image:-unknown}" "${result:-unknown}" "$health"
  [[ $health == healthy ]]
}

run_manual_rollback() {
  local backup_dir
  log '[1/3] Checking rollback state'
  validate_environment || return 1
  backup_dir=$(latest_rollback_backup || true)
  if [[ -z $backup_dir ]]; then
    log_error 'No valid rollback backup is available'
    return 1
  fi
  log "[2/3] Restoring $(basename "$backup_dir")"
  if ! restore_backup "$backup_dir" manual-rollback; then
    log_error 'Rollback failed; recovery artifacts were preserved'
    return 3
  fi
  log '[3/3] Verifying previous release'
  log "Rollback succeeded: $(read_key "$CURRENT_RELEASE_FILE" DEPLOYED_COMMIT)"
  append_summary manual-rollback "$(read_key "$CURRENT_RELEASE_FILE" DEPLOYED_COMMIT)" success
}

run_deploy() {
  local deployed_commit metadata_file
  log '[1/6] Checking production state'
  validate_environment || return 1
  capture_previous_release || return 1
  fetch_target || return 1

  deployed_commit=$(read_key "$CURRENT_RELEASE_FILE" DEPLOYED_COMMIT || true)
  if [[ $deployed_commit == "$TARGET_COMMIT" ]]; then
    if wait_for_health; then
      log "Commit ${TARGET_COMMIT:0:12} is already deployed and healthy"
      return 0
    fi
    log_error 'The recorded release is current but unhealthy'
    return 1
  fi

  log '[2/6] Backing up current release'
  if ! create_release_backup; then
    append_summary failed "$TARGET_COMMIT" backup
    return 1
  fi
  metadata_file="$CURRENT_BACKUP_DIR/release.env"

  log "[3/6] Preparing ${REMOTE_NAME}/${BRANCH_NAME}"
  if ! prepare_target; then
    restore_source_and_environment "$CURRENT_BACKUP_DIR" || true
    set_key "$metadata_file" DEPLOY_RESULT failed-before-switch
    set_key "$metadata_file" ROLLBACK_AVAILABLE false
    cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
    append_summary failed "$TARGET_COMMIT" preparation
    return 1
  fi

  log "[4/6] Building $TARGET_VERSION"
  if ! build_target; then
    restore_source_and_environment "$CURRENT_BACKUP_DIR" || true
    set_key "$metadata_file" DEPLOY_RESULT failed-before-switch
    set_key "$metadata_file" ROLLBACK_AVAILABLE false
    cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
    append_summary failed "$TARGET_COMMIT" build
    return 1
  fi

  log '[5/6] Switching container'
  if ! run_logged compose up -d gitea; then
    log_error 'New container failed to start; beginning automatic rollback'
  fi

  log '[6/6] Verifying health'
  if ! wait_for_health; then
    log_error 'New release health verification failed; beginning automatic rollback'
    if restore_backup "$CURRENT_BACKUP_DIR" automatic-rollback; then
      set_key "$metadata_file" DEPLOY_RESULT automatic-rollback
      cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
      append_summary rolled-back "$TARGET_COMMIT" health
      log 'Automatic rollback succeeded'
      return 2
    fi
    set_key "$metadata_file" DEPLOY_RESULT rollback-failed
    cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
    append_summary critical "$TARGET_COMMIT" rollback
    log_error 'Automatic rollback failed; operator intervention is required'
    return 3
  fi

  set_key "$metadata_file" DEPLOY_RESULT success
  write_current_release "$TARGET_COMMIT" "$TARGET_IMAGE" "$TARGET_VERSION" success
  cp "$RUN_LOG" "$CURRENT_BACKUP_DIR/deploy.log"
  append_summary success "$TARGET_COMMIT" "$TARGET_IMAGE"
  prune_release_backups
  log "Deployment succeeded: ${TARGET_COMMIT:0:12}"
}

case "$COMMAND" in
  deploy) run_deploy ;;
  status) run_status ;;
  rollback) run_manual_rollback ;;
esac
