# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Entry point and validator dispatch.

import sys

from .aws import validate_aws
from .azure import validate_azure
from .cobra import COBRA_CLIS, validate_cobra_cli
from .common import emit_github_annotations
from .digitalocean import validate_digitalocean
from .gcloud import validate_gcloud
from .nutanix import validate_nutanix
from .oci import validate_oci
from .openapi import API_PROVIDERS, validate_api_provider
from .vercel import validate_vercel

# CLI validators, dispatched by name. The Cobra CLI and REST API
# validators come from their registries and are appended below.
CLI_VALIDATORS = {
    "aws": validate_aws,
    "azure": validate_azure,
    "oci": validate_oci,
    "gcp": validate_gcloud,
    "digitalocean": validate_digitalocean,
    "nutanix": validate_nutanix,
    "vercel": validate_vercel,
}

VALIDATORS = list(CLI_VALIDATORS) + list(COBRA_CLIS) + list(API_PROVIDERS)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = sys.argv[1:]
    github_actions = False
    target = "all"

    # Parse flags
    positional = []
    for arg in args:
        if arg == "--github-actions":
            github_actions = True
        else:
            positional.append(arg)

    if positional:
        target = positional[0]

    if target not in ("all", *VALIDATORS):
        print(
            f"Unknown validator: {target}\n"
            f"Usage: {sys.argv[0]} [{'|'.join(['all'] + VALIDATORS)}] [--github-actions]",
            file=sys.stderr,
        )
        sys.exit(2)

    total_pass = 0
    total_fail = 0

    for name, validator in CLI_VALIDATORS.items():
        if target in ("all", name):
            p, f = validator()
            total_pass += p
            total_fail += f

    for name in COBRA_CLIS:
        if target in ("all", name):
            p, f = validate_cobra_cli(name)
            total_pass += p
            total_fail += f

    for name in API_PROVIDERS:
        if target in ("all", name):
            p, f = validate_api_provider(name)
            total_pass += p
            total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
