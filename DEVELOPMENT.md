# Development

This document is targeted at project developers. It helps people to make their
first steps. It also serves as a general entry to development documentation like
tooling configuration and usage.

## Requirements

Your environment should fulfill the following basic requirements:

- [Go](https://go.dev). See [`go.mod`](go.mod) for the minimum required version.
- [Pre-commit](https://pre-commit.com). For managing and maintaining pre-commit
  Git hooks. Optional.
- [Task](https://taskfile.dev). Task runner as simple alternative to Make.
  Optional.
- Unix-like. Not required by itself, but assumed as the standard.

In addition to the following sections in this document, note that the
[`devel`](devel) directory contains more documentation including further
information about the tooling listed above.

## Initial Setup

### Pre-commit Hooks

Ensure that [pre-commit](https://pre-commit.com) is installed globally. Setup
the pre-commit hooks:

```sh
pre-commit install --install-hooks
pre-commit install --install-hooks --hook-type commit-msg
```

Run all hooks to make sure things are alright:

```sh
pre-commit run -a
```

Read [`devel/pre-commit.md`](devel/pre-commit.md) for more info.

### Taskfile

Ensure that [Task](https://taskfile.dev) is installed. It is used to wrap common
commands abd Can be compared to Make and its phony targets.

Read [`devel/task.md`](devel/task.md) for more info.

### Running Tests

Run tests to make sure everything is setup correctly:

```sh
task test
```

## Local Config

The local configuration is used for everything that goes beyond plain unit
tests. For example if you want to run the code against a Kubernetes cluster and
an AWS account.

Example configurations are located in the [`assets`](assets) directory.

- [`config-example.yaml`](assets/config-example.yaml)
- [`config-minimal.yaml`](assets/config-minimal.yaml)

Duplicate one of these files and place it as `config.yaml` in the root of this
repository. It is already listed in `.gitignore` and so Git will ignore it.
Consider setting `logging.pretty=true` for human-friendly logs.
