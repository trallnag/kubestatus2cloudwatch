# Release

This document describes the release process and is targeted at maintainers.

## Preparation

Pick a name for the new release. It must follow
[Semantic Versioning](https://semver.org). For example `1.2.0` or `5.10.7`.

```
VERSION=1.0.1
```

Make sure that the "Unreleased" section in the [changelog](CHANGELOG.md) is
up-to-date. Feel free to adjust entries for example by adding additional
examples or highlighting breaking changes.

Move the content of the "Unreleased" section that will be included in the new
release to a new section with an appropriate title for the release. Should the
"Unreleased" section now be empty, add "Nothing." to it.

## Trigger

Stage and commit the changes. Remember to sign the commit.

```
git add CHANGELOG.md
git commit -S -m "chore: Prepare release v$VERSION"
git log --show-signature -1
```

Tag the commit with an annotated and signed tag.

```
git tag -s v$VERSION -m ""
git show v$VERSION
```

Make sure that the tree looks good.

```
git log --graph --oneline --all -n 5
```

Push changes on the master branch.

```
git push origin master
```

Check workflow runs in GitHub Actions and ensure everything is fine.

Now push the tag itself.

```
git push origin v$VERSION
```

This triggers the release workflow which will build binaries, build and push
container images, and draft a GitHub release.

## Wrap Up

Ensure that the new set of images has been pushed to projects repository on
Docker Hub
[here](https://hub.docker.com/repository/docker/trallnag/kubestatus2cloudwatch).

Go to the release page of this project on GitHub
[here](https://github.com/trallnag/kubestatus2cloudwatch/releases) and review
the automatically created release draft.

Set the release title to `$VERSION / $DATE`. For example "1.0.0 / 2023-01-01".

Publish the draft.
