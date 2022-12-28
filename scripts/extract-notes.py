#!/usr/bin/env python

"""\
Usage: extract-notes.py [options...]

Extract notes for the latest release from a changelog file. Used to get text
for GitHub Release. Changelog must follow a specific format for this to work.

Options:
  --input INPUT    Changelog file. Defaults to "CHANGELOG.md".
  --output OUTPUT  File where extraction should be written to.

Examples:
  extract-notes.py --output tmp/notes.md
  extract-notes.py --input CHANGELOG.md
"""

import argparse
import sys


class ArgumentParser(argparse.ArgumentParser):
    def print_help(self):
        sys.stdout.write(__doc__)


# Parse args.
parser = ArgumentParser()
parser.add_argument("--input", type=str, default="CHANGELOG.md")
parser.add_argument("--output", type=str)
args = parser.parse_args()

# Map to vars.
arg_input = args.input
arg_output = args.output

# Count number of H2 headers.
prefix_counter = 0

# Extracted lines.
lines = []

# Extract lines between second and third H2 header. Includes the second H2
# header. All headers are upgraded by one level, for example from H3 to H2.
with open(arg_input) as f:
    for line in f:
        if line.startswith("## "):
            prefix_counter += 1

        if prefix_counter == 2:
            if line.startswith("#"):
                lines.append(line[1:])
            else:
                lines.append(line)
        elif prefix_counter == 3:
            break


# Remove first line because it is a header we don't want.
lines = lines[1:]

# Remove line breaks and trim.
lines = [line.strip() for line in lines]

# # Remove empty strings from the beginning of the reversed list.
while len(lines) > 0 and lines[0] == "":
    lines.pop(0)

# Remove empty strings from the end of the list.
lines = list(reversed(lines))
while len(lines) > 0 and lines[0] == "":
    lines.pop(0)
lines = list(reversed(lines))

if arg_output:
    with open(arg_output, "w") as f:
        for line in lines:
            f.write(line + "\n")
else:
    for line in lines:
        print(line)
