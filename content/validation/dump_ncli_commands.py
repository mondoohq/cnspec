#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps the full nCLI command grammar (entities, operations, operation
# aliases, and required/optional parameters) from the Nutanix AOS Command
# Reference published on portal.nutanix.com.
#
# Unlike aws/gcloud/doctl, ncli cannot be introspected in CI: it is a
# proprietary Java CLI that Nutanix only distributes from a running Prism
# cluster. The support portal, however, serves the Command Reference through
# a JSON API that needs no authentication:
#
#   /api/v1/documents/<book>:<page>        -> metadata + full book TOC
#   /api/v1/documents/<book>:<page>/html   -> rendered page HTML
#
# Each nCLI entity page contains one <pre class="pre codeblock"> per
# operation with a regular grammar:
#
#   ncli> <entity> { <op> | <op-alias> } req-param="v" [opt-param="v" ]
#
# We pin the book version (like CLOUDFLARE_OPENAPI_SHA in
# validate_remediation_commands.py) so the output is deterministic; bumping
# NCLI_BOOK to a newer AOS release is a deliberate maintainer action.
#
# Output JSON:
#   {
#     "_meta": {"book": "...", "source": "..."},
#     "entities": {
#       "cluster": {
#         "operations": {
#           "edit-params": {
#             "aliases": ["edit-info"],
#             "required": [],
#             "optional": ["new-name", ...]
#           }
#         }
#       }
#     }
#   }
#
# Usage: python3 dump_ncli_commands.py [--output cmd_data/ncli_commands.json]

import argparse
import concurrent.futures
import html
import json
import re
import sys
import urllib.request
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
DEFAULT_OUTPUT = SCRIPT_DIR / "cmd_data" / "ncli_commands.json"

# AOS Command Reference book on portal.nutanix.com. The unversioned alias
# "Command-Ref-AOS" always resolves to the latest release; we pin a specific
# version so regeneration is reproducible.
NCLI_BOOK = "Command-Ref-AOS-v7_5"

PORTAL_API = "https://portal.nutanix.com/api/v1/documents"

# The ncli landing page; its metadata response carries the TOC for the
# entire Command Reference book.
NCLI_TOC_PAGE = "man-ncli-c.html"

# Matches one parameter assignment in a syntax line. Parameter values in the
# docs are placeholder identifiers or brace-delimited choice lists; neither
# contains '=', so anchoring on 'name=' is unambiguous.
_PARAM_RE = re.compile(r"([a-z0-9-]+)=")
_OPTIONAL_PARAM_RE = re.compile(r"\[\s*([a-z0-9-]+)=")

_PRE_BLOCK_RE = re.compile(r'<pre class="pre codeblock">(.*?)</pre>', re.DOTALL)
_TAG_RE = re.compile(r"<[^>]+>")

# 'ncli> <entity> { op | alias }' prefix of every syntax block. The operation
# group always comes before any parameter, so the first brace group is safe
# to take even though choice values later in the line also use braces.
_SYNTAX_PREFIX_RE = re.compile(r"^ncli>\s*([a-z0-9-]+)\s*\{([^}]*)\}")


def fetch(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.read()


def fetch_entity_pages() -> list[tuple[str, str]]:
    """Return (entity_name, page_id) for every nCLI entity in the book TOC."""
    url = f"{PORTAL_API}/{NCLI_BOOK}%3A{NCLI_TOC_PAGE}"
    meta = json.loads(fetch(url))
    if not meta.get("toc"):
        print(
            f"Error: no TOC in portal response for {NCLI_BOOK}:{NCLI_TOC_PAGE}.\n"
            "The portal API may have changed or the book id may be stale.",
            file=sys.stderr,
        )
        sys.exit(1)

    pages = []
    for entry in meta["toc"]:
        page = entry["path"].split(":", 1)[1]
        m = re.match(r"acl-ncli-(.+)-auto-r\.html$", page)
        # Entity pages sit at level 4 under "nCLI Entities"; the level-3
        # index page acl-ncli-toc-auto-r.html must not match.
        if m and entry.get("level") == 4:
            entity = entry["title"].split(":", 1)[0].strip()
            pages.append((entity, page))
    return pages


def parse_entity_page(entity: str, page: str) -> dict:
    """Parse one entity page into {operation: {aliases, required, optional}}."""
    raw = fetch(f"{PORTAL_API}/{NCLI_BOOK}%3A{page}/html").decode("utf-8")
    operations: dict[str, dict] = {}

    for block in _PRE_BLOCK_RE.findall(raw):
        text = html.unescape(_TAG_RE.sub("", block)).replace("\xa0", " ")
        text = " ".join(text.split())

        m = _SYNTAX_PREFIX_RE.match(text)
        if not m:
            print(
                f"Warning: unparseable syntax block on {entity} page: "
                f"{text[:100]}",
                file=sys.stderr,
            )
            continue
        block_entity, op_group = m.groups()
        if block_entity != entity:
            print(
                f"Warning: entity mismatch on {entity} page: {text[:100]}",
                file=sys.stderr,
            )
            continue

        ops = [o.strip() for o in op_group.split("|") if o.strip()]
        rest = text[m.end():]
        all_params = set(_PARAM_RE.findall(rest))
        optional = set(_OPTIONAL_PARAM_RE.findall(rest))
        required = all_params - optional

        name, aliases = ops[0], ops[1:]
        if name in operations:
            # Same operation documented in multiple blocks: union the
            # parameter sets and keep a param required only if every block
            # requires it.
            existing = operations[name]
            existing["aliases"] = sorted(set(existing["aliases"]) | set(aliases))
            prev_required = set(existing["required"])
            prev_optional = set(existing["optional"])
            new_required = prev_required & required
            new_optional = (prev_optional | optional | prev_required | required) - new_required
            existing["required"] = sorted(new_required)
            existing["optional"] = sorted(new_optional)
        else:
            operations[name] = {
                "aliases": sorted(aliases),
                "required": sorted(required),
                "optional": sorted(optional),
            }

    if not operations:
        print(f"Warning: no operations parsed for entity '{entity}'", file=sys.stderr)
    return operations


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    args = parser.parse_args()

    pages = fetch_entity_pages()
    print(f"Found {len(pages)} nCLI entity pages in {NCLI_BOOK}", file=sys.stderr)

    entities: dict[str, dict] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        futures = {
            pool.submit(parse_entity_page, entity, page): entity
            for entity, page in pages
        }
        for future in concurrent.futures.as_completed(futures):
            entity = futures[future]
            operations = future.result()
            if operations:
                entities[entity] = {"operations": operations}

    result = {
        "_meta": {
            "book": NCLI_BOOK,
            "source": f"{PORTAL_API}/{NCLI_BOOK}:{NCLI_TOC_PAGE}",
            "generated_by": "dump_ncli_commands.py",
        },
        "entities": {k: entities[k] for k in sorted(entities)},
    }

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=False) + "\n")
    op_count = sum(len(e["operations"]) for e in entities.values())
    print(
        f"Wrote {len(entities)} entities, {op_count} operations to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
