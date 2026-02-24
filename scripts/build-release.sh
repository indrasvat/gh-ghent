#!/bin/bash
# build-release.sh — Cross-compile gh-ghent with version ldflags.
# Used by cli/gh-extension-precompile@v2 via build_script_override.
# Receives the release tag (e.g., v0.1.0) as $1.
set -eo pipefail

TAG="${1:-dev}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

MODULE="github.com/indrasvat/gh-ghent"
LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X ${MODULE}/internal/version.Version=${TAG}"
LDFLAGS="${LDFLAGS} -X ${MODULE}/internal/version.Commit=${COMMIT}"
LDFLAGS="${LDFLAGS} -X ${MODULE}/internal/version.BuildDate=${BUILD_DATE}"

platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
)

mkdir -p dist

for platform in "${platforms[@]}"; do
    goos="${platform%/*}"
    goarch="${platform#*/}"
    ext=""
    [ "$goos" = "windows" ] && ext=".exe"

    echo "Building ${goos}/${goarch}..."
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -ldflags="${LDFLAGS}" \
        -o "dist/${goos}-${goarch}${ext}" ./cmd/ghent
done

echo "Build complete."
ls -lh dist/
