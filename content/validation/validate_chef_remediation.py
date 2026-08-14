#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates Chef Infra recipe code blocks found in remediation sections of
# cnspec policies by running cookstyle against each snippet.
#
# Only validates `- id: chef` remediation blocks.
#
# cookstyle is Chef's RuboCop distribution. It catches Ruby syntax errors plus
# Chef-specific correctness and modernization problems — templating sysctl
# values instead of using the sysctl resource, integer file modes, node
# attribute comparisons that should use the platform helpers, and so on.
#
# Usage:
#   python3 validate_chef_remediation.py                  # validate all
#   python3 validate_chef_remediation.py linux            # validate Linux only
#   python3 validate_chef_remediation.py chef             # validate Chef policies only
#   python3 validate_chef_remediation.py --github-actions # emit GH annotations

import concurrent.futures
import json
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
        SCRIPT_DIR / ".." / "mondoo-linux-operational-policy.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-snmp-policy.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-linux-workstation-security.mql.yaml",
    ],
    "chef": [
        SCRIPT_DIR / ".." / "mondoo-chef-infra-client.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-chef-infra-server.mql.yaml",
    ],
}

# cookstyle cops to disable — these are too noisy for remediation snippets that
# are reference examples rather than complete cookbooks.
#
# Chef/Deprecations/ResourceWithoutUnifiedTrue and Chef/Sharing/* only apply to
# custom resources and cookbook metadata, neither of which a snippet carries.
DISABLED_COPS = [
    "Chef/Sharing/InvalidLicenseString",
]

FAILURES: list[dict] = []


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------

@dataclass
class ChefBlock:
    code: str
    line: int
    uid: str
    file: Path


@dataclass
class CookstyleResult:
    success: bool
    issues: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def extract_chef_blocks(content: str, filepath: Path) -> list[ChefBlock]:
    """Extract Ruby code blocks from `- id: chef` remediation sections."""
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
        r"- id: chef\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        chef_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(chef_line)

        for fence in re.finditer(r"```ruby\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).rstrip()
            if block.strip():
                code_offset = desc_start + fence.start(1)
                line_number = content[:code_offset].count("\n") + 1
                blocks.append(ChefBlock(
                    code=block, line=line_number, uid=uid, file=filepath,
                ))
    return blocks


# ---------------------------------------------------------------------------
# Snippet processing
# ---------------------------------------------------------------------------

def sanitize_snippet(code: str) -> str:
    """Clean up a recipe snippet so it stands alone as a recipe file."""
    code = textwrap.dedent(code)
    if not code.endswith("\n"):
        code += "\n"
    return code


# ---------------------------------------------------------------------------
# cookstyle execution
# ---------------------------------------------------------------------------

def run_cookstyle(recipe: Path) -> CookstyleResult:
    """Run cookstyle on a recipe file and return structured results."""
    cmd = ["cookstyle", "--format", "json", "--force-default-config"]
    for cop in DISABLED_COPS:
        cmd += ["--except", cop]
    cmd.append(str(recipe))

    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)

    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        stderr = result.stderr.strip()
        return CookstyleResult(
            success=False,
            issues=[stderr or f"cookstyle exited with code {result.returncode}"],
        )

    issues = []
    for f in data.get("files", []):
        for offense in f.get("offenses", []):
            cop = offense.get("cop_name", "")
            line = offense.get("location", {}).get("line", "")
            msg = offense.get("message", "")
            issues.append(f"{cop} (line {line}): {msg}")

    return CookstyleResult(success=not issues, issues=issues)


# ---------------------------------------------------------------------------
# Validation orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    """Show the first meaningful line of the snippet, truncated."""
    for line in code.split("\n"):
        stripped = line.strip()
        if stripped and not stripped.startswith("#"):
            if len(stripped) > max_len:
                stripped = stripped[: max_len - 3] + "..."
            return stripped
    return code[:max_len]


def validate_block(block: ChefBlock) -> tuple[ChefBlock, bool, list[str]]:
    """Validate a single Chef recipe block."""
    sanitized = sanitize_snippet(block.code)

    with tempfile.TemporaryDirectory(prefix="cookstyle_") as tmp:
        # cookstyle keys some cops off the cookbook layout; `recipes/` is what
        # makes it treat the file as a recipe rather than a library file.
        recipes = Path(tmp) / "recipes"
        recipes.mkdir()
        recipe = recipes / "default.rb"
        recipe.write_text(sanitized)

        result = run_cookstyle(recipe)
        return block, result.success, result.issues


def validate_policy_file(filepath: Path, workers: int) -> tuple[int, int]:
    """Validate all Chef blocks in a policy file."""
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_chef_blocks(content, filepath)

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
        futures = [pool.submit(validate_block, b) for b in blocks]
        results = [f.result() for f in concurrent.futures.as_completed(futures)]

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
        title = f"Cookstyle ({r['uid']})"
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

    if not shutil.which("cookstyle"):
        print(
            "Error: cookstyle not found in PATH.\n"
            "Install with: gem install cookstyle",
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
