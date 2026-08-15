#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Ratchet on how much remediation each validator actually sees.
#
# Every validator finds its work by regex over the policy YAML, and the failure
# mode that costs us most is not a crash — it is a validator that quietly stops
# matching and reports success over an empty set. That has happened more than
# once in this directory's history: a `desc: |-` the pattern did not allow for,
# a policy never added to a `TARGETS` list, an enrichment step scoped narrower
# than the checking step. Every one of them looked like a green build.
#
# So this records how many blocks each validator extracts, per policy, and
# fails when a number goes *down* without the checked-in counts being updated in
# the same change. Growth needs no action; a drop has to be explained.
#
# It deliberately runs no linters and needs nothing installed — it calls each
# validator's own extractor, so it is a fast CI job and it measures the same
# code path the real run uses. Sharing the extractor is the point: a regex that
# breaks breaks here too.
#
# Usage:
#   python3 content/validation/check_validation_coverage.py            # report
#   python3 content/validation/check_validation_coverage.py --check    # CI gate
#   python3 content/validation/check_validation_coverage.py --update   # accept

import argparse
import importlib.util
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CONTENT_DIR = SCRIPT_DIR.parent
BUDGET_FILE = SCRIPT_DIR / "remediation-coverage.json"

sys.path.insert(0, str(SCRIPT_DIR))

from validators.common import extract_bash_blocks, split_commands  # noqa: E402


