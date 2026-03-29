#!/usr/bin/env python3
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps all valid gcloud CLI subcommands and their flags for services used
# in the GCP security policy. Reads from the pre-built static completion
# tree bundled with the Google Cloud SDK.
#
# Output is a JSON file mapping:
#   { "compute": ["instances", "networks", ...],
#     "compute instances list": ["--filter", "--project", ...], ... }
#
# Usage: python3 dump_gcloud_commands.py [--output gcloud_commands.json]
#
# Requires: the Google Cloud SDK (`gcloud`) must be installed.

import argparse
import concurrent.futures
import json
import re
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "gcloud_commands.json"
GCLOUD_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-gcp-security.mql.yaml"


def detect_services_from_policy() -> list[str]:
    """Auto-detect gcloud services used in the policy file.

    Scans `gcloud <service> ...` commands in bash code blocks within cli
    remediation sections and returns the unique set of top-level service
    names.
    """
    if not GCLOUD_POLICY_FILE.exists():
        print(
            f"Warning: Policy file not found: {GCLOUD_POLICY_FILE}\n"
            f"Cannot auto-detect services.",
            file=sys.stderr,
        )
        return []

    content = GCLOUD_POLICY_FILE.read_text()
    services = set()
    for match in re.finditer(r"```bash\s*\n(.*?)```", content, re.DOTALL):
        block = match.group(1)
        joined = re.sub(r"\\\s*\n\s*", " ", block)
        for line in joined.split("\n"):
            line = line.strip()
            if line.startswith("gcloud "):
                parts = line.split()
                if len(parts) >= 2 and not parts[1].startswith("-"):
                    services.add(parts[1])
    return sorted(services)


def find_gcloud_completions_path() -> str:
    """Find the gcloud static completion tree file."""
    gcloud_path = subprocess.run(
        ["which", "gcloud"], capture_output=True, text=True
    ).stdout.strip()

    if not gcloud_path:
        print("Error: gcloud CLI not found in PATH.", file=sys.stderr)
        sys.exit(1)

    # gcloud is typically at <sdk>/bin/gcloud; completions at <sdk>/data/cli/
    real_path = Path(gcloud_path).resolve()
    sdk_root = real_path.parent.parent

    # Try standard location first
    completions = sdk_root / "data" / "cli" / "gcloud_completions.py"
    if completions.exists():
        return str(completions.parent)

    # Search more broadly
    candidates = list(sdk_root.glob("**/data/cli/gcloud_completions.py"))
    if candidates:
        return str(candidates[0].parent)

    # Try share directory (Homebrew layout)
    share_candidates = list(
        Path("/opt/homebrew/share/google-cloud-sdk/data/cli").glob(
            "gcloud_completions.py"
        )
    )
    if share_candidates:
        return str(share_candidates[0].parent)

    print(
        "Error: Could not find gcloud_completions.py.\n"
        "Is the Google Cloud SDK installed?",
        file=sys.stderr,
    )
    sys.exit(1)


def walk_tree(node: dict, prefix: str = "") -> dict[str, list[str]]:
    """Recursively walk the gcloud completion tree.

    Returns a flat dict mapping command paths to their subcommand names
    (for groups) or flag names (for leaf commands). Boolean flags
    automatically get --no- variants since gcloud generates these.
    """
    results = {}
    commands = node.get("commands", {})
    if not commands:
        return results

    subcommand_names = []
    for name, child in sorted(commands.items()):
        path = f"{prefix} {name}".strip()
        child_commands = child.get("commands", {})

        if child_commands:
            # This is a group — recurse
            subcommand_names.append(name)
            results.update(walk_tree(child, path))
        else:
            # This is a leaf command — extract flags with --no- variants
            subcommand_names.append(name)
            flags_dict = child.get("flags", {})
            flags = []
            for flag_name, flag_type in flags_dict.items():
                flags.append(flag_name)
                # gcloud generates --no- variants for boolean flags
                if flag_type == "bool" and not flag_name.startswith("--no-"):
                    no_flag = "--no-" + flag_name[2:]
                    flags.append(no_flag)
            results[path] = sorted(set(flags))

    if prefix:
        results[prefix] = sorted(subcommand_names)

    return results


