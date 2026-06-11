# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
# Entry point and validator dispatch.

import sys

from .aws import validate_aws
from .azure import validate_azure
from .common import emit_github_annotations
from .digitalocean import validate_digitalocean
from .gcloud import validate_gcloud
from .nutanix import validate_nutanix
from .oci import validate_oci
from .openapi import validate_cloudflare

VALIDATORS = ["aws", "azure", "oci", "gcp", "digitalocean", "cloudflare", "nutanix"]


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

    if target in ("all", "aws"):
        p, f = validate_aws()
        total_pass += p
        total_fail += f

    if target in ("all", "azure"):
        p, f = validate_azure()
        total_pass += p
        total_fail += f

    if target in ("all", "oci"):
        p, f = validate_oci()
        total_pass += p
        total_fail += f

    if target in ("all", "gcp"):
        p, f = validate_gcloud()
        total_pass += p
        total_fail += f

    if target in ("all", "digitalocean"):
        p, f = validate_digitalocean()
        total_pass += p
        total_fail += f

    if target in ("all", "cloudflare"):
        p, f = validate_cloudflare()
        total_pass += p
        total_fail += f

    if target in ("all", "nutanix"):
        p, f = validate_nutanix()
        total_pass += p
        total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
