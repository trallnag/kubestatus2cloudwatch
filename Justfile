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

  # Set up mise environment.
  mise --version
  mise install

  # Check mise tool availability.
  mise exec -- exec-cmds-defer-errors --version
  mise exec -- filter-pre-commit-hooks --version
  mise exec -- gofumpt --version
  mise exec -- golangci-lint --version
  mise exec -- goreleaser --version
  mise exec -- mdformat --version
  mise exec -- pre-commit --version
  mise exec -- shellcheck --version
  mise exec -- shfmt --version
  mise exec -- yamlfmt --version

  # Check tool availability.
  go version
  uv --version

  # Install pre-commit hooks.
  mise exec -- pre-commit install --install-hooks
  mise exec -- pre-commit install --install-hooks --hook-type commit-msg
  mise exec -- pre-commit install --install-hooks --hook-type post-checkout
  mise exec -- pre-commit install --install-hooks --hook-type post-merge

  # Download dependencies for Go project.
  go mod download

# Update dependencies.
update:
  # Update tools managed with Homebrew.
  brew upgrade

  # Update tools managed with mise.
  mise upgrade --bump

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
    "just fix--gofumpt" \
    "just fix--golangci"

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

# Format Go files with formatters in golangci-lint.
fix--golangci:
  golangci-lint fmt

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
  go test --race --covermode=atomic --coverprofile=coverage.out

# Run Go unit tests.
test--short:
  go test --race --covermode=atomic --coverprofile=coverage.out --skip ^TestIntegration.+$

# Run Go integration tests.
test--long:
  go test --race --covermode=atomic --coverprofile=coverage.out --run ^TestIntegration.+$

# Create release notes based on changelog.
[group('misc')]
create-release-notes:
  ./scripts/create-release-notes.bash CHANGELOG.md .tmp/release-notes.md