def get_flags_from_help(cmd_path: str) -> list[str]:
    """Parse `gcloud <cmd> --help` to extract flags.

    The static completion tree may be incomplete for some commands.
    Parsing --help gives the authoritative set of user-facing flags.
    """
    result = subprocess.run(
        ["gcloud"] + cmd_path.split() + ["--help"],
        capture_output=True,
        text=True,
        timeout=30,
    )
    output = result.stdout + result.stderr

    flags = set()
    for match in re.finditer(r"(--[a-z][a-z0-9-]*)", output):
        flag = match.group(1)
        # Skip flags that appear in example text or URLs
        if flag not in ("--help", "--format"):
            flags.add(flag)
    return sorted(flags)


def detect_policy_commands(commands_db: dict[str, list[str]]) -> set[str]:
    """Find gcloud commands used in the policy file."""
    if not GCLOUD_POLICY_FILE.exists():
        return set()

    content = GCLOUD_POLICY_FILE.read_text()
    policy_commands = set()
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
                if not line.startswith("gcloud "):
                    continue
                parts = line.split()
                cmd_parts = []
                for p in parts[1:]:
                    if p.startswith("-"):
                        break
                    cmd_parts.append(p)
                candidate = " ".join(cmd_parts)
                if candidate in commands_db:
                    policy_commands.add(candidate)
    return policy_commands


def main():
    parser = argparse.ArgumentParser(
        description="Dump gcloud CLI commands and flags to JSON"
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"Output JSON file (default: {DEFAULT_OUTPUT})",
    )
    args = parser.parse_args()

    services = detect_services_from_policy()
    if not services:
        print(
            "Error: No gcloud services detected from policy file.",
            file=sys.stderr,
        )
        sys.exit(1)
    print(
        f"Detected services from policy: {', '.join(services)}",
        file=sys.stderr,
    )

    completions_dir = find_gcloud_completions_path()
    print(f"Using completions from: {completions_dir}", file=sys.stderr)

    # Import the static completion tree
    sys.path.insert(0, completions_dir)
    try:
        from gcloud_completions import STATIC_COMPLETION_CLI_TREE as tree
    except ImportError as e:
        print(f"Error: Could not import gcloud_completions: {e}", file=sys.stderr)
        sys.exit(1)

    # Extract global flags from the root node
    global_flags = sorted(tree.get("flags", {}).keys())

    # Walk only the services used in the policy
    commands = {}
    all_commands = tree.get("commands", {})
    for svc in services:
        if svc in all_commands:
            svc_node = all_commands[svc]
            commands.update(walk_tree({"commands": {svc: svc_node}}, ""))
        else:
            print(
                f"Warning: service '{svc}' not found in gcloud CLI",
                file=sys.stderr,
            )

    # Add global flags to every leaf command entry
    for key, value in commands.items():
        if value and isinstance(value[0], str) and value[0].startswith("-"):
            commands[key] = sorted(set(value + global_flags))

    # Phase 2: Supplement flags via `gcloud --help` for commands used in the
    # policy. The static completion tree may be incomplete for some commands.
    policy_commands = detect_policy_commands(commands)
    if policy_commands:
        print(
            f"Phase 2: Parsing --help for {len(policy_commands)} policy commands...",
            file=sys.stderr,
        )

        def process_cmd(cmd_name: str) -> tuple[str, list[str]]:
            try:
                help_flags = get_flags_from_help(cmd_name)
            except Exception:
                help_flags = []
            existing_flags = commands.get(cmd_name, [])
            merged = sorted(set(existing_flags + help_flags))
            return cmd_name, merged

        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
            futures = {
                executor.submit(process_cmd, cmd): cmd for cmd in policy_commands
            }
            for future in concurrent.futures.as_completed(futures):
                cmd_name, merged_flags = future.result()
                commands[cmd_name] = merged_flags

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(commands, indent=2, sort_keys=True) + "\n")

    total_subcommands = sum(
        1
        for k, v in commands.items()
        if v and isinstance(v[0], str) and v[0].startswith("-")
    )
    total_groups = sum(
        1
        for k, v in commands.items()
        if not v or (isinstance(v[0], str) and not v[0].startswith("-"))
    )
    print(
        f"Wrote {total_subcommands} commands across {total_groups} groups to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
