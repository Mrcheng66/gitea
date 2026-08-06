#!/usr/bin/env bash
set -euo pipefail

OPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
NGINX_CONFIG="$OPS_DIR/deploy/nginx/gitea.conf"

[[ -r $NGINX_CONFIG ]] || {
  printf 'Nginx Gitea route file is missing\n' >&2
  exit 1
}

grep -Eq '^location[[:space:]]+/[[:space:]]*\{' "$NGINX_CONFIG" || {
  printf 'Nginx config must define the Gitea location / fallback\n' >&2
  exit 1
}

proxy_count=$(grep -c 'proxy_pass http://127.0.0.1:3000;' "$NGINX_CONFIG")
[[ $proxy_count -eq 1 ]] || {
  printf 'All web routes must use the single Gitea upstream on 127.0.0.1:3000\n' >&2
  exit 1
}

if grep -Eq '8081|_workbench|workbench' "$NGINX_CONFIG"; then
  printf 'Legacy Workbench routing remains in Nginx config\n' >&2
  exit 1
fi

for header in Host X-Real-IP X-Forwarded-For X-Forwarded-Proto; do
  grep -q "proxy_set_header $header" "$NGINX_CONFIG" || {
    printf 'Missing proxy header: %s\n' "$header" >&2
    exit 1
  }
done

printf 'Nginx routing tests passed\n'
