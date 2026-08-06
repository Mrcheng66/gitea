#!/usr/bin/env bash
set -uo pipefail

STACK_DIR=${STACK_DIR:-/opt/gitea-stack}
COMPOSE_FILE=${COMPOSE_FILE:-$STACK_DIR/ops/compose/compose.yaml}
COMPOSE_ENV_FILE=${COMPOSE_ENV_FILE:-$STACK_DIR/ops/compose/.env}
GITEA_DATA_DIR=${GITEA_DATA_DIR:-/srv/gitea/data}
BACKUP_STATE_FILE=${BACKUP_STATE_FILE:-/var/lib/gitea-stack/last-successful-backup.json}
BACKUP_LOCK_DIR=${BACKUP_LOCK_DIR:-/run/gitea-stack-backup.lock}
BACKUP_MAX_AGE_SECONDS=${BACKUP_MAX_AGE_SECONDS:-86400}
BACKUP_LOCK_MAX_AGE_SECONDS=${BACKUP_LOCK_MAX_AGE_SECONDS:-3600}
CERT_MIN_VALID_SECONDS=${CERT_MIN_VALID_SECONDS:-604800}
DOCKER_BIN=${DOCKER_BIN:-docker}
CURL_BIN=${CURL_BIN:-curl}
OPENSSL_BIN=${OPENSSL_BIN:-openssl}
JQ_BIN=${JQ_BIN:-jq}
DF_BIN=${DF_BIN:-df}

if [[ -z ${GITEA_DOMAIN:-} ]]; then
  printf 'GITEA_DOMAIN is required\n' >&2
  exit 1
fi

compose() {
  "$DOCKER_BIN" compose \
    --project-directory "$STACK_DIR" \
    --env-file "$COMPOSE_ENV_FILE" \
    -f "$COMPOSE_FILE" \
    "$@"
}

failures=()
now=$(date +%s)

if ! "$CURL_BIN" -fsS --max-time 10 "https://${GITEA_DOMAIN}/" >/dev/null; then
  failures+=("Gitea root health check failed")
fi

if ! "$CURL_BIN" -fsS --max-time 10 "https://${GITEA_DOMAIN}/api/healthz" >/dev/null; then
  failures+=("Gitea API health check failed")
fi

database_type=
if ! database_type=$(compose exec -T gitea sh -c 'printf %s "$GITEA__database__DB_TYPE"' 2>/dev/null); then
  failures+=("Organization project database type could not be read")
elif [[ $database_type != sqlite3 ]]; then
  failures+=("Organization project database type is not sqlite3")
fi

org_project_enabled=
if ! org_project_enabled=$(compose exec -T gitea sh -c 'printf %s "$GITEA__org_project__ENABLED"' 2>/dev/null); then
  failures+=("Organization project enabled state could not be read")
elif [[ $org_project_enabled != true ]]; then
  failures+=("Organization project module is not enabled")
fi

readiness_sql="SELECT json_extract('{\"ready\":1}', '\$.ready');
SELECT COUNT(*)
FROM org_project_config_pointer AS pointer
LEFT JOIN org_project_config_version AS draft
  ON draft.id = pointer.draft_version_id
  AND draft.owner_id = pointer.owner_id
  AND draft.state = 'draft'
LEFT JOIN org_project_config_version AS published
  ON published.id = pointer.published_version_id
  AND published.owner_id = pointer.owner_id
  AND published.state = 'published'
WHERE pointer.draft_version_id = 0
  OR draft.id IS NULL
  OR (pointer.published_version_id <> 0 AND published.id IS NULL);"
readiness_output=
if ! readiness_output=$(compose exec -T gitea sqlite3 -batch -noheader /data/gitea/gitea.db "$readiness_sql" 2>/dev/null); then
  failures+=("Organization project readiness query failed")
else
  json1_result=$(printf '%s\n' "$readiness_output" | sed -n '1p')
  pointer_failures=$(printf '%s\n' "$readiness_output" | sed -n '2p')
  if [[ $json1_result != 1 ]]; then
    failures+=("SQLite JSON1 self-check failed")
  fi
  if [[ $pointer_failures != 0 ]]; then
    failures+=("Organization project configuration pointers are inconsistent")
  fi
fi

if ! "$OPENSSL_BIN" s_client -servername "$GITEA_DOMAIN" -connect "${GITEA_DOMAIN}:443" </dev/null 2>/dev/null \
  | "$OPENSSL_BIN" x509 -checkend "$CERT_MIN_VALID_SECONDS" -noout >/dev/null 2>&1; then
  failures+=("TLS certificate expires within 7 days or cannot be read")
fi

disk_percent=$("$DF_BIN" -P "$GITEA_DATA_DIR" 2>/dev/null | awk 'NR == 2 {gsub(/%/, "", $5); print $5}')
if [[ ! $disk_percent =~ ^[0-9]+$ ]]; then
  failures+=("Disk usage could not be read for ${GITEA_DATA_DIR}")
elif [[ $disk_percent -ge 80 ]]; then
  failures+=("Disk usage for ${GITEA_DATA_DIR} is ${disk_percent}%")
fi

backup_running=0
if [[ -r "$BACKUP_LOCK_DIR/started_at" ]]; then
  lock_started=$(<"$BACKUP_LOCK_DIR/started_at")
  if [[ $lock_started =~ ^[0-9]+$ && $((now - lock_started)) -le $BACKUP_LOCK_MAX_AGE_SECONDS ]]; then
    backup_running=1
  else
    failures+=("Backup lock is stale")
  fi
fi

if [[ $backup_running -eq 0 ]]; then
  if [[ ! -r "$BACKUP_STATE_FILE" ]]; then
    failures+=("No successful backup state exists")
  else
    completed_epoch=$("$JQ_BIN" -r '.completed_at_epoch // empty' "$BACKUP_STATE_FILE" 2>/dev/null)
    if [[ ! $completed_epoch =~ ^[0-9]+$ ]]; then
      failures+=("Backup state is invalid")
    elif [[ $((now - completed_epoch)) -gt $BACKUP_MAX_AGE_SECONDS ]]; then
      failures+=("Last successful backup is older than 24 hours")
    fi
  fi
fi

if [[ ${#failures[@]} -gt 0 ]]; then
  message=$(printf '%s\n' "${failures[@]}")
  printf '[ERROR] Platform health check failed on %s:\n%s\n' "$GITEA_DOMAIN" "$message" >&2
  exit 1
fi

printf 'Platform health checks passed for %s\n' "$GITEA_DOMAIN"
