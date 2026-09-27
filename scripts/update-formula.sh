#!/usr/bin/env bash
#
# Refreshes the version and sha256 values in Formula/pgqv.rb from the release
# checksums in dist/SHA256SUMS (produced by scripts/build-release.sh).
#
# The Homebrew formula installs prebuilt release archives, so its checksums must
# match the archives that were actually attached to the release.
#
# Usage:
#   scripts/update-formula.sh 1.0.0
#
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "usage: $(basename "$0") <version>    e.g. $(basename "$0") 1.0.0" >&2
  exit 1
fi

FORMULA=Formula/pgqv.rb
SUMS=dist/SHA256SUMS

if [ ! -f "$SUMS" ]; then
  echo "error: $SUMS not found; run scripts/build-release.sh first" >&2
  exit 1
fi

sum_for() {
  awk -v asset="$1" '$2 == asset { print $1; exit }' "$SUMS"
}

# set_platform <platform> rewrites the url and sha256 lines for that platform.
set_platform() {
  local platform="$1" asset sha url
  asset="pgqv_${VERSION}_${platform}.tar.gz"
  sha="$(sum_for "$asset")"
  if [ -z "$sha" ]; then
    echo "error: no checksum for $asset in $SUMS" >&2
    exit 1
  fi
  url="https://github.com/serhii-chechun/pg-query-validate/releases/download/v${VERSION}/${asset}"

  awk -v platform="$platform" -v url="$url" -v sha="$sha" '
    /url / && index($0, platform) {
      print "      url \"" url "\""
      getline
      print "      sha256 \"" sha "\""
      next
    }
    { print }
  ' "$FORMULA" >"$FORMULA.tmp"
  mv "$FORMULA.tmp" "$FORMULA"
}

set_platform darwin_arm64
set_platform darwin_amd64
set_platform linux_arm64
set_platform linux_amd64

echo "Updated $FORMULA to v$VERSION"
