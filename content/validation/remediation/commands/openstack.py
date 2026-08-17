# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# openstack (OpenStackClient) CLI validation.

import os
import re
import shlex
import subprocess
import sys

from common import (
    FAILURES,
    CONTENT_DIR,
    extract_bash_blocks,
    policy_relpath,
    split_commands,
    truncate_cmd,
)

# ---------------------------------------------------------------------------
# openstack (OpenStackClient) validation
# ---------------------------------------------------------------------------
#
# OpenStackClient is built on cliff, which gives every command tree a
# `complete` subcommand that prints the whole grammar as a bash-completion
# script. That is one subprocess for the entire tree, where doctl needs a
# breadth-first walk of several hundred `--help` invocations:
#
#   cmds='access address aggregate ... volume vpn'
#   cmds_security_group='create delete list set show unset'
#   cmds_security_group_rule_create='-h --help ... --protocol --dst-port'
#
# A node's value is either its subcommand names or its flags; a leaf is
# recognizable because cliff always emits `-h --help` first. The keys join
# path words with `_`, which would be ambiguous if a command name ever
# contained one, so the tree is rebuilt by walking down from the root rather
# than by splitting keys: each level's names come from its parent's value
# list, so a name containing any character resolves exactly.
#
# `complete` is generated from the loaded entry points and never contacts a
# cloud, so the walk needs no credentials and is deterministic. Auth
# environment variables are cleared anyway, so a developer's sourced
# openrc cannot change the grammar.

OPENSTACK_POLICY_FILE = CONTENT_DIR / "mondoo-openstack-security.mql.yaml"

# OpenStackClient ships the core services (compute, identity, image, network,
# volume, object store, shared file systems). Every other service is a separate
# PyPI package that registers its own cliff entry points, and `openstack
# complete` only knows about the ones installed — exactly like the Azure CLI
# and its extensions, where a missing extension silently shrinks the grammar
# and turns a real command into "unknown command".
#
# So the plugins the policy needs are declared here rather than left to
# whatever the machine happens to have. Each maps its root command to the
# package that provides it; a plugin that contributes no node is a hard error,
# because the alternative is a grammar that quietly validates less than it
# claims to.
OPENSTACK_PLUGINS = {
    "loadbalancer": "python-octaviaclient",   # Octavia, load balancing
    "coe": "python-magnumclient",             # Magnum, container orchestration
    "database": "python-troveclient",         # Trove, database as a service
    "datastore": "python-troveclient",
    "secret": "python-barbicanclient",        # Barbican, key management
}

_CMDS_LINE = re.compile(r"^\s*cmds(_[A-Za-z0-9_]+)?='([^']*)'", re.M)


def _openstack_complete() -> str:
    """Return the bash-completion grammar `openstack complete` prints."""
    env = {k: v for k, v in os.environ.items() if not k.startswith("OS_")}
    r = subprocess.run(
        ["openstack", "complete"],
        capture_output=True, text=True, timeout=120, env=env,
    )
    if r.returncode != 0 or "cmds=" not in r.stdout:
        print(
            "Error: `openstack complete` produced no grammar "
            f"(exit {r.returncode}).\n"
            f"stderr:\n{r.stderr[:1000]}",
            file=sys.stderr,
        )
        sys.exit(1)
    return r.stdout


def _parse_completion(text: str) -> dict[str, list[str]]:
    """Parse the completion script into its raw `cmds_<key>` node map."""
    nodes: dict[str, list[str]] = {}
    for m in _CMDS_LINE.finditer(text):
        nodes[(m.group(1) or "")[1:]] = m.group(2).split()
    return nodes


def _is_leaf(values: list[str]) -> bool:
    """A leaf node holds flags; cliff emits `-h`/`--help` on every one."""
    return any(v.startswith("-") for v in values)


