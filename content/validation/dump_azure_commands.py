#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps all valid Azure CLI commands and their flags.
#
# Uses a two-phase approach:
#   1. Load the full command table from az CLI internals (fast, gets all command
#      names but flags may have incorrect aliases for some commands).
#   2. For each command, supplement flags by parsing `az <cmd> --help` output
#      which gives the actual user-facing flag names.
#
# Output is a JSON file mapping:
#   { "network nsg rule create": ["--access", "--name", ...], ... }
#
# Usage: python3 dump_azure_commands.py [--output azure_commands.json]

import argparse
import concurrent.futures
import json
import re
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "azure_commands.json"

# Global flags available on every az CLI command
GLOBAL_FLAGS = [
    "--debug",
    "--help",
    "--only-show-errors",
    "--output",
    "--query",
    "--subscription",
    "--verbose",
]

# The script runs inside the az CLI's bundled Python to extract the command
# table and initial flags. Some flags may have internal names rather than
# user-facing aliases (e.g. --resource-group-name instead of --resource-group).
_EXTRACT_SCRIPT = r"""
import json, sys

sys.path.insert(0, sys.argv[1])

# Pre-import requests to avoid deadlocks when Azure CLI modules
# lazy-import it concurrently during command table loading (affects
# containerapp and cdn modules in az CLI 2.85.0+).
import requests  # noqa: F401

from azure.cli.core import get_default_cli, MainCommandsLoader
from azure.cli.core.commands import AzCliCommandInvoker
from azure.cli.core.parser import AzCliCommandParser

cli = get_default_cli()
parser = AzCliCommandParser(cli)
cli.invocation = AzCliCommandInvoker(
    cli, parser_cls=AzCliCommandParser, commands_loader_cls=MainCommandsLoader
)
cli.invocation.data = {"command_string": ""}

loader = MainCommandsLoader(cli)
cmd_table = loader.load_command_table(None)
print(f"Loaded {len(cmd_table)} commands", file=sys.stderr)

result = {}
errors = 0
for cmd_name in sorted(cmd_table.keys()):
    try:
        cli.invocation.data["command_string"] = cmd_name
        loader.load_arguments(cmd_name)
        cmd = cmd_table[cmd_name]
        cmd.load_arguments()
        flags = []
        for name, arg in cmd.arguments.items():
            opts = arg.type.settings.get("options_list", [])
            flags.extend([o for o in opts if o.startswith("--")])
        result[cmd_name] = sorted(
            f for f in set(flags) if not f.endswith("-")
        )
    except Exception:
        errors += 1
        result[cmd_name] = []

print(f"Loaded args for {len(result)} commands ({errors} errors)", file=sys.stderr)
print(json.dumps(result))
"""


