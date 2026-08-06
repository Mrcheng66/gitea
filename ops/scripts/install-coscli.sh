#!/usr/bin/env bash
set -euo pipefail

COSCLI_VERSION=1.0.8
INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}
CURL_BIN=${CURL_BIN:-curl}

if [[ $(uname -s) != Linux ]]; then
  printf 'COSCLI installer supports Linux deployment hosts only.\n' >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64 | amd64)
    cos_arch=amd64
    expected_sha256=7165f2ae16c5f7ac495864c963ca574a76e04ec72680d7bc8a8eee3234d8cf91
    ;;
  aarch64 | arm64)
    cos_arch=arm64
    expected_sha256=0404b4da5b1d0c230c7d7522cb3bbec2909e314ab998889a0aeb8dc6094a2d21
    ;;
  *)
    printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

work_dir=$(mktemp -d)
trap 'rm -rf -- "$work_dir"' EXIT
binary="$work_dir/coscli"
url="https://github.com/tencentyun/coscli/releases/download/v${COSCLI_VERSION}/coscli-v${COSCLI_VERSION}-linux-${cos_arch}"

"$CURL_BIN" -fL --retry 3 --connect-timeout 10 --output "$binary" "$url"
printf '%s  %s\n' "$expected_sha256" "$binary" | sha256sum --check --status
install -d -m 0755 "$INSTALL_DIR"
install -m 0755 "$binary" "$INSTALL_DIR/coscli"

"$INSTALL_DIR/coscli" --version
