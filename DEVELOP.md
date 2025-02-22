# Development

This document is targeted at project developers. It helps people to make their
first steps. It also serves as a general entry to development documentation like
tooling configuration and usage.

The environment is expected to be Unix-like.

For core development activities, [Go](https://go.dev/) is sufficient. Other
stuff just comes on top and is in any case handled by GitHub Actions. For a
complete development environment (Git hooks, Markdown formatting, etc.), the
following tools are required:

- [Copier](https://copier.readthedocs.io/en/stable/) for project templating.
- [Gofumpt](https://github.com/mvdan/gofumpt) for formatting Go code.
- [Golangci-lint](https://golangci-lint.run/) for linting Go code.
- [Just](https://github.com/casey/just) for running self-documenting tasks.
- [Mdformat](https://github.com/hukkin/mdformat) for Markdown formatting.
- [Pre-commit](https://pre-commit.com/) for managing pre-commit hooks.
- [ShellCheck](https://github.com/koalaman/shellcheck) for shell script linting.
- [Shfmt](https://github.com/mvdan/sh) for shell script formatting.

Same goes for the following utilities:

- [Exec-cmds-defer-errors](https://pypi.org/project/exec-cmds-defer-errors/).
- [Filter-pre-commit-hooks](https://pypi.org/project/filter-pre-commit-hooks/).

Common tasks like initialization and runnings tests are covered by and
documented in [`Justfile`](./Justfile). To run a complete suite of tasks, just
invoke `just` without arguments.

This projects supports [Development Containers](https://containers.dev/). Check
out [`.devcontainer/README.md`](./.devcontainer/README.md) for more information.

## Local configuration

The local configuration is used for everything that goes beyond plain unit
tests. For example if you want to run the code against a Kubernetes cluster and
an AWS account.

Example configurations are located in the [`assets`](./assets) directory.

- [`config-example.yaml`](./assets/config-example.yaml)
- [`config-minimal.yaml`](./assets/config-minimal.yaml)

Duplicate one of these files and place it as `config.yaml` in the root of this
repository. It is already listed in `.gitignore` and so Git will ignore it.
Consider setting `logging.pretty=true` for human-friendly logs.
