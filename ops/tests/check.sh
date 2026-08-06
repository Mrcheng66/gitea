#!/usr/bin/env bash
set -euo pipefail

OPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
REPO_DIR=$(cd "$OPS_DIR/.." && pwd)
COMPOSE_DIR="$OPS_DIR/compose"
cd "$OPS_DIR"

bash -n scripts/*.sh tests/*.sh tests/stubs/*

for executable in scripts/*.sh tests/*.sh tests/stubs/*; do
  [[ -x $executable ]] || {
    printf 'File is not executable: %s\n' "$executable" >&2
    exit 1
  }
done

compose_config=$(docker compose \
  --project-directory "$REPO_DIR" \
  --env-file "$COMPOSE_DIR/.env.example" \
  -f "$COMPOSE_DIR/compose.yaml" \
  config --format json)
printf '%s' "$compose_config" | jq -e '
  (.services | keys) == ["gitea"]
' >/dev/null

printf '%s' "$compose_config" | jq -e --arg repo_dir "$REPO_DIR" '
  .services.gitea.image == "code-lab/gitea:1.27.1-internal.1" and
  .services.gitea.build.context == $repo_dir and
  .services.gitea.environment.GITEA__org_project__ENABLED == "true" and
  .services.gitea.environment.GITEA__database__DB_TYPE == "sqlite3"
' >/dev/null

if grep -Eq 'image:.*latest|_IMAGE=.*latest' "$COMPOSE_DIR/compose.yaml" "$COMPOSE_DIR/.env.example"; then
  printf 'Floating latest image tag found\n' >&2
  exit 1
fi

if ! printf '%s' "$compose_config" | jq -e '
  any(.services.gitea.ports[];
    .target == 3000 and .published == "3000" and .host_ip == "127.0.0.1")
' >/dev/null; then
  printf 'Gitea web port 3000 must bind to 127.0.0.1\n' >&2
  exit 1
fi

if printf '%s' "$compose_config" | grep -q '8081'; then
  printf 'Legacy Workbench port 8081 remains in Compose\n' >&2
  exit 1
fi

for metadata_arg in GITEA_VERSION GITEA_UPSTREAM_COMMIT GITEA_INTERNAL_COMMIT GITEA_BUILD_DATE; do
  grep -q "$metadata_arg" "$REPO_DIR/Dockerfile" || {
    printf 'Dockerfile is missing build metadata argument: %s\n' "$metadata_arg" >&2
    exit 1
  }
done

grep -q '^OnCalendar=\*-\*-\* 03:00:00$' deploy/systemd/gitea-backup.timer
grep -q '^OnUnitActiveSec=5min$' deploy/systemd/gitea-health.timer
grep -q '^EnvironmentFile=/etc/gitea-stack/backup.env$' deploy/systemd/gitea-backup.service
if grep -Rqi 'Workbench' compose scripts deploy/systemd deploy/nginx; then
  printf 'Legacy Workbench deployment reference remains\n' >&2
  exit 1
fi

tests/test-configure-cos.sh
tests/test-backup.sh
tests/test-health-check.sh
tests/test-nginx-routing.sh

printf 'all checks passed\n'
