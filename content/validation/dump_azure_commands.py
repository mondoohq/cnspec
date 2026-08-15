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

sys.path.insert(0, str(SCRIPT_DIR))
from validators.common import extract_bash_blocks  # noqa: E402

# Remediation ids whose bash blocks may contain `az` commands. The Azure
# policy uses `cli`; the M365 policy documents a few DNS fixes there too.
# Keep in sync with what validators/azure.py reads.
AZURE_REMEDIATION_IDS = ("cli",)

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


def _parse_help_flags(output: str) -> list[str]:
    """Extract user-facing flags from `az <cmd> --help` output.

    Only extracts flags from argument sections (e.g. "Arguments",
    "Global Arguments"), stopping at "Examples" to avoid picking up
    flags from example text.
    """
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


def get_flags_from_help(cmd_name: str, retries: int = 4) -> list[str]:
    """Parse `az <cmd> --help` to extract user-facing flags, with retries.

    Under the 8-way concurrent Phase 2 load, individual `az --help`
    invocations occasionally fail transiently (nonzero exit, or empty
    output from a cold-starting interpreter). Every real leaf command's
    help lists at least the "Global Arguments" (--debug, --help, ...), so
    an empty flag set is never legitimate — it always signals a failed
    invocation. Retry those, and raise if they never succeed, so the
    caller can fail loudly rather than silently checking in a command with
    a truncated flag list (which then rejects valid remediation commands).
    """
    last_err = ""
    for attempt in range(retries):
        try:
            result = subprocess.run(
                ["az"] + cmd_name.split() + ["--help"],
                capture_output=True,
                text=True,
                timeout=30,
            )
        except subprocess.SubprocessError as e:
            last_err = f"subprocess error: {e}"
            continue
        flags = _parse_help_flags(result.stdout + result.stderr)
        if flags:
            return flags
        last_err = (
            f"rc={result.returncode}, no flags parsed "
            f"(stdout={len(result.stdout)}B, stderr={len(result.stderr)}B)"
        )
    raise RuntimeError(f"az {cmd_name} --help failed after {retries} tries: {last_err}")


def detect_policy_commands(commands: dict) -> set[str]:
    """Return every `az` command path the validator will check.

    That is a wider set than the `- id: cli` remediation blocks alone: the
    validator also reads `audit:` blocks, and the M365 policy documents some
    fixes under other remediation ids. A command missed here keeps phase 1's
    destination-name flags, which is precisely the defect this scoping used
    to cause — so the extraction mirrors the validator's own reader rather
    than re-implementing a narrower regex.
    """
    policy_dir = SCRIPT_DIR / ".."
    policy_files = [
        policy_dir / "mondoo-azure-security.mql.yaml",
        policy_dir / "mondoo-m365-security.mql.yaml",
    ]

    policy_commands = set()
    for policy_file in policy_files:
        if not policy_file.exists():
            continue
        content = policy_file.read_text()
        blocks = extract_bash_blocks(
            content, include_audit=True, remediation_ids=AZURE_REMEDIATION_IDS
        )
        for block, _line, _uid in blocks:
            joined = re.sub(r"\\\s*\n\s*", " ", block)
            for line in joined.split("\n"):
                line = line.strip()
                if not line.startswith("az "):
                    continue
                # Command path is every token before the first flag.
                cmd_parts = []
                for p in line.split()[1:]:
                    if p.startswith("-"):
                        break
                    cmd_parts.append(p)
                # Walk back to the longest prefix that names a real command,
                # so a trailing positional argument does not hide it.
                while cmd_parts:
                    candidate = " ".join(cmd_parts)
                    if candidate in commands:
                        policy_commands.add(candidate)
                        break
                    cmd_parts.pop()

    return policy_commands


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

    # Phase 2: Resolve real flags via `az --help` for each command.
    #
    # Phase 1's flags are argparse *destination* names for any argument whose
    # user-facing alias is registered globally rather than on the command
    # (`resource_group_name` is exposed as `--resource-group`/`-g`, not
    # `--resource-group-name`). Only `--help` reports what the CLI actually
    # accepts, so phase 2 REPLACES the phase-1 flags rather than adding to
    # them — unioning the two lets an invented flag through validation.
    #
    # Parsing --help is slow (~0.5s per command), so it runs only for the
    # commands the validator will actually check: every `az` invocation in
    # the policy files. For all other commands the phase-1 flags remain as a
    # permissive fallback; they only ever serve the "does this command
    # exist" half of validation.
    policy_commands = detect_policy_commands(commands)

    print(
        f"Phase 2: Parsing --help for {len(policy_commands)} policy commands...",
        file=sys.stderr,
    )

    def process_cmd(cmd_name: str) -> tuple[str, list[str] | None]:
        try:
            help_flags = get_flags_from_help(cmd_name)
        except Exception as e:
            print(f"  WARNING: {e}", file=sys.stderr)
            return cmd_name, None
        # `--help` is authoritative: it lists exactly the options this command
        # parses. Phase 1's flags are deliberately discarded here.
        resolved = sorted(
            f for f in set(help_flags + GLOBAL_FLAGS) if not f.endswith("-")
        )
        return cmd_name, resolved

    failed = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
        futures = {
            executor.submit(process_cmd, cmd): cmd for cmd in policy_commands
        }
        for future in concurrent.futures.as_completed(futures):
            cmd_name, merged_flags = future.result()
            if merged_flags is None:
                failed.append(cmd_name)
                continue
            commands[cmd_name] = merged_flags

    # A policy command whose --help never resolved would be checked in with
    # only its (often differently-named) API flags, silently rejecting valid
    # remediation commands. Refuse to write truncated data — fail loudly so
    # the maintainer re-runs instead.
    if failed:
        print(
            f"\nError: could not resolve --help flags for {len(failed)} policy "
            f"command(s): {', '.join(sorted(failed))}\n"
            "Not writing output; re-run the script.",
            file=sys.stderr,
        )
        sys.exit(1)

    # Add global flags to remaining commands
    for key in commands:
        if key not in policy_commands:
            commands[key] = sorted(set(commands[key] + GLOBAL_FLAGS))

    args.output.write_text(json.dumps(commands, indent=2, sort_keys=True) + "\n")
    print(f"Wrote {len(commands)} commands to {args.output}", file=sys.stderr)


if __name__ == "__main__":
    main()
