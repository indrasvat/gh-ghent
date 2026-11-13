#!/usr/bin/env bash
# verify-release-checksums.sh — fail unless a published release carries a
# checksums.txt that lists every binary asset with a matching SHA-256.
# install.sh refuses to install without it, so a release missing it is broken.
# Usage: scripts/verify-release-checksums.sh vX.Y.Z [owner/repo]
set -euo pipefail

tag="${1:?usage: $0 vX.Y.Z [owner/repo]}"
repo="${2:-${GITHUB_REPOSITORY:-indrasvat/gh-ghent}}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# Authenticated downloads see drafts; install.sh (anonymous) does not.
if [ "$(gh release view "$tag" -R "$repo" --json isDraft -q .isDraft)" != "false" ]; then
  echo "FAIL: $tag is a draft (or missing), so install.sh cannot download it" >&2
  exit 1
fi

gh release download "$tag" -R "$repo" -D "$tmp"
cd "$tmp"

if [ ! -f checksums.txt ]; then
  echo "FAIL: $tag has no checksums.txt" >&2
  exit 1
fi

missing=0
for f in gh-ghent_*; do
  if ! grep -Eq "^[0-9a-f]{64}  ${f}\$" checksums.txt; then
    echo "FAIL: $tag checksums.txt has no valid SHA-256 entry for $f" >&2
    missing=1
  fi
done
[ "$missing" -eq 0 ] || exit 1

shasum -a 256 --strict -c checksums.txt
echo "OK: $tag checksums.txt covers $(find . -name 'gh-ghent_*' | wc -l | tr -d ' ') binaries"
