#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps OpenAPI specs for vendors that don't serve them from a pinnable
# URL, for the REST API validators in validators/openapi.py.
#
# Most API providers (Cloudflare, Slack, Grafana) publish their spec in a
# git repository, so the validator downloads it at validation time from a
# raw URL pinned to a commit SHA. The vendors below serve their spec from
# a live, unversioned endpoint instead — Tailscale even documents its
# spec as unstable — so a runtime download could change or break between
# CI runs. Their specs are checked into cmd_data/ and refreshed by
# re-running this script, the same model as azure_commands.json and
# ncli_commands.json. Never hand-edit the output files.
#
# Tailscale serves YAML; it is converted to JSON so the validators stay
# stdlib-only (PyYAML is needed only here, when regenerating).
#
# Usage: python3 dump_api_specs.py

import json
import sys
import urllib.request
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CMD_DATA_DIR = SCRIPT_DIR / "cmd_data"

SPECS = [
    {
        "source": "https://api.tailscale.com/api/v2?openapi",
        "format": "yaml",
        "output": "tailscale_openapi.json",
    },
    {
        "source": "https://dac-static.atlassian.com/cloud/admin/organization/swagger.v3.json",
        "format": "json",
        "output": "atlassian_org_openapi.json",
    },
    {
        "source": "https://dac-static.atlassian.com/cloud/admin/user-management/swagger.v3.json",
        "format": "json",
        "output": "atlassian_user_management_openapi.json",
    },
    {
        # Vercel serves its spec from a live, unversioned endpoint. It is
        # large (~9.5 MiB pretty-printed), so it is stored minified (~2.9
        # MiB, comparable to azure_commands.json).
        "source": "https://openapi.vercel.sh/",
        "format": "json",
        "output": "vercel_openapi.json",
        "minify": True,
    },
]


def fetch(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": "cnspec-validation"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.read()


def main():
    try:
        import yaml
    except ImportError:
        print(
            "Error: PyYAML is required to regenerate the spec dumps "
            "(the Tailscale spec is served as YAML).\n"
            "Install it with: pip install pyyaml",
            file=sys.stderr,
        )
        sys.exit(1)

    CMD_DATA_DIR.mkdir(exist_ok=True)
    for entry in SPECS:
        print(f"Fetching {entry['source']} ...", file=sys.stderr)
        raw = fetch(entry["source"])
        if entry["format"] == "yaml":
            spec = yaml.safe_load(raw)
        else:
            spec = json.loads(raw)
        spec["_meta"] = {"source": entry["source"]}
        out = CMD_DATA_DIR / entry["output"]
        if entry.get("minify"):
            out.write_text(json.dumps(spec, separators=(",", ":"), sort_keys=True) + "\n")
        else:
            out.write_text(json.dumps(spec, indent=1, sort_keys=True) + "\n")
        print(f"  wrote {out} ({out.stat().st_size // 1024} KiB)", file=sys.stderr)


if __name__ == "__main__":
    main()
