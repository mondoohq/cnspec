# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# gcloud CLI validation.

import concurrent.futures
import re
import shlex
import subprocess
import sys

from collections.abc import Iterator
from pathlib import Path

from .common import FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands, truncate_cmd


# ---------------------------------------------------------------------------
# gcloud validation
# ---------------------------------------------------------------------------
#
# Unlike aws/azure/oci, the gcloud commands database is built in-memory at
# validation time from the Google Cloud SDK's static completion tree. This
# avoids shipping a large, frequently-churning gcloud_commands.json in the
# repo. Requires gcloud to be installed in the environment running validation.

GCLOUD_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-gcp-security.mql.yaml"


def detect_gcloud_services_from_policy() -> list[str]:
    """Return the set of top-level gcloud services used in the GCP policy."""
    if not GCLOUD_POLICY_FILE.exists():
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


def find_gcloud_completions_path() -> str | None:
    """Locate the gcloud static completion tree directory, or None if missing."""
    which = subprocess.run(
        ["which", "gcloud"], capture_output=True, text=True
    )
    gcloud_path = which.stdout.strip()
    if not gcloud_path:
        return None

    real_path = Path(gcloud_path).resolve()
    sdk_root = real_path.parent.parent

    completions = sdk_root / "data" / "cli" / "gcloud_completions.py"
    if completions.exists():
        return str(completions.parent)

    candidates = list(sdk_root.glob("**/data/cli/gcloud_completions.py"))
    if candidates:
        return str(candidates[0].parent)

    share_candidates = list(
        Path("/opt/homebrew/share/google-cloud-sdk/data/cli").glob(
            "gcloud_completions.py"
        )
    )
    if share_candidates:
        return str(share_candidates[0].parent)

    return None


def walk_gcloud_tree(node: dict, prefix: str = "") -> dict[str, list[str]]:
    """Flatten the gcloud completion tree into {path: [subcommands|flags]}.

    Boolean flags get synthetic --no- variants since gcloud generates these.
    """
    results: dict[str, list[str]] = {}
    commands = node.get("commands", {})
    if not commands:
        return results

    subcommand_names = []
    for name, child in sorted(commands.items()):
        path = f"{prefix} {name}".strip()
        child_commands = child.get("commands", {})

        if child_commands:
            subcommand_names.append(name)
            results.update(walk_gcloud_tree(child, path))
        else:
            subcommand_names.append(name)
            flags_dict = child.get("flags", {})
            flags = []
            for flag_name, flag_type in flags_dict.items():
                flags.append(flag_name)
                if flag_type == "bool" and not flag_name.startswith("--no-"):
                    flags.append("--no-" + flag_name[2:])
            results[path] = sorted(set(flags))

    if prefix:
        results[prefix] = sorted(subcommand_names)

    return results


