# Release

This document describes the release process and is targeted at maintainers.

## Preparation

Start by picking a name for the new release. It must follow
[Semantic Versioning](https://semver.org).

Make sure that the "Unreleased" section in the [changelog](CHANGELOG.md) is
up-to-date. Feel free to adjust entries.

Move the content of the "Unreleased" section that will be included in the new
release to a new section with an appropiate title for the release. Should the
"Unreleased" section now be empty, add "Nothing." to it.

## Trigger

Commit the changes. Remember to sign the commit.

```
git commit -S -m "chore: Prepare release v$VERSION"
```

Ensure that the commit is signed.

```
git log --show-signature -1
```

Tag the commit with an annotated and signed tag.

```
git tag -s v$VERSION -m ""
```

Ensure that the tag is signed.

```
git show v$VERSION
```

Push changes, but not the tag.

```
git push
```

Check workflow runs in GitHub Actions and ensure everything is fine. Now push
the tag itself.

```
git push v$VERSION
```

This triggers the release workflow which will build binaries, build and push
container images, and draft a GitHub release.

## Wrap Up

Ensure that the new set of images has been pushed to projects repository on
Docker Hub
[here](https://hub.docker.com/repository/docker/trallnag/kubestatus2cloudwatch).

Go to the release page of this project on GitHub
[here](https://github.com/trallnag/kubestatus2cloudwatch/releases) and
review the automatically created release draft.

Publish the draft.
