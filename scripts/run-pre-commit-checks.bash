#!/usr/bin/env bash

#
# Runs pre-commit hooks that are tagged with "check" and "task".
#

set -euo pipefail

declare -x SKIP

SKIP=$(filter-pre-commit-hooks check task)

pre-commit run --all-files \
  | (grep --invert-match --regexp='Skipped' || true)
