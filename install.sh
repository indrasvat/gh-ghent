#!/usr/bin/env bash
set -euo pipefail

repo="${GH_GHENT_REPO:-indrasvat/gh-ghent}"
version="latest"
install_dir="${GH_GHENT_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
  cat <<USAGE
install gh-ghent

Usage:
  install.sh [--version vX.Y.Z] [--dir DIR] [--repo owner/name]

Environment:
  GH_GHENT_REPO          override release repo (default: indrasvat/gh-ghent)
  GH_GHENT_INSTALL_DIR   override install dir (default: ~/.local/bin)
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      version="${2:?missing version}"
      shift 2
      ;;
    --dir)
      install_dir="${2:?missing dir}"
      shift 2
      ;;
    --repo)
      repo="${2:?missing repo}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need curl
need shasum

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

if [ "$version" = "latest" ]; then
  latest_json="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest")"
  version_label="$(printf '%s\n' "$latest_json" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$version_label" ]; then
    echo "could not resolve latest release tag for ${repo}" >&2
    exit 1
  fi
  release_url="https://github.com/${repo}/releases/download/${version_label}"
else
  release_url="https://github.com/${repo}/releases/download/${version}"
  version_label="$version"
fi

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

asset="gh-ghent_${version_label}_${os}-${arch}"
binary_url="${release_url}/${asset}"
checksums_url="${release_url}/checksums.txt"

echo "downloading ${binary_url}"
curl -fsSL "$binary_url" -o "$tmp/$asset"
curl -fsSL "$checksums_url" -o "$tmp/checksums.txt"

(
  cd "$tmp"
  grep "  ${asset}$" checksums.txt >checksums.selected
  shasum -a 256 -c checksums.selected
)

install -d "$install_dir"
install -m 755 "$tmp/$asset" "$install_dir/gh-ghent"

if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$install_dir/gh-ghent" >/dev/null 2>&1 || true
fi

"$install_dir/gh-ghent" version >/dev/null

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "warning: $install_dir is not on PATH" >&2 ;;
esac

echo "installed gh-ghent ${version_label} to $install_dir/gh-ghent"
