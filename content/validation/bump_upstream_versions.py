#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Rewrites a validator's pinned upstream to the current release.
#
# `check_upstream_versions.py` says which pins are behind; this applies the
# bump, in the file that declares the pin. For a CLI installed from a release
# artifact it also re-downloads that artifact and recomputes the SHA-256, using
# the download URL read back out of the workflow, so the version and the digest
# can never move apart.
#
# It changes files and stops. It does not decide whether the bump is *good* --
# a new cfn-lint or a new tflint ruleset can legitimately start failing
# snippets that were fine, and a Terraform provider major can rename resources
# out from under a hundred of them. That judgement belongs to whoever reviews
# the pull request the weekly workflow opens, with CI's verdict in front of
# them.
#
# Usage:
#   python3 content/validation/bump_upstream_versions.py --list
#   python3 content/validation/bump_upstream_versions.py --only cfn-lint
#   python3 content/validation/bump_upstream_versions.py --only terraform-provider
#   python3 content/validation/bump_upstream_versions.py --all --dry-run
#   python3 content/validation/bump_upstream_versions.py --only doctl --json -

import argparse
import json
import os
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from upstream_pins import (  # noqa: E402
    REPO_ROOT,
    Pin,
    discover,
    sync_dump_script_pins,
    verify_workflow_checksums,
)


# What a red CI run on this bump most likely means, per kind of pin. The point
# is to save the reviewer the first ten minutes of triage: for most of these,
# failing checks are the *expected* outcome of a real upstream change, not a
# broken bump.
CAVEATS = {
    "linter": (
        "A new linter release can legitimately start failing snippets that passed "
        "before, either from a new rule or a tightened one. If CI is red, the "
        "question is whether the content or the pin is wrong."
    ),
    "cli": (
        "The SHA-256 in the workflow was recomputed from the artifact this bump "
        "points at, downloaded from the same URL CI uses. A red run means the "
        "vendor changed a command or flag that remediation or audit steps rely on."
    ),
    "tflint-ruleset": (
        "Rulesets ship new rules enabled by default, so a red run here is usually "
        "new findings in existing snippets rather than a regression. Rules that do "
        "not apply to documentation examples belong in DISABLED_RULES, with a reason."
    ),
    "terraform-provider": (
        "This moves a `~>` constraint across a **major** version. Providers rename, "
        "retype, and remove resources across majors, so expect the Terraform "
        "validator to surface genuine content work rather than a clean pass. Merging "
        "this and fixing the snippets are two separate decisions."
    ),
    "spec": (
        "The validator now checks curl calls against the API as of a newer commit of "
        "the vendor's spec. A red run means a remediation or audit step names an "
        "endpoint, method, or field the vendor has since changed."
    ),
}


# How to name a pull request that carries every behind pin of one kind. Every
# kind lives in one dense block of one file -- 24 Terraform providers on 24
# consecutive lines of PROVIDER_MAP, four spec SHAs on four consecutive lines
# of openapi.py -- so a pull request per *pin* means every pull request of that
# kind conflicts with every other one the moment the first is merged. Grouping
# by kind is what makes them mergeable in any order.
KIND_LABELS = {
    "linter": "linter pins",
    "cli": "pinned CLI releases",
    "tflint-ruleset": "tflint ruleset pins",
    "terraform-provider": "Terraform provider constraints",
    "spec": "pinned OpenAPI specs",
}


def pr_title(applied: list[dict]) -> str:
    """Pull request title for a group of applied bumps."""
    if len(applied) == 1:
        # A one-pin group still says which pin and which version, which is what
        # the commit history reads like today.
        return f"🧹 validation: bump {applied[0]['name']} to {applied[0]['to']}"

    kinds = dict.fromkeys(a["kind"] for a in applied)
    if len(kinds) == 1:
        kind = next(iter(kinds))
        return f"🧹 validation: bump {len(applied)} {KIND_LABELS.get(kind, f'{kind} pins')}"
    return f"🧹 validation: bump {len(applied)} pinned upstreams"


def pr_branch(applied: list[dict]) -> str:
    """Stable branch for a group of applied bumps.

    Named for the *kind*, never for a version and no longer for a single pin:
    the branch belongs to the group, so next week's run updates the open pull
    request instead of opening a second one beside it.
    """
    kinds = dict.fromkeys(a["kind"] for a in applied)
    if len(kinds) == 1:
        return f"deps/validation/{re.sub(r'[^a-z0-9]+', '-', next(iter(kinds)).lower()).strip('-')}"
    return "deps/validation/upstream-pins"


def pr_body(applied: list[dict]) -> str:
    lines = [
        "Opened by the weekly `Bump validation dependencies` workflow.",
        "",
        "These pins decide what the remediation and IaC-variant validators check "
        "against. Nothing else watches them: Dependabot covers `gomod` and "
        "`github-actions` only.",
        "",
        "| Pinned artifact | Kind | From | To |",
        "| --- | --- | --- | --- |",
    ]
    for a in applied:
        lines.append(f"| {a['name']} | {a['kind']} | `{a['from']}` | `{a['to']}` |")

    for kind in dict.fromkeys(a["kind"] for a in applied):
        if kind in CAVEATS:
            lines += ["", f"**{kind}** — {CAVEATS[kind]}"]

    lines += [
        "",
        "A bump is a judgement call, which is why this is a pull request and not a "
        "commit. Close it if a pin should stay where it is; the workflow will "
        "reopen the same branch when upstream moves again.",
    ]

    if len(applied) > 1:
        lines += [
            "",
            "Every pin above is declared in the same block of the same file, so one "
            "pull request per pin would mean every one of them conflicting with the "
            "rest the moment the first was merged. They are grouped instead. If one "
            "row needs work the others should not wait on, drop that line from this "
            "branch and merge the rest -- next week's run will offer it again.",
        ]

    return "\n".join(lines)


