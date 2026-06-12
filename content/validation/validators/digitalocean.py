# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# doctl (DigitalOcean) CLI validation.

import concurrent.futures
import re
import shlex
import subprocess
import sys

from pathlib import Path

from .common import FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands, truncate_cmd


# ---------------------------------------------------------------------------
# doctl (DigitalOcean) validation
# ---------------------------------------------------------------------------
#
# Unlike aws/oci/gcp, doctl is a Go binary with no Python introspection
# surface and no static completion tree we can parse. We walk `doctl --help`
# breadth-first, parsing each subcommand and flag list out of the Cobra-
# rendered help text. The walk is parallelized; a full traversal of the
# ~475-command tree completes in roughly one second.

DO_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-digitalocean-security.mql.yaml"

# Cobra-rendered section headers like "Available Commands:", "Manage
# DigitalOcean Resources:", "View Billing:". The top-level `doctl --help`
# uses category headers instead of "Available Commands:".
_DOCTL_SECTION_HEADER = re.compile(r"^[A-Z][^:\n]*:\s*$")

# Subcommand line: "  <name>   <description>" — two-space indent, then a
# lowercase identifier, then whitespace before the description. Cobra pads
# names to the longest one plus a single space, so the longest subcommand in
# a group has exactly one space before its description.
_DOCTL_SUBCOMMAND_LINE = re.compile(r"^  ([a-z][a-z0-9-]*)\s+\S")

# Flag line: "      --flag-name <type>   <description>" or
# "  -X, --flag-name <type>   <description>". We only capture the long form.
_DOCTL_FLAG_LINE = re.compile(r"^\s+(?:-\w,\s+)?(--[a-z][a-z0-9-]*)")


def _doctl_help(path: str) -> str:
    """Return stdout+stderr of `doctl <path> --help`."""
    args = ["doctl"] + (path.split() if path else []) + ["--help"]
    r = subprocess.run(args, capture_output=True, text=True, timeout=15)
    return r.stdout + r.stderr


def _parse_doctl_help(text: str) -> tuple[list[str], list[str]]:
    """Extract (subcommands, flags) from a doctl --help blob.

    Handles both the top-level layout (category headers like "Manage
    DigitalOcean Resources:") and group/leaf layouts ("Available Commands:"
    / "Flags:" / "Global Flags:").
    """
    subcommands: list[str] = []
    flags: list[str] = []
    in_flags = False
    in_cmds = False

    for line in text.split("\n"):
        if line.startswith("Flags:") or line.startswith("Global Flags:"):
            in_flags = True
            in_cmds = False
            continue
        if _DOCTL_SECTION_HEADER.match(line):
            in_flags = False
            # Example invocations are indented like subcommand lines but
            # start with "doctl"; don't mine the Examples/Usage sections.
            in_cmds = not line.startswith(("Examples:", "Usage:"))
            continue
        if line.strip() == "":
            continue

        if in_flags:
            m = _DOCTL_FLAG_LINE.match(line)
            if m:
                flags.append(m.group(1))
        elif in_cmds:
            m = _DOCTL_SUBCOMMAND_LINE.match(line)
            if m:
                subcommands.append(m.group(1))

    return sorted(set(subcommands)), sorted(set(flags))


def detect_doctl_services_from_policy() -> list[str]:
    """Return the set of top-level doctl services used in the DO policy."""
    if not DO_POLICY_FILE.exists():
        return []

    content = DO_POLICY_FILE.read_text()
    services = set()
    for match in re.finditer(r"```bash\s*\n(.*?)```", content, re.DOTALL):
        block = match.group(1)
        joined = re.sub(r"\\\s*\n\s*", " ", block)
        for line in joined.split("\n"):
            line = line.strip()
            if line.startswith("doctl "):
                parts = line.split()
                if len(parts) >= 2 and not parts[1].startswith("-"):
                    services.add(parts[1])
    return sorted(services)


