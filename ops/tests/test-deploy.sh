#!/usr/bin/env bash
set -euo pipefail

OPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_SCRIPT="$OPS_DIR/scripts/deploy.sh"
test_root=$(mktemp -d)
trap 'status=$?; if [[ ${KEEP_TEST_ROOT:-0} == 1 ]]; then printf "preserved test root: %s\n" "$test_root" >&2; else rm -rf -- "$test_root"; fi; exit $status' EXIT

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local file=$1
  local pattern=$2
  grep -q -- "$pattern" "$file" || fail "Expected $file to contain: $pattern"
}

assert_not_contains() {
  local file=$1
  local pattern=$2
  if grep -q -- "$pattern" "$file"; then
    fail "Expected $file not to contain: $pattern"
  fi
}

bin_dir="$test_root/bin"
mkdir -p "$bin_dir"

docker_stub="$bin_dir/docker"
cat >"$docker_stub" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail

: "${DOCKER_LOG:?DOCKER_LOG is required}"
printf '%s\n' "$*" >>"$DOCKER_LOG"

if [[ -n ${DOCKER_FAIL_MATCH:-} && "$*" == *"$DOCKER_FAIL_MATCH"* ]]; then
  exit 1
fi

if [[ "$*" == *'up -d --no-build gitea'* ]]; then
  if [[ ${DOCKER_FAIL_ROLLBACK:-0} == 1 ]]; then
    exit 1
  fi
  [[ -z ${DOCKER_SWITCH_MARKER:-} ]] || rm -f -- "$DOCKER_SWITCH_MARKER"
elif [[ "$*" == *'start gitea'* ]]; then
  [[ -z ${DOCKER_SWITCH_MARKER:-} ]] || rm -f -- "$DOCKER_SWITCH_MARKER"
elif [[ "$*" == *'up -d gitea'* ]]; then
  [[ -z ${DOCKER_SWITCH_MARKER:-} ]] || : >"$DOCKER_SWITCH_MARKER"
  if [[ -n ${DOCKER_MUTATE_FILE:-} ]]; then
    printf 'new release data\n' >"$DOCKER_MUTATE_FILE"
  fi
elif [[ "$*" == *'compose'* && "$*" == *'ps -q gitea'* ]]; then
  printf 'test-container\n'
elif [[ "$1" == inspect && "$*" == *'.State.Health'* ]]; then
  printf '%s\n' "${DOCKER_HEALTH_STATUS:-healthy}"
elif [[ "$*" == *'image inspect'* && "$*" == *'org.opencontainers.image.revision'* ]]; then
  printf '%s\n' "${DOCKER_IMAGE_REVISION:-}"
fi
STUB
chmod +x "$docker_stub"

curl_stub="$bin_dir/curl"
cat >"$curl_stub" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail

[[ -z ${CURL_LOG:-} ]] || printf '%s\n' "$*" >>"$CURL_LOG"
if [[ ${CURL_FAIL_AFTER_SWITCH:-0} == 1 && -n ${DOCKER_SWITCH_MARKER:-} && -e $DOCKER_SWITCH_MARKER ]]; then
  exit 1
fi
exit "${CURL_STATUS:-0}"
STUB
chmod +x "$curl_stub"

flock_stub="$bin_dir/flock"
cat >"$flock_stub" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
exit "${FLOCK_STATUS:-0}"
STUB
chmod +x "$flock_stub"

tar_stub="$bin_dir/tar"
real_tar=$(command -v tar)
cat >"$tar_stub" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ ${TAR_FAIL_CREATE:-0} == 1 && $1 == -czf ]]; then
  exit 1
fi
exec "${REAL_TAR:?REAL_TAR is required}" "$@"
STUB
chmod +x "$tar_stub"

