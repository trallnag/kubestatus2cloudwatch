#!/usr/bin/env bash

#
# This script runs pre-commit hooks that are tagged with "fix" and "task".
#
# Pre-commit is executed twice so that the script only fails if there is
# something really wrong and not just a successful fix.
#

set -euo pipefail

declare -x SKIP

SKIP=$(./scripts/filter_pre_commit_hooks.py fix task)

(pre-commit run --all-files || pre-commit run --all-files) \
  | (grep --invert-match --regexp='Skipped' || true)
