#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates the CloudFormation snippets in `- id: cloudformation` remediation
# blocks with cfn-lint, which checks resource types and property names against
# the AWS resource specification.
#
# Mirrors validate_terraform_remediation.py: extract the fenced blocks from one
# `- id:`, make each snippet stand on its own, and run the ecosystem's own
# linter over it.
#
# Usage: python3 validate_cloudformation_remediation.py [aws] [--github-actions]

import argparse
import concurrent.futures
import json
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
REPO_ROOT = SCRIPT_DIR.parent.parent

TARGETS = {
    "aws": [SCRIPT_DIR / ".." / "mondoo-aws-security.mql.yaml"],
}

# A remediation snippet opens at `Resources:` and demonstrates the one setting
# its check is about. cfn-lint needs a whole template, so the missing preamble
# is supplied here.
TEMPLATE_HEADER = "AWSTemplateFormatVersion: '2010-09-09'\n"

# Rules that only fire because a documentation snippet is a fragment.
#
# E1010 resolves `!GetAtt Target.Attr` against the template's own resources. A
# snippet that wires up, say, a KMS key it does not declare will always fail
# it, and unlike `!Ref` there is no way to stub the target: GetAtt requires a
# real resource of a known type, and the snippet never says which type. Ref and
# Sub targets ARE stubbed as parameters (see collect_undeclared_names), so
# their equivalents stay enabled.
IGNORED_RULES = ("E1010",)

# Warnings worth failing on. cfn-lint reports deprecation as a warning, but a
# remediation that tells a customer to deploy a runtime AWS has already
# disabled is exactly the staleness this validator exists to catch. These are
# value-level: the fix is to bump a version string.
WARNINGS_AS_ERRORS = {
    "W2531",  # deprecated Lambda runtime
    "W3690",  # deprecated RDS engine version
}

# Service-level retirement. Worth surfacing — a remediation pointing at a
# service AWS is winding down has a shelf life — but the fix is to migrate the
# check to a different service, which is a content decision rather than a typo.
# Reported as [INFO] so the signal is visible without blocking the build.
WARNINGS_INFORMATIONAL = {
    "W3696",  # property from a service being shut down
    "W3697",  # resource type from a service in maintenance mode
}

# Warnings that only fire because of the synthetic parameters this script
# generates to stand in for a fragment's undeclared references.
IGNORED_WARNINGS = {
    "W2501",  # "NoEcho should be True" on a stub parameter
    "W1011",  # "use dynamic references for secrets" on a stub parameter
}

FAILURES: list[dict] = []


@dataclass
class CfnBlock:
    code: str
    line: int  # 1-based line in the policy file of the snippet's first line
    uid: str
    file: Path


@dataclass
class LintResult:
    success: bool = True
    issues: list[str] = field(default_factory=list)
    # (in-snippet line offset, message) so failures can be reported against
    # the policy file rather than the temp template.
    located: list[tuple[int, str]] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def dedent_block(block: str) -> str:
    """Strip the YAML block-scalar indentation the fence sits inside.

    The snippets live inside a `desc: |` scalar, so every line carries the
    scalar's indentation. HCL does not care, which is why the Terraform
    validator can skip this — YAML very much does: a template indented by 12
    spaces is not a template, it is a parse error.
    """
    lines = block.split("\n")
    indents = [
        len(line) - len(line.lstrip(" ")) for line in lines if line.strip()
    ]
    if not indents:
        return block
    common = min(indents)
    return "\n".join(line[common:] if line.strip() else "" for line in lines)


