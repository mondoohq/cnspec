#!/usr/bin/env python3
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1
#
# Validates Bash script code blocks found in remediation sections of cnspec
# policies by running shellcheck against each snippet.
#
# Only validates `- id: bash` remediation blocks (not `- id: cli`).
#
# Usage:
#   python3 validate_bash_remediation.py                  # validate all
#   python3 validate_bash_remediation.py linux            # validate Linux only
#   python3 validate_bash_remediation.py --github-actions # emit GH annotations

import concurrent.futures
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import textwrap
from dataclasses import dataclass, field
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent

TARGETS = {
    "linux": [
        SCRIPT_DIR / ".." / "mondoo-linux-security.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-snmp-policy.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-workstation-security.mql.yaml",
    ],
    "kubernetes": [SCRIPT_DIR / ".." / "mondoo-kubernetes-security.mql.yaml"],
}

# shellcheck codes to exclude:
# SC2034 - variable appears unused (common in example snippets)
# SC2312 - consider invoking separately to avoid masking return values (noisy for examples)
EXCLUDE_CODES = ["SC2034", "SC2312"]

FAILURES: list[dict] = []


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------

@dataclass
class BashBlock:
    code: str
    line: int
    uid: str
    file: Path


@dataclass
class ShellcheckResult:
    success: bool
    issues: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def extract_bash_blocks(content: str, filepath: Path) -> list[BashBlock]:
    """Extract bash code blocks from bash remediation sections.

    Only extracts from `- id: bash` sections, NOT `- id: cli`.
    """
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
        r"- id: bash\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        bash_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(bash_line)

        for fence in re.finditer(r"```bash\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).rstrip()
            if block.strip():
                code_offset = desc_start + fence.start(1)
                line_number = content[:code_offset].count("\n") + 1
                blocks.append(BashBlock(
                    code=block, line=line_number, uid=uid, file=filepath,
                ))
    return blocks


# ---------------------------------------------------------------------------
# Snippet processing
# ---------------------------------------------------------------------------

def sanitize_snippet(code: str) -> str:
    """Clean up bash snippet for shellcheck."""
    code = textwrap.dedent(code)
    # Replace <placeholder> tokens with valid shell strings
    code = re.sub(r'"<[a-zA-Z][a-zA-Z0-9_-]*>"', '"placeholder"', code)
    code = re.sub(r"<[a-zA-Z][a-zA-Z0-9_-]*>", "placeholder", code)
    # Ensure shebang is present — shellcheck needs it to detect shell dialect
    if not code.lstrip().startswith("#!"):
        code = "#!/bin/bash\n" + code
    # Ensure trailing newline
    if not code.endswith("\n"):
        code += "\n"
    return code


# ---------------------------------------------------------------------------
# shellcheck execution
# ---------------------------------------------------------------------------

def run_shellcheck(script_path: Path) -> ShellcheckResult:
    """Run shellcheck on a script file and return structured results."""
    exclude = ",".join(EXCLUDE_CODES)
    result = subprocess.run(
        [
            "shellcheck",
            "--format=json",
            "--severity=warning",
            f"--exclude={exclude}",
            str(script_path),
        ],
        capture_output=True,
        text=True,
        timeout=30,
    )

    if result.returncode == 0:
        return ShellcheckResult(success=True)

    issues = []
    try:
        data = json.loads(result.stdout)
        for issue in data:
            code = issue.get("code", "")
            level = issue.get("level", "")
            msg = issue.get("message", "")
            line = issue.get("line", "")
            issues.append(f"SC{code} ({level}, line {line}): {msg}")
    except (json.JSONDecodeError, TypeError):
        stderr = result.stderr.strip()
        if stderr:
            issues.append(stderr)

    if not issues:
        if result.returncode not in (0, 1):
            issues.append(f"shellcheck exited with code {result.returncode}")
        else:
            return ShellcheckResult(success=True)

    return ShellcheckResult(success=False, issues=issues)


# ---------------------------------------------------------------------------
# Validation orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    """Show first meaningful line of bash snippet, truncated."""
    for line in code.split("\n"):
        stripped = line.strip()
        if stripped and not stripped.startswith("#!"):
            if len(stripped) > max_len:
                stripped = stripped[: max_len - 3] + "..."
            return stripped
    return code[:max_len]


def validate_block(block: BashBlock) -> tuple[BashBlock, bool, list[str]]:
    """Validate a single bash block."""
    sanitized = sanitize_snippet(block.code)

    with tempfile.TemporaryDirectory(prefix="shellcheck_") as tmp:
        tmp_path = Path(tmp)
        script = tmp_path / "script.sh"
        script.write_text(sanitized)
        result = run_shellcheck(script)
        return block, result.success, result.issues


def validate_policy_file(
    filepath: Path, workers: int
) -> tuple[int, int]:
    """Validate all bash blocks in a policy file."""
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_bash_blocks(content, filepath)

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
        title = f"Shellcheck ({r['uid']})"
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
            try:
                workers = int(args[i + 1])
            except ValueError:
                print(f"Error: --workers requires an integer, got '{args[i + 1]}'", file=sys.stderr)
                sys.exit(2)
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

    if not shutil.which("shellcheck"):
        print(
            "Error: shellcheck not found in PATH.\n"
            "Install with: apt-get install shellcheck",
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
