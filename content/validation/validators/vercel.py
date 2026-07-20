# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Vercel CLI validation.
#
# `vercel` is a Node.js CLI with no completion surface to introspect in CI
# (unlike aws/gcloud/doctl) and no Cobra `__complete` command (unlike
# kubectl/gh/glab/hcloud). Its help text is regular, though, so the command
# grammar is generated once by dump_vercel_commands.py and checked into
# cmd_data/vercel_commands.json — the same model as azure_commands.json and
# ncli_commands.json. See that script for the pinned CLI version and how to
# regenerate.

import json
import shlex
import sys
from pathlib import Path

from .common import (
    CMD_DATA_DIR,
    FAILURES,
    SCRIPT_DIR,
    extract_bash_blocks,
    policy_relpath,
    split_commands,
    truncate_cmd,
)

VERCEL_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-vercel-security.mql.yaml"
VERCEL_COMMANDS_FILE = CMD_DATA_DIR / "vercel_commands.json"

# Command aliases `vercel` accepts consistently but does not surface in the
# per-subcommand `--help` output the dump parses (help only shows the
# canonical `list` / `remove`, yet `vercel env ls` and `vercel env rm`
# both work). Only add entries attested by running the CLI. Discoverable
# aliases (the top-level `ls | list` pairs) come from the grammar's own
# `aliases` map and need no entry here.
VERCEL_EXTRA_ALIASES = {
    "ls": "list",
    "rm": "remove",
}


def _load_vercel_db() -> tuple[dict, dict]:
    """Load the generated grammar; exits if it has not been generated."""
    if not VERCEL_COMMANDS_FILE.exists():
        print(
            f"Error: Commands database not found: {VERCEL_COMMANDS_FILE}\n"
            f"Run dump_vercel_commands.py first to generate it.",
            file=sys.stderr,
        )
        sys.exit(1)
    data = json.loads(VERCEL_COMMANDS_FILE.read_text())
    aliases = {**data.get("aliases", {}), **VERCEL_EXTRA_ALIASES}
    return data["commands"], aliases


def parse_vercel_command(
    cmd: str, commands: dict, aliases: dict
) -> tuple[str, str | None, list[str]]:
    """Parse a `vercel` command into (command_path, next_token, flags).

    Matches the longest known command path, resolving alias tokens (`ls` ->
    `list`) to their canonical name so the path key stays canonical. Tokens
    after the matched path (env-var names, domains, ids) are positionals and
    are not validated. next_token is the first positional after the path,
    which the caller reports as a misspelled subcommand when the path lands
    on a command group. Returns the raw pre-flag tokens as the path when
    nothing matches, so the caller reports it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "vercel":
        return "", None, []

    parts = tokens[1:]

    def resolve(token: str, parent: str) -> str | None:
        """Canonical subcommand name of token under parent, or None."""
        subs = commands.get(parent, {}).get("subcommands", [])
        if token in subs:
            return token
        canonical = aliases.get(token)
        if canonical in subs:
            return canonical
        return None

    # Positional (non-flag) tokens in order; the command path is the
    # longest resolvable prefix of these.
    positionals = [t for t in parts if not t.startswith("-")]

    command_path = ""
    matched = 0
    parent = ""
    for token in positionals:
        canonical = resolve(token, parent)
        if canonical is None:
            break
        command_path = f"{parent} {canonical}".strip()
        parent = command_path
        matched += 1

    # The first positional after a matched command group is a candidate
    # misspelled subcommand; after a leaf it is just an argument, which the
    # validator ignores.
    next_token = positionals[matched] if matched < len(positionals) else None

    if not command_path and positionals:
        # Nothing resolved: report the leading positionals as the unknown
        # command (a top-level typo like `vercel enviroment`).
        command_path = positionals[0]

    flags = [t.split("=")[0] for t in parts if t.startswith("--")]
    return command_path, next_token, flags


def validate_vercel_command(
    command_path: str,
    next_token: str | None,
    flags: list[str],
    commands: dict,
) -> tuple[bool, list[str]]:
    """Validate a parsed `vercel` command against the generated grammar."""
    errors: list[str] = []

    node = commands.get(command_path)
    if node is None:
        errors.append(f"unknown command 'vercel {command_path}'")
        return False, errors

    # A node with subcommands is a command group and needs one; the next
    # positional (if any) is a misspelled subcommand rather than an
    # argument, because a real subcommand would have extended the path.
    if node["subcommands"]:
        if next_token is not None:
            errors.append(
                f"unknown subcommand '{next_token}' for 'vercel {command_path}'"
            )
        else:
            errors.append(
                f"'vercel {command_path}' is a command group; missing subcommand"
            )
        return False, errors

    valid_flags = set(node["flags"])
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for 'vercel {command_path}'")

    return len(errors) == 0, errors


def validate_vercel() -> tuple[int, int]:
    """Validate Vercel CLI commands. Returns (pass_count, fail_count)."""
    commands, aliases = _load_vercel_db()

    if not VERCEL_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {VERCEL_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    content = VERCEL_POLICY_FILE.read_text()
    # Vercel's audit paths use the CLI directly (`### Audit via CLI`), so
    # audit blocks are validated alongside cli remediation blocks. This is
    # a new validator with no backlog of unvalidated audit blocks.
    blocks = extract_bash_blocks(content, include_audit=True)
    relpath = policy_relpath(VERCEL_POLICY_FILE)

    pass_count = 0
    fail_count = 0

    for block_text, block_line, uid in blocks:
        for cmd, line_num in split_commands(block_text, "vercel", block_line):
            command_path, next_token, flags = parse_vercel_command(
                cmd, commands, aliases
            )
            if not command_path:
                continue

            is_valid, errors = validate_vercel_command(
                command_path, next_token, flags, commands
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
                    "cloud": "vercel",
                })

    return pass_count, fail_count