def extract_cfn_blocks(content: str, filepath: Path) -> list[CfnBlock]:
    """Extract YAML code blocks from cloudformation remediation sections."""
    lines = content.split("\n")
    uid_positions: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = re.match(r"^  - uid:\s+(\S+)", line)
        if m:
            uid_positions.append((i + 1, m.group(1)))

    def find_uid_for_line(line_num: int) -> str:
        result = ""
        for pos, uid in uid_positions:
            if pos <= line_num:
                result = uid
            else:
                break
        return result

    pattern = re.compile(
        r"- id: cloudformation\s*\n\s+desc: \|-?\s*\n"
        r"(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )

    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        cfn_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(cfn_line)

        # One logical template may be split across several ```yaml fences
        # interleaved with prose. Concatenate them so cfn-lint sees the whole
        # template rather than fragments.
        fences = []
        first_line = None
        for fence in re.finditer(r"```yaml\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).rstrip()
            if not block.strip():
                continue
            if first_line is None:
                code_offset = desc_start + fence.start(1)
                first_line = content[:code_offset].count("\n") + 1
            fences.append(dedent_block(block))

        if fences:
            blocks.append(
                CfnBlock(
                    code="\n".join(fences), line=first_line, uid=uid, file=filepath
                )
            )
    return blocks


# ---------------------------------------------------------------------------
# Snippet processing
# ---------------------------------------------------------------------------

# Names CloudFormation resolves itself; never stub these.
PSEUDO_PARAMS = re.compile(r"^AWS::")

REF_SHORT = re.compile(r"!Ref\s+([A-Za-z][A-Za-z0-9]*)")
REF_LONG = re.compile(r'"?Ref"?\s*:\s*"?([A-Za-z][A-Za-z0-9]*)"?')
SUB_VAR = re.compile(r"\$\{([A-Za-z][A-Za-z0-9]*)\}")


def declared_names(code: str) -> set[str]:
    """Top-level logical IDs the snippet declares under any section.

    Section keys sit at column 0 and their children at column 2, which is the
    shape every snippet in the policy uses.
    """
    return set(re.findall(r"^ {2}([A-Za-z][A-Za-z0-9]*):\s*$", code, re.M))


def collect_undeclared_names(code: str) -> set[str]:
    """Logical IDs the snippet references but never declares.

    A remediation example demonstrates one setting and is meant to be pasted
    into a configuration that already has the surrounding resources, so these
    references are expected. Declaring them as parameters lets cfn-lint check
    everything else without drowning the run in unresolved-reference errors.
    """
    referenced = set()
    referenced.update(REF_SHORT.findall(code))
    referenced.update(REF_LONG.findall(code))
    referenced.update(SUB_VAR.findall(code))
    return {
        n
        for n in referenced - declared_names(code)
        if not PSEUDO_PARAMS.match(n)
    }


def generate_template(code: str) -> tuple[str, int]:
    """Wrap a snippet into a complete template.

    Returns (template_text, preamble_line_count) so cfn-lint line numbers can
    be translated back to the policy file.
    """
    stubs = sorted(collect_undeclared_names(code))
    parts = [TEMPLATE_HEADER]

    if stubs:
        # If the snippet already has its own Parameters section, adding a
        # second one would produce a duplicate key and a YAML parse error, so
        # merge into the existing one instead.
        if re.search(r"^Parameters:\s*$", code, re.M):
            stub_yaml = "".join(
                f"  {name}:\n    Type: String\n" for name in stubs
            )
            code = re.sub(
                r"^(Parameters:\s*\n)", r"\1" + stub_yaml, code, count=1, flags=re.M
            )
        else:
            parts.append("Parameters:\n")
            for name in stubs:
                parts.append(f"  {name}:\n    Type: String\n")

    preamble = "".join(parts)
    return preamble + code + "\n", preamble.count("\n")


# ---------------------------------------------------------------------------
# cfn-lint execution
# ---------------------------------------------------------------------------

PARSEABLE = re.compile(
    r"^(?P<path>.+?):(?P<line>\d+):(?P<col>\d+):(?P<eline>\d+):(?P<ecol>\d+):"
    r"(?P<rule>[EWI]\d+):(?P<msg>.*)$"
)


def run_cfn_lint(paths: list[Path]) -> dict[str, list[tuple[int, str, str]]]:
    """Lint templates in one batch.

    cfn-lint loads the AWS resource specification on startup, which dominates
    runtime, so one invocation over every template is far cheaper than one per
    template. Returns {filename: [(line, rule, message), ...]}.
    """
    cmd = ["cfn-lint", "--format", "parseable"]
    for rule in IGNORED_RULES:
        cmd.extend(["--ignore-checks", rule])
    # `--` terminates --ignore-checks, which accepts multiple values and would
    # otherwise swallow every template path that follows it.
    cmd.append("--")
    cmd.extend(str(p) for p in paths)

    result = subprocess.run(cmd, capture_output=True, text=True, timeout=900)

    findings: dict[str, list[tuple[int, str, str]]] = {}
    for line in (result.stdout + result.stderr).splitlines():
        m = PARSEABLE.match(line.strip())
        if not m:
            continue
        name = Path(m.group("path")).name
        findings.setdefault(name, []).append(
            (int(m.group("line")), m.group("rule"), m.group("msg").strip())
        )
    return findings


def is_failure(rule: str) -> bool:
    if rule in IGNORED_WARNINGS:
        return False
    if rule.startswith("E"):
        return True
    return rule in WARNINGS_AS_ERRORS


# ---------------------------------------------------------------------------
# Validation orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    """Show the first resource type in the snippet, or its first line."""
    m = re.search(r"Type:\s*(\S+)", code)
    if m:
        return m.group(1)
    first_line = code.split("\n")[0].strip()
    if len(first_line) > max_len:
        first_line = first_line[: max_len - 3] + "..."
    return first_line


def policy_relpath(filepath: Path) -> str:
    return str(filepath.resolve().relative_to(REPO_ROOT))


def validate_policy_file(filepath: Path) -> tuple[int, int]:
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_cfn_blocks(content, filepath)
    if not blocks:
        return 0, 0

    relpath = policy_relpath(filepath)
    pass_count = 0
    fail_count = 0

    with tempfile.TemporaryDirectory(prefix="cfnlint_") as tmp:
        tmp_path = Path(tmp)
        index: dict[str, tuple[CfnBlock, int]] = {}

        for i, block in enumerate(blocks):
            template, preamble_lines = generate_template(block.code)
            name = f"t{i:05d}.yaml"
            (tmp_path / name).write_text(template)
            index[name] = (block, preamble_lines)

        findings = run_cfn_lint(sorted(tmp_path.glob("t*.yaml")))

    for name in sorted(index):
        block, preamble_lines = index[name]
        snippet = truncate_snippet(block.code)
        issues = []
        notes = []
        for lint_line, rule, msg in findings.get(name, []):
            if rule in WARNINGS_INFORMATIONAL:
                notes.append(f"{rule}: {msg}")
                continue
            if not is_failure(rule):
                continue
            # Translate the temp-template line back to the policy file.
            snippet_line = max(lint_line - preamble_lines, 1)
            policy_line = block.line + snippet_line - 1
            issues.append((policy_line, f"{rule}: {msg}"))

        for note in notes:
            print(f"[INFO] {block.uid}")
            print(f"       {note}")

        if not issues:
            print(f"[PASS] {block.uid}")
            print(f"       {snippet}")
            pass_count += 1
            continue

        print(f"[FAIL] {block.uid}")
        print(f"       {snippet}")
        for _line, text in issues:
            print(f"       {text}")
        fail_count += 1
        FAILURES.append(
            {
                "file": relpath,
                "line": issues[0][0],
                "uid": block.uid,
                "snippet": snippet,
                "errors": [text for _l, text in issues],
            }
        )

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# GitHub Actions annotations
# ---------------------------------------------------------------------------

def emit_github_annotations() -> None:
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['snippet']}"
        title = f"CloudFormation validation ({r['uid']})"
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = (
            title.replace("%", "%25")
            .replace("\r", "%0D")
            .replace("\n", "%0A")
            .replace(",", "%2C")
            .replace("::", "%3A%3A")
        )
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Validate CloudFormation remediation snippets with cfn-lint"
    )
    parser.add_argument(
        "target", nargs="?", default="all", choices=["all", *TARGETS]
    )
    parser.add_argument("--github-actions", action="store_true")
    args = parser.parse_args()

    if not shutil.which("cfn-lint"):
        print(
            "Error: cfn-lint not found in PATH.\n"
            "Install with: pip install cfn-lint",
            file=sys.stderr,
        )
        sys.exit(1)

    total_pass = 0
    total_fail = 0

    targets_to_run = TARGETS.keys() if args.target == "all" else [args.target]
    for t in targets_to_run:
        for filepath in TARGETS[t]:
            p, f = validate_policy_file(filepath)
            total_pass += p
            total_fail += f

    if args.github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
