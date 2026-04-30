#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
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
#   python3 validate_remediation_commands.py gcp         # validate gcloud commands only

import concurrent.futures
import json
import re
import shlex
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CMD_DATA_DIR = SCRIPT_DIR / "cmd_data"

VALIDATORS = ["aws", "azure", "oci", "gcp", "digitalocean"]

# Collected failures for annotation output.  Each entry is a dict with keys:
# file, line, uid, command, errors, cloud
FAILURES: list[dict] = []


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
#
# The AWS commands database is built in-memory at validation time by
# introspecting the botocore service models bundled with AWS CLI v2. This
# avoids shipping a large, frequently-churning aws_commands.json in the repo.
# Requires the AWS CLI v2 to be installed in the environment running validation.

AWS_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-aws-security.mql.yaml"

# CLI-name → botocore-model-name aliases. Most CLI services match their
# botocore names; these differ and need an explicit mapping.
AWS_CLI_TO_BOTOCORE = {
    "configservice": "config",
    "s3api": "s3",
}

# Global flags available on every AWS CLI command.
AWS_GLOBAL_FLAGS = [
    "--ca-bundle",
    "--cli-auto-prompt",
    "--cli-binary-format",
    "--cli-connect-timeout",
    "--cli-error-format",
    "--cli-input-json",
    "--cli-input-yaml",
    "--cli-read-timeout",
    "--color",
    "--debug",
    "--endpoint-url",
    "--generate-cli-skeleton",
    "--no-cli-auto-prompt",
    "--no-cli-pager",
    "--no-paginate",
    "--no-sign-request",
    "--no-verify-ssl",
    "--output",
    "--profile",
    "--query",
    "--region",
    "--version",
]

_XFORM_RE1 = re.compile(r"([a-z0-9])([A-Z])")
_XFORM_RE2 = re.compile(r"([A-Z])([A-Z][a-z])")


def _xform_name(name: str) -> str:
    """CamelCase → snake_case (mirrors botocore.xform_name)."""
    s = _XFORM_RE1.sub(r"\1_\2", name)
    s = _XFORM_RE2.sub(r"\1_\2", s)
    return s.lower()


def _load_aws_service_ops(service_dir: Path) -> tuple[list[str], dict[str, list[str]]]:
    """Parse `<service>/<api-version>/service-2.json` and return
    (subcommands, {subcommand: flags}) exactly as the AWS CLI exposes them.
    """
    candidates = sorted(service_dir.glob("*/service-2.json"))
    if not candidates:
        return [], {}
    data = json.loads(candidates[-1].read_text())
    shapes = data.get("shapes", {})
    subcommands: list[str] = []
    flag_map: dict[str, list[str]] = {}
    for op_name, op in data.get("operations", {}).items():
        subcmd = _xform_name(op_name).replace("_", "-")
        subcommands.append(subcmd)
        flags: list[str] = []
        input_ref = op.get("input")
        if input_ref:
            input_shape = shapes.get(input_ref.get("shape", ""), {})
            for m_name, m_ref in input_shape.get("members", {}).items():
                kebab = _xform_name(m_name).replace("_", "-")
                flags.append(f"--{kebab}")
                m_shape = shapes.get(m_ref.get("shape", ""), {})
                if m_shape.get("type") == "boolean":
                    flags.append(f"--no-{kebab}")
        flag_map[subcmd] = sorted(set(flags))
    return sorted(subcommands), flag_map


def detect_aws_services_from_policy() -> list[str]:
    """Return the set of top-level aws CLI services used in the AWS policy."""
    if not AWS_POLICY_FILE.exists():
        return []

    content = AWS_POLICY_FILE.read_text()
    services = set()
    for match in re.finditer(r"```bash\s*\n(.*?)```", content, re.DOTALL):
        block = match.group(1)
        joined = re.sub(r"\\\s*\n\s*", " ", block)
        for line in joined.split("\n"):
            line = line.strip()
            if line.startswith("aws "):
                parts = line.split()
                if len(parts) >= 2 and not parts[1].startswith("-"):
                    services.add(parts[1])
    return sorted(services)