setup_case() {
  local name=$1
  case_dir="$test_root/$name"
  remote_dir="$case_dir/remote.git"
  seed_dir="$case_dir/seed"
  stack_dir="$case_dir/stack"
  compose_dir="$stack_dir/ops/compose"
  data_dir="$case_dir/storage/data"
  state_root="$case_dir/state"
  releases_dir="$state_root/releases"
  run_dir="$state_root/run"
  lock_file="$case_dir/deploy.lock"
  docker_log="$case_dir/docker.log"
  curl_log="$case_dir/curl.log"
  switch_marker="$case_dir/switched"
  app_data_file="$data_dir/gitea/state.txt"

  mkdir -p "$case_dir" "$data_dir/gitea" "$state_root" "$run_dir"
  : >"$docker_log"
  : >"$curl_log"
  printf 'old release data\n' >"$app_data_file"

  git init --bare --initial-branch=main "$remote_dir" >/dev/null
  git init -b main "$seed_dir" >/dev/null
  git -C "$seed_dir" config user.name 'Deploy Test'
  git -C "$seed_dir" config user.email 'deploy-test@example.com'
  mkdir -p "$seed_dir/ops/compose"
  cp "$OPS_DIR/compose/compose.yaml" "$seed_dir/ops/compose/compose.yaml"
  printf 'initial\n' >"$seed_dir/release.txt"
  git -C "$seed_dir" add .
  git -C "$seed_dir" commit -m 'initial release' >/dev/null
  initial_commit=$(git -C "$seed_dir" rev-parse HEAD)

  git -C "$seed_dir" remote add origin "$remote_dir"
  git -C "$seed_dir" push -u origin main >/dev/null 2>&1
  git clone "$remote_dir" "$stack_dir" >/dev/null 2>&1
  git -C "$stack_dir" checkout main >/dev/null 2>&1

  cat >"$compose_dir/.env" <<ENV
GITEA_DOMAIN=git.test.example
GITEA_IMAGE=code-lab/gitea
GITEA_VERSION=1.27.1-internal.initial
GITEA_UPSTREAM_COMMIT=a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f
GITEA_INTERNAL_COMMIT=$initial_commit
GITEA_BUILD_DATE=2026-08-05T00:00:00Z
GITEA_DATA_DIR=$data_dir
ENV
  mkdir -p "$releases_dir"
  cat >"$state_root/current-release.env" <<STATE
DEPLOYED_COMMIT=$initial_commit
GITEA_IMAGE=code-lab/gitea:1.27.1-internal.initial
GITEA_VERSION=1.27.1-internal.initial
DEPLOYED_AT=2026-08-05T00:00:00Z
RESULT=success
STATE

  export STACK_DIR="$stack_dir"
  export COMPOSE_FILE="$compose_dir/compose.yaml"
  export COMPOSE_ENV_FILE="$compose_dir/.env"
  export GITEA_DATA_DIR="$data_dir"
  export EXPECTED_GITEA_DATA_DIR="$data_dir"
  export STATE_ROOT="$state_root"
  export RELEASES_DIR="$releases_dir"
  export RUN_DIR="$run_dir"
  export LOCK_FILE="$lock_file"
  export DOCKER_BIN="$docker_stub"
  export CURL_BIN="$curl_stub"
  export FLOCK_BIN="$flock_stub"
  export TAR_BIN="$tar_stub"
  export REAL_TAR="$real_tar"
  export DEPLOY_SKIP_ROOT_CHECK=1
  export DEPLOY_SKIP_REEXEC=1
  export DEPLOY_SKIP_CHOWN=1
  export HEALTH_ATTEMPTS=2
  export HEALTH_SLEEP_SECONDS=0
  export PUBLIC_HEALTH_URL=https://git.test.example/api/healthz
  export LOCAL_HEALTH_URL=http://127.0.0.1:3000/api/healthz
  export DOCKER_LOG="$docker_log"
  export CURL_LOG="$curl_log"
  export DOCKER_SWITCH_MARKER="$switch_marker"
  export DOCKER_MUTATE_FILE="$app_data_file"
  export DOCKER_FAIL_MATCH=
  export DOCKER_FAIL_ROLLBACK=0
  export CURL_FAIL_AFTER_SWITCH=0
  export CURL_STATUS=0
  export DOCKER_HEALTH_STATUS=healthy
  export FLOCK_STATUS=0
  export TAR_FAIL_CREATE=0
}

