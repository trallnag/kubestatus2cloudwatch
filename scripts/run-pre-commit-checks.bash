#!/usr/bin/env bash

#
# Runs pre-commit hooks that are tagged with "check" and "task".
#

set -euo pipefail

mx() {
  mise exec -- "$@"
}

declare -x SKIP

SKIP=$(mx filter-pre-commit-hooks check task)

mx pre-commit run --all-files \
  | (grep --invert-match --regexp='Skipped' || true)