def select(pins: list[Pin], only: str | None) -> list[Pin]:
    if only is None:
        return [p for p in pins if p.state == "behind"]

    picked: set[int] = set()
    # Each term names one pin or a whole kind, and several may be given at once.
    # A kind is what the weekly workflow passes -- one job per kind, so the
    # pins that share a block of lines also share a pull request.
    for term in (t.strip() for t in only.split(",")):
        if not term:
            continue
        match = [p for p in pins if term in (p.name, p.slug) or term == p.kind]
        if not match:
            known = ", ".join(sorted({p.slug for p in pins} | {p.kind for p in pins}))
            raise SystemExit(f"no pin or kind named {term!r}. Known: {known}")
        picked.update(id(p) for p in match)

    # An explicitly named pin that is already current is not an error: the
    # weekly workflow builds its matrix from one run of the checker and applies
    # each entry in a later job, and upstream can publish in between. Filtering
    # over `pins` rather than the matches keeps discovery order and drops the
    # duplicates that overlapping terms produce.
    return [p for p in pins if id(p) in picked and p.state == "behind"]


def main() -> None:
    parser = argparse.ArgumentParser(description="Bump a pinned validator upstream in place")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        "--only",
        metavar="PIN",
        help="bump these pins: a comma-separated list of pin names, slugs, or kinds",
    )
    group.add_argument("--all", action="store_true", help="bump every pin that is behind")
    group.add_argument("--list", action="store_true", help="list bumpable pins and exit")
    group.add_argument(
        "--verify-checksums",
        action="store_true",
        help="re-derive each checksummed CLI's digest at its current pin and compare",
    )
    group.add_argument(
        "--sync-dump-pins",
        action="store_true",
        help="realign a dump script's version constant with the grammar it emitted",
    )
    parser.add_argument("--dry-run", action="store_true", help="report without writing")
    parser.add_argument(
        "--json",
        metavar="PATH",
        help="write a manifest of applied bumps ('-' for stdout)",
    )
    parser.add_argument(
        "--pr-body",
        metavar="PATH",
        help="write a pull request body describing the applied bumps",
    )
    parser.add_argument(
        "--github-output",
        action="store_true",
        help="append title/branch/applied to $GITHUB_OUTPUT for the calling workflow",
    )
    args = parser.parse_args()

    if args.sync_dump_pins:
        for line in sync_dump_script_pins() or ["dump script pins already in sync"]:
            print(line)
        return

    if args.verify_checksums:
        results = verify_workflow_checksums()
        for name, ok, detail in results:
            print(f"[{'PASS' if ok else 'FAIL'}] {name:<12} {detail}")
        if any(not ok for _, ok, _ in results):
            raise SystemExit(1)
        return

    pins = discover()

    if args.list:
        for p in pins:
            flag = "auto" if p.automatable else "manual"
            print(f"{p.slug:<32} {p.kind:<20} {p.state:<10} {flag}")
        return

    targets = select(pins, None if args.all else args.only)

    applied, skipped = [], []
    for pin in targets:
        if not pin.automatable:
            # Grammars are the usual case: the pin records which tool produced a
            # checked-in JSON, so moving it means re-running the dump script
            # against that tool, not editing a string.
            skipped.append({"name": pin.name, "slug": pin.slug, "reason": pin.note or "no mechanical bump"})
            print(f"skip {pin.name}: needs a manual refresh ({pin.note})", file=sys.stderr)
            continue

        print(f"bump {pin.name}: {pin.pinned} -> {pin.latest}")
        if args.dry_run:
            continue

        try:
            changed = pin.apply(pin.latest)
        except ValueError as err:
            # A pattern that stopped matching or an artifact that would not
            # download must fail loudly. Half-writing a version without its
            # checksum would hand CI a file that fails `sha256sum -c` with no
            # hint of why.
            raise SystemExit(f"failed to bump {pin.name}: {err}") from err

        applied.append({
            "name": pin.name,
            "slug": pin.slug,
            "kind": pin.kind,
            "from": pin.pinned,
            "to": pin.latest,
            "files": [str(f.relative_to(REPO_ROOT)) for f in changed],
        })

    if args.json:
        manifest = json.dumps({"applied": applied, "skipped": skipped}, indent=2)
        if args.json == "-":
            print(manifest)
        else:
            Path(args.json).write_text(manifest + "\n")

    if args.pr_body and applied:
        Path(args.pr_body).write_text(pr_body(applied) + "\n")

    if args.github_output and applied:
        # The title interpolates an upstream-supplied version, which reaches
        # $GITHUB_OUTPUT and from there the pull request title and commit
        # message. `as_token` already refuses anything but a single version-
        # shaped token, so a newline cannot get this far -- the heredoc form is
        # the second layer, and the one that stays correct if a future resolver
        # returns something richer. The branch needs neither: it is generated
        # from a `[^a-z0-9]+` substitution.
        with open(os.environ["GITHUB_OUTPUT"], "a") as fh:
            fh.write(f"applied={len(applied)}\n")
            fh.write(f"title<<PR_TITLE_EOF\n{pr_title(applied)}\nPR_TITLE_EOF\n")
            fh.write(f"branch={pr_branch(applied)}\n")

    if not applied and not args.dry_run:
        print("nothing to bump", file=sys.stderr)


if __name__ == "__main__":
    main()
