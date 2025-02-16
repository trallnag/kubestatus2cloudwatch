#!/usr/bin/env -S uv run --script

#
# This work is available under the ISC license.
#
# Copyright Tim Schwenke <tim@trallnag.com>
#
# Permission to use, copy, modify, and/or distribute this software for any
# purpose with or without fee is hereby granted, provided that the above
# copyright notice and this permission notice appear in all copies.
#
# THE SOFTWARE IS PROVIDED “AS IS” AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
# REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
# AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
# INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
# LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
# OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
# PERFORMANCE OF THIS SOFTWARE.
#
# /// script
#
# requires-python = ">= 3.12"
#
# dependencies = [
#   "click == 8.1.8",
#   "pyyaml == 6.0.2",
# ]
#
# ///
#

import re
from enum import StrEnum
from pathlib import Path
from typing import TypedDict, override

import click
import yaml

VERSION = "2.0.1"

HELP = """
Filter pre-commit hooks.

The script output can be used to populate the SKIP environment variable, so
that only a subset of hooks is executed when running pre-commit.

Here it is used to run all hooks that are tagged with "fix" and "task":

\b
SKIP=$(uv run --script filter_pre_commit_hooks.py fix task) pre-commit run -a

Note that in the example the script is executed with "uv run", a subcommand of
uv, which is a package manager for Python. This is because the script contains
inline script metadata specifying required dependencies. The script also
contains a shebang, so it can be executed directly.

Tags are extracted from the "alias" field of every hook. Tags are declared by
putting them into parenthesis at the end of the respective alias. Individual
tags are separated by commas. Here are two exemplary aliases:

\b
- forbid-new-submodules (check, task)
- mixed-line-ending (fix, task)

Options can be passed to change the behavior of the script. For example, to
filter hooks by their identifier instead of their tags.
"""

EPILOG = """
\b
For more information, check out
<https://github.com/trallnag/filter-pre-commit-hooks>.
"""


class Command(click.Command):
    """Custom command. Only used to customize the epilog formatting."""

    @override
    def format_epilog(self, ctx: click.Context, formatter: click.HelpFormatter) -> None:
        """Format the epilog. Just like the original, but without indentation."""

        if self.epilog:
            formatter.write_paragraph()
            formatter.write_text(self.epilog)


class Target(StrEnum):
    """Target to filter hooks by."""

    ID = "id"
    """Filter hooks by their identifier."""

    TAG = "tag"
    """Filter hooks by their tag."""


class Mode(StrEnum):
    """Mode to filter hooks by target."""

    ALL_OF = "all-of"
    """Filter hooks that have all of the given values for target."""

    ANY_OF = "any-of"
    """Filter hooks that have any of the given values for target."""


class Orient(StrEnum):
    """Orient to filter hooks by."""

    INVERT = "invert"
    """Output hook identifiers that don't match the filters."""

    NO_INVERT = "no-invert"
    """Output hook identifiers that match the filters."""


class Format(StrEnum):
    """Format to output hooks."""

    COMMA = "comma"
    """Output hooks as comma-separated list."""

    NEWLINE = "newline"
    """Output hooks as newline-separated list."""


class Hook(TypedDict):
    """Pre-commit hook."""

    id: str
    alias: str


class Repo(TypedDict):
    """Pre-commit repository."""

    hooks: list[Hook]


class Config(TypedDict):
    """Pre-commit config."""

    repos: list[Repo]


def extract_tags(alias: str | None) -> set[str]:
    """Extract tags from alias."""

    if alias is None:
        return set()

    match = re.match(r".*\((?P<tags>.*)\)$", alias)

    if match is None:
        return set()

    return {tag.strip() for tag in match.group("tags").split(",")}


def is_hook_filtered(
    hook: Hook,
    filters: list[str],
    target: Target,
    mode: Mode,
    orient: Orient,
) -> bool:
    """Decide if hook is filtered or not."""

    filtered = False

    if target == Target.TAG:
        tags = extract_tags(hook.get("alias"))

        if mode == Mode.ALL_OF:
            filtered = all(f in tags for f in filters)
        else:
            filtered = any(f in tags for f in filters)
    elif mode == Mode.ALL_OF:
        filtered = all(f == hook["id"] for f in filters)
    else:
        filtered = any(f == hook["id"] for f in filters)

    return not filtered if orient == Orient.INVERT else filtered


def format_hooks(
    hooks: set[str],
    output_format: Format,
) -> str:
    """Format hooks."""

    if output_format == Format.COMMA:
        return ", ".join(sorted(hooks))

    return "\n".join(sorted(hooks))


@click.command(
    cls=Command,
    context_settings={
        "help_option_names": ["-h", "--help"],
        "show_default": True,
    },
    help=HELP,
    epilog=EPILOG,
)
@click.argument(
    "filters",
    envvar="FILTERS",
    type=click.STRING,
    nargs=-1,
)
@click.option(
    "--config",
    type=click.Path(
        exists=True,
        file_okay=True,
        readable=True,
        resolve_path=True,
        allow_dash=True,
        path_type=Path,
    ),
    default=".pre-commit-config.yaml",
    help="Path to the pre-commit config file.",
)
@click.option(
    "--target",
    type=click.Choice([target.value for target in Target]),
    default=Target.TAG,
    help=(
        "Target to filter hooks by. "
        "With `id`, the hook identifier is filtered. "
        "With `tag`, the hook tags are filtered."
    ),
)
@click.option(
    "--mode",
    type=click.Choice([mode.value for mode in Mode]),
    default=Mode.ALL_OF,
    help=(
        "Mode to filter hooks by target. "
        "With `all_of`, all filters must match. "
        "With `any_of`, any filter must match."
    ),
)
@click.option(
    "--orient",
    type=click.Choice([orient.value for orient in Orient]),
    default=Orient.INVERT,
    help=(
        "Invert the filter. "
        "With invert, hooks that match the filter are excluded from output. "
        "With no-invert, only hooks that match the filter are included in output."
    ),
)
@click.option(
    "--format",
    "o_format",
    type=click.Choice([o_format.value for o_format in Format]),
    default=Format.COMMA,
)
@click.version_option(VERSION)
def filter_pre_commit_hooks(  # noqa: PLR0913
    filters: list[str],
    config: Path,
    target: Target,
    mode: Mode,
    orient: Orient,
    o_format: Format,
) -> None:
    """Filter pre-commit hooks."""

    with Path.open(config) as pre_commit_config:
        data: Config = yaml.safe_load(pre_commit_config)

    filtered_hooks = set()

    for repo in data["repos"]:
        for hook in repo["hooks"]:
            if is_hook_filtered(hook, filters, target, mode, orient):
                filtered_hooks.add(hook["id"])

    click.echo(format_hooks(filtered_hooks, o_format))


if __name__ == "__main__":
    filter_pre_commit_hooks()
