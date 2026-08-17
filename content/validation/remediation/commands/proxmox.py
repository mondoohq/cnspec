# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# pvesh (Proxmox VE) CLI validation.

import json
import shlex
import sys

from common import (
    FAILURES,
    CONTENT_DIR,
    DATA_DIR,
    extract_bash_blocks,
    policy_relpath,
    split_commands,
    truncate_cmd,
)

# ---------------------------------------------------------------------------
# pvesh (Proxmox VE) validation
# ---------------------------------------------------------------------------
#
# Proxmox VE defines its entire API in JSON Schema and generates the API
# viewer, the REST endpoints and the CLI tools from it. `pvesh` is the thinnest
# of those: it takes an API path directly and turns the remaining options into
# request parameters.
#
#   pvesh set /cluster/firewall/options --enable 1
#   pvesh create /cluster/firewall/rules --type in --action DROP
#
# So the schema validates a pvesh invocation exactly: the path must exist, the
# verb must map to a method the path serves, and every option must be a
# parameter of that method. The grammar is checked in at data/proxmox_api.json
# by upstream/dump/proxmox.py, which means this validator needs no Proxmox host
# and no network — unlike every other CLI here, there is nothing to install.
#
# SCOPE: pvesh only. The policy also uses pct, qm, pveum, pvenode, pvesm and
# pvecm, and those are generated from this same schema — but by Perl code that
# maps each subcommand to an API path, and that mapping is not published in any
# machine-readable form. pve-docs generates the man pages from it at build
# time and does not check the generated synopsis files in. Validating those
# would mean hand-writing the subcommand-to-path map, which is inventing the
# oracle rather than reading one, so they are deliberately left unchecked
# rather than checked against something made up here.

PROXMOX_POLICY_FILE = CONTENT_DIR / "mondoo-proxmox-security.mql.yaml"
PROXMOX_API_FILE = DATA_DIR / "proxmox_api.json"

# pvesh's verbs and the HTTP methods they issue. `ls` and `usage` are viewer
# commands: `ls` lists child nodes and `usage` prints the schema, so both are
# satisfied by the path existing and take no API parameters.
PVESH_METHODS = {
    "get": "GET",
    "set": "PUT",
    "create": "POST",
    "delete": "DELETE",
    "ls": None,
    "usage": None,
}

# pvesh's own output options, which it consumes rather than sending to the API.
PVESH_OPTIONS = {
    "output-format", "human-readable", "noborder", "noheader",
    "quiet", "verbose", "help",
}


def load_proxmox_api() -> dict:
    """Load the checked-in Proxmox API grammar."""
    if not PROXMOX_API_FILE.exists():
        print(
            f"Error: {PROXMOX_API_FILE} not found.\n"
            "Regenerate it with:\n"
            "  python3 content/validation/upstream/dump/proxmox.py",
            file=sys.stderr,
        )
        sys.exit(1)
    return json.loads(PROXMOX_API_FILE.read_text())["paths"]


def resolve_api_path(path: str, api: dict) -> str | None:
    """Match a written path against the schema, honouring `{param}` segments.

    A remediation writes the concrete path a reader would type
    (`/nodes/$node/firewall/options`), while the schema declares the template
    (`/nodes/{node}/firewall/options`). A template segment matches any single
    written segment; everything else must match literally, so a wrong segment
    at any depth is still caught.
    """
    written = path.strip("/").split("/")
    for candidate in api:
        segments = candidate.strip("/").split("/")
        if len(segments) != len(written):
            continue
        if all(s.startswith("{") or s == w for s, w in zip(segments, written)):
            return candidate
    return None


def parse_pvesh_command(cmd: str) -> tuple[str, str, list[str]]:
    """Parse a pvesh invocation into (verb, api path, option names)."""
    try:
        tokens = shlex.split(cmd)
    except ValueError:
        tokens = cmd.split()

    if len(tokens) < 2 or tokens[0] != "pvesh":
        return "", "", []

    verb = tokens[1]
    path = ""
    options = []
    for token in tokens[2:]:
        if token.startswith("-"):
            options.append(token.lstrip("-").split("=")[0])
        elif not path:
            path = token

    return verb, path, options


def validate_pvesh_command(
    verb: str, path: str, options: list[str], api: dict
) -> tuple[bool, list[str]]:
    """Validate a parsed pvesh invocation against the API schema."""
    errors = []

    if verb not in PVESH_METHODS:
        errors.append(
            f"'{verb}' is not a pvesh verb (expected one of "
            f"{', '.join(sorted(PVESH_METHODS))})"
        )
        return False, errors

    if not path:
        errors.append(f"'pvesh {verb}' names no API path")
        return False, errors

    resolved = resolve_api_path(path, api)
    if resolved is None:
        errors.append(f"no such Proxmox API path '{path}'")
        return False, errors

    method = PVESH_METHODS[verb]
    if method is None:
        return True, []

    methods = api[resolved]
    if method not in methods:
        errors.append(
            f"'{resolved}' does not serve {method}; it serves "
            f"{', '.join(sorted(methods))}"
        )
        return False, errors

    valid = set(methods[method]["params"])
    for option in options:
        if option in PVESH_OPTIONS:
            continue
        if option not in valid:
            errors.append(f"'{option}' is not a parameter of {method} {resolved}")

    return len(errors) == 0, errors


def validate_proxmox() -> tuple[int, int]:
    """Validate pvesh commands. Returns (pass_count, fail_count)."""
    if not PROXMOX_POLICY_FILE.exists():
        print(f"Error: Policy file not found: {PROXMOX_POLICY_FILE}", file=sys.stderr)
        sys.exit(1)

    content = PROXMOX_POLICY_FILE.read_text()
    # This policy documents each fix twice: `- id: cli` for the single command
    # and `- id: bash` for the same fix wrapped in a script that loops over the
    # nodes. Reading only `cli` would leave 12 of the 29 pvesh invocations
    # unchecked — shellcheck lints those blocks, but nothing there knows
    # whether the command exists.
    blocks = extract_bash_blocks(
        content, include_audit=True, remediation_ids=("cli", "bash")
    )
    if not blocks:
        return 0, 0

    api = load_proxmox_api()
    relpath = policy_relpath(PROXMOX_POLICY_FILE)

    pass_count = fail_count = 0
    for block_text, block_line, uid in blocks:
        for cmd, line_num in split_commands(block_text, "pvesh", block_line):
            verb, path, options = parse_pvesh_command(cmd)
            if not verb:
                continue

            is_valid, errors = validate_pvesh_command(verb, path, options, api)

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
                "cloud": "proxmox",
            })

    return pass_count, fail_count
