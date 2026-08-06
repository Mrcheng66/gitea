#!/usr/bin/env bash
set -euo pipefail

OPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf -- "$test_dir"' EXIT

data_dir="$test_dir/data"
state_file="$test_dir/last-successful-backup.json"
lock_dir="$test_dir/backup.lock"
docker_log="$test_dir/docker.log"
mkdir -p "$data_dir"
: >"$docker_log"

export GITEA_DOMAIN=git.test.example
export GITEA_DATA_DIR="$data_dir"
export BACKUP_STATE_FILE="$state_file"
export BACKUP_LOCK_DIR="$lock_dir"
export CURL_BIN="$OPS_DIR/tests/stubs/curl"
export OPENSSL_BIN="$OPS_DIR/tests/stubs/openssl"
export DF_BIN="$OPS_DIR/tests/stubs/df"
export DOCKER_BIN="$OPS_DIR/tests/stubs/docker"
export DOCKER_LOG="$docker_log"
export CURL_STATUS=0
export OPENSSL_SCLIENT_STATUS=0
export OPENSSL_X509_STATUS=0
export DF_PERCENT=10
export DOCKER_DATABASE_TYPE=sqlite3
export DOCKER_ORG_PROJECT_ENABLED=true
export DOCKER_ORG_PROJECT_READINESS=$'1\n0'

jq -n --argjson completed_at_epoch "$(date +%s)" '{completed_at_epoch:$completed_at_epoch}' >"$state_file"
"$OPS_DIR/scripts/health-check.sh"

grep -q 'sqlite3 -batch -noheader /data/gitea/gitea.db' "$docker_log"

jq -n '{completed_at_epoch:1}' >"$state_file"
health_error="$test_dir/health-error.log"
if "$OPS_DIR/scripts/health-check.sh" 2>"$health_error"; then
  printf 'Expected a stale backup health check failure\n' >&2
  exit 1
fi
grep -q 'Last successful backup is older than 24 hours' "$health_error"

jq -n --argjson completed_at_epoch "$(date +%s)" '{completed_at_epoch:$completed_at_epoch}' >"$state_file"
export DOCKER_DATABASE_TYPE=mysql
sqlite_error="$test_dir/sqlite-error.log"
if "$OPS_DIR/scripts/health-check.sh" 2>"$sqlite_error"; then
  printf 'Expected a non-SQLite database readiness failure\n' >&2
  exit 1
fi
grep -q 'Organization project database type is not sqlite3' "$sqlite_error"

export DOCKER_DATABASE_TYPE=sqlite3
export DOCKER_ORG_PROJECT_READINESS=$'1\n1'
pointer_error="$test_dir/pointer-error.log"
if "$OPS_DIR/scripts/health-check.sh" 2>"$pointer_error"; then
  printf 'Expected a configuration pointer readiness failure\n' >&2
  exit 1
fi
grep -q 'Organization project configuration pointers are inconsistent' "$pointer_error"

export DOCKER_ORG_PROJECT_READINESS=$'0\n0'
json_error="$test_dir/json-error.log"
if "$OPS_DIR/scripts/health-check.sh" 2>"$json_error"; then
  printf 'Expected a SQLite JSON1 readiness failure\n' >&2
  exit 1
fi
grep -q 'SQLite JSON1 self-check failed' "$json_error"

printf 'health check tests passed\n'