push_release() {
  local content=$1
  printf '%s\n' "$content" >"$seed_dir/release.txt"
  git -C "$seed_dir" add release.txt
  git -C "$seed_dir" commit -m "$content" >/dev/null
  git -C "$seed_dir" push origin main >/dev/null 2>&1
  target_commit=$(git -C "$seed_dir" rev-parse HEAD)
  export DOCKER_IMAGE_REVISION="$target_commit"
}

run_deploy() {
  local output_file=$1
  shift
  DEPLOY_TIMESTAMP=${DEPLOY_TIMESTAMP:-20260806T120000Z} "$DEPLOY_SCRIPT" "$@" >"$output_file" 2>&1
}

[[ -x $DEPLOY_SCRIPT ]] || fail "Deployment script is not executable: $DEPLOY_SCRIPT"

help_output="$test_root/help.log"
"$DEPLOY_SCRIPT" help >"$help_output"
assert_contains "$help_output" 'Usage:'
assert_contains "$help_output" 'rollback'

setup_case workflow
push_release 'frontend release'
first_target=$target_commit
success_output="$case_dir/success.log"
export DEPLOY_TIMESTAMP=20260806T120000Z
run_deploy "$success_output"
assert_contains "$success_output" 'Deployment succeeded'
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$first_target" ]]
assert_contains "$compose_dir/.env" "GITEA_INTERNAL_COMMIT=$first_target"
assert_contains "$compose_dir/.env" "GITEA_VERSION=1.27.1-internal.${first_target:0:12}"
assert_contains "$docker_log" 'stop --timeout 60 gitea'
assert_contains "$docker_log" 'start gitea'
assert_contains "$docker_log" 'build --pull gitea'
assert_contains "$docker_log" 'up -d gitea'
assert_contains "$state_root/current-release.env" "DEPLOYED_COMMIT=$first_target"
[[ $(find "$releases_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ') == 1 ]]

: >"$docker_log"
unchanged_output="$case_dir/unchanged.log"
export DEPLOY_TIMESTAMP=20260806T120100Z
run_deploy "$unchanged_output"
assert_contains "$unchanged_output" 'already deployed and healthy'
assert_not_contains "$docker_log" 'build --pull gitea'
[[ $(find "$releases_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ') == 1 ]]

status_output="$case_dir/status.log"
run_deploy "$status_output" status
assert_contains "$status_output" "$first_target"
assert_contains "$status_output" 'healthy'

export FLOCK_STATUS=1
lock_output="$case_dir/lock.log"
set +e
run_deploy "$lock_output" status
lock_status=$?
set -e
[[ $lock_status == 75 ]] || fail "Expected lock exit 75, got $lock_status"
assert_contains "$lock_output" 'Another deployment is already running'
export FLOCK_STATUS=0

printf 'dirty\n' >>"$stack_dir/release.txt"
dirty_output="$case_dir/dirty.log"
set +e
run_deploy "$dirty_output"
dirty_status=$?
set -e
[[ $dirty_status == 1 ]] || fail "Expected dirty worktree exit 1, got $dirty_status"
assert_contains "$dirty_output" 'contain local changes'
git -C "$stack_dir" checkout -- release.txt

mkdir -p "$case_dir/other-data"
export EXPECTED_GITEA_DATA_DIR="$case_dir/other-data"
unsafe_output="$case_dir/unsafe.log"
set +e
run_deploy "$unsafe_output" status
unsafe_status=$?
set -e
[[ $unsafe_status == 1 ]] || fail "Expected unsafe data path exit 1, got $unsafe_status"
assert_contains "$unsafe_output" 'Unexpected Gitea data directory'
export EXPECTED_GITEA_DATA_DIR="$data_dir"

printf 'stable target one data\n' >"$app_data_file"
push_release 'business logic release'
second_target=$target_commit

export TAR_FAIL_CREATE=1
backup_error="$case_dir/backup-error.log"
export DEPLOY_TIMESTAMP=20260806T121000Z
set +e
run_deploy "$backup_error"
backup_status=$?
set -e
[[ $backup_status == 1 ]] || fail "Expected backup failure exit 1, got $backup_status"
assert_contains "$backup_error" 'Unable to create the release backup'
assert_contains "$docker_log" 'start gitea'
assert_not_contains "$docker_log" "build --pull gitea"
export TAR_FAIL_CREATE=0

export DOCKER_FAIL_MATCH='build --pull gitea'
build_error="$case_dir/build-error.log"
export DEPLOY_TIMESTAMP=20260806T121100Z
set +e
run_deploy "$build_error"
build_status=$?
set -e
[[ $build_status == 1 ]] || fail "Expected image build failure, got $build_status"
assert_contains "$build_error" 'Build failed; previous release remains active'
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$first_target" ]]
assert_contains "$app_data_file" 'stable target one data'
assert_not_contains "$docker_log" 'up -d gitea'
build_backup=$(find "$releases_dir" -mindepth 1 -maxdepth 1 -type d -name '20260806T121100Z-*')
assert_contains "$build_backup/release.env" 'ROLLBACK_AVAILABLE=false'
export DOCKER_FAIL_MATCH=

export CURL_FAIL_AFTER_SWITCH=1
automatic_output="$case_dir/automatic-rollback.log"
export DEPLOY_TIMESTAMP=20260806T121200Z
set +e
run_deploy "$automatic_output"
automatic_status=$?
set -e
[[ $automatic_status == 2 ]] || fail "Expected automatic rollback exit 2, got $automatic_status"
assert_contains "$automatic_output" 'Automatic rollback succeeded'
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$first_target" ]]
assert_contains "$app_data_file" 'stable target one data'
assert_contains "$docker_log" 'up -d --no-build gitea'
export CURL_FAIL_AFTER_SWITCH=0

export DEPLOY_TIMESTAMP=20260806T121300Z
run_deploy "$case_dir/second-success.log"
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$second_target" ]]

printf 'operator changed data\n' >"$app_data_file"
manual_output="$case_dir/manual.log"
export DEPLOY_TIMESTAMP=20260806T121400Z
run_deploy "$manual_output" rollback
assert_contains "$manual_output" 'Rollback succeeded'
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$first_target" ]]
assert_contains "$app_data_file" 'stable target one data'

for index in 1 2 3 4 5 6; do
  mkdir -p "$releases_dir/2026070${index}T000000Z-old${index}"
done
push_release 'retention release'
third_target=$target_commit
export DEPLOY_TIMESTAMP=20260806T124000Z
run_deploy "$case_dir/retention.log"
[[ $(git -C "$stack_dir" rev-parse HEAD) == "$third_target" ]]
backup_count=$(find "$releases_dir" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
[[ $backup_count == 5 ]] || fail "Expected 5 retained backups, got $backup_count"

push_release 'broken rollback release'
export CURL_FAIL_AFTER_SWITCH=1
export DOCKER_FAIL_ROLLBACK=1
export DEPLOY_TIMESTAMP=20260806T125000Z
rollback_failure_output="$case_dir/rollback-failure.log"
set +e
run_deploy "$rollback_failure_output"
rollback_failure_status=$?
set -e
[[ $rollback_failure_status == 3 ]] || fail "Expected rollback failure exit 3, got $rollback_failure_status"
assert_contains "$rollback_failure_output" 'Automatic rollback failed'
find "$releases_dir" -mindepth 2 -maxdepth 2 -type d -name 'failed-data-*' | grep -q .

printf 'deployment tests passed\n'
