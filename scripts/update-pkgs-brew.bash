#!/usr/bin/env bash

#
# This script updates the given Homebrew formulae.
#

set -euo pipefail

if command -v brew &> /dev/null; then
  brew update

  installed=$(brew list -q --formula -1)

  for formula in "$@"; do
    if echo "$installed" | grep -q "^$formula$"; then
      brew upgrade -q --formula "$formula"
    fi
  done
fi
