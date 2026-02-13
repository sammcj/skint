#!/usr/bin/env bash
set -euo pipefail

# Usage: get-version.sh <github_ref> [bump_version_tag]
# Extracts a clean version string for use in build ldflags and release names.

GITHUB_REF="${1:-}"
BUMP_TAG="${2:-}"

if [[ "$GITHUB_REF" == refs/tags/v* ]]; then
  echo "${GITHUB_REF#refs/tags/v}"
elif [[ "$GITHUB_REF" == "refs/heads/main" && -n "$BUMP_TAG" ]]; then
  echo "${BUMP_TAG#v}"
else
  echo "sha-$(git rev-parse --short HEAD)"
fi
