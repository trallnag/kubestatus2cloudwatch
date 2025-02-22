set dotenv-load

set shell := [
  "bash",
  "-o", "errexit", "-o", "nounset", "-o", "pipefail",
  "-O", "extglob", "-O", "globstar", "-O", "nullglob",
  "-c"
]

# Init, fix, check, and test.
default: init fix check test

# Initialize environment.
init:
  # Create local-only directories.
  mkdir -p \
    .cache \
    .local \
    .tmp

  # Check tool availability.
  exec-cmds-defer-errors --version
  filter-pre-commit-hooks --version
  go version
  gofumpt --version
  golangci-lint --version
  goreleaser --version
  mdformat --version
  pre-commit --version
  shellcheck --version
  shfmt --version
  uv --version
  yamlfmt --version

  # Install pre-commit hooks.
  pre-commit install --install-hooks
  pre-commit install --install-hooks --hook-type commit-msg
  pre-commit install --install-hooks --hook-type post-checkout
  pre-commit install --install-hooks --hook-type post-merge

  # Download dependencies for Go project.
  go mod download

# Update dependencies.
update:
  # Try to update tools managed with Homebrew.
  ./scripts/update-pkgs-brew.bash \
    go \
    gofumpt \
    golangci-lint \
    goreleaser \
    just \
    shellcheck \
    shfmt \
    uv \
    yamlfmt

  # Try to update tools managed with uv.
  ./scripts/update-pkgs-uv.bash \
    copier \
    exec-cmds-defer-errors \
    filter-pre-commit-hooks \
    mdformat \
    pre-commit

  # Update pre-commit repositories and hooks.
  pre-commit autoupdate

  # Update dependencies for Go project.
  go get -u
  go mod tidy

# Run recipes that fix stuff.
fix:
  exec-cmds-defer-errors \
    "just fix--pre-commit" \
    "just fix--mdformat" \
    "just fix--shfmt" \
    "just fix--gofumpt"

# Run pre-commit hooks that fix stuff.
fix--pre-commit:
  ./scripts/run-pre-commit-fixes.bash

# Format Markdown files with mdformat.
fix--mdformat:
  mdformat **/*.md

# Format shell scripts with shfmt.
fix--shfmt:
  shfmt --list --write **/*.bash **/*.sh

# Format Go files with gofumpt.
fix--gofumpt:
  gofumpt -w .

# Run recipes that check stuff.
check:
  exec-cmds-defer-errors \
    "just check--pre-commit" \
    "just check--shellcheck" \
    "just check--golangci"

# Run pre-commit hooks that check stuff.
check--pre-commit:
  ./scripts/run-pre-commit-checks.bash

# Lint shell scripts with ShellCheck.
check--shellcheck:
  shellcheck **/*.bash **/*.sh

# Lint Go files with golangci-lint.
check--golangci:
  golangci-lint run

# Run Go tests.
test:
  go test -v -race -covermode=atomic -coverprofile=coverage.out

# Create release notes based on changelog.
[group('misc')]
create-release-notes:
  ./scripts/create-release-notes.bash CHANGELOG.md .tmp/release-notes.md
