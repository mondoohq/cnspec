#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates the Bicep snippets in `- id: bicep` remediation blocks with the
# Bicep CLI, which resolves resource types and apiVersions against the ARM type
# index and rejects properties a type does not define.
#
# Mirrors validate_cloudformation_remediation.py: extract the fenced blocks
# from one `- id:`, make each snippet stand on its own, run the ecosystem's own
# compiler over it.
#
# Usage: python3 validate_bicep_remediation.py [azure|m365] [--github-actions]

import argparse
import concurrent.futures
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
    "azure": [SCRIPT_DIR / ".." / "mondoo-azure-security.mql.yaml"],
    "m365": [SCRIPT_DIR / ".." / "mondoo-m365-security.mql.yaml"],
}

# `bicep build` needs a resolvable module/type cache. Snippets never use
# registry modules, so --no-restore keeps the run offline and fast.
BICEP_ARGS = ("build", "--stdout", "--no-restore")

# Diagnostics that only fire because a documentation snippet is a fragment.
#
# BCP057 ("the name X does not exist in the current context") is the Bicep
# equivalent of CloudFormation's unresolved-!Ref: a remediation example wires
# itself to a key vault or subnet it does not declare. There is no type-safe
# stand-in — declaring the name as `param x object` merely trades BCP057 for
# BCP036/BCP240, because Bicep requires a genuine resource reference in the
# positions these names occupy. So the rule is ignored, exactly as the
# CloudFormation validator ignores E1010 for `!GetAtt` targets.
IGNORED_DIAGNOSTICS = {"BCP057"}

# Informational: reported, but not a build failure. The fix is a migration or
# a deliberate documentation choice rather than a typo.
INFORMATIONAL = {
    "BCP081",  # no types available for this resource type/apiVersion
    # A remediation example names a specific cloud on purpose; telling readers
    # to write environment().resourceManager in a two-line snippet is noise.
    "no-hardcoded-env-urls",
}

FAILURES: list[dict] = []

DIAGNOSTIC = re.compile(
    r"^(?P<path>.+?)\((?P<line>\d+),(?P<col>\d+)\)\s*:\s*"
    r"(?P<sev>Error|Warning)\s+(?P<code>[A-Za-z0-9-]+):\s*(?P<msg>.*)$"
)


@dataclass
class BicepBlock:
    code: str
    line: int
    uid: str
    file: Path


@dataclass
class LintResult:
    issues: list[str] = field(default_factory=list)
    notes: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def dedent_block(block: str) -> str:
    """Strip the YAML block-scalar indentation the fence sits inside."""
    lines = block.split("\n")
    indents = [len(l) - len(l.lstrip(" ")) for l in lines if l.strip()]
    if not indents:
        return block
    common = min(indents)
    return "\n".join(l[common:] if l.strip() else "" for l in lines)


def extract_bicep_blocks(content: str, filepath: Path) -> list[BicepBlock]:
    """Extract Bicep code blocks from `- id: bicep` remediation sections."""
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
        r"- id: bicep\s*\n\s+desc: \|-?\s*\n"
        r"(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )

    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        bicep_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(bicep_line)

        fences, first_line = [], None
        for fence in re.finditer(r"```bicep\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).rstrip()
            if not block.strip():
                continue
            if first_line is None:
                code_offset = desc_start + fence.start(1)
                first_line = content[:code_offset].count("\n") + 1
            fences.append(dedent_block(block))

        if fences:
            blocks.append(
                BicepBlock(code="\n\n".join(fences), line=first_line, uid=uid, file=filepath)
            )
    return blocks


# ---------------------------------------------------------------------------
# bicep execution
# ---------------------------------------------------------------------------

def run_bicep(path: Path) -> list[tuple[int, str, str, str]]:
    """Compile one file. Returns [(line, severity, code, message), ...]."""
    result = subprocess.run(
        ["bicep", *BICEP_ARGS, str(path)],
        capture_output=True,
        text=True,
        timeout=120,
    )
    out = []
    for line in (result.stdout + result.stderr).splitlines():
        m = DIAGNOSTIC.match(line.strip())
        if m:
            out.append(
                (int(m.group("line")), m.group("sev"), m.group("code"), m.group("msg").strip())
            )
    return out


def validate_block(block: BicepBlock) -> tuple[BicepBlock, LintResult]:
    """Compile one snippet and classify its diagnostics."""
    with tempfile.TemporaryDirectory(prefix="bicep_") as tmp:
        path = Path(tmp) / "main.bicep"
        path.write_text(block.code + "\n")

        result = LintResult()
        for line, _sev, code, msg in run_bicep(path):
            if code in IGNORED_DIAGNOSTICS:
                continue
            if code in INFORMATIONAL:
                result.notes.append(f"{code}: {msg}")
                continue
            result.issues.append((block.line + max(line, 1) - 1, f"{code}: {msg}"))
        return block, result


# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    m = re.search(r"resource\s+\S+\s+'([^']+)'", code)
    if m:
        return m.group(1)
    first_line = code.split("\n")[0].strip()
    return first_line[: max_len - 3] + "..." if len(first_line) > max_len else first_line


def validate_policy_file(filepath: Path, workers: int) -> tuple[int, int]:
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_bicep_blocks(content, filepath)
    if not blocks:
        return 0, 0

    relpath = str(filepath.resolve().relative_to(REPO_ROOT))
    pass_count = fail_count = 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        results = list(pool.map(lambda b: validate_block(b), blocks))
    results.sort(key=lambda r: r[0].line)

    for block, result in results:
        snippet = truncate_snippet(block.code)
        for note in result.notes:
            print(f"[INFO] {block.uid}")
            print(f"       {note}")
        if not result.issues:
            print(f"[PASS] {block.uid}")
            print(f"       {snippet}")
            pass_count += 1
            continue
        print(f"[FAIL] {block.uid}")
        print(f"       {snippet}")
        for _line, text in result.issues:
            print(f"       {text}")
        fail_count += 1
        FAILURES.append({
            "file": relpath,
            "line": result.issues[0][0],
            "uid": block.uid,
            "snippet": snippet,
            "errors": [t for _l, t in result.issues],
        })

    return pass_count, fail_count


def emit_github_annotations() -> None:
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['snippet']}"
        title = f"Bicep validation ({r['uid']})"
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = (
            title.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
            .replace(",", "%2C").replace("::", "%3A%3A")
        )
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")


def main():
    parser = argparse.ArgumentParser(
        description="Validate Bicep remediation snippets with the Bicep CLI"
    )
    parser.add_argument("target", nargs="?", default="all", choices=["all", *TARGETS])
    parser.add_argument("--github-actions", action="store_true")
    parser.add_argument("--workers", type=int, default=8)
    args = parser.parse_args()

    if not shutil.which("bicep"):
        print(
            "Error: bicep not found in PATH.\n"
            "Install with: az bicep install && "
            'export PATH="$HOME/.azure/bin:$PATH"',
            file=sys.stderr,
        )
        sys.exit(1)

    total_pass = total_fail = 0
    for t in (TARGETS if args.target == "all" else [args.target]):
        for filepath in TARGETS[t]:
            p, f = validate_policy_file(filepath, args.workers)
            total_pass += p
            total_fail += f

    if args.github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
