#!/usr/bin/env python

"""\
Usage: build-dist.py [options...]

Build distribution. Includes cross-compiled binaries, archives, and checksums.

Options:
  --name     Name of app. Defaults to "app".
  --dir      Directory for dist. Defaults to "dist".
  --version  Version of dist.

Examples:
  build-dist.py --name token2go-server --version 1.2.3
  build-dist.py --dir tmp --name justatest
"""

import argparse
import hashlib
import os
import shutil
import subprocess
import sys
import tarfile
from collections import namedtuple
from os.path import basename, exists, isdir, isfile, join
from zipfile import ZipFile


class ArgumentParser(argparse.ArgumentParser):
    def print_help(self):
        sys.stdout.write(__doc__)


def sha256sum(file):
    hash = hashlib.sha256()
    with open(file, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            hash.update(byte_block)
    return hash.hexdigest() + "  " + basename(file)


# Parse args.
parser = ArgumentParser()
parser.add_argument("--name", type=str, default="app")
parser.add_argument("--dir", type=str, default="dist")
parser.add_argument("--version", type=str)
args = parser.parse_args()

# Map to vars.
name = args.name
dist_dir = args.dir
version = args.version

# Supported platforms.
Platform = namedtuple("Platform", ["os", "arch"])
platforms = [
    Platform("darwin", "amd64"),
    Platform("darwin", "arm64"),
    Platform("linux", "amd64"),
    Platform("linux", "arm64"),
    Platform("windows", "amd64"),
    Platform("windows", "arm64"),
]

# Delete and recreate dist dir.
if exists(dist_dir):
    shutil.rmtree(dist_dir)
os.mkdir(dist_dir)

for pf in platforms:
    # Output for platform is placed here.
    odir_name = name
    odir_name += f"-{version}" if version else ""
    odir_name += f"-{pf.os}-{pf.arch}"

    # Relative path to platform output dir.
    odir_path = join(dist_dir, odir_name)

    # Ensure output directory exists.
    if not isdir(odir_path):
        os.mkdir(odir_path)

    # Base name of binary including extension.
    bin_name = f"{name}.exe" if pf.os == "windows" else name

    # Relative path to binary.
    bin_path = join(odir_path, bin_name)

    os.environ["CGO_ENABLED"] = "0"
    os.environ["GOOS"] = pf.os
    os.environ["GOARCH"] = pf.arch

    ldflags = "-s -w"
    if version:
        ldflags += f" -X 'main.version={version}'"

    subprocess.run(["go", "build", "-o", bin_path, "-ldflags", ldflags], env=os.environ)

    # Include other files.
    if isfile("LICENSE"):
        shutil.copy("LICENSE", odir_path)
    if isfile("CHANGELOG.md"):
        shutil.copy("CHANGELOG.md", odir_path)

    # Archive contents of output directory.
    if pf.os == "windows":
        arc_ext = ".zip"
        with ZipFile(join(dist_dir, odir_name + arc_ext), "w") as archive:
            for root, dirs, files in os.walk(odir_path):
                for f in files:
                    archive.write(join(root, f), arcname=basename(f))
    else:
        arc_ext = ".tar.gz"
        with tarfile.open(join(dist_dir, odir_name + arc_ext), "w:gz") as archive:
            for root, dirs, files in os.walk(odir_path):
                for f in files:
                    archive.add(join(root, f), arcname=basename(f))

    # Add sha256sum type of checksum string to txt file.
    with open(join(dist_dir, "sha256sums.txt"), "a") as f:
        f.write(sha256sum(join(dist_dir, odir_name + arc_ext)) + "\n")
