# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Generic validation for Cobra-based CLIs (kubectl, gh, glab, hcloud).
#
# Like doctl, these are Go binaries with no Python introspection surface.
# Unlike doctl, their --help layouts differ wildly (gh and glab render
# custom uppercase-header templates, kubectl uses "Options:" with
# `--flag=default:` lines), so instead of parsing help text per CLI we
# walk the hidden `__complete` command that Cobra builds into every CLI
# for shell completion:
#
#   <cli> __complete <path...> ""    -> subcommand candidates (name\tdesc)
#   <cli> __complete <path...> "--"  -> flag candidates (--name\tdesc)
#
# The output is machine-readable and identical across all Cobra CLIs.
# Two caveats shape the implementation:
#
#   - Flag completion can be filtered by custom completion logic (hcloud
#     only suggests required flags that haven't been given yet), so the
#     valid flag set is the UNION of flag completions and flag-shaped
#     lines parsed from `<cli> <path> --help`.
#   - Leaf commands may complete positional VALUES (pod names, server
#     names) instead of subcommands when the CLI can reach a live
#     API. The walk subprocesses run with cloud credentials stripped
#     (KUBECONFIG=/dev/null, tokens unset) so value completions come
#     back empty and the walk is deterministic.

import concurrent.futures
import os
import re
import shlex
import subprocess
import sys

from .common import (
    FAILURES,
    SCRIPT_DIR,
    extract_bash_blocks,
    policy_relpath,
    split_commands,
    truncate_cmd,
)

# ---------------------------------------------------------------------------
# CLI registry
# ---------------------------------------------------------------------------
#
# Each entry:
#   cli           — binary name (must be on PATH)
#   policies      — policy files (relative to content/) to scan
#   include_audit — validate ```bash blocks in audit: sections too. On for
#                   all of these: they are new validators with no backlog
#                   of unvalidated audit blocks (kubectl in particular
#                   appears almost exclusively in audit sections — the
#                   kubernetes policies remediate via manifests).
#   install       — install hint printed when the binary is missing

COBRA_CLIS = {
    "kubernetes": {
        "cli": "kubectl",
        "policies": [
            "mondoo-kubernetes-security.mql.yaml",
            "mondoo-kubernetes-best-practices.mql.yaml",
        ],
        "include_audit": True,
        "install": (
            "  macOS (Homebrew): brew install kubectl\n"
            "  Linux:            https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/"
        ),
    },
    "github": {
        "cli": "gh",
        "policies": [
            "mondoo-github-security.mql.yaml",
            "mondoo-github-best-practices.mql.yaml",
        ],
        "include_audit": True,
        "install": (
            "  macOS (Homebrew): brew install gh\n"
            "  Linux:            https://github.com/cli/cli#installation"
        ),
    },
    "gitlab": {
        "cli": "glab",
        "policies": ["mondoo-gitlab-security.mql.yaml"],
        "include_audit": True,
        "install": (
            "  macOS (Homebrew): brew install glab\n"
            "  Linux:            https://gitlab.com/gitlab-org/cli#installation"
        ),
    },
    "hetzner": {
        "cli": "hcloud",
        "policies": ["mondoo-hetzner-security.mql.yaml"],
        "include_audit": True,
        "install": (
            "  macOS (Homebrew): brew install hcloud\n"
            "  Linux:            https://github.com/hetznercloud/cli/releases"
        ),
    },
}


def _walk_env() -> dict:
    """Subprocess environment with cloud/API credentials stripped, so
    completion callbacks can't reach a live backend and positional-value
    completions (pod names, server names) come back empty."""
    env = dict(os.environ)
    env["KUBECONFIG"] = "/dev/null"
    for var in (
        "GH_TOKEN", "GITHUB_TOKEN",
        "GITLAB_TOKEN", "GLAB_TOKEN",
        "HCLOUD_TOKEN",
    ):
        env.pop(var, None)
    return env


# A completion candidate that is a plausible subcommand name. Cobra
# candidates are "name\tdescription"; real subcommands are short
# lowercase identifiers, which also filters out any stray value
# completions a CLI might produce without credentials.
_SUBCOMMAND_CANDIDATE_RE = re.compile(r"^[a-z][a-z0-9_-]*$")

# A flag-definition line in --help output:
#   "      --description string        Description of the rule"
#   "  -A, --all-namespaces=false:"    (kubectl)
_HELP_FLAG_LINE_RE = re.compile(r"^\s+(?:-\w,\s+)?(--[a-z][a-z0-9-]*)")


def _run_cli(args: list[str], env: dict) -> str:
    r = subprocess.run(args, capture_output=True, text=True, timeout=20, env=env)
    return r.stdout


