#!/usr/bin/env bash
set -euo pipefail

COS_ENV_FILE=${1:-/etc/gitea-stack/cos.env}
COSCLI_BIN=${COSCLI_BIN:-coscli}
JQ_BIN=${JQ_BIN:-jq}

if [[ ! -r "$COS_ENV_FILE" ]]; then
  printf 'COS environment file is not readable: %s\n' "$COS_ENV_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$COS_ENV_FILE"

required_vars=(
  COS_BUCKET
  COS_REGION
  COS_ENDPOINT
  COS_SECRET_ID
  COS_SECRET_KEY
  COS_CONFIG_PATH
)
for var_name in "${required_vars[@]}"; do
  if [[ -z ${!var_name:-} ]]; then
    printf 'Missing required variable: %s\n' "$var_name" >&2
    exit 1
  fi
done

install -d -m 0700 "$(dirname "$COS_CONFIG_PATH")"
temp_config=$(mktemp "${COS_CONFIG_PATH}.tmp.XXXXXX")
trap 'rm -f -- "$temp_config"' EXIT
umask 077

"$JQ_BIN" -n \
  --arg secret_id "$COS_SECRET_ID" \
  --arg secret_key "$COS_SECRET_KEY" \
  --arg bucket "$COS_BUCKET" \
  --arg region "$COS_REGION" \
  --arg endpoint "$COS_ENDPOINT" \
  '{
    cos: {
      base: {
        secretid: $secret_id,
        secretkey: $secret_key,
        sessiontoken: "",
        protocol: "https",
        mode: "SecretKey",
        closeautoswitchhost: "false",
        disableencryption: "true",
        disableautofetchbuckettype: "false"
      },
      buckets: [{
        name: $bucket,
        alias: $bucket,
        region: $region,
        endpoint: $endpoint,
        ofs: false
      }]
    }
  }' >"$temp_config"

install -m 0600 "$temp_config" "$COS_CONFIG_PATH"
unset COS_SECRET_ID COS_SECRET_KEY

coscli=("$COSCLI_BIN" --config-path "$COS_CONFIG_PATH" --disable-log)
"${coscli[@]}" bucket-versioning --method put "cos://$COS_BUCKET" Enabled
"${coscli[@]}" bucket-encryption --method put "cos://$COS_BUCKET" --sse-algorithm AES256
"${coscli[@]}" bucket-versioning --method get "cos://$COS_BUCKET"
"${coscli[@]}" bucket-encryption --method get "cos://$COS_BUCKET"

printf 'COSCLI configured at %s. Add the lifecycle rules described in ops/docs/deployment.md.\n' "$COS_CONFIG_PATH"
