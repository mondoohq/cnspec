#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
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

CONTENT_DIR = SCRIPT_DIR / ".."

# Unlike the other validators, this one has no policy allowlist. A `TARGETS`
# list is a standing invitation to the failure it is supposed to prevent: a
# policy gains its first shell snippet, nobody adds it to the list, and the
# snippet ships unlinted with CI green. shellcheck needs no per-policy setup —
# it lints a shell script, and every policy's shell snippets are shell scripts —
# so the default target is simply every policy in content/.
#
# The named groups below stay as a convenience for running one area locally
# (`validate_bash_remediation.py linux`), not as a definition of what CI covers.
TARGETS = {
    "linux": [
        CONTENT_DIR / "mondoo-linux-security.mql.yaml",
        CONTENT_DIR / "mondoo-linux-operational-policy.mql.yaml",
        CONTENT_DIR / "mondoo-linux-snmp-policy.mql.yaml",
        CONTENT_DIR / "mondoo-linux-workstation-security.mql.yaml",
    ],
    "kubernetes": [CONTENT_DIR / "mondoo-kubernetes-security.mql.yaml"],
    "edr": [CONTENT_DIR / "mondoo-edr-policy.mql.yaml"],
    "freebsd": [CONTENT_DIR / "mondoo-freebsd-security.mql.yaml"],
    "mariadb": [CONTENT_DIR / "mondoo-mariadb-security.mql.yaml"],
    "mysql": [CONTENT_DIR / "mondoo-mysql-security.mql.yaml"],
    "proxmox": [CONTENT_DIR / "mondoo-proxmox-security.mql.yaml"],
}


def all_policy_files() -> list[Path]:
    """Every policy bundle in content/ — what `all` (and therefore CI) covers."""
    return sorted(CONTENT_DIR.glob("*.mql.yaml"))


# shellcheck codes to exclude.
#
# The first group is about these snippets being *examples*, not programs:
# SC2016 - expressions don't expand in single quotes (intentional for config file content)
# SC2312 - consider invoking separately to avoid masking return values (noisy for examples)
# SC2009 - "use pgrep instead of ps | grep" (style; the ps form is what the vendor docs show)
# SC2013 - "read lines with a while loop instead of for" (style, and the loop bodies here are one-liners)
#
# The second group is shellcheck reading vendor CLI syntax as shell syntax:
# SC1083 - literal { }. AWS CLI shorthand (`EncryptionAtRest={DataVolumeKMSKeyId=...}`)
#          and doc placeholders (`/subscriptions/{subscriptionId}`) contain braces
#          that are meant literally; without a comma the shell leaves them alone.
# SC2102 - "ranges can only match single chars". gcloud's own docs write placeholders
#          as `[key_ring_name]`, which shellcheck reads as a glob character range.
# SC2046 - quote `$(...)` to prevent word splitting. `az resource update --ids
#          $(az ... -o tsv)` is the documented idiom and deliberately splits when the
#          inner query returns more than one id.
# SC2162 - `read` without -r. The snippets that use it read vendor identifiers,
#          which never contain backslashes.
#
# Deliberately NOT excluded: SC2086 (unquoted variable) and SC2140 (`"a"b"c"`),
# which flag real quoting mistakes. Both were fixed in the content instead.
EXCLUDE_CODES = [
    "SC2016",
    "SC2312",
    "SC2009",
    "SC2013",
    "SC1083",
    "SC2102",
    "SC2046",
    "SC2162",
]

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

# The fence language decides what gets linted, not the remediation method id.
#
# This used to read `- id: bash|script|sh` only, which left the ~690 ```bash
# fences under `- id: cli` unlinted in every policy without a CLI grammar
# validator (linux, freebsd, proxmox, macos, the database policies). A shell
# snippet is a shell snippet whichever method documents it, and for the cloud
# policies this is complementary rather than redundant: the grammar validators
# check that `aws ...` is a real command, shellcheck checks that the shell around
# it is correct — quoting, loops, redirection.
#
# Reading the fence rather than the id is also what keeps a `script` entry
# holding PowerShell (the Windows convention in content/CLAUDE.md) out of
# shellcheck: it is fenced ```powershell, so it is simply not matched.
SHELL_FENCE_LANGUAGES = ("bash", "sh")


def extract_bash_blocks(content: str, filepath: Path) -> list[BashBlock]:
    """Extract shell code blocks from every remediation method.

    Any `- id:` entry may hold a shell snippet; the fence language decides
    whether it is one (see SHELL_FENCE_LANGUAGES).
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
        r"- id: \S+\s*\n\s+desc: \|-?\s*\n"
        r"(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        bash_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(bash_line)

        lang_alt = "|".join(SHELL_FENCE_LANGUAGES)
        for fence in re.finditer(rf"```(?:{lang_alt})\s*\n(.*?)```", desc_block, re.DOTALL):
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
    # Replace <placeholder> tokens with valid shell strings.
    #
    # The character class has to admit spaces and dots. Real placeholders in
    # this content include `<Key Vault Resource ID>`, `<new login-name>`,
    # `<cert.pem>` and `<AKIA...>`, and a pattern limited to
    # [a-zA-Z0-9_-] leaves those in place — where shellcheck then reads the
    # `<` as a redirection and reports SC1073 "couldn't parse this
    # redirection" against a snippet that is perfectly fine. Six of those were
    # the entire "failure" list when this validator's scope was widened.
    #
    # Bounded to one line and 60 characters so a genuine `<` redirection
    # followed by a later `>` cannot be swallowed as one giant placeholder,
    # and `(?<!<)` keeps it off heredocs: in `cat <<EOF > /etc/foo` the span
    # `<EOF >` otherwise reads as a placeholder, the heredoc marker is
    # destroyed, and shellcheck reports the heredoc *body* as bad commands.
    code = re.sub(r'"(?<!<)<[^<>\n]{1,60}>"', '"placeholder"', code)
    code = re.sub(r"(?<!<)<[^<>\n]{1,60}>", "placeholder", code)
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
            "--severity=info",
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

    # `all` means every policy in content/, not the union of the named groups —
    # see the comment on TARGETS.
    files = all_policy_files() if target == "all" else TARGETS[target]

    for filepath in files:
        p, f = validate_policy_file(filepath, workers)
        total_pass += p
        total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
