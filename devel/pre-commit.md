# Pre-Commit

Used for maintaining Git hooks. Must be installed globally on the respective
system. As it is written in Python, for example
[`pipx`](https://github.com/pypa/pipx) can be used to install it.

- [pre-commit.com](https://pre-commit.com)
- [github.com/pre-commit/pre-commit](https://github.com/pre-commit/pre-commit)

Whenever this repository is initially cloned, the following should be executed:

```
pre-commit install --install-hooks
pre-commit install --install-hooks --hook-type commit-msg
```

Pre-commit should now run on every commit.

Pre-commit is configured via
[`.pre-commit-config.yaml`](../.pre-commit-config.yaml).

## GitHub Actions

While pre-commit is used in GitHub Actions, there is no explicit job or workflow
where pre-commit is executed. This happens through https://pre-commit.ci a
GitHub App called [pre-commit ci](https://github.com/marketplace/pre-commit-ci).

Configuration for this is done in the repository owner's settings and the
https://pre-commit.ci web user interface.

## Housekeeping

### Update hooks

```
pre-commit autoupdate
```

## Cheat Sheet

### Run pre-commit against all files

```
pre-commit run -a
```

### Run specific hook against all files

```
pre-commit run -a <hook>
```
