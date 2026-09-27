#!/usr/bin/env bash
#
# Builds the pgqv release archives for Linux, macOS and Windows into dist/.
#
# macOS binaries are built natively: clang targets both arm64 and amd64.
# Linux and Windows binaries need a C toolchain, because pgqv is built with cgo
# and cannot be cross-compiled without one, so they are built inside a golang
# container (docker or podman).
#
# Windows cross-compilation also needs CGO_LDFLAGS=-lssp: Debian's mingw-w64
# enables the stack protector but does not link the support library, which
# otherwise fails with "undefined reference to __stack_chk_fail".
#
# Usage:
#   scripts/build-release.sh
#   VERSION=1.0.0 scripts/build-release.sh
#
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
VERSION="${VERSION#v}"

DIST=dist
IMAGE=golang:1.27-bookworm

find_container() {
  local candidate
  for candidate in docker podman; do
    if command -v "$candidate" >/dev/null 2>&1; then
      echo "$candidate"
      return 0
    fi
  done
  echo ""
}

CONTAINER="$(find_container)"

# native_build <goos> <goarch> <output-name>
native_build() {
  echo "==> $1/$2 (native)"
  CGO_ENABLED=1 GOOS="$1" GOARCH="$2" \
    go build -trimpath -ldflags '-s -w' -o "$DIST/$3" ./cmd/pgqv
}

# container_build <goos> <goarch> <output-name> <platform-arg> <apt-pkgs> <cc> <cgo-ldflags>
container_build() {
  local goos="$1" goarch="$2" out="$3" platform="$4" pkgs="$5" cc="$6" ldflags="$7"
  local inner="go build -trimpath -ldflags '-s -w' -o /out/${out} ./cmd/pgqv"

  if [ -n "$pkgs" ]; then
    inner="apt-get update -qq >/dev/null && apt-get install -y -qq ${pkgs} >/dev/null && ${inner}"
  fi

  echo "==> ${goos}/${goarch} (container)"
  # ${platform} is intentionally unquoted so that an empty value disappears.
  # shellcheck disable=SC2086
  "$CONTAINER" run --rm ${platform} \
    -v "$PWD":/src \
    -v "$PWD/$DIST":/out \
    -w /src \
    -e CGO_ENABLED=1 \
    -e GOOS="$goos" \
    -e GOARCH="$goarch" \
    -e CC="$cc" \
    -e CGO_LDFLAGS="$ldflags" \
    "$IMAGE" sh -c "set -e; ${inner}"
}

# package_unix <goos> <goarch>
package_unix() {
  ( cd "$DIST" && tar -czf "pgqv_${VERSION}_$1_$2.tar.gz" pgqv && rm -f pgqv )
}

mkdir -p "$DIST"

if [ "$(uname -s)" = "Darwin" ]; then
  native_build darwin arm64 pgqv && package_unix darwin arm64
  native_build darwin amd64 pgqv && package_unix darwin amd64
else
  echo "skipping darwin builds: not running on macOS" >&2
fi

if [ -z "$CONTAINER" ]; then
  echo "error: docker or podman is required for the linux and windows builds" >&2
  exit 1
fi

container_build linux arm64 pgqv "--platform=linux/arm64" "" gcc "" && package_unix linux arm64
container_build linux amd64 pgqv "--platform=linux/amd64" "" gcc "" && package_unix linux amd64

container_build windows amd64 pgqv.exe "" gcc-mingw-w64-x86-64 x86_64-w64-mingw32-gcc "-lssp" &&
  ( cd "$DIST" && zip -q "pgqv_${VERSION}_windows_amd64.zip" pgqv.exe && rm -f pgqv.exe )

if command -v sha256sum >/dev/null 2>&1; then
  SUM="sha256sum"
else
  SUM="shasum -a 256"
fi
# shellcheck disable=SC2086
( cd "$DIST" && $SUM pgqv_* >SHA256SUMS )

echo
echo "Artifacts for v${VERSION}:"
ls -lh "$DIST" | tail -n +2
