#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf -- "$test_dir"' EXIT

env_file="$test_dir/cos.env"
config_file="$test_dir/cos.yaml"
coscli_log="$test_dir/coscli.log"

printf '%s\n' \
  'COS_BUCKET=test-bucket-1250000000' \
  'COS_REGION=ap-guangzhou' \
  'COS_ENDPOINT=cos.ap-guangzhou.myqcloud.com' \
  'COS_SECRET_ID=test-secret-id' \
  'COS_SECRET_KEY=test-secret-key' \
  "COS_CONFIG_PATH=$config_file" >"$env_file"

export COSCLI_BIN="$PROJECT_DIR/tests/stubs/coscli"
export COS_REMOTE_ROOT="$test_dir/remote"
export COSCLI_LOG="$coscli_log"

"$PROJECT_DIR/scripts/configure-cos.sh" "$env_file"

[[ $(stat -f '%Lp' "$config_file" 2>/dev/null || stat -c '%a' "$config_file") == 600 ]]
[[ $(jq -r '.cos.base.secretid' "$config_file") == test-secret-id ]]
[[ $(jq -r '.cos.buckets[0].name' "$config_file") == test-bucket-1250000000 ]]
grep -q 'bucket-versioning --method put cos://test-bucket-1250000000 Enabled' "$coscli_log"
grep -q 'bucket-encryption --method put cos://test-bucket-1250000000 --sse-algorithm AES256' "$coscli_log"

printf 'COS configuration tests passed\n'
