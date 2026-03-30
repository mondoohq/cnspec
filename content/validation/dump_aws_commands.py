#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps all valid AWS CLI subcommands and their flags for services used in
# the AWS security policy. Reads directly from botocore service models bundled
# with the AWS CLI for speed.
#
# Output is a JSON file mapping:
#   { "service": ["subcommand1", ...],
#     "service subcommand": ["--flag1", "--flag2", ...], ... }
#
# Usage: python3 dump_aws_commands.py [--output aws_commands.json]

import argparse
import json
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "aws_commands.json"

# Services referenced in the AWS security policy.
# The CLI exposes some services under aliases (e.g. `aws configservice` maps to
# the botocore "config" service model, `aws s3api` maps to "s3"). We load using
# the botocore name but store results under the CLI name so validation matches.
SERVICES = [
    "accessanalyzer",
    "acm",
    "apigateway",
    "appstream",
    "athena",
    "autoscaling",
    "backup",
    "cloudfront",
    "cloudtrail",
    "cloudwatch",
    "codebuild",
    "cognito-identity",
    "cognito-idp",
    "config",
    "dax",
    "dms",
    "docdb",
    "drs",
    "ds",
    "dynamodb",
    "ec2",
    "ecr",
    "ecs",
    "efs",
    "eks",
    "elasticache",
    "elasticbeanstalk",
    "elbv2",
    "emr",
    "es",
    "firehose",
    "fsx",
    "glue",
    "guardduty",
    "iam",
    "inspector2",
    "kafka",
    "kinesis",
    "kms",
    "lambda",
    "logs",
    "memorydb",
    "mq",
    "neptune",
    "opensearch",
    "rds",
    "redshift",
    "route53",
    "route53domains",
    "s3",
    "s3control",
    "sagemaker",
    "secretsmanager",
    "securityhub",
    "sns",
    "sqs",
    "ssm",
    "timestream-influxdb",
    "timestream-write",
    "workspaces",
]

# Map from CLI service name to botocore model name where they differ
CLI_TO_BOTOCORE = {
    "configservice": "config",
    "s3api": "s3",
}

# Map from botocore model name back to CLI service name
BOTOCORE_TO_CLI = {v: k for k, v in CLI_TO_BOTOCORE.items()}

# Global flags available on every AWS CLI command
GLOBAL_FLAGS = [
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

# The script that runs inside the awscli's bundled Python to extract service
# models from botocore. This avoids needing botocore installed system-wide.
_EXTRACT_SCRIPT = r"""
import sys, json

sys.path.insert(0, sys.argv[1])

from awscli.botocore import xform_name
from awscli.botocore.session import Session

# argv[2] is a JSON object mapping botocore model name -> CLI service name
service_map = json.loads(sys.argv[2])
result = {}

session = Session()

for botocore_name, cli_name in service_map.items():
    try:
        model = session.get_service_model(botocore_name)
    except Exception as e:
        print(f"Warning: could not load model for {botocore_name}: {e}", file=sys.stderr)
        continue

    subcommands = []
    for op_name in model.operation_names:
        subcmd = xform_name(op_name).replace("_", "-")
        subcommands.append(subcmd)

        # Get flags from operation input shape
        op = model.operation_model(op_name)
        flags = []
        if op.input_shape:
            for member in op.input_shape.members:
                flag = "--" + xform_name(member).replace("_", "-")
                flags.append(flag)
                # Boolean members also get a --no- variant
                member_shape = op.input_shape.members[member]
                if member_shape.type_name == "boolean":
                    flags.append("--no-" + xform_name(member).replace("_", "-"))
        result[f"{cli_name} {subcmd}"] = sorted(flags)

    result[cli_name] = sorted(subcommands)

print(json.dumps(result))
"""


def find_awscli_site_packages() -> str:
    """Find the site-packages directory for the AWS CLI's bundled Python."""
    # Find via the aws binary's real path
    aws_path = subprocess.run(
        ["which", "aws"], capture_output=True, text=True
    ).stdout.strip()

    # Resolve symlinks to find the actual installation
    real_path = Path(aws_path).resolve()
    # Walk up to find site-packages
    base = real_path.parent.parent  # e.g., /opt/homebrew/Cellar/awscli/2.x.x/libexec
    candidates = list(base.glob("**/site-packages/awscli/botocore/data"))
    if not candidates:
        # Try the libexec path pattern
        candidates = list(
            Path(aws_path).resolve().parent.parent.glob(
                "libexec/**/site-packages/awscli/botocore/data"
            )
        )
    if not candidates:
        print(
            "Error: Could not find awscli botocore data directory. "
            "Is the AWS CLI v2 installed?",
            file=sys.stderr,
        )
        sys.exit(1)

    # site-packages is 3 levels up from awscli/botocore/data
    return str(candidates[0].parent.parent.parent)


def main():
    parser = argparse.ArgumentParser(
        description="Dump AWS CLI commands and flags to JSON"
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"Output JSON file (default: {DEFAULT_OUTPUT})",
    )
    args = parser.parse_args()

    site_packages = find_awscli_site_packages()
    print(f"Using awscli from: {site_packages}", file=sys.stderr)

    # Build mapping: botocore model name -> CLI service name
    # Most are the same, but some differ (e.g. config -> configservice)
    service_map = {}
    for svc in SERVICES:
        cli_name = BOTOCORE_TO_CLI.get(svc, svc)
        service_map[svc] = cli_name

    # Run extraction in a subprocess using the system python3 with awscli's
    # site-packages on the path
    result = subprocess.run(
        [
            sys.executable,
            "-c",
            _EXTRACT_SCRIPT,
            site_packages,
            json.dumps(service_map),
        ],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error extracting commands:\n{result.stderr}", file=sys.stderr)
        sys.exit(1)

    if result.stderr:
        print(result.stderr, end="", file=sys.stderr)

    commands = json.loads(result.stdout)

    # Add global flags to every subcommand entry
    for key, value in commands.items():
        if " " in key:  # subcommand entry (has flags)
            merged = sorted(set(value + GLOBAL_FLAGS))
            commands[key] = merged

    args.output.write_text(json.dumps(commands, indent=2, sort_keys=True) + "\n")

    total_subcommands = sum(1 for k in commands if " " in k)
    total_services = sum(1 for k in commands if " " not in k)
    print(
        f"Wrote {total_subcommands} subcommands across {total_services} services to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