def find_aws_botocore_data_dir() -> Path | None:
    """Locate the `botocore/data/` directory bundled with AWS CLI v2.

    Handles both install layouts:
      - Homebrew / pip:      `.../lib/python3.X/site-packages/awscli/botocore/data/`
      - Official installer:  `.../aws-cli/v2/<ver>/dist/awscli/botocore/data/`

    Returns None if AWS CLI v2 is not installed or the data dir is missing.
    """
    aws_path = subprocess.run(
        ["which", "aws"], capture_output=True, text=True
    ).stdout.strip()
    if not aws_path:
        return None

    real_path = Path(aws_path).resolve()
    # The install root is usually 2–3 ancestors above the aws binary.
    search_roots = [real_path.parent.parent, real_path.parent.parent.parent]
    for root in search_roots:
        for pattern in (
            "**/dist/awscli/botocore/data",
            "**/site-packages/awscli/botocore/data",
        ):
            candidates = list(root.glob(pattern))
            if candidates:
                return candidates[0]
    return None


def build_aws_commands_db() -> dict[str, list[str]]:
    """Build an in-memory AWS commands database by parsing the botocore
    service-2.json files bundled with AWS CLI v2, scoped to services used
    in the AWS policy.

    Returns an empty dict if the policy file has no aws commands. Exits with
    a helpful error if the AWS CLI v2 is not installed.
    """
    services = detect_aws_services_from_policy()
    if not services:
        return {}

    data_dir = find_aws_botocore_data_dir()
    if not data_dir:
        print(
            "Error: AWS CLI v2 not found in PATH.\n"
            "\n"
            "Validating aws remediation commands requires the AWS CLI v2 to\n"
            "be installed locally — the botocore service models bundled with\n"
            "the CLI are the source of truth for valid commands and flags.\n"
            "The AWS CLI v1 (pip-installed) is not supported.\n"
            "\n"
            "Install options:\n"
            "  macOS (Homebrew): brew install awscli\n"
            "  macOS (pkg):      https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html#getting-started-install-macos\n"
            "  Linux:            https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html#cliv2-linux-install\n"
            "  Windows:          https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html#cliv2-windows-install\n"
            "\n"
            "After installing, ensure `aws` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} aws\n"
            "\n"
            "To skip aws validation, run a different target instead\n"
            "(e.g. `azure`, `oci`, or `gcp`).",
            file=sys.stderr,
        )
        sys.exit(1)

    commands: dict[str, list[str]] = {}
    for cli_name in services:
        botocore_name = AWS_CLI_TO_BOTOCORE.get(cli_name, cli_name)
        service_dir = data_dir / botocore_name
        if not service_dir.is_dir():
            print(
                f"Warning: AWS service '{botocore_name}' not found under {data_dir}",
                file=sys.stderr,
            )
            continue
        subcommands, flag_map = _load_aws_service_ops(service_dir)
        commands[cli_name] = subcommands
        for subcmd, flags in flag_map.items():
            commands[f"{cli_name} {subcmd}"] = sorted(set(flags + AWS_GLOBAL_FLAGS))
    return commands


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

    commands_db = build_aws_commands_db()
    if not commands_db:
        return 0, 0

    content = AWS_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    policy_relpath = str(AWS_POLICY_FILE.resolve().relative_to(Path.cwd()))

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
                FAILURES.append({
                    "file": policy_relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "aws",
                })

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

    policy_relpath = str(AZURE_POLICY_FILE.resolve().relative_to(Path.cwd()))

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
                    "file": policy_relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "azure",
                })

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# OCI validation
# ---------------------------------------------------------------------------
#
# The OCI commands database is built in-memory at validation time by walking
# the Click command tree from the oci_cli Python package. This avoids shipping
# a large oci_commands.json in the repo. Requires the OCI CLI to be installed
# in the environment running validation.

OCI_POLICY_FILE = SCRIPT_DIR / ".." / "mondoo-oci-security.mql.yaml"

# Global flags available on every OCI CLI command.
OCI_GLOBAL_FLAGS = [
    "--auth",
    "--auth-purpose",
    "--cert-bundle",
    "--cli-auto-prompt",
    "--cli-rc-file",
    "--config-file",
    "--connection-timeout",
    "--debug",
    "--defaults-file",
    "--enable-dual-stack",
    "--enable-propagation",
    "--endpoint",
    "--generate-full-command-json-input",
    "--generate-param-json-input",
    "--help",
    "--max-retries",
    "--no-retry",
    "--opc-client-request-id",
    "--opc-request-id",
    "--output",
    "--profile",
    "--proxy",
    "--query",
    "--raw-output",
    "--read-timeout",
    "--realm-specific-endpoint",
    "--region",
    "--request-id",
]

