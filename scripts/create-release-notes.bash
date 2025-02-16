#!/usr/bin/env bash

#
# This script generates the release notes for the latest release by extracting
# them from the changelog, formatting them, and writing them to a file in tmp.
#

set -euo pipefail

mkdir -p tmp

release_notes=tmp/release-notes.md

awk '/^## /{count++} count==2{print} count==3{exit}' CHANGELOG.md \
  | tail +2 \
  | awk 'NF {p=1} p' | tac \
  | awk 'NF {p=1} p' | tac \
  | sed 's/^### /## /' \
    > $release_notes

mdformat --wrap=no $release_notes