def _complete(
    cli: str, path: str, to_complete: str, env: dict, described_only: bool = False
) -> list[str]:
    """Return completion candidate names for `<cli> __complete <path...>
    <to_complete>`. The trailing `:<n>` directive line is dropped, as is
    each candidate's tab-separated description.

    With described_only=True, candidates without a tab-separated
    description are dropped: real subcommands always carry their Short
    description, while static positional-value completions (kubectl's
    `rollout restart` completes resource types like `deployment`) are
    bare names. This is what keeps value completions from masquerading
    as subcommands and turning a leaf into a phantom command group.
    """
    args = [cli, "__complete"] + (path.split() if path else []) + [to_complete]
    out = _run_cli(args, env)
    names = []
    for line in out.split("\n"):
        if not line or line.startswith(":"):
            continue
        if described_only and ("\t" not in line or not line.split("\t", 1)[1].strip()):
            continue
        names.append(line.split("\t", 1)[0])
    return names


def _node_info(cli: str, path: str, env: dict) -> dict:
    """Discover one command node: its subcommands and valid flags."""
    subcommands = [
        c for c in _complete(cli, path, "", env, described_only=True)
        if _SUBCOMMAND_CANDIDATE_RE.match(c)
    ]
    flags = {f for f in _complete(cli, path, "--", env) if f.startswith("--")}
    help_text = _run_cli([cli] + (path.split() if path else []) + ["--help"], env)
    for line in help_text.split("\n"):
        m = _HELP_FLAG_LINE_RE.match(line)
        if m:
            flags.add(m.group(1))
    return {"subcommands": sorted(set(subcommands)), "flags": sorted(flags)}


def detect_cli_services(cli: str, policy_files: list, include_audit: bool) -> list[str]:
    """Top-level <cli> subcommands used across the given policies.

    Global flags may precede the subcommand (`kubectl -n kube-system get
    pod`), and whether a leading flag consumes the next token as its
    value isn't knowable without the CLI's flag table — so collect every
    plausible candidate. Bogus candidates (flag values like
    `kube-system`) introspect to empty nodes, which the walk drops.
    """
    services = set()
    for policy_file in policy_files:
        if not policy_file.exists():
            continue
        content = policy_file.read_text()
        for block, line, _ in extract_bash_blocks(content, include_audit=include_audit):
            for cmd, _ in split_commands(block, cli, line):
                parts = cmd.split()[1:]
                for token in parts[:4]:
                    if token.startswith("-"):
                        continue
                    if _SUBCOMMAND_CANDIDATE_RE.match(token):
                        services.add(token)
                    break
                # Also take the token after a possible `-f value` pair.
                if (
                    len(parts) >= 3
                    and parts[0].startswith("-")
                    and "=" not in parts[0]
                    and not parts[1].startswith("-")
                    and _SUBCOMMAND_CANDIDATE_RE.match(parts[2] or "-")
                ):
                    services.add(parts[2])
    return sorted(services)


def build_cobra_commands_db(key: str, extra_services: list[str] | None = None) -> dict[str, dict]:
    """Walk a Cobra CLI's command tree breadth-first via __complete,
    scoped to the top-level subcommands the policies actually use
    (plus extra_services, used by tests to widen the walk).

    Returns {} when the policies contain no commands for this CLI. Exits
    with an install hint when the binary is missing.
    """
    entry = COBRA_CLIS[key]
    cli = entry["cli"]
    policy_files = [SCRIPT_DIR / ".." / p for p in entry["policies"]]

    services = detect_cli_services(cli, policy_files, entry["include_audit"])
    services = sorted(set(services) | set(extra_services or []))
    if not services:
        return {}

    cli_path = subprocess.run(
        ["which", cli], capture_output=True, text=True
    ).stdout.strip()
    if not cli_path:
        print(
            f"Error: {cli} CLI not found in PATH.\n"
            "\n"
            f"Validating {cli} commands requires the {cli} CLI to be installed\n"
            "locally — its Cobra completion tree is the source of truth for\n"
            "valid commands and flags.\n"
            "\n"
            "Install options:\n"
            f"{entry['install']}\n"
            "\n"
            f"After installing, ensure `{cli}` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} {key}",
            file=sys.stderr,
        )
        sys.exit(1)

    env = _walk_env()

    # Breadth-first walk, parallelizing the subprocess calls in each
    # round (same scheme as the doctl walker).
    visited: set[str] = set()
    db: dict[str, dict] = {}
    queue = list(services)

    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
        while queue:
            future_to_path = {}
            for path in queue:
                if path in visited:
                    continue
                visited.add(path)
                future_to_path[pool.submit(_node_info, cli, path, env)] = path
            next_queue: list[str] = []
            for fut, path in future_to_path.items():
                try:
                    node = fut.result()
                except subprocess.TimeoutExpired:
                    print(
                        f"Warning: timed out introspecting `{cli} {path}`",
                        file=sys.stderr,
                    )
                    continue
                # A path that introspects to nothing — no subcommands and
                # no flags (a real leaf always has at least --help) — is
                # not a command. This drops bogus service candidates from
                # detect_cli_services (e.g. a leading flag's value) and
                # keeps misspelled commands out of the db so validation
                # reports them.
                if not node["subcommands"] and not node["flags"]:
                    continue
                db[path] = node
                for s in node["subcommands"]:
                    sub_path = f"{path} {s}".strip()
                    if sub_path not in visited:
                        next_queue.append(sub_path)
            queue = next_queue

    return db


