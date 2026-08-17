#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Dumps OpenAPI specs the REST API validators in
# remediation/commands/openapi.py cannot download at validation time.
#
# Most API providers (Cloudflare, Slack, Grafana, MongoDB Atlas) publish
# their spec as JSON in a git repository, so the validator downloads it at
# validation time from a raw URL pinned to a commit SHA. A spec lands here
# instead for one of two reasons:
#
#   unpinnable  the vendor serves the spec from a live, unversioned
#               endpoint — Tailscale even documents its spec as unstable —
#               so a runtime download could change or break between CI
#               runs. Tailscale, Atlassian and Vercel are these.
#   YAML        the spec is pinnable but published only as YAML, and the
#               validators are stdlib-only (no PyYAML in the validate jobs),
#               so it cannot be parsed at validation time. Okta and
#               Portainer are these, and their entries carry a `pin` naming
#               the exact upstream revision this dump came from.
#
# Either way the checked-in JSON is the stable artifact, refreshed by
# re-running this script — the same model as azure_commands.json and
# ncli_commands.json. Never hand-edit the output files.
#
# Every spec is written pretty-printed with sorted keys, and that is a
# reviewability decision rather than a stylistic one. These files are only ever
# read by a machine, but they are refreshed by an automated weekly pull request
# that a human has to approve, and the whole point of that pull request is to
# show which endpoints, parameters and schemas moved. Two of them used to be
# stored minified to save disk: a one-line 3 MiB file renders as a single
# changed line, which no reviewer and no code-review tool can read, so the
# approval step degraded to trusting the vendor.
#
# The size argument that motivated minifying does not survive being measured.
# Uncompressed the two files grow 3.05 -> 6.61 MiB and 2.67 -> 3.58 MiB, but
# git stores compressed blobs, and there the same change is 0.37 -> 0.54 MiB
# and 0.44 -> 0.48 MiB: about 0.2 MiB total against a repository whose .git is
# already ~390 MiB. Readability is worth four hundredths of a percent.
#
# PyYAML is needed only here, when regenerating.
#
# Usage: python3 content/validation/upstream/dump/api_specs.py

import json
import sys
import urllib.request
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))  # content/validation
from paths import CONTENT_DIR, DATA_DIR, VALIDATION_DIR  # noqa: E402
CMD_DATA_DIR = DATA_DIR

# Pinned revisions for the YAML-only specs. upstream/pins.py watches both and
# reports when upstream has moved; bumping one means editing it here and
# re-running this script.
#
# Okta versions its spec by a dated directory under dist/. There is also a
# `dist/current/`, deliberately not used: it moves, which would make a
# regeneration silently pick up a different spec than the one this dump was
# reviewed against.
OKTA_SPEC_VERSION = "2026.08.0"
# Portainer commits its generated spec only on `develop` — the release tags
# do not carry api/docs/ at all — so this pins the commit, like the raw-URL
# specs in openapi.py.
PORTAINER_SPEC_SHA = "ecb352db9cd0a5b31aa05a10766a4a4e001e8998"

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
        # Vercel serves its spec from a live, unversioned endpoint, so every
        # regeneration can pick up whatever changed since the last one. That
        # makes it the file whose diff most needs to be readable.
        "source": "https://openapi.vercel.sh/",
        "format": "json",
        "output": "vercel_openapi.json",
    },
    {
        # Okta publishes the Management API spec as YAML only. `-minimal` is
        # the variant without the response examples, which the validator never
        # reads; it is ~1 MiB smaller than the full document and identical in
        # paths, methods and schemas.
        "source": (
            "https://raw.githubusercontent.com/okta/okta-management-openapi-spec/"
            f"master/dist/{OKTA_SPEC_VERSION}/management-minimal.yaml"
        ),
        "pin": OKTA_SPEC_VERSION,
        "format": "yaml",
        "output": "okta_openapi.json",
    },
    {
        # Portainer publishes both a Swagger 2.0 (swagger.yaml) and an
        # OpenAPI 3 (openapi.yaml) rendering of the same API. The OpenAPI 3
        # one is used: same 223 paths, and request bodies are expressed as
        # `requestBody` schemas rather than `in: body` parameters.
        "source": (
            "https://raw.githubusercontent.com/portainer/portainer/"
            f"{PORTAINER_SPEC_SHA}/api/docs/openapi.yaml"
        ),
        "pin": PORTAINER_SPEC_SHA,
        "format": "yaml",
        "output": "portainer_openapi.json",
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

    # Regenerating everything rewrites the three specs served from live
    # endpoints too, so a change that only touches one spec arrives as a diff
    # across all of them. Naming one keeps the diff to what you meant.
    wanted = [a for a in sys.argv[1:] if not a.startswith("-")]
    specs = [s for s in SPECS if not wanted or any(w in s["output"] for w in wanted)]
    if not specs:
        names = ", ".join(s["output"].removesuffix("_openapi.json") for s in SPECS)
        print(f"No spec matches {wanted}. Known: {names}", file=sys.stderr)
        sys.exit(2)

    CMD_DATA_DIR.mkdir(exist_ok=True)
    for entry in specs:
        print(f"Fetching {entry['source']} ...", file=sys.stderr)
        raw = fetch(entry["source"])
        if entry["format"] == "yaml":
            spec = yaml.safe_load(raw)
        else:
            spec = json.loads(raw)
        spec["_meta"] = {"source": entry["source"]}
        if "pin" in entry:
            # Stamped so upstream/pins.py can compare it against upstream
            # without re-deriving it from the URL.
            spec["_meta"]["pin"] = entry["pin"]
        out = CMD_DATA_DIR / entry["output"]
        out.write_text(json.dumps(spec, indent=1, sort_keys=True) + "\n")
        print(f"  wrote {out} ({out.stat().st_size // 1024} KiB)", file=sys.stderr)


if __name__ == "__main__":
    main()