def find_az_site_packages() -> str:
    """Find the site-packages directory for the Azure CLI's bundled Python."""
    az_path = subprocess.run(
        ["which", "az"], capture_output=True, text=True
    ).stdout.strip()

    if not az_path:
        print("Error: az CLI not found in PATH.", file=sys.stderr)
        sys.exit(1)

    wrapper = Path(az_path).read_text()
    match = re.search(r"(/\S+/libexec/bin/python\S*)", wrapper)
    if match:
        py_path = Path(match.group(1))
        site_packages = py_path.parent.parent / "lib"
        candidates = list(site_packages.glob("python*/site-packages"))
        if candidates:
            return str(candidates[0])

    real_path = Path(az_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/site-packages/azure/cli/core"))
    if candidates:
        return str(candidates[0].parent.parent.parent)

    print(
        "Error: Could not find Azure CLI site-packages directory.\n"
        "Is the Azure CLI installed?",
        file=sys.stderr,
    )
    sys.exit(1)


def get_flags_from_help(cmd_name: str) -> list[str]:
    """Parse `az <cmd> --help` to extract user-facing flags.

    Only extracts flags from argument sections (e.g. "Arguments",
    "Global Arguments"), stopping at "Examples" to avoid picking up
    flags from example text.
    """
    result = subprocess.run(
        ["az"] + cmd_name.split() + ["--help"],
        capture_output=True,
        text=True,
        timeout=30,
    )
    output = result.stdout + result.stderr

    # Only parse lines within argument sections, stop at Examples
    flags = set()
    in_args = False
    for line in output.splitlines():
        # Section headers are left-aligned capitalized words
        if re.match(r"^[A-Z]", line):
            header = line.strip().rstrip(":")
            if "Arguments" in header or "Parameters" in header:
                in_args = True
            else:
                in_args = False
            # Stop entirely at Examples section
            if header == "Examples":
                break
            continue
        if in_args:
            for match in re.finditer(r"(--[a-z][a-z0-9-]*)", line):
                flags.add(match.group(1))
    return sorted(flags)


def main():
    parser = argparse.ArgumentParser(
        description="Dump Azure CLI commands and flags to JSON"
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"Output JSON file (default: {DEFAULT_OUTPUT})",
    )
    args = parser.parse_args()

    site_packages = find_az_site_packages()
    print(f"Using Azure CLI from: {site_packages}", file=sys.stderr)

    # Phase 1: Bulk extract command names and initial flags from Python API
    print("Phase 1: Loading command table...", file=sys.stderr)
    result = subprocess.run(
        [sys.executable, "-c", _EXTRACT_SCRIPT, site_packages],
        capture_output=True,
        text=True,
        timeout=300,
    )

    if result.returncode != 0:
        print(f"Error extracting commands:\n{result.stderr}", file=sys.stderr)
        sys.exit(1)

    if result.stderr:
        for line in result.stderr.splitlines():
            if "SyntaxWarning" not in line and line.strip():
                print(line, file=sys.stderr)

    commands = json.loads(result.stdout)

    # Phase 2: Supplement flags via `az --help` for each command.
    # The Python API misses some user-facing aliases (e.g. --resource-group
    # may appear as --resource-group-name). Parsing --help is accurate but
    # slow (~0.5s per command), so we only do it for commands that appear in
    # the policy files we validate against. For all other commands, the API
    # flags plus global flags are sufficient for existence checking.
    policy_dir = SCRIPT_DIR / ".."
    policy_commands = set()
    for policy_file in policy_dir.glob("*azure*.mql.yaml"):
        content = policy_file.read_text()
        # Extract az commands from bash blocks in cli remediation sections
        for match in re.finditer(
            r"- id: cli\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
            content,
            re.DOTALL,
        ):
            for fence in re.finditer(r"```bash\s*\n(.*?)```", match.group(1), re.DOTALL):
                block = fence.group(1)
                joined = re.sub(r"\\\s*\n\s*", " ", block)
                for line in joined.split("\n"):
                    line = line.strip()
                    if not line.startswith("az "):
                        continue
                    # Extract command path (tokens before first flag)
                    parts = line.split()
                    cmd_parts = []
                    for p in parts[1:]:
                        if p.startswith("-"):
                            break
                        cmd_parts.append(p)
                    candidate = " ".join(cmd_parts)
                    if candidate in commands:
                        policy_commands.add(candidate)

    print(
        f"Phase 2: Parsing --help for {len(policy_commands)} policy commands...",
        file=sys.stderr,
    )

    def process_cmd(cmd_name: str) -> tuple[str, list[str]]:
        try:
            help_flags = get_flags_from_help(cmd_name)
        except Exception:
            help_flags = []
        api_flags = commands.get(cmd_name, [])
        merged = sorted(
            f for f in set(api_flags + help_flags + GLOBAL_FLAGS)
            if not f.endswith("-")
        )
        return cmd_name, merged

    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
        futures = {
            executor.submit(process_cmd, cmd): cmd for cmd in policy_commands
        }
        for future in concurrent.futures.as_completed(futures):
            cmd_name, merged_flags = future.result()
            commands[cmd_name] = merged_flags

    # Add global flags to remaining commands
    for key in commands:
        if key not in policy_commands:
            commands[key] = sorted(set(commands[key] + GLOBAL_FLAGS))

    args.output.write_text(json.dumps(commands, indent=2, sort_keys=True) + "\n")
    print(f"Wrote {len(commands)} commands to {args.output}", file=sys.stderr)


if __name__ == "__main__":
    main()
