#!/usr/bin/env python3
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1
#
# Validates CLI commands found in remediation sections of cnspec policies
# against known-good sets of subcommands and flags.
#
# Usage:
#   python3 validate_remediation_commands.py            # validate all
#   python3 validate_remediation_commands.py aws         # validate AWS commands only
#   python3 validate_remediation_commands.py azure       # validate Azure commands only
#   python3 validate_remediation_commands.py oci         # validate OCI commands only

import json
import re
import shlex
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CMD_DATA_DIR = SCRIPT_DIR / "cmd_data"

VALIDATORS = ["aws", "azure", "oci"]


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def extract_bash_blocks(content: str) -> list[tuple[str, int, str]]:
    """Extract bash code blocks from cli remediation sections.

    Returns a list of (block_text, line_number, uid) tuples where line_number
    is the 1-based line of the first code line in the block, and uid is the
    check UID that contains this remediation block.
    """
    # Pre-compute a list of (line_number, uid) from all `- uid:` lines so we
    # can look up the enclosing check for any position in the file.
    lines = content.split("\n")
    uid_positions: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = re.match(r"^  - uid:\s+(\S+)", line)
        if m:
            uid_positions.append((i + 1, m.group(1)))

    def find_uid_for_line(line_num: int) -> str:
        """Find the nearest uid defined before line_num."""
        result = ""
        for pos, uid in uid_positions:
            if pos <= line_num:
                result = uid
            else:
                break
        return result

    pattern = re.compile(
        r"- id: cli\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        # Line number of the cli remediation block itself
        cli_line = content[:match.start()].count("\n") + 1
        uid = find_uid_for_line(cli_line)

        for fence in re.finditer(r"```bash\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).strip()
            if block:
                code_offset = desc_start + fence.start(1)
                line_number = content[:code_offset].count("\n") + 1
                blocks.append((block, line_number, uid))
    return blocks


def split_commands(block: str, prefix: str, block_start_line: int) -> list[tuple[str, int]]:
    """Split a code block into individual commands starting with prefix.

    Returns a list of (command, line_number) tuples.
    """
    lines = block.split("\n")
    commands = []
    i = 0

    while i < len(lines):
        line = lines[i]
        raw_line_num = block_start_line + i

        # Join continuation lines
        full_line = line
        cont_lines = 0
        while full_line.rstrip().endswith("\\") and i + cont_lines + 1 < len(lines):
            cont_lines += 1
            full_line = full_line.rstrip()[:-1] + " " + lines[i + cont_lines].strip()

        stripped = full_line.strip()
        if stripped and not stripped.startswith("#"):
            # Use shlex to handle quoted values containing | or ;
            # then re-join and split on unquoted pipes/semicolons
            try:
                tokens = shlex.split(stripped)
            except ValueError:
                tokens = stripped.split()
            rejoined = " ".join(tokens)
            # Split on pipe/semicolon boundaries
            for segment in re.split(r"\s*[|;]\s*", rejoined):
                segment = segment.strip()
                if segment.startswith(f"{prefix} "):
                    commands.append((segment, raw_line_num))

        i += 1 + cont_lines

    return commands


def truncate_cmd(cmd: str, max_len: int = 120) -> str:
    """Collapse whitespace and truncate a command for display."""
    display = " ".join(cmd.split())
    if len(display) > max_len:
        display = display[: max_len - 3] + "..."
    return display


# ---------------------------------------------------------------------------
# AWS validation
# ---------------------------------------------------------------------------

AWS_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-aws-security.mql.yaml"
AWS_COMMANDS_FILE = CMD_DATA_DIR / "aws_commands.json"


def parse_aws_command(cmd: str) -> tuple[str, str, list[str]]:
    """Parse an aws command into (service, subcommand, flags).

    Always returns a non-empty service when the command starts with 'aws'
    followed by at least one non-flag token, so the caller can report
    unknown services/subcommands instead of silently skipping them.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "aws":
        return "", "", []

    service = tokens[1]
    if service.startswith("-"):
        return "", "", []

    subcommand = ""
    flags = []
    for token in tokens[2:]:
        if token.startswith("--"):
            flags.append(token.split("=")[0])
        elif not subcommand and not token.startswith("-"):
            subcommand = token

    return service, subcommand, flags


def validate_aws_command(
    service: str,
    subcommand: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed AWS command against the commands database."""
    errors = []

    if service not in commands_db:
        errors.append(f"unknown service '{service}'")
        return False, errors

    if not subcommand:
        errors.append(f"missing subcommand for '{service}'")
        return False, errors

    valid_subcommands = commands_db[service]
    if subcommand not in valid_subcommands:
        errors.append(f"unknown subcommand '{service} {subcommand}'")
        return False, errors

    key = f"{service} {subcommand}"
    if key in commands_db:
        valid_flags = set(commands_db[key])
        for flag in flags:
            if flag not in valid_flags:
                errors.append(f"unknown flag '{flag}' for '{service} {subcommand}'")

    return len(errors) == 0, errors


def validate_aws() -> tuple[int, int]:
    """Validate AWS CLI commands. Returns (pass_count, fail_count)."""
    if not AWS_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {AWS_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    if not AWS_COMMANDS_FILE.exists():
        print(
            f"Error: Commands database not found: {AWS_COMMANDS_FILE}\n"
            f"Run dump_aws_commands.py first to generate it.",
            file=sys.stderr,
        )
        sys.exit(1)

    commands_db = json.loads(AWS_COMMANDS_FILE.read_text())
    content = AWS_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    for block_text, block_line, uid in blocks:
        commands = split_commands(block_text, "aws", block_line)
        for cmd, line_num in commands:
            service, subcommand, flags = parse_aws_command(cmd)

            if not service:
                continue

            is_valid, errors = validate_aws_command(
                service, subcommand, flags, commands_db
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

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# Azure validation
# ---------------------------------------------------------------------------

AZURE_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-azure-security.mql.yaml"
AZURE_COMMANDS_FILE = CMD_DATA_DIR / "azure_commands.json"


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
    """Validate Azure CLI commands. Returns (pass_count, fail_count)."""
    if not AZURE_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {AZURE_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    if not AZURE_COMMANDS_FILE.exists():
        print(
            f"Error: Commands database not found: {AZURE_COMMANDS_FILE}\n"
            f"Run dump_azure_commands.py first to generate it.",
            file=sys.stderr,
        )
        sys.exit(1)

    commands_db = json.loads(AZURE_COMMANDS_FILE.read_text())
    content = AZURE_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

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

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# OCI validation
# ---------------------------------------------------------------------------

OCI_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-oci-security.mql.yaml"
OCI_COMMANDS_FILE = CMD_DATA_DIR / "oci_commands.json"


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

    if not OCI_COMMANDS_FILE.exists():
        print(
            f"Error: Commands database not found: {OCI_COMMANDS_FILE}\n"
            f"Run dump_oci_commands.py first to generate it.",
            file=sys.stderr,
        )
        sys.exit(1)

    commands_db = json.loads(OCI_COMMANDS_FILE.read_text())
    content = OCI_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

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

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    target = sys.argv[1] if len(sys.argv) > 1 else "all"

    if target not in ("all", *VALIDATORS):
        print(
            f"Unknown validator: {target}\n"
            f"Usage: {sys.argv[0]} [{'|'.join(['all'] + VALIDATORS)}]",
            file=sys.stderr,
        )
        sys.exit(2)

    total_pass = 0
    total_fail = 0

    if target in ("all", "aws"):
        p, f = validate_aws()
        total_pass += p
        total_fail += f

    if target in ("all", "azure"):
        p, f = validate_azure()
        total_pass += p
        total_fail += f

    if target in ("all", "oci"):
        p, f = validate_oci()
        total_pass += p
        total_fail += f

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
