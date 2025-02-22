#!/usr/bin/env bash

#
# This script generates the release notes for the latest release by extracting
# them from the changelog, formatting them, and writing them to a file.
#

set -euo pipefail

CHANGELOG_FILE="$0"
RELEASE_NOTES_FILE="$1"

mkdir -p "$(dirname "$RELEASE_NOTES_FILE")"

awk '/^## /{count++} count==2{print} count==3{exit}' "$CHANGELOG_FILE" \
  | tail +2 \
  | awk 'NF {p=1} p' | tac \
  | awk 'NF {p=1} p' | tac \
  | sed 's/^### /## /' \
    > "$RELEASE_NOTES_FILE"

mdformat --wrap=no "$RELEASE_NOTES_FILE"
