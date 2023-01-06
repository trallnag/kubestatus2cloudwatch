# Contributing <!-- omit in toc -->

Thank you for your interest in improving this project. Your contributions are
appreciated.

In the following you can find a collection of frequently asked questions and
hopefully good answers.

- [How to setup local dev environment?](#how-to-setup-local-dev-environment)

Also consider taking a look at the development documentation at
[`docs/devel`](docs/devel).

## How to setup local dev environment?

### Pre-commit <!-- omit in toc -->

Tool written in Python used for maintaining Git hooks.

- <https://pre-commit.com/>

Whenever this repository is initially cloned, execute:

```
pre-commit install --install-hooks
pre-commit install --install-hooks --hook-type commit-msg
```

Pre-commit should now run on every commit.

Note that Go must be installed for some hooks to work properly.

Read [`docs/devel/pre-commit.md`](docs/devel/pre-commit.md) for more.

### Taskfile <!-- omit in toc -->

Like Make but for tasks instead of builds. Written in Go.

- <https://taskfile.dev/>

It can be installed with Homebrew (other options are available as well).

```
brew install go-task/tap/go-task
```

Read [`docs/devel/taskfile.md`](docs/devel/taskfile.md) for more.

### Local Config <!-- omit in toc -->

Place `config.yaml` in the root of this repository. It is already listed in
`.gitignore` and so Git will ignore it. Consider setting `logging.pretty=true`
for human-friendly logs.
