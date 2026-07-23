#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps the `vercel` CLI command grammar (command groups, subcommands,
# discoverable aliases, and valid flags) into cmd_data/vercel_commands.json
# for the CLI validator in validators/vercel.py.
#
# Unlike aws/gcloud/doctl (introspected live in CI from botocore models, a
# Click tree, or the SDK completion tree) and unlike the Cobra CLIs
# (kubectl/gh/glab/hcloud, which expose a machine-readable `__complete`
# command), `vercel` is a Node.js CLI with no completion surface. Its help
# text, however, is regular: every command prints a `Commands:` section
# whose entries sit at a fixed indent
#
#     add     name [environment] [git-branch]  Add an Environment Variable
#     list    [environment] [git-branch]       List all Environment Variables
#
# and flags appear in `Options:` / `Global Options:` sections as `--flag`
# lines. We walk the tree by running `vercel <path> --help` for every
# discovered command and parse those two shapes.
#
# The grammar is pinned to a `vercel` version (like NCLI_BOOK or the
# AWS/doctl CLI version): bumping VERCEL_VERSION and regenerating is a
# deliberate maintainer action so the checked-in output is reproducible.
#
# Output JSON:
#   {
#     "_meta": {"vercel_version": "...", "generated_by": "..."},
#     "commands": {
#       "":          {"subcommands": ["env", "tokens", ...], "flags": [...]},
#       "env":       {"subcommands": ["add", "list", "remove", ...], "flags": [...]},
#       "env add":   {"subcommands": [], "flags": ["--sensitive", "--force", ...]},
#       ...
#     },
#     "aliases": {"ls": "list", "i": "install", "ir": "integration-resource"}
#   }
#
# Usage: python3 dump_vercel_commands.py [--output cmd_data/vercel_commands.json]

import argparse
import concurrent.futures
import json
import re
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "vercel_commands.json"
POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-vercel-security.mql.yaml"

# Pin the CLI version the grammar was generated from. `vercel` is installed
# from npm; regenerate after `npm i -g vercel@<new>` and bump this.
VERCEL_VERSION = "56.4.1"

# A command entry in a `Commands:` section: leading indent, a lowercase
# command name, an optional `short | long` alias pair, then whitespace
# before the argument/description columns. Section sub-headers in the
# top-level help ("Basic", "Advanced") are Capitalized, so the lowercase
# anchor excludes them.
_COMMAND_LINE_RE = re.compile(
    r"^(\s+)([a-z][a-z0-9-]*)\s*(?:\|\s*([a-z][a-z0-9-]*))?\s{2,}\S"
)

# A section header line like "Commands:", "Options:", "Global Options:".
_SECTION_HEADER_RE = re.compile(r"^\s*[A-Z][A-Za-z ]*:\s*$")

# A long flag anywhere in help text. Short flags (-y) are not validated
# because remediation snippets use long forms.
_FLAG_RE = re.compile(r"(?<![-\w])(--[a-z][a-z0-9-]+)")


def run_help(path: str) -> str:
    """Return the combined help output for `vercel <path> --help`."""
    args = ["vercel"] + (path.split() if path else []) + ["--help"]
    r = subprocess.run(args, capture_output=True, text=True, timeout=30)
    # `vercel <group> --help` prints to stdout; some paths print usage to
    # stderr. Merge both so parsing is layout-independent.
    return (r.stdout or "") + "\n" + (r.stderr or "")


def parse_commands_section(text: str) -> list[tuple[str, str | None]]:
    """Return (name, alias) pairs from the `Commands:` section, or []."""
    lines = text.splitlines()
    start = None
    for i, line in enumerate(lines):
        if re.match(r"^\s*Commands:\s*$", line):
            start = i + 1
            break
    if start is None:
        return []

    # The section runs to the next header (Options:, Global Options:,
    # Examples:, …) or EOF.
    block = []
    for line in lines[start:]:
        if _SECTION_HEADER_RE.match(line):
            break
        block.append(line)

    candidates = [
        (m.group(1), m.group(2), m.group(3))
        for line in block
        if (m := _COMMAND_LINE_RE.match(line))
    ]
    if not candidates:
        return []

    # Command entries share the shallowest indent in the block; anything
    # deeper is a wrapped description line that happens to start with a
    # lowercase word ("for a Project").
    base_indent = min(len(indent) for indent, _, _ in candidates)
    return [
        (name, alias)
        for indent, name, alias in candidates
        if len(indent) == base_indent
    ]


