#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  printf 'Usage: %s <username> <email>\n' "$0" >&2
  exit 64
fi

STACK_DIR=${STACK_DIR:-/opt/gitea-stack}
DOCKER_BIN=${DOCKER_BIN:-docker}
username=$1
email=$2

"$DOCKER_BIN" compose \
  --project-directory "$STACK_DIR" \
  --env-file "$STACK_DIR/.env" \
  exec -T --user git gitea \
  gitea admin user create \
  --admin \
  --must-change-password \
  --random-password \
  --username "$username" \
  --email "$email"

printf 'Sign in with the generated password, change it, and enable 2FA immediately.\n'
