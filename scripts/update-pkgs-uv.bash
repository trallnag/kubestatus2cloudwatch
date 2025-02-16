#!/usr/bin/env bash

#
# This script updates the given tools managed with uv.
#

set -euo pipefail

if command -v uv &> /dev/null; then
  installed=$(uv tool list)

  for pkg in "$@"; do
    if echo "$installed" | grep -q "^$pkg .*$"; then
      uv tool upgrade "$pkg"
    fi
  done
fi