def parse_cobra_command(
    cli: str, cmd: str, db: dict[str, dict]
) -> tuple[str, str | None, list[str]]:
    """Parse a CLI command into (command_path, next_token, flags).

    Matches the longest known command path from the database; tokens
    after it (resource types, names, file args) are positionals and not
    validated. next_token is the first positional after the matched
    path, which the caller reports as a misspelled subcommand when the
    path lands on a command group. Returns the raw pre-flag tokens as
    the path when nothing matches, so the caller reports it as unknown.
    """
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != cli:
        return "", None, []

    parts = tokens[1:]

    # Global flags may precede the subcommand (`kubectl -n kube-system
    # get pod`). Start path matching at the first token that is a known
    # top-level command; flag values like `kube-system` won't match.
    start = 0
    for i, token in enumerate(parts):
        if not token.startswith("-") and token in db:
            start = i
            break

    command_path = ""
    next_token = None
    raw_path_parts = []
    for i in range(start, len(parts)):
        if parts[i].startswith("-"):
            break
        raw_path_parts.append(parts[i])
        candidate = " ".join(parts[start: i + 1])
        if candidate in db:
            command_path = candidate
            depth = i + 1
            next_token = (
                parts[depth]
                if depth < len(parts) and not parts[depth].startswith("-")
                else None
            )

    if not command_path and raw_path_parts:
        command_path = " ".join(raw_path_parts)

    flags = []
    for token in parts:
        if token.startswith("--"):
            flags.append(token.split("=")[0])

    return command_path, next_token, flags


def validate_cobra_command(
    cli: str,
    command_path: str,
    raw_next_token: str | None,
    flags: list[str],
    db: dict[str, dict],
) -> tuple[bool, list[str]]:
    """Validate a parsed command against the walked command tree."""
    errors = []

    if command_path not in db:
        errors.append(f"unknown command '{cli} {command_path}'")
        return False, errors

    node = db[command_path]

    # A node with subcommands needs one; the next positional token (if
    # any) was already rejected by the longest-path match, so it's a
    # misspelled subcommand rather than a positional argument.
    if node["subcommands"]:
        if raw_next_token is not None:
            errors.append(
                f"unknown subcommand '{raw_next_token}' for '{cli} {command_path}'"
            )
        else:
            errors.append(
                f"'{cli} {command_path}' is a command group; missing subcommand"
            )
        return False, errors

    valid_flags = set(node["flags"])
    for flag in flags:
        if flag not in valid_flags:
            errors.append(f"unknown flag '{flag}' for '{cli} {command_path}'")

    return len(errors) == 0, errors


def validate_cobra_cli(key: str) -> tuple[int, int]:
    """Validate one registered Cobra CLI. Returns (pass, fail)."""
    entry = COBRA_CLIS[key]
    cli = entry["cli"]

    db = build_cobra_commands_db(key)
    if not db:
        return 0, 0

    pass_count = 0
    fail_count = 0

    for policy_name in entry["policies"]:
        policy_file = SCRIPT_DIR / ".." / policy_name
        if not policy_file.exists():
            print(f"Error: Policy file not found: {policy_file}", file=sys.stderr)
            sys.exit(1)

        content = policy_file.read_text()
        blocks = extract_bash_blocks(content, include_audit=entry["include_audit"])
        relpath = policy_relpath(policy_file)

        for block_text, block_line, uid in blocks:
            commands = split_commands(block_text, cli, block_line)
            for cmd, line_num in commands:
                command_path, next_token, flags = parse_cobra_command(cli, cmd, db)
                if not command_path:
                    continue

                is_valid, errors = validate_cobra_command(
                    cli, command_path, next_token, flags, db
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
                        "cloud": key,
                    })

    return pass_count, fail_count
