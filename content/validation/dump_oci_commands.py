#!/usr/bin/env python3
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps all valid OCI CLI subcommands and their flags for services used in
# the OCI security policy. Uses the oci_cli Python package's Click command
# tree for fast, accurate introspection.
#
# Output is a JSON file mapping:
#   { "iam": ["auth-token", "user", ...],
#     "iam user list": ["--compartment-id", "--all", ...], ... }
#
# Usage: python3 dump_oci_commands.py [--output oci_commands.json]
#
# Requires: the OCI CLI (`oci`) must be installed. The script finds its
# bundled Python and runs the extraction there.

import argparse
import json
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "oci_commands.json"

# Services referenced in the OCI security policy.
SERVICES = [
    "iam",
    "network",
    "os",
]

# Global flags available on every OCI CLI command.
GLOBAL_FLAGS = [
    "--auth",
    "--auth-purpose",
    "--cert-bundle",
    "--cli-auto-prompt",
    "--cli-rc-file",
    "--config-file",
    "--connection-timeout",
    "--debug",
    "--defaults-file",
    "--enable-dual-stack",
    "--enable-propagation",
    "--endpoint",
    "--generate-full-command-json-input",
    "--generate-param-json-input",
    "--help",
    "--max-retries",
    "--no-retry",
    "--opc-client-request-id",
    "--opc-request-id",
    "--output",
    "--profile",
    "--proxy",
    "--query",
    "--raw-output",
    "--read-timeout",
    "--realm-specific-endpoint",
    "--region",
    "--request-id",
]

# The script that runs inside the OCI CLI's bundled Python to extract the
# command tree from the Click-based CLI.
_EXTRACT_SCRIPT = r"""
import json, sys
sys.path.insert(0, sys.argv[1])

import click
from oci_cli import dynamic_loader
from oci_cli.cli import cli

# Load all service modules so the full command tree is available
dynamic_loader.load_all_services()

services = json.loads(sys.argv[2])

def walk(group, prefix=""):
    results = {}
    if isinstance(group, click.Group):
        subcommands = []
        for name, cmd in sorted(group.commands.items()):
            path = f"{prefix} {name}".strip()
            if isinstance(cmd, click.Group):
                subcommands.append(name)
                results.update(walk(cmd, path))
            else:
                subcommands.append(name)
                flags = []
                for param in cmd.params:
                    for opt in param.opts:
                        if opt.startswith("--"):
                            flags.append(opt)
                    if hasattr(param, "secondary_opts"):
                        for opt in param.secondary_opts:
                            if opt.startswith("--"):
                                flags.append(opt)
                results[path] = sorted(set(flags))
        if prefix:
            results[prefix] = sorted(subcommands)
    return results

result = {}
for svc in services:
    if svc in cli.commands:
        svc_cmd = cli.commands[svc]
        result.update(walk(svc_cmd, svc))
    else:
        print(f"Warning: service '{svc}' not found in OCI CLI", file=sys.stderr)

print(json.dumps(result))
"""


def find_oci_cli_site_packages() -> str:
    """Find the site-packages directory for the OCI CLI's bundled Python."""
    oci_path = subprocess.run(
        ["which", "oci"], capture_output=True, text=True
    ).stdout.strip()

    if not oci_path:
        print("Error: oci CLI not found in PATH.", file=sys.stderr)
        sys.exit(1)

    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/site-packages/oci_cli/__init__.py"))
    if not candidates:
        print(
            "Error: Could not find oci_cli package directory.\n"
            "Is the OCI CLI installed?",
            file=sys.stderr,
        )
        sys.exit(1)

    # site-packages is 2 levels up from oci_cli/__init__.py
    return str(candidates[0].parent.parent)


def find_oci_python() -> str:
    """Find the Python interpreter used by the OCI CLI."""
    oci_path = subprocess.run(
        ["which", "oci"], capture_output=True, text=True
    ).stdout.strip()

    if not oci_path:
        print("Error: oci CLI not found in PATH.", file=sys.stderr)
        sys.exit(1)

    # Read the shebang line to find the Python interpreter
    with open(oci_path) as f:
        first_line = f.readline().strip()

    if first_line.startswith("#!"):
        python_path = first_line[2:].strip()
        if Path(python_path).exists():
            return python_path

    # Fallback: resolve symlinks and look for python in the same prefix
    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/bin/python3"))
    if candidates:
        return str(candidates[0])

    print(
        "Error: Could not find Python interpreter for OCI CLI.",
        file=sys.stderr,
    )
    sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Dump OCI CLI commands and flags to JSON"
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"Output JSON file (default: {DEFAULT_OUTPUT})",
    )
    args = parser.parse_args()

    site_packages = find_oci_cli_site_packages()
    python_path = find_oci_python()
    print(f"Using OCI CLI from: {site_packages}", file=sys.stderr)
    print(f"Using Python: {python_path}", file=sys.stderr)

    # Run extraction in the OCI CLI's own Python environment
    result = subprocess.run(
        [
            python_path,
            "-c",
            _EXTRACT_SCRIPT,
            site_packages,
            json.dumps(SERVICES),
        ],
        capture_output=True,
        text=True,
        timeout=120,
    )

    if result.returncode != 0:
        print(f"Error extracting commands:\n{result.stderr}", file=sys.stderr)
        sys.exit(1)

    if result.stderr:
        for line in result.stderr.splitlines():
            if "Warning" not in line and line.strip():
                print(line, file=sys.stderr)

    commands = json.loads(result.stdout)

    # Add global flags to every leaf command entry (entries with flags, not
    # subcommand lists)
    for key, value in commands.items():
        # Leaf commands have flag lists (strings starting with --)
        if value and isinstance(value[0], str) and value[0].startswith("-"):
            commands[key] = sorted(set(value + GLOBAL_FLAGS))

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(commands, indent=2, sort_keys=True) + "\n")

    total_subcommands = sum(
        1 for k, v in commands.items() if v and isinstance(v[0], str) and v[0].startswith("-")
    )
    total_groups = sum(
        1 for k, v in commands.items() if not v or (isinstance(v[0], str) and not v[0].startswith("-"))
    )
    print(
        f"Wrote {total_subcommands} commands across {total_groups} groups to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
