# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Azure CLI validation.

import json
import shlex
import sys

from pathlib import Path

from .common import CMD_DATA_DIR, FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands, truncate_cmd


# ---------------------------------------------------------------------------
# Azure validation
# ---------------------------------------------------------------------------

AZURE_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-azure-security.mql.yaml"
M365_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-m365-security.mql.yaml"
AZURE_COMMANDS_FILE = CMD_DATA_DIR / "azure_commands.json"

# Policy files whose `id: cli` remediations use the Azure CLI (`az`). The M365
# policy is included here because its CLI remediations also target `az`.
AZURE_CLI_POLICY_FILES = [AZURE_POLICY_FILE, M365_POLICY_FILE]


def parse_az_command(cmd: str, commands_db: dict[str, list[str]]) -> tuple[str, list[str]]:
    """Parse an az command into (command_path, flags).

    Azure CLI commands have variable-depth paths (e.g. 'network nsg rule create').
    We match the longest known command path from the database. If no match is
    found, the raw command path (tokens before the first flag) is returned so
    the caller can report it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "az":
        return "", []

    # Find the longest matching command path by consuming tokens until
    # we hit a flag or run out of matching commands
    parts = tokens[1:]  # skip 'az'
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


def validate_az_command(
    command_path: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed Azure CLI command against the commands database."""
    errors = []

    if command_path not in commands_db:
        errors.append(f"unknown command 'az {command_path}'")
        return False, errors

    valid_flags = set(commands_db[command_path])
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for 'az {command_path}'")

    return len(errors) == 0, errors


def validate_azure() -> tuple[int, int]:
    """Validate Azure CLI commands. Returns (pass_count, fail_count).

    Scans both the Azure security policy and the M365 security policy, since
    M365 `id: cli` remediations also use the Azure CLI (`az`).
    """
    if not AZURE_COMMANDS_FILE.exists():
        print(
            f"Error: Commands database not found: {AZURE_COMMANDS_FILE}\n"
            f"Run dump_azure_commands.py first to generate it.",
            file=sys.stderr,
        )
        sys.exit(1)

    commands_db = json.loads(AZURE_COMMANDS_FILE.read_text())

    pass_count = 0
    fail_count = 0

    for policy_file in AZURE_CLI_POLICY_FILES:
        if not policy_file.exists():
            print(f"Error: Policy file not found: {policy_file}", file=sys.stderr)
            sys.exit(1)

        content = policy_file.read_text()
        blocks = extract_bash_blocks(content)
        relpath = policy_relpath(policy_file)

        for block_text, block_line, uid in blocks:
            commands = split_commands(block_text, "az", block_line)
            for cmd, line_num in commands:
                command_path, flags = parse_az_command(cmd, commands_db)

                if not command_path:
                    continue

                is_valid, errors = validate_az_command(
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
                        "cloud": "azure",
                    })

    return pass_count, fail_count
