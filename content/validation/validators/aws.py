# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# AWS CLI validation.

import json
import re
import shlex
import subprocess
import sys

from pathlib import Path

from .common import FAILURES, SCRIPT_DIR, extract_bash_blocks, policy_relpath, split_commands, truncate_cmd


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


# The AWS CLI synthesizes these subcommands and flags at runtime; they are
# real CLI surface but never appear in the botocore service-2.json model the
# database is built from.
AWS_CLI_CUSTOM_SUBCOMMANDS = {
    # Every service with a waiters-2.json model gets an `aws <service> wait`
    # subcommand whose own arguments are waiter names, not model operations.
    "wait",
}

# Service-scoped subcommands the AWS CLI synthesizes on top of the botocore
# model. They are real CLI surface but have no service-2.json operation, so the
# generated database never contains them. Each maps to the flags the CLI
# exposes for that customization.
AWS_CLI_CUSTOM_SERVICE_SUBCOMMANDS = {
    "emr": {
        # `modify-cluster-attributes` wraps SetVisibleToAllUsers /
        # SetTerminationProtection / etc.; it replaces the removed
        # set-visible-to-all-users command in CLI v2.
        "modify-cluster-attributes": [
            "--cluster-id",
            "--visible-to-all-users",
            "--no-visible-to-all-users",
            "--termination-protected",
            "--no-termination-protected",
            "--unhealthy-node-replacement",
            "--no-unhealthy-node-replacement",
            "--auto-terminate",
            "--no-auto-terminate",
        ],
    },
}

AWS_CLI_CUSTOM_FLAGS = {
    # CLI customization that writes the seed material to a local file.
    "iam create-virtual-mfa-device": ["--outfile", "--bootstrap-method"],
    # Members typed as AttributeBooleanValue structures get boolean-style
    # --flag/--no-flag forms in the CLI instead of a value argument.
    "ec2 modify-subnet-attribute": [
        "--map-public-ip-on-launch",
        "--no-map-public-ip-on-launch",
        "--assign-ipv6-address-on-creation",
        "--no-assign-ipv6-address-on-creation",
    ],
}


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

    if subcommand in AWS_CLI_CUSTOM_SUBCOMMANDS:
        return True, []

    custom_subcommands = AWS_CLI_CUSTOM_SERVICE_SUBCOMMANDS.get(service, {})
    if subcommand in custom_subcommands:
        valid_flags = set(custom_subcommands[subcommand])
        for flag in flags:
            if flag not in valid_flags:
                errors.append(f"unknown flag '{flag}' for '{service} {subcommand}'")
        return len(errors) == 0, errors

    valid_subcommands = commands_db[service]
    if subcommand not in valid_subcommands:
        errors.append(f"unknown subcommand '{service} {subcommand}'")
        return False, errors

    key = f"{service} {subcommand}"
    if key in commands_db:
        valid_flags = set(commands_db[key])
        valid_flags.update(AWS_CLI_CUSTOM_FLAGS.get(key, []))
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

    relpath = policy_relpath(AWS_POLICY_FILE)

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
                    "file": relpath,
                    "line": line_num,
                    "uid": uid,
                    "command": truncate_cmd(cmd),
                    "errors": errors,
                    "cloud": "aws",
                })

    return pass_count, fail_count
