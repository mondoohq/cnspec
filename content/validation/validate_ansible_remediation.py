#!/usr/bin/env python3
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1
#
# Validates Ansible playbook code blocks found in remediation sections of
# cnspec policies by running ansible-lint against each snippet.
#
# Usage:
#   python3 validate_ansible_remediation.py                  # validate all
#   python3 validate_ansible_remediation.py linux            # validate Linux only
#   python3 validate_ansible_remediation.py windows          # validate Windows only
#   python3 validate_ansible_remediation.py --github-actions # emit GH annotations

import concurrent.futures
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent

TARGETS = {
    "linux": [
        SCRIPT_DIR / ".." / "mondoo-linux-security.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-operational-policy.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-snmp-policy.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-workstation-security.mql.yaml",
    ],
    "windows": [
        SCRIPT_DIR / ".." / "mondoo-windows-security.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-windows-workstation-security.mql.yaml",
    ],
    "macos": [SCRIPT_DIR / ".." / "mondoo-macos-security.mql.yaml"],
    "kubernetes": [SCRIPT_DIR / ".." / "mondoo-kubernetes-security.mql.yaml"],
}

# ansible-lint rules to skip — these are too noisy for remediation snippets
# that are designed as reference examples, not production playbooks.
SKIP_RULES = [
    "yaml[truthy]",  # snippets use `become: true` not `become: "true"`
]

# Rules to ignore in results (can't be skipped via -x but are false positives)
IGNORE_CHECKS = {
    "syntax-check[unknown-module]",  # offline mode can't resolve collections
}

FAILURES: list[dict] = []


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------

@dataclass
class AnsibleBlock:
    code: str
    line: int
    uid: str
    file: Path


