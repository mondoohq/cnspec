# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Shared helpers for the remediation command validators.

import os
import re
import shlex
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))  # content/validation
# Re-exported so the per-CLI validators in this directory import their paths
# from one place. See ../../paths.py for why they are resolved centrally.
from paths import CONTENT_DIR, DATA_DIR, DUMP_DIR, REPO_ROOT  # noqa: E402,F401

# Collected failures for annotation output.  Each entry is a dict with keys:
# file, line, uid, command, errors, cloud
FAILURES: list[dict] = []

# `$(...)` command substitution. The character class excludes parens, so a
# single match is always the innermost substitution; extract_substitutions()
# loops to unwrap nested ones.
COMMAND_SUBSTITUTION = re.compile(r"\$\(([^()]*)\)")

# Shell operators that end one command and start the next. Deliberately the
# same pipe and semicolon boundaries the splitter has always recognized, so
# this change is about *where* they are found rather than which ones count.
# `&&` is lexed as its own token but left inline, as it was before.
SHELL_OPERATORS = {"|", "||", ";"}


def lex_shell(text: str) -> list[str]:
    """Tokenize a shell command, keeping unquoted `|` and `;` as their own
    tokens and leaving quoted ones inside the token they belong to.

    `shlex.split` cannot make that distinction: it discards quoting, so a
    filter expression such as `'{ ($.eventName = A) || ($.eventName = B) }'`
    comes back indistinguishable from a real pipeline, and splitting the
    result truncates the command at the first `||`. Every flag after that
    point then goes unvalidated. `punctuation_chars` keeps the operators
    separate at lex time, while quoted text stays in one piece.

    Raises ValueError on an unterminated quote, like `shlex.split`.
    """
    lexer = shlex.shlex(text, posix=True, punctuation_chars="|;")
    lexer.whitespace_split = True
    # `shlex.split` clears this; without it an inline `#` truncates the command.
    lexer.commenters = ""
    return list(lexer)


def extract_substitutions(text: str) -> tuple[list[str], str]:
    """Pull every `$(...)` command out of `text`.

    Returns (inner commands, text with the substitutions blanked out).

    Replacing in one pass would mishandle nesting: substituting the inner
    `$(cmd2)` of `$(cmd1 $(cmd2))` leaves `$(cmd1 )` behind, and re.sub
    resumes *after* the match rather than rescanning, so the outer command
    would stay inline and its flags would read as flags of whatever command
    surrounds it. Looping until no match remains unwraps both.
    """
    commands = []
    while True:
        m = COMMAND_SUBSTITUTION.search(text)
        if not m:
            return commands, text
        commands.append(m.group(1).strip())
        text = text[: m.start()] + " " + text[m.end() :]


def policy_relpath(policy_file: Path) -> str:
    """Repo-root-relative path for a policy file, as GitHub annotations
    expect. Independent of the caller's working directory."""
    return os.path.relpath(policy_file.resolve(), REPO_ROOT)


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def extract_bash_blocks(
    content: str,
    include_audit: bool = False,
    remediation_ids: tuple[str, ...] = ("cli",),
) -> list[tuple[str, int, str]]:
    """Extract bash code blocks from remediation sections.

    Returns a list of (block_text, line_number, uid) tuples where line_number
    is the 1-based line of the first code line in the block, and uid is the
    check UID that contains this remediation block.

    remediation_ids selects which `- id: <method>` remediation entries are
    scanned. It defaults to `("cli",)` — the vendor-CLI method every CLI
    validator reads. API-first products that document their fix as a REST
    call under `- id: api` (e.g. the Vercel policy) pass `("cli", "api")`
    so both are validated; the lookahead already stops each block at the
    next `- id:` regardless of method, so adding ids never bleeds one
    block into the next.

    With include_audit=True, bash blocks in `audit: |` sections are
    extracted as well. A wrong audit command misleads users exactly like a
    wrong remediation, so every validator that can enable this does: the
    REST API and Cobra validators from the start, and the cloud CLI
    validators as of this change. `azure` included, now that
    upstream/dump/azure.py records CLI option strings rather than argparse
    destination names — with the old grammar, commands used only in audit
    blocks kept names like `--resource-group-name`, and turning this on
    reported ~170 failures that were the grammar's fault, not the content's.
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

    id_alt = "|".join(re.escape(i) for i in remediation_ids)
    pattern = re.compile(
        rf"- id: (?:{id_alt})\s*\n\s+desc: \|\s*\n"
        r"(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
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

    if include_audit:
        # An `audit: |` scalar runs until the next line at the same
        # indentation (its sibling key, usually `remediation:`).
        audit_pattern = re.compile(
            r"\n(\s+)audit: \|\s*\n(.*?)(?=\n\1\S|\Z)", re.DOTALL
        )
        for match in audit_pattern.finditer(content):
            audit_block = match.group(2)
            audit_start = match.start(2)
            audit_line = content[:match.start()].count("\n") + 2
            uid = find_uid_for_line(audit_line)

            for fence in re.finditer(r"```bash\s*\n(.*?)```", audit_block, re.DOTALL):
                block = fence.group(1).strip()
                if block:
                    code_offset = audit_start + fence.start(1)
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
            # then re-join and split on unquoted pipes/semicolons.
            # A command can also continue across lines inside an open
            # quote — multi-line JSON --data payloads — which shlex
            # reports as an unclosed-quote ValueError; keep appending
            # lines until the quote closes (or the block ends, in which
            # case fall back to a naive split of what we have).
            segments: list[str] = []
            while True:
                try:
                    tokens = lex_shell(stripped)
                    # Break the token stream at the unquoted operators; an
                    # operator inside a quoted value stayed inside its token.
                    current: list[str] = []
                    segments = []
                    for token in tokens:
                        if token in SHELL_OPERATORS:
                            segments.append(" ".join(current))
                            current = []
                        else:
                            current.append(token)
                    segments.append(" ".join(current))
                    break
                except ValueError:
                    if i + cont_lines + 1 >= len(lines):
                        # The quote never closes (malformed snippet).
                        # Keep the text intact rather than whitespace-
                        # splitting it, which would mangle any JSON
                        # payload structure; downstream parsing will
                        # surface the malformation.
                        segments = [stripped]
                        break
                    cont_lines += 1
                    stripped = stripped + "\n" + lines[i + cont_lines]

            for segment in segments:
                # A `$(...)` substitution holds a command in its own right, and
                # audit blocks use them to feed one query into the next. Pull
                # each one out as a separate command and drop it from the text
                # around it — left inline, its flags read as flags of the outer
                # command.
                inner_commands, segment = extract_substitutions(segment)
                for inner in inner_commands:
                    if inner.startswith(f"{prefix} "):
                        commands.append((inner, raw_line_num))

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

def _encode_annotation(s: str, in_properties: bool = False) -> str:
    """Encode workflow-command special characters.

    Property values (like title=) additionally need ',' and '::' encoded
    because they would otherwise terminate the property list.
    """
    s = s.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
    if in_properties:
        s = s.replace(",", "%2C").replace("::", "%3A%3A")
    return s


def emit_github_annotations() -> None:
    """Print GitHub Actions workflow commands for each failure.

    These produce inline annotations on the PR Files tab, regardless of
    whether the annotated file is part of the PR diff.
    See https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions#setting-an-error-message
    """
    for r in FAILURES:
        msg = _encode_annotation("; ".join(r["errors"]) + f" — {r['command']}")
        title = _encode_annotation(
            f"{r['cloud'].upper()} CLI validation ({r['uid']})", in_properties=True
        )
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")