# Script that runs inside the OCI CLI's bundled Python to extract the command
# tree from the Click-based CLI.
_OCI_EXTRACT_SCRIPT = r"""
import json, sys
sys.path.insert(0, sys.argv[1])

import click
from oci_cli import dynamic_loader
from oci_cli.cli import cli

dynamic_loader.load_all_services()

services = json.loads(sys.argv[2])

def walk(group, prefix=""):
    results = {}
    if isinstance(group, click.Group):
        subcommands = []
        for name, cmd in sorted(group.commands.items()):
            path = f"{prefix} {name}".strip()
            if isinstance(cmd, click.Group):
                subcommands.append(name)
                results.update(walk(cmd, path))
            else:
                subcommands.append(name)
                flags = []
                for param in cmd.params:
                    for opt in param.opts:
                        if opt.startswith("--"):
                            flags.append(opt)
                    if hasattr(param, "secondary_opts"):
                        for opt in param.secondary_opts:
                            if opt.startswith("--"):
                                flags.append(opt)
                results[path] = sorted(set(flags))
        if prefix:
            results[prefix] = sorted(subcommands)
    return results

result = {}
for svc in services:
    if svc in cli.commands:
        result.update(walk(cli.commands[svc], svc))
    else:
        print(f"Warning: service '{svc}' not found in OCI CLI", file=sys.stderr)

print(json.dumps(result))
"""


def detect_oci_services_from_policy() -> list[str]:
    """Return the set of top-level oci services used in the OCI policy."""
    if not OCI_POLICY_FILE.exists():
        return []

    content = OCI_POLICY_FILE.read_text()
    services = set()
    for match in re.finditer(r"```bash\s*\n(.*?)```", content, re.DOTALL):
        block = match.group(1)
        joined = re.sub(r"\\\s*\n\s*", " ", block)
        for line in joined.split("\n"):
            line = line.strip()
            if line.startswith("oci "):
                parts = line.split()
                if len(parts) >= 2 and not parts[1].startswith("-"):
                    services.add(parts[1])
    return sorted(services)


def _find_oci_binary() -> str | None:
    oci_path = subprocess.run(
        ["which", "oci"], capture_output=True, text=True
    ).stdout.strip()
    return oci_path or None


def find_oci_cli_site_packages(oci_path: str) -> str | None:
    """Locate the site-packages directory containing oci_cli."""
    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/site-packages/oci_cli/__init__.py"))
    if not candidates:
        return None
    return str(candidates[0].parent.parent)


def find_oci_python(oci_path: str) -> str | None:
    """Return the Python interpreter used by the OCI CLI."""
    try:
        with open(oci_path) as f:
            first_line = f.readline().strip()
    except OSError:
        first_line = ""

    if first_line.startswith("#!"):
        python_path = first_line[2:].strip()
        if Path(python_path).exists():
            return python_path

    real_path = Path(oci_path).resolve()
    base = real_path.parent.parent
    candidates = list(base.glob("**/bin/python3"))
    if candidates:
        return str(candidates[0])
    return None