def build_openstack_commands_db() -> dict[str, list[str]]:
    """Build the openstack command grammar, keyed by space-joined path.

    Walks down from the root instead of splitting the underscore-joined keys,
    so a command name is never re-split at a character that happens to be the
    key separator. Exits with an install hint if the CLI is missing, or if a
    declared plugin contributed nothing.
    """
    if not subprocess.run(["which", "openstack"], capture_output=True).stdout.strip():
        print(
            "Error: openstack CLI not found in PATH.\n"
            "\n"
            "Validating openstack remediation commands requires OpenStackClient\n"
            "locally — `openstack complete` is the source of truth for valid\n"
            "commands and flags.\n"
            "\n"
            "Install (the service plugins are separate packages, and the\n"
            "grammar omits any that is missing):\n"
            "  pip install python-openstackclient python-octaviaclient \\\n"
            "              python-magnumclient python-troveclient \\\n"
            "              python-barbicanclient\n"
            "\n"
            "After installing, re-run:\n"
            f"  python3 {sys.argv[0]} openstack",
            file=sys.stderr,
        )
        sys.exit(1)

    nodes = _parse_completion(_openstack_complete())

    commands: dict[str, list[str]] = {}
    # (key in the raw node map, path as the policy writes it)
    queue = [(w, w) for w in nodes.get("", [])]
    while queue:
        key, path = queue.pop()
        values = nodes.get(key)
        if values is None:
            continue
        commands[path] = values
        if not _is_leaf(values):
            for name in values:
                queue.append((f"{key}_{name}", f"{path} {name}"))

    missing = sorted({
        pkg for root, pkg in OPENSTACK_PLUGINS.items() if root not in commands
    })
    if missing:
        print(
            "Error: these OpenStackClient plugins are declared in\n"
            "OPENSTACK_PLUGINS but contributed no commands, so the grammar is\n"
            "smaller than the policy needs and real commands would be reported\n"
            "as unknown:\n"
            + "".join(f"  {pkg}\n" for pkg in missing)
            + "\nInstall them with:\n"
            f"  pip install {' '.join(missing)}",
            file=sys.stderr,
        )
        sys.exit(1)

    return commands


def parse_openstack_command(
    cmd: str, commands_db: dict[str, list[str]]
) -> tuple[str, list[str], str]:
    """Parse an openstack invocation into (command_path, long flags, next word).

    Command paths are variable-depth ('security group rule create') and are
    followed by positional arguments that look exactly like path words, so the
    longest path present in the grammar wins and the rest are positionals.

    The word immediately after the matched path comes back too. On a leaf it is
    a positional and is ignored; on a group it is whatever was written where a
    subcommand belongs, which is the word the reader got wrong.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "openstack":
        return "", [], ""

    words: list[str] = []
    for token in tokens[1:]:
        if token.startswith("-"):
            break
        words.append(token)

    command_path = ""
    depth = 0
    for i in range(len(words)):
        candidate = " ".join(words[: i + 1])
        if candidate in commands_db:
            command_path = candidate
            depth = i + 1

    if not command_path and words:
        command_path = " ".join(words)
        depth = len(words)

    flags = [t.split("=")[0] for t in tokens if t.startswith("--")]
    next_word = words[depth] if depth < len(words) else ""
    return command_path, flags, next_word


def validate_openstack_command(
    command_path: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
    next_word: str = "",
) -> tuple[bool, list[str]]:
    """Validate a parsed openstack command against the grammar."""
    errors = []

    if command_path not in commands_db:
        errors.append(f"unknown command 'openstack {command_path}'")
        return False, errors

    values = commands_db[command_path]
    if not _is_leaf(values):
        if next_word:
            errors.append(
                f"'{next_word}' is not a subcommand of 'openstack {command_path}'"
            )
        else:
            errors.append(
                f"'openstack {command_path}' is a command group; missing subcommand"
            )
        return False, errors

    valid = set(values)
    for flag in flags:
        if flag not in valid:
            errors.append(f"unknown flag '{flag}' for 'openstack {command_path}'")

    return len(errors) == 0, errors


def validate_openstack() -> tuple[int, int]:
    """Validate openstack CLI commands. Returns (pass_count, fail_count)."""
    if not OPENSTACK_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {OPENSTACK_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    content = OPENSTACK_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content, include_audit=True)
    if not blocks:
        return 0, 0

    commands_db = build_openstack_commands_db()
    relpath = policy_relpath(OPENSTACK_POLICY_FILE)

    pass_count = fail_count = 0
    for block_text, block_line, uid in blocks:
        for cmd, line_num in split_commands(block_text, "openstack", block_line):
            command_path, flags, next_word = parse_openstack_command(cmd, commands_db)
            if not command_path:
                continue

            is_valid, errors = validate_openstack_command(
                command_path, flags, commands_db, next_word
            )

            print(f"[{'PASS' if is_valid else 'FAIL'}] {uid}")
            print(f"       {truncate_cmd(cmd)}")
            if is_valid:
                pass_count += 1
                continue

            for error in errors:
                print(f"       {error}")
            fail_count += 1
            FAILURES.append({
                "file": relpath,
                "line": line_num,
                "uid": uid,
                "command": truncate_cmd(cmd),
                "errors": errors,
                "cloud": "openstack",
            })

    return pass_count, fail_count