def get_gcloud_flags_from_help(cmd_path: str) -> list[str]:
    """Parse `gcloud <cmd> --help` output to extract flags.

    The static completion tree is occasionally incomplete for specific
    commands; --help is the authoritative source of user-facing flags.
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
        flags.add(match.group(1))
    return sorted(flags)


def probe_gcloud_command(cmd_path: str) -> list[str] | None:
    """Run `gcloud <cmd_path> --help` and return the parsed flags if the
    command exists, otherwise None.

    Used to backfill commands missing from the SDK's static completion
    tree — gcloud hides some real subcommands such as
    `gcloud logging cmek-settings update`.
    """
    try:
        result = subprocess.run(
            ["gcloud"] + cmd_path.split() + ["--help"],
            capture_output=True,
            text=True,
            timeout=30,
        )
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return None
    if result.returncode != 0:
        return None
    output = result.stdout + result.stderr
    flags = set()
    for match in re.finditer(r"(--[a-z][a-z0-9-]*)", output):
        flags.add(match.group(1))
    return sorted(flags)


_GCLOUD_SUBCOMMAND_RE = re.compile(r"^[a-z][a-z0-9-]*$")


def _iter_gcloud_policy_invocations() -> Iterator[list[str]]:
    """Yield the post-``gcloud`` token list for every gcloud invocation in an
    ``id: cli`` remediation in the GCP policy.

    Centralises the YAML + fenced-block + line-continuation parsing so the
    per-token filtering logic stays in the callers.
    """
    if not GCLOUD_POLICY_FILE.exists():
        return

    content = GCLOUD_POLICY_FILE.read_text()
    for match in re.finditer(
        r"- id: cli\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        content,
        re.DOTALL,
    ):
        for fence in re.finditer(
            r"```bash\s*\n(.*?)```", match.group(1), re.DOTALL
        ):
            joined = re.sub(r"\\\s*\n\s*", " ", fence.group(1))
            for line in joined.split("\n"):
                line = line.strip()
                if not line.startswith("gcloud "):
                    continue
                yield line.split()[1:]


def detect_gcloud_policy_command_paths() -> set[str]:
    """Return the bare gcloud command paths referenced by `id: cli` policy
    remediations.

    A path is the longest leading run of lowercase-hyphen tokens before any
    flag or positional placeholder (`INSTANCE_NAME`, `[ZONE]`, `<KEY>`).
    The result is suitable for direct lookup against the in-memory
    commands DB.
    """
    paths: set[str] = set()
    for tokens in _iter_gcloud_policy_invocations():
        cmd_parts: list[str] = []
        for p in tokens:
            if p.startswith("-") or not _GCLOUD_SUBCOMMAND_RE.match(p):
                break
            cmd_parts.append(p)
        if cmd_parts:
            paths.add(" ".join(cmd_parts))
    return paths


def detect_gcloud_policy_commands(commands_db: dict[str, list[str]]) -> set[str]:
    """Return the set of known gcloud command paths used in the GCP policy."""
    policy_commands: set[str] = set()
    for tokens in _iter_gcloud_policy_invocations():
        cmd_parts: list[str] = []
        for p in tokens:
            if p.startswith("-"):
                break
            cmd_parts.append(p)
        candidate = " ".join(cmd_parts)
        if candidate in commands_db:
            policy_commands.add(candidate)
    return policy_commands


def build_gcloud_commands_db() -> dict[str, list[str]]:
    """Build an in-memory gcloud commands database by walking the SDK's
    static completion tree, scoped to services used in the GCP policy, and
    supplementing policy-referenced leaf commands with flags parsed from
    `gcloud --help`.

    Returns an empty dict if the policy file has no gcloud commands or the
    gcloud SDK is not installed.
    """
    services = detect_gcloud_services_from_policy()
    if not services:
        return {}

    completions_dir = find_gcloud_completions_path()
    if not completions_dir:
        print(
            "Error: gcloud CLI not found in PATH.\n"
            "\n"
            "Validating gcloud remediation commands requires the Google Cloud\n"
            "SDK to be installed locally — the static completion tree bundled\n"
            "with the SDK is the source of truth for valid commands and flags.\n"
            "\n"
            "Install options:\n"
            "  macOS (Homebrew): brew install --cask google-cloud-sdk\n"
            "  Debian/Ubuntu:    https://cloud.google.com/sdk/docs/install-sdk#deb\n"
            "  RHEL/Fedora:      https://cloud.google.com/sdk/docs/install-sdk#rpm\n"
            "  Other:            https://cloud.google.com/sdk/docs/install\n"
            "\n"
            "After installing, ensure `gcloud` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} gcp\n"
            "\n"
            "To skip gcloud validation, run a non-gcp target instead\n"
            "(e.g. `aws`, `azure`, or `oci`).",
            file=sys.stderr,
        )
        sys.exit(1)

    sys.path.insert(0, completions_dir)
    try:
        from gcloud_completions import STATIC_COMPLETION_CLI_TREE as tree
    except ImportError as e:
        print(
            f"Error: Found gcloud at '{completions_dir}' but could not import\n"
            f"gcloud_completions: {e}\n"
            "\n"
            "This usually indicates a broken or partial SDK install. Try\n"
            "reinstalling the Google Cloud SDK (see\n"
            "https://cloud.google.com/sdk/docs/install).",
            file=sys.stderr,
        )
        sys.exit(1)

    global_flags = sorted(tree.get("flags", {}).keys())

    commands: dict[str, list[str]] = {}
    all_commands = tree.get("commands", {})
    for svc in services:
        svc_node = all_commands.get(svc)
        if svc_node is None:
            print(
                f"Warning: gcloud service '{svc}' not found in SDK",
                file=sys.stderr,
            )
            continue
        commands.update(walk_gcloud_tree({"commands": {svc: svc_node}}, ""))

    for key, value in commands.items():
        if value and isinstance(value[0], str) and value[0].startswith("-"):
            commands[key] = sorted(set(value + global_flags))

    policy_commands = detect_gcloud_policy_commands(commands)
    if policy_commands:
        def _merge(cmd_name: str) -> tuple[str, list[str]]:
            try:
                help_flags = get_gcloud_flags_from_help(cmd_name)
            except Exception:
                help_flags = []
            existing = commands.get(cmd_name, [])
            return cmd_name, sorted(set(existing + help_flags))

        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            for cmd_name, merged in pool.map(_merge, policy_commands):
                commands[cmd_name] = merged

    # Backfill policy-referenced commands that aren't in the static
    # completion tree (e.g. `gcloud logging cmek-settings update`) by
    # probing `gcloud <cmd> --help` and using its exit code as ground truth.
    policy_paths = detect_gcloud_policy_command_paths()
    missing_paths = sorted(p for p in policy_paths if p not in commands)
    if missing_paths:
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            for cmd_path, flags in zip(
                missing_paths, pool.map(probe_gcloud_command, missing_paths)
            ):
                if flags is None:
                    continue
                commands[cmd_path] = sorted(set(flags + global_flags))

    return commands


def parse_gcloud_command(cmd: str, commands_db: dict[str, list[str]]) -> tuple[str, list[str]]:
    """Parse a gcloud command into (command_path, flags).

    gcloud commands have variable-depth paths (e.g. 'compute instances create').
    We match the longest known command path from the database. If no match is
    found, the raw command path is returned so the caller can report it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "gcloud":
        return "", []

    # Find the longest matching command path by consuming tokens until
    # we hit a flag or run out of matching commands
    parts = tokens[1:]  # skip 'gcloud'
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


def validate_gcloud_command(
    command_path: str,
    flags: list[str],
    commands_db: dict[str, list[str]],
) -> tuple[bool, list[str]]:
    """Validate a parsed gcloud command against the commands database."""
    errors = []

    if command_path not in commands_db:
        errors.append(f"unknown command 'gcloud {command_path}'")
        return False, errors

    valid_flags = set(commands_db[command_path])
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for 'gcloud {command_path}'")

    return len(errors) == 0, errors


def validate_gcloud() -> tuple[int, int]:
    """Validate gcloud CLI commands. Returns (pass_count, fail_count)."""
    if not GCLOUD_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {GCLOUD_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    commands_db = build_gcloud_commands_db()
    if not commands_db:
        return 0, 0

    content = GCLOUD_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    relpath = policy_relpath(GCLOUD_POLICY_FILE)

    for block_text, block_line, uid in blocks:
        commands = split_commands(block_text, "gcloud", block_line)
        for cmd, line_num in commands:
            command_path, flags = parse_gcloud_command(cmd, commands_db)

            if not command_path:
                continue

            is_valid, errors = validate_gcloud_command(
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
                    "cloud": "gcp",
                })

    return pass_count, fail_count
