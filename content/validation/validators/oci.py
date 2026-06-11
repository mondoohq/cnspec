# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# OCI CLI validation.

import json
import re
import shlex
import subprocess
import sys

from pathlib import Path

from .common import FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands, truncate_cmd


# ---------------------------------------------------------------------------
# OCI validation
# ---------------------------------------------------------------------------
#
# The OCI commands database is built in-memory at validation time by walking
# the Click command tree from the oci_cli Python package. This avoids shipping
# a large oci_commands.json in the repo. Requires the OCI CLI to be installed
# in the environment running validation.

OCI_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-oci-security.mql.yaml"

# Global flags available on every OCI CLI command.
OCI_GLOBAL_FLAGS = [
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

# Script that runs inside the OCI CLI's bundled Python to extract the command
# tree from the Click-based CLI.
_OCI_EXTRACT_SCRIPT = r"""
import json, sys
sys.path.insert(0, sys.argv[1])

import click
from oci_cli import dynamic_loader
from oci_cli.cli import cli

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
        result.update(walk(cli.commands[svc], svc))
    else:
        print(f"Warning: service '{svc}' not found in OCI CLI", file=sys.stderr)

print(json.dumps(result))
"""


def detect_oci_services_from_policy() -> list[str]:
    """Return the set of top-level oci services used in the OCI policy."""
    if not OCI_POLICY_FILE.exists():
        return []

    content = OCI_POLICY_FILE.read_text()
    services = set()
    for match in re.finditer(r"```bash\s*\n(.*?)```", content, re.DOTALL):
        block = match.group(1)
        joined = re.sub(r"\\\s*\n\s*", " ", block)
        for line in joined.split("\n"):
            line = line.strip()
            if line.startswith("oci "):
                parts = line.split()
                if len(parts) >= 2 and not parts[1].startswith("-"):
                    services.add(parts[1])
    return sorted(services)


def _find_oci_binary() -> str | None:
    oci_path = subprocess.run(
        ["which", "oci"], capture_output=True, text=True
    ).stdout.strip()
    return oci_path or None


def find_oci_cli_site_packages(oci_path: str) -> str | None:
    """Locate the site-packages directory containing oci_cli."""
    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/site-packages/oci_cli/__init__.py"))
    if not candidates:
        return None
    return str(candidates[0].parent.parent)


def find_oci_python(oci_path: str) -> str | None:
    """Return the Python interpreter used by the OCI CLI."""
    try:
        with open(oci_path) as f:
            first_line = f.readline().strip()
    except OSError:
        first_line = ""

    if first_line.startswith("#!"):
        python_path = first_line[2:].strip()
        if Path(python_path).exists():
            return python_path

    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/bin/python3"))
    if candidates:
        return str(candidates[0])
    return None


def build_oci_commands_db() -> dict[str, list[str]]:
    """Build an in-memory OCI commands database by walking the oci_cli Click
    command tree, scoped to services used in the OCI policy.

    Returns an empty dict if the policy file has no oci commands. Exits with
    a helpful error if the OCI CLI is not installed.
    """
    services = detect_oci_services_from_policy()
    if not services:
        return {}

    oci_path = _find_oci_binary()
    if not oci_path:
        print(
            "Error: oci CLI not found in PATH.\n"
            "\n"
            "Validating oci remediation commands requires the OCI CLI to be\n"
            "installed locally — the oci_cli Python package's Click command\n"
            "tree is the source of truth for valid commands and flags.\n"
            "\n"
            "Install options:\n"
            "  pip:              pip install oci-cli\n"
            "  macOS (Homebrew): brew install oci-cli\n"
            "  Installer script: https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm\n"
            "\n"
            "After installing, ensure `oci` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} oci\n"
            "\n"
            "To skip oci validation, run a different target instead\n"
            "(e.g. `aws`, `azure`, or `gcp`).",
            file=sys.stderr,
        )
        sys.exit(1)

    site_packages = find_oci_cli_site_packages(oci_path)
    python_path = find_oci_python(oci_path)
    if not site_packages or not python_path:
        print(
            f"Error: Found `oci` at '{oci_path}' but could not locate its\n"
            f"bundled Python environment (site-packages: {site_packages}, "
            f"python: {python_path}).\n"
            "\n"
            "This usually indicates a broken or partial CLI install. Try\n"
            "reinstalling the OCI CLI (see install options above).",
            file=sys.stderr,
        )
        sys.exit(1)

    result = subprocess.run(
        [
            python_path,
            "-c",
            _OCI_EXTRACT_SCRIPT,
            site_packages,
            json.dumps(services),
        ],
        capture_output=True,
        text=True,
        timeout=120,
    )

    if result.returncode != 0:
        print(
            f"Error extracting OCI commands from oci_cli:\n{result.stderr}",
            file=sys.stderr,
        )
        sys.exit(1)
    if result.stderr:
        for line in result.stderr.splitlines():
            if line.strip() and "Warning" not in line:
                print(line, file=sys.stderr)

    commands = json.loads(result.stdout)
    for key, value in commands.items():
        if value and isinstance(value[0], str) and value[0].startswith("-"):
            commands[key] = sorted(set(value + OCI_GLOBAL_FLAGS))
    return commands


def parse_oci_command(cmd: str, commands_db: dict[str, list[str]]) -> tuple[str, list[str]]:
    """Parse an oci command into (command_path, flags).

    OCI CLI commands have variable-depth paths (e.g. 'iam user api-key list').
    We match the longest known command path from the database. If no match is
    found, the raw command path (tokens before the first flag) is returned so
    the caller can report it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "oci":
        return "", []

    # Find the longest matching command path by consuming tokens until
    # we hit a flag or run out of matching commands
    parts = tokens[1:]  # skip 'oci'
    command_path = ""
    raw_path_parts = []
    for i in range(len(parts)):
        if parts[i].startswith("-"):
            break
        raw_path_parts.append(parts[i])
        candidate = " ".join(parts[: i + 1])
        if candidate in commands_db:
            command_path = candidate

    # If no match found, return the raw path so the caller can report it
    if not command_path and raw_path_parts:
        command_path = " ".join(raw_path_parts)

    # Extract flags from remaining tokens
    flags = []
    for token in parts:
        if token.startswith("--"):
            flag = token.split("=")[0]
            flags.append(flag)

    return command_path, flags


def validate_oci_command(
    command_path: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed OCI CLI command against the commands database."""
    errors = []

    if command_path not in commands_db:
        errors.append(f"unknown command 'oci {command_path}'")
        return False, errors

    valid_flags = set(commands_db[command_path])
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for 'oci {command_path}'")

    return len(errors) == 0, errors


def validate_oci() -> tuple[int, int]:
    """Validate OCI CLI commands. Returns (pass_count, fail_count)."""
    if not OCI_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {OCI_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    commands_db = build_oci_commands_db()
    if not commands_db:
        return 0, 0

    content = OCI_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    relpath = policy_relpath(OCI_POLICY_FILE)

    for block_text, block_line, uid in blocks:
        commands = split_commands(block_text, "oci", block_line)
        for cmd, line_num in commands:
            command_path, flags = parse_oci_command(cmd, commands_db)

            if not command_path:
                continue

            is_valid, errors = validate_oci_command(
                command_path, flags, commands_db
            )

            if is_valid:
                print(f"[PASS] {uid}")
                print(f"       {truncate_cmd(cmd)}")
                pass_count += 1
            else:
                print(f"[FAIL] {uid}")
                print(f"       {truncate_cmd(cmd)}")
                for error in errors:
                    print(f"       {error}")
                fail_count += 1
                FAILURES.append({
                    "file": relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "oci",
                })

    return pass_count, fail_count