def build_doctl_commands_db() -> dict[str, list[str]]:
    """Build an in-memory doctl commands database by walking the Cobra help
    tree breadth-first, scoped to services used in the DO policy.

    Returns an empty dict if the policy file has no doctl commands. Exits
    with a helpful error if doctl is not installed.
    """
    services = detect_doctl_services_from_policy()
    if not services:
        return {}

    doctl_path = subprocess.run(
        ["which", "doctl"], capture_output=True, text=True
    ).stdout.strip()
    if not doctl_path:
        print(
            "Error: doctl CLI not found in PATH.\n"
            "\n"
            "Validating doctl remediation commands requires the doctl CLI to\n"
            "be installed locally — its `--help` output is the source of truth\n"
            "for valid commands and flags.\n"
            "\n"
            "Install options:\n"
            "  macOS (Homebrew): brew install doctl\n"
            "  Linux (snap):     snap install doctl\n"
            "  Other:            https://docs.digitalocean.com/reference/doctl/how-to/install/\n"
            "\n"
            "After installing, ensure `doctl` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} digitalocean\n"
            "\n"
            "To skip doctl validation, run a different target instead\n"
            "(e.g. `aws`, `azure`, `oci`, or `gcp`).",
            file=sys.stderr,
        )
        sys.exit(1)

    # Breadth-first walk, parallelizing the subprocess calls in each round.
    # The executor is created once and reused across rounds so we don't pay
    # thread-pool startup cost on every level of the tree.
    visited: set[str] = set()
    results: dict[str, dict] = {}
    queue = list(services)

    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
        while queue:
            future_to_path = {}
            for path in queue:
                if path in visited:
                    continue
                visited.add(path)
                future_to_path[pool.submit(_doctl_help, path)] = path
            next_queue: list[str] = []
            for fut, path in future_to_path.items():
                try:
                    text = fut.result()
                except subprocess.TimeoutExpired:
                    print(
                        f"Warning: timed out fetching `doctl {path} --help`",
                        file=sys.stderr,
                    )
                    continue
                subs, flags = _parse_doctl_help(text)
                results[path] = {"subcommands": subs, "flags": flags}
                for s in subs:
                    sub_path = f"{path} {s}".strip()
                    if sub_path not in visited:
                        next_queue.append(sub_path)
            queue = next_queue

    # Flatten {"compute firewall": {subcommands, flags}, ...} into the
    # validator's expected shape: a flat map where group entries hold the
    # list of valid subcommand names and leaf entries hold the flag list.
    commands: dict[str, list[str]] = {}
    for path, node in results.items():
        if node["subcommands"]:
            commands[path] = node["subcommands"]
        else:
            commands[path] = node["flags"]
    return commands


def parse_doctl_command(cmd: str, commands_db: dict[str, list[str]]) -> tuple[str, list[str]]:
    """Parse a doctl command into (command_path, flags).

    doctl commands have variable-depth paths (e.g. 'compute firewall add-rules').
    We match the longest known command path from the database. If no match is
    found, the raw command path (tokens before the first flag) is returned so
    the caller can report it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "doctl":
        return "", []

    parts = tokens[1:]  # skip 'doctl'
    command_path = ""
    raw_path_parts = []
    for i in range(len(parts)):
        if parts[i].startswith("-"):
            break
        raw_path_parts.append(parts[i])
        candidate = " ".join(parts[: i + 1])
        if candidate in commands_db:
            command_path = candidate

    if not command_path and raw_path_parts:
        command_path = " ".join(raw_path_parts)

    flags = []
    for token in parts:
        if token.startswith("--"):
            flags.append(token.split("=")[0])

    return command_path, flags


def validate_doctl_command(
    command_path: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed doctl command against the commands database."""
    errors = []

    if command_path not in commands_db:
        errors.append(f"unknown command 'doctl {command_path}'")
        return False, errors

    entry = commands_db[command_path]
    # Group entries hold subcommand names; leaf entries hold flag names
    # (which start with `--`). If we landed on a group, the user is missing
    # a required subcommand.
    if entry and not entry[0].startswith("-"):
        errors.append(
            f"'doctl {command_path}' is a command group; missing subcommand"
        )
        return False, errors

    valid_flags = set(entry)
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for 'doctl {command_path}'")

    return len(errors) == 0, errors


def validate_digitalocean() -> tuple[int, int]:
    """Validate doctl CLI commands. Returns (pass_count, fail_count)."""
    if not DO_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {DO_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    commands_db = build_doctl_commands_db()
    if not commands_db:
        return 0, 0

    content = DO_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    relpath = policy_relpath(DO_POLICY_FILE)

    for block_text, block_line, uid in blocks:
        commands = split_commands(block_text, "doctl", block_line)
        for cmd, line_num in commands:
            command_path, flags = parse_doctl_command(cmd, commands_db)

            if not command_path:
                continue

            is_valid, errors = validate_doctl_command(
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
                    "cloud": "digitalocean",
                })

    return pass_count, fail_count