def build_oci_commands_db() -> dict[str, list[str]]:
    """Build an in-memory OCI commands database by walking the oci_cli Click
    command tree, scoped to services used in the OCI policy.

    Returns an empty dict if the policy file has no oci commands. Exits with
    a helpful error if the OCI CLI is not installed.
    """
    services = detect_oci_services_from_policy()
    if not services:
        return {}

    oci_path = _find_oci_binary()
    if not oci_path:
        print(
            "Error: oci CLI not found in PATH.\n"
            "\n"
            "Validating oci remediation commands requires the OCI CLI to be\n"
            "installed locally — the oci_cli Python package's Click command\n"
            "tree is the source of truth for valid commands and flags.\n"
            "\n"
            "Install options:\n"
            "  pip:              pip install oci-cli\n"
            "  macOS (Homebrew): brew install oci-cli\n"
            "  Installer script: https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm\n"
            "\n"
            "After installing, ensure `oci` is on your PATH and re-run:\n"
            f"  python3 {sys.argv[0]} oci\n"
            "\n"
            "To skip oci validation, run a different target instead\n"
            "(e.g. `aws`, `azure`, or `gcp`).",
            file=sys.stderr,
        )
        sys.exit(1)

    site_packages = find_oci_cli_site_packages(oci_path)
    python_path = find_oci_python(oci_path)
    if not site_packages or not python_path:
        print(
            f"Error: Found `oci` at '{oci_path}' but could not locate its\n"
            f"bundled Python environment (site-packages: {site_packages}, "
            f"python: {python_path}).\n"
            "\n"
            "This usually indicates a broken or partial CLI install. Try\n"
            "reinstalling the OCI CLI (see install options above).",
            file=sys.stderr,
        )
        sys.exit(1)

    result = subprocess.run(
        [
            python_path,
            "-c",
            _OCI_EXTRACT_SCRIPT,
            site_packages,
            json.dumps(services),
        ],
        capture_output=True,
        text=True,
        timeout=120,
    )

    if result.returncode != 0:
        print(
            f"Error extracting OCI commands from oci_cli:\n{result.stderr}",
            file=sys.stderr,
        )
        sys.exit(1)
    if result.stderr:
        for line in result.stderr.splitlines():
            if line.strip() and "Warning" not in line:
                print(line, file=sys.stderr)

    commands = json.loads(result.stdout)
    for key, value in commands.items():
        if value and isinstance(value[0], str) and value[0].startswith("-"):
            commands[key] = sorted(set(value + OCI_GLOBAL_FLAGS))
    return commands


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

    commands_db = build_oci_commands_db()
    if not commands_db:
        return 0, 0

    content = OCI_POLICY_FILE.read_text()
    blocks = extract_bash_blocks(content)

    pass_count = 0
    fail_count = 0

    policy_relpath = str(OCI_POLICY_FILE.resolve().relative_to(Path.cwd()))

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
                FAILURES.append({
                    "file": policy_relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "oci",
                })

    return pass_count, fail_count


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


def detect_gcloud_policy_commands(commands_db: dict[str, list[str]]) -> set[str]:
    """Return the set of known gcloud command paths used in the GCP policy."""
    if not GCLOUD_POLICY_FILE.exists():
        return set()

    content = GCLOUD_POLICY_FILE.read_text()
    policy_commands = set()
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

    policy_relpath = str(GCLOUD_POLICY_FILE.resolve().relative_to(Path.cwd()))

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
                    "file": policy_relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "gcp",
                })

    return pass_count, fail_count


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
# lowercase identifier, then at least two spaces before the description.
_DOCTL_SUBCOMMAND_LINE = re.compile(r"^  ([a-z][a-z0-9-]*)\s{2,}")

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
            in_cmds = True
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
    visited: set[str] = set()
    results: dict[str, dict] = {}
    queue: list[str] = [s for s in services]

    while queue:
        with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
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

    policy_relpath = str(DO_POLICY_FILE.resolve().relative_to(Path.cwd()))

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
                    "file": policy_relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "digitalocean",
                })

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# GitHub Actions annotations
# ---------------------------------------------------------------------------

def emit_github_annotations() -> None:
    """Print GitHub Actions workflow commands for each failure.

    These produce inline annotations on the PR Files tab, regardless of
    whether the annotated file is part of the PR diff.
    See https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions#setting-an-error-message
    """
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['command']}"
        title = f"{r['cloud'].upper()} CLI validation ({r['uid']})"
        # Workflow command special characters must be encoded
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = title.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A").replace(",", "%2C").replace("::", "%3A%3A")
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = sys.argv[1:]
    github_actions = False
    target = "all"

    # Parse flags
    positional = []
    for arg in args:
        if arg == "--github-actions":
            github_actions = True
        else:
            positional.append(arg)

    if positional:
        target = positional[0]

    if target not in ("all", *VALIDATORS):
        print(
            f"Unknown validator: {target}\n"
            f"Usage: {sys.argv[0]} [{'|'.join(['all'] + VALIDATORS)}] [--github-actions]",
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

    if target in ("all", "gcp"):
        p, f = validate_gcloud()
        total_pass += p
        total_fail += f

    if target in ("all", "digitalocean"):
        p, f = validate_digitalocean()
        total_pass += p
        total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
