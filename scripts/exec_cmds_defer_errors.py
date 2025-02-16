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
# ]
#
# ///
#

import subprocess
import sys
from dataclasses import dataclass
from typing import override

import click

VERSION = "2.0.0"

HELP = """
Execute commands and defer errors.

Here it is used to run three commands:

\b
uv run --script exec_cmds_defer_errors.py "echo hi" "exit 1" "echo world"

Every command is executed in its own shell.
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


@dataclass
class CmdExecErrorInfo:
    """Information about a command execution error."""

    cmd_index: int
    """Index of the command."""

    cmd_short: str
    """Shortened version of the command."""

    exit_code: int
    """Exit code of the command."""


@click.command(
    cls=Command,
    context_settings={
        "help_option_names": ["-h", "--help"],
        "show_default": True,
    },
    help=HELP,
    epilog=EPILOG,
)
@click.argument("cmds", nargs=-1)
@click.version_option(VERSION)
def exec_cmds_defer_errors(cmds: list[str]) -> None:
    """Execute commands and defer errors."""

    cmd_exec_errors: list[CmdExecErrorInfo] = []

    for index, cmd in enumerate(cmds):
        click.secho(f"Executing command {index + 1}...", fg="blue")
        click.secho(cmd.strip(), fg="blue")

        # Rule S602 disabled as running arbitrary code in shell is intended here.
        completed_proc = subprocess.run(cmd, shell=True, check=False)  # noqa: S602

        if completed_proc.returncode == 0:
            click.secho(f"Executed command {index + 1} successfully.", fg="green")
        else:
            click.secho(
                (
                    f"Executed command {index + 1} failed "
                    f"with exit code {completed_proc.returncode}."
                ),
                fg="red",
            )

            cut_off_limit = 30
            cmd_short = f"{cmd[:cut_off_limit]}..." if len(cmd) > cut_off_limit else cmd

            cmd_short = cmd_short.replace("\r\n", " ").replace("\n", " ")

            cmd_exec_errors.append(
                CmdExecErrorInfo(
                    cmd_index=index + 1,
                    cmd_short=cmd_short,
                    exit_code=completed_proc.returncode,
                )
            )

    if not cmd_exec_errors:
        click.secho(
            f"All {len(cmds)} commands executed successfully.",
            fg="green",
        )

        sys.exit(0)
    else:
        click.secho(
            f"{len(cmd_exec_errors)} out of {len(cmds)} command(s) failed.", fg="red"
        )

        for cmd_exec_error in cmd_exec_errors:
            click.secho(
                (
                    f"Command {cmd_exec_error.cmd_index} failed "
                    f"with exit code {cmd_exec_error.exit_code}: "
                    f"{cmd_exec_error.cmd_short}"
                ),
                fg="red",
            )

        sys.exit(1)


if __name__ == "__main__":
    exec_cmds_defer_errors()
