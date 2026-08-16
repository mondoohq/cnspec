#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Reports which of the validators' pinned upstreams have moved.
#
# The validators are only as current as the things they check against: a linter
# release, a vendor CLI grammar, a tflint ruleset, a Terraform provider, an
# OpenAPI spec. Every one of those is pinned somewhere in the repo and none of
# them are watched by Dependabot, which covers gomod and github-actions only.
#
# The pins themselves, and the code that reads them out of the files that
# declare them, live in `upstream/pins.py`. This script only renders them; its
# sibling `upstream/bump.py` rewrites them.
#
# It is a report, not a gate: a bump is a judgement call, and a GitHub API blip
# must not fail a build.
#
# Usage:
#   python3 content/validation/upstream/check.py
#   python3 content/validation/upstream/check.py --format markdown
#   python3 content/validation/upstream/check.py --format json
#   python3 content/validation/upstream/check.py --exit-code   # 1 if behind

import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))  # content/validation/upstream
from pins import Pin, discover  # noqa: E402

STATE_MARK = {
    "behind": "BEHIND",
    "current": "ok",
    "unstamped": "NO VERSION RECORDED",
    "unchecked": "could not reach upstream",
    "manual": "no machine-readable upstream, review by hand",
}


def render_text(pins: list[Pin]) -> str:
    # `default=0` rather than assuming a non-empty list: every discoverer
    # returns nothing when the file it reads is absent, so a repo layout change
    # would turn this into a ValueError instead of an empty report.
    width = max((len(p.name) for p in pins), default=0)
    return "\n".join(
        f"{p.name:<{width}}  pinned={p.pinned:<12} upstream={p.latest:<12} {STATE_MARK[p.state]}"
        for p in pins
    )


def render_markdown(pins: list[Pin]) -> str:
    lines = [
        "| Pinned artifact | Kind | Pinned at | Upstream | State |",
        "| --- | --- | --- | --- | --- |",
    ]
    for p in pins:
        # A pin that is behind but has no mechanical bump needs a person, so
        # say so here rather than leaving the reader to wonder why no PR
        # showed up for it.
        mark = STATE_MARK[p.state]
        if p.state == "behind" and not p.automatable:
            mark += " (manual bump)"
        lines.append(f"| {p.name} | {p.kind} | `{p.pinned}` | `{p.latest}` | {mark} |")
    return "\n".join(lines)


def render_json(pins: list[Pin]) -> str:
    """Machine-readable form, consumed by the weekly auto-bump workflow to
    build its matrix of pins to open PRs for."""
    return json.dumps(
        [
            {
                "name": p.name,
                "slug": p.slug,
                "kind": p.kind,
                "pinned": p.pinned,
                "latest": p.latest,
                "state": p.state,
                "automatable": p.automatable,
                "note": p.note,
            }
            for p in pins
        ],
        indent=2,
    )


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Report which validator upstreams have moved past their pin"
    )
    parser.add_argument("--format", choices=["text", "markdown", "json"], default="text")
    parser.add_argument(
        "--exit-code",
        action="store_true",
        help="exit 1 when any pin is behind (default: always exit 0, this is a report)",
    )
    args = parser.parse_args()

    pins = discover()

    if args.format == "json":
        print(render_json(pins))
    elif args.format == "markdown":
        print(render_markdown(pins))
    else:
        print(render_text(pins))

    behind = [p for p in pins if p.state == "behind"]
    unstamped = [p for p in pins if p.state == "unstamped"]
    print(
        f"\n{len(behind)} behind, {len(unstamped)} unstamped, {len(pins)} checked",
        file=sys.stderr,
    )
    if args.exit_code and behind:
        sys.exit(1)


if __name__ == "__main__":
    main()