@dataclass
class LintResult:
    success: bool
    issues: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def extract_ansible_blocks(content: str, filepath: Path) -> list[AnsibleBlock]:
    """Extract YAML code blocks from ansible remediation sections."""
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
        r"- id: ansible\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        ansible_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(ansible_line)

        for fence in re.finditer(r"```yaml\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).rstrip()
            if block.strip():
                code_offset = desc_start + fence.start(1)
                line_number = content[:code_offset].count("\n") + 1
                blocks.append(AnsibleBlock(
                    code=block, line=line_number, uid=uid, file=filepath,
                ))
    return blocks


# ---------------------------------------------------------------------------
# Snippet processing
# ---------------------------------------------------------------------------

def dedent_snippet(code: str) -> str:
    """Remove common leading whitespace from all lines.

    Uses the indentation of the first non-empty content line (after ---)
    as the baseline, since the --- document marker may have been stripped
    to column 0 by the extraction process.
    """
    import textwrap
    return textwrap.dedent(code)


def sanitize_snippet(code: str) -> str:
    """Clean up ansible snippet for linting."""
    code = dedent_snippet(code)
    # Replace <placeholder> tokens with valid strings
    code = re.sub(r'"<[a-zA-Z][a-zA-Z0-9_-]*>"', '"placeholder"', code)
    code = re.sub(r"<[a-zA-Z][a-zA-Z0-9_-]*>", "placeholder", code)
    # Ensure trailing newline
    if not code.endswith("\n"):
        code += "\n"
    return code


# ---------------------------------------------------------------------------
# ansible-lint execution
# ---------------------------------------------------------------------------

def run_ansible_lint(playbook_path: Path) -> LintResult:
    """Run ansible-lint on a playbook file and return structured results."""
    skip_args = []
    for rule in SKIP_RULES:
        skip_args.extend(["-x", rule])

    result = subprocess.run(
        [
            "ansible-lint",
            "--format=json",
            "--offline",
            "--profile=basic",
            *skip_args,
            str(playbook_path),
        ],
        capture_output=True,
        text=True,
        timeout=60,
    )

    issues = []
    try:
        data = json.loads(result.stdout)
        for issue in data:
            check = issue.get("check_name", "unknown")
            if check in IGNORE_CHECKS:
                continue
            desc = issue.get("description", "")
            body = issue.get("content", {}).get("body", "")
            msg = f"{check}: {desc}"
            if body:
                msg += f" ({body})"
            issues.append(msg)
    except (json.JSONDecodeError, TypeError):
        # Only report stderr if we can't parse JSON and it's not just warnings
        stderr = result.stderr.strip()
        if stderr:
            lines = [
                l for l in stderr.split("\n")
                if not any(s in l for s in [
                    "caching", "WARNING", "UserWarning",
                    "cache_dir", "unskippable",
                ])
            ]
            if lines:
                issues.append("\n".join(lines))

    if not issues:
        return LintResult(success=True)

    return LintResult(success=False, issues=issues)


# ---------------------------------------------------------------------------
# Validation orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    """Show first meaningful line of ansible snippet, truncated."""
    for line in code.split("\n"):
        stripped = line.strip()
        if stripped and stripped != "---":
            if len(stripped) > max_len:
                stripped = stripped[: max_len - 3] + "..."
            return stripped
    return code[:max_len]


def validate_block(block: AnsibleBlock) -> tuple[AnsibleBlock, bool, list[str]]:
    """Validate a single ansible block. Returns (block, success, issues)."""
    sanitized = sanitize_snippet(block.code)

    with tempfile.TemporaryDirectory(prefix="ansible_lint_") as tmp:
        tmp_path = Path(tmp)
        playbook = tmp_path / "playbook.yml"
        playbook.write_text(sanitized)
        result = run_ansible_lint(playbook)
        return block, result.success, result.issues


def validate_policy_file(
    filepath: Path, workers: int
) -> tuple[int, int]:
    """Validate all ansible blocks in a policy file."""
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_ansible_blocks(content, filepath)

    if not blocks:
        return 0, 0

    resolved = filepath.resolve()
    try:
        policy_relpath = str(resolved.relative_to(Path.cwd()))
    except ValueError:
        policy_relpath = str(resolved.relative_to(SCRIPT_DIR.resolve().parent.parent))

    pass_count = 0
    fail_count = 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {pool.submit(validate_block, b): b for b in blocks}
        results = []
        for future in concurrent.futures.as_completed(futures):
            results.append(future.result())

    # Sort by line number for stable output
    results.sort(key=lambda r: r[0].line)

    for block, success, issues in results:
        snippet = truncate_snippet(block.code)
        if success:
            print(f"[PASS] {block.uid}")
            print(f"       {snippet}")
            pass_count += 1
        else:
            print(f"[FAIL] {block.uid}")
            print(f"       {snippet}")
            for issue in issues:
                print(f"       {issue}")
            fail_count += 1
            FAILURES.append({
                "file": policy_relpath,
                "line": block.line,
                "uid": block.uid,
                "snippet": snippet,
                "errors": issues,
            })

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# GitHub Actions annotations
# ---------------------------------------------------------------------------

def emit_github_annotations() -> None:
    """Print GitHub Actions workflow commands for each failure."""
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['snippet']}"
        title = f"Ansible lint ({r['uid']})"
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = (
            title.replace("%", "%25")
            .replace("\r", "%0D")
            .replace("\n", "%0A")
            .replace(",", "%2C")
            .replace("::", "%3A%3A")
        )
        print(
            f"::error file={r['file']},line={r['line']},title={title}::{msg}"
        )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = sys.argv[1:]
    github_actions = False
    workers = 8
    target = "all"

    positional = []
    i = 0
    while i < len(args):
        if args[i] == "--github-actions":
            github_actions = True
        elif args[i] == "--workers" and i + 1 < len(args):
            workers = int(args[i + 1])
            i += 1
        else:
            positional.append(args[i])
        i += 1

    if positional:
        target = positional[0]

    valid_targets = ["all"] + list(TARGETS.keys())
    if target not in valid_targets:
        print(
            f"Unknown target: {target}\n"
            f"Usage: {sys.argv[0]} [{'|'.join(valid_targets)}] "
            f"[--github-actions] [--workers N]",
            file=sys.stderr,
        )
        sys.exit(2)

    if not shutil.which("ansible-lint"):
        print(
            "Error: ansible-lint not found in PATH.\n"
            "Install with: pipx install ansible-lint",
            file=sys.stderr,
        )
        sys.exit(1)

    total_pass = 0
    total_fail = 0

    targets_to_run = TARGETS.keys() if target == "all" else [target]

    for t in targets_to_run:
        for filepath in TARGETS[t]:
            p, f = validate_policy_file(filepath, workers)
            total_pass += p
            total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
