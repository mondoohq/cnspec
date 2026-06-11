# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Shared helpers for the remediation command validators.

import os
import re
import shlex
from pathlib import Path

# content/validation/ — the directory holding this package, cmd_data/, and
# the dump scripts. Policy files live one level up in content/.
SCRIPT_DIR = Path(__file__).parent.parent
CMD_DATA_DIR = SCRIPT_DIR / "cmd_data"
REPO_ROOT = SCRIPT_DIR.parent.parent

# Collected failures for annotation output.  Each entry is a dict with keys:
# file, line, uid, command, errors, cloud
FAILURES: list[dict] = []


def policy_relpath(policy_file: Path) -> str:
    """Repo-root-relative path for a policy file, as GitHub annotations
    expect. Independent of the caller's working directory."""
    return os.path.relpath(policy_file.resolve(), REPO_ROOT)


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def extract_bash_blocks(content: str) -> list[tuple[str, int, str]]:
    """Extract bash code blocks from cli remediation sections.

    Returns a list of (block_text, line_number, uid) tuples where line_number
    is the 1-based line of the first code line in the block, and uid is the
    check UID that contains this remediation block.
    """
    # Pre-compute a list of (line_number, uid) from all `- uid:` lines so we
    # can look up the enclosing check for any position in the file.
    lines = content.split("\n")
    uid_positions: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = re.match(r"^  - uid:\s+(\S+)", line)
        if m:
            uid_positions.append((i + 1, m.group(1)))

    def find_uid_for_line(line_num: int) -> str:
        """Find the nearest uid defined before line_num."""
        result = ""
        for pos, uid in uid_positions:
            if pos <= line_num:
                result = uid
            else:
                break
        return result

    pattern = re.compile(
        r"- id: cli\s*\n\s+desc: \|\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        # Line number of the cli remediation block itself
        cli_line = content[:match.start()].count("\n") + 1
        uid = find_uid_for_line(cli_line)

        for fence in re.finditer(r"```bash\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).strip()
            if block:
                code_offset = desc_start + fence.start(1)
                line_number = content[:code_offset].count("\n") + 1
                blocks.append((block, line_number, uid))
    return blocks


def split_commands(block: str, prefix: str, block_start_line: int) -> list[tuple[str, int]]:
    """Split a code block into individual commands starting with prefix.

    Returns a list of (command, line_number) tuples.
    """
    lines = block.split("\n")
    commands = []
    i = 0

    while i < len(lines):
        line = lines[i]
        raw_line_num = block_start_line + i

        # Join continuation lines
        full_line = line
        cont_lines = 0
        while full_line.rstrip().endswith("\\") and i + cont_lines + 1 < len(lines):
            cont_lines += 1
            full_line = full_line.rstrip()[:-1] + " " + lines[i + cont_lines].strip()

        stripped = full_line.strip()
        if stripped and not stripped.startswith("#"):
            # Use shlex to handle quoted values containing | or ;
            # then re-join and split on unquoted pipes/semicolons
            try:
                tokens = shlex.split(stripped)
            except ValueError:
                tokens = stripped.split()
            rejoined = " ".join(tokens)
            # Split on pipe/semicolon boundaries
            for segment in re.split(r"\s*[|;]\s*", rejoined):
                segment = segment.strip()
                if segment.startswith(f"{prefix} "):
                    commands.append((segment, raw_line_num))

        i += 1 + cont_lines

    return commands


def truncate_cmd(cmd: str, max_len: int = 120) -> str:
    """Collapse whitespace and truncate a command for display."""
    display = " ".join(cmd.split())
    if len(display) > max_len:
        display = display[: max_len - 3] + "..."
    return display


# ---------------------------------------------------------------------------
# GitHub Actions annotations
# ---------------------------------------------------------------------------

def emit_github_annotations() -> None:
    """Print GitHub Actions workflow commands for each failure.

    These produce inline annotations on the PR Files tab, regardless of
    whether the annotated file is part of the PR diff.
    See https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions#setting-an-error-message
    """
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['command']}"
        title = f"{r['cloud'].upper()} CLI validation ({r['uid']})"
        # Workflow command special characters must be encoded
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = title.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A").replace(",", "%2C").replace("::", "%3A%3A")
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")