def parse_flags(text: str) -> list[str]:
    """Return the long flags (own + global) documented in the help text."""
    start = text.find("Options:")
    scope = text[start:] if start != -1 else text
    return sorted(set(_FLAG_RE.findall(scope)))


def _node_info(path: str) -> tuple[dict, dict[str, str]]:
    """Introspect one command node: its {subcommands, flags} and any
    discoverable aliases (each `short | long` pair in the help)."""
    help_text = run_help(path)
    aliases: dict[str, str] = {}
    subcommands: list[str] = []
    for name, alias in parse_commands_section(help_text):
        canonical = name
        if alias and alias != name:
            # Help renders alias pairs as `short | long`; the longer is
            # canonical. Record the shorter as an alias of it.
            short, canonical = sorted((name, alias), key=len)
            aliases[short] = canonical
        if canonical not in subcommands:
            subcommands.append(canonical)
    node = {"subcommands": sorted(set(subcommands)), "flags": parse_flags(help_text)}
    return node, aliases


def detect_root_commands(policy_file: Path) -> list[str]:
    """Top-level `vercel` subcommands referenced by the policy.

    The full `vercel` tree is large and every `--help` is a ~1.5s Node
    startup, so — like the Cobra walkers — we scope the walk to the command
    groups the policy actually uses (env, tokens, certs, …) rather than
    enumerating the whole CLI.
    """
    if not policy_file.exists():
        print(f"Error: policy file not found: {policy_file}", file=sys.stderr)
        sys.exit(1)
    text = policy_file.read_text()
    return sorted(set(re.findall(r"\bvercel\s+([a-z][a-z0-9-]*)", text)))


def walk(roots: list[str]) -> tuple[dict[str, dict], dict[str, str]]:
    """Walk the `vercel` command tree via `--help`, scoped to `roots` and
    their descendants, parallelizing the `--help` subprocesses in each
    round (each is a fresh Node process).

    The root node ("") is always captured for its global flags, top-level
    subcommand list, and discoverable aliases, but is not descended into —
    only the requested roots are. Returns (commands, aliases): commands
    maps each canonical command path to its {subcommands, flags}; aliases
    maps each discoverable alias token to its canonical name.
    """
    commands: dict[str, dict] = {}
    aliases: dict[str, str] = {}
    visited: set[str] = set()

    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
        # Root node first: its subcommand list is what makes `vercel <root>`
        # resolve, but descending it would walk the entire CLI.
        root_node, root_aliases = _node_info("")
        commands[""] = root_node
        aliases.update(root_aliases)
        visited.add("")

        queue = [r for r in roots if r in root_node["subcommands"]
                 or aliases.get(r) in root_node["subcommands"]]
        while queue:
            round_paths = [p for p in queue if p not in visited]
            visited.update(round_paths)
            futures = {pool.submit(_node_info, p): p for p in round_paths}
            next_queue: list[str] = []
            for fut, path in futures.items():
                try:
                    node, node_aliases = fut.result()
                except subprocess.TimeoutExpired:
                    print(f"Warning: timed out on `vercel {path} --help`", file=sys.stderr)
                    continue
                commands[path] = node
                aliases.update(node_aliases)
                for sub in node["subcommands"]:
                    child = f"{path} {sub}".strip()
                    if child not in visited:
                        next_queue.append(child)
            queue = next_queue

    return commands, aliases


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    args = parser.parse_args()

    installed = subprocess.run(
        ["vercel", "--version"], capture_output=True, text=True
    ).stdout.strip()
    if not installed:
        print(
            "Error: `vercel` CLI not found in PATH.\n"
            "Install it with: npm i -g vercel@" + VERCEL_VERSION,
            file=sys.stderr,
        )
        sys.exit(1)
    if installed != VERCEL_VERSION:
        print(
            f"Warning: installed vercel {installed} != pinned {VERCEL_VERSION}; "
            "update VERCEL_VERSION if this is an intentional bump.",
            file=sys.stderr,
        )

    roots = detect_root_commands(POLICY_FILE)
    print(f"Scanning {len(roots)} command groups used by the policy: "
          f"{', '.join(roots)}", file=sys.stderr)
    commands, aliases = walk(roots)

    result = {
        "_meta": {
            "vercel_version": installed,
            "generated_by": "dump_vercel_commands.py",
        },
        "commands": {k: commands[k] for k in sorted(commands)},
        "aliases": {k: aliases[k] for k in sorted(aliases)},
    }

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(
        f"Wrote {len(commands)} command nodes, {len(aliases)} aliases "
        f"to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
