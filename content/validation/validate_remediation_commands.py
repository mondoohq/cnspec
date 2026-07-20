#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates CLI commands found in remediation sections of cnspec policies
# against known-good sets of subcommands and flags.
#
# Usage:
#   python3 validate_remediation_commands.py               # validate all
#   python3 validate_remediation_commands.py aws           # validate AWS commands only
#   python3 validate_remediation_commands.py azure         # validate Azure commands only
#   python3 validate_remediation_commands.py oci           # validate OCI commands only
#   python3 validate_remediation_commands.py gcp           # validate gcloud commands only
#   python3 validate_remediation_commands.py digitalocean  # validate doctl commands only
#   python3 validate_remediation_commands.py alicloud      # validate aliyun commands only
#   python3 validate_remediation_commands.py cloudflare    # validate Cloudflare API curl commands only
#   python3 validate_remediation_commands.py mongodbatlas  # validate MongoDB Atlas API curl commands only
#   python3 validate_remediation_commands.py nutanix       # validate Nutanix ncli commands only

from validators.main import main

if __name__ == "__main__":
    main()