def load_validator(module_name: str):
    """Import a validate_*.py script as a module without running it."""
    path = SCRIPT_DIR / f"{module_name}.py"
    spec = importlib.util.spec_from_file_location(module_name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


# ---------------------------------------------------------------------------
# Code-block validators: each owns a TARGETS map and an extractor
# ---------------------------------------------------------------------------

BLOCK_VALIDATORS = [
    ("bash", "validate_bash_remediation", "extract_bash_blocks"),
    ("ansible", "validate_ansible_remediation", "extract_ansible_blocks"),
    ("chef", "validate_chef_remediation", "extract_chef_blocks"),
    ("cloudformation", "validate_cloudformation_remediation", "extract_cfn_blocks"),
    ("bicep", "validate_bicep_remediation", "extract_bicep_blocks"),
    ("terraform", "validate_terraform_remediation", "extract_hcl_blocks"),
]


def count_block_validators() -> dict[str, dict[str, int]]:
    counts: dict[str, dict[str, int]] = {}
    for name, module_name, extractor_name in BLOCK_VALIDATORS:
        module = load_validator(module_name)
        extractor = getattr(module, extractor_name)
        per_policy: dict[str, int] = {}
        for paths in module.TARGETS.values():
            for entry in paths:
                path = Path(entry)
                if not path.exists():
                    continue
                blocks = extractor(path.read_text(), path)
                if blocks:
                    per_policy[path.name] = len(blocks)
        counts[name] = dict(sorted(per_policy.items()))
    return counts


# ---------------------------------------------------------------------------
# CLI/API grammar validators
#
# Counting blocks would understate these: a block holds many commands and the
# validator checks the ones that start with its own tool. The command count is
# what actually moves when an extractor or a `TARGETS` entry breaks, so that is
# what is recorded.
# ---------------------------------------------------------------------------

CLI_TARGETS = [
    # (label, policy filename, command prefix, remediation ids)
    ("aws", "mondoo-aws-security.mql.yaml", "aws", ("cli",)),
    ("azure", "mondoo-azure-security.mql.yaml", "az", ("cli",)),
    ("azure-m365", "mondoo-m365-security.mql.yaml", "az", ("cli",)),
    ("gcp", "mondoo-gcp-security.mql.yaml", "gcloud", ("cli",)),
    ("oci", "mondoo-oci-security.mql.yaml", "oci", ("cli",)),
    ("digitalocean", "mondoo-digitalocean-security.mql.yaml", "doctl", ("cli",)),
    ("nutanix", "mondoo-nutanix-security.mql.yaml", "ncli", ("cli",)),
    ("alicloud", "mondoo-alibaba-security.mql.yaml", "aliyun", ("cli",)),
    ("vercel", "mondoo-vercel-security.mql.yaml", "vercel", ("cli", "api")),
    ("kubernetes", "mondoo-kubernetes-security.mql.yaml", "kubectl", ("cli",)),
    ("kubernetes-bp", "mondoo-kubernetes-best-practices.mql.yaml", "kubectl", ("cli",)),
    ("github", "mondoo-github-security.mql.yaml", "gh", ("cli",)),
    ("github-bp", "mondoo-github-best-practices.mql.yaml", "gh", ("cli",)),
    ("gitlab", "mondoo-gitlab-security.mql.yaml", "glab", ("cli",)),
    ("hetzner", "mondoo-hetzner-security.mql.yaml", "hcloud", ("cli",)),
    ("databricks", "mondoo-databricks-security.mql.yaml", "databricks", ("cli",)),
    ("cloudflare", "mondoo-cloudflare-security.mql.yaml", "curl", ("cli",)),
    ("tailscale", "mondoo-tailscale-security.mql.yaml", "curl", ("cli",)),
    ("slack", "mondoo-slack-security.mql.yaml", "curl", ("cli",)),
    ("atlassian", "mondoo-atlassian-security.mql.yaml", "curl", ("cli",)),
    ("grafana", "mondoo-grafana-security.mql.yaml", "curl", ("cli",)),
    ("mongodbatlas", "mondoo-mongodbatlas-security.mql.yaml", "curl", ("api",)),
]


def count_cli_validators() -> dict[str, int]:
    counts: dict[str, int] = {}
    for label, filename, prefix, ids in CLI_TARGETS:
        path = CONTENT_DIR / filename
        if not path.exists():
            continue
        total = 0
        for block, line, _uid in extract_bash_blocks(
            path.read_text(), include_audit=True, remediation_ids=ids
        ):
            total += len(split_commands(block, prefix, line))
        counts[label] = total
    return counts


def collect() -> dict:
    return {
        "blocks": count_block_validators(),
        "commands": dict(sorted(count_cli_validators().items())),
    }


# ---------------------------------------------------------------------------
# Comparison
# ---------------------------------------------------------------------------

def flatten(counts: dict) -> dict[str, int]:
    """One flat name -> count map, so comparison is a single pass."""
    flat: dict[str, int] = {}
    for validator, per_policy in counts["blocks"].items():
        for policy, n in per_policy.items():
            flat[f"blocks/{validator}/{policy}"] = n
    for label, n in counts["commands"].items():
        flat[f"commands/{label}"] = n
    return flat


def compare(recorded: dict, current: dict) -> tuple[list[str], list[str]]:
    """Returns (regressions, growth)."""
    old, new = flatten(recorded), flatten(current)
    regressions, growth = [], []
    for key, was in sorted(old.items()):
        now = new.get(key, 0)
        if now < was:
            regressions.append(
                f"{key}: {was} -> {now}"
                + (" (nothing extracted at all)" if now == 0 else "")
            )
        elif now > was:
            growth.append(f"{key}: {was} -> {now}")
    for key, now in sorted(new.items()):
        if key not in old:
            growth.append(f"{key}: new, {now}")
    return regressions, growth


def totals(counts: dict) -> tuple[int, int]:
    blocks = sum(n for per in counts["blocks"].values() for n in per.values())
    commands = sum(counts["commands"].values())
    return blocks, commands


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Ratchet on how much remediation each validator extracts"
    )
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--check", action="store_true", help="fail if any count dropped")
    group.add_argument("--update", action="store_true", help="record the current counts")
    args = parser.parse_args()

    current = collect()
    blocks, commands = totals(current)

    if args.update:
        BUDGET_FILE.write_text(json.dumps(current, indent=2, sort_keys=True) + "\n")
        print(f"Recorded {blocks} blocks and {commands} commands to {BUDGET_FILE.name}")
        return

    if not BUDGET_FILE.exists():
        print(
            f"Error: {BUDGET_FILE} not found. Create it with --update.",
            file=sys.stderr,
        )
        sys.exit(1)

    recorded = json.loads(BUDGET_FILE.read_text())
    regressions, growth = compare(recorded, current)

    if not args.check:
        for validator, per_policy in current["blocks"].items():
            print(f"{validator:<16} {sum(per_policy.values()):5d} blocks "
                  f"across {len(per_policy)} policies")
        for label, n in current["commands"].items():
            print(f"{label:<16} {n:5d} commands")

    print(f"\n{blocks} blocks, {commands} commands extracted", file=sys.stderr)

    if growth:
        print(f"\n{len(growth)} count(s) grew (fine, no action needed):", file=sys.stderr)
        for line in growth:
            print(f"  {line}", file=sys.stderr)

    if regressions:
        print(
            f"\n{len(regressions)} validator count(s) DROPPED:\n  "
            + "\n  ".join(regressions)
            + "\n\nA validator extracting less than it used to is either content that "
            "was removed\nor — the reason this check exists — an extractor that "
            "silently stopped matching.\nIf the drop is intended, re-run with "
            "--update in the same change so the new\nfloor is reviewable in the diff.",
            file=sys.stderr,
        )
        if args.check:
            sys.exit(1)


if __name__ == "__main__":
    main()
