#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Backfills the 6 derived compliance frameworks (bsi-grundschutz-sys15,
# csa-cloud-controls-matrix-4, dora, hipaa, pci-dss-4, vda-isa-5) on parent
# checks that already carry the 7 hand-mapped frameworks (iso-27001-2022,
# nis-2, nist-csf-1, nist-csf-2, nist-sp-800-53-rev5, nist-sp-800-171,
# soc2-2017).
#
# How it works:
#   1. Each parent check's existing (iso-27001-2022, nist-csf-2,
#      nist-sp-800-53-rev5) tuple is mapped to one of ~10 categories
#      (ENC_AT_REST, NET_SEG, AUDIT_LOG, AUTHENTICATOR, AUTHZ, HARDENING,
#      PATCHING, BACKUP, INTEGRITY, ALL_FALSE).
#   2. Each category has a 6-framework template derived from the AWS
#      source-of-truth policy (mondoo-aws-security.mql.yaml), where every
#      mapping was verified against the framework YAML in
#      cnspec-enterprise-policies/frameworks/.
#   3. Missing framework keys are filled in from the template; existing keys
#      are never overwritten. Tags are reordered into canonical order on
#      output.
#
# Limitations:
#   - Does NOT hand-map the base 7 frameworks. Author the iso/nis-2/csf/
#     sp-800-53/sp-800-171/soc2 tags first per content/CLAUDE.md.
#   - Categorization is heuristic. Novel (iso, csf2, sp53) combinations may
#     fall through to ALL_FALSE — extend `categorize()` when that happens.
#   - Templates are frozen against the AWS policy at the time this tool was
#     written. Refresh the TEMPLATES dict when framework universes evolve.
#
# Usage:
#   python3 content/validation/backfill_compliance_frameworks.py PATH...
#   python3 content/validation/backfill_compliance_frameworks.py --dry-run PATH...
#   python3 content/validation/backfill_compliance_frameworks.py --all
#
# The --all flag scans every content/mondoo-*-security.mql.yaml file.

import argparse
import glob
import os
import re
import sys
from collections import Counter
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CONTENT_DIR = SCRIPT_DIR.parent

# 6-framework templates by category. Values were derived from the AWS policy
# (mondoo-aws-security.mql.yaml) and verified against the framework definitions
# in cnspec-enterprise-policies/frameworks/. Each template covers exactly the
# 6 compliance keys that the categorizer derives — the other 7 must already
# be hand-mapped.
TEMPLATES = {
    "ENC_AT_REST": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-cek-03",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-a-2-iv-access-control-encryption-and-decryption",
        "compliance/pci-dss-4": "pcidss-requirement-3-5-1",
        "compliance/vda-isa-5": "vda-isa-5-5-1-1",
    },
    "ENC_IN_TRANSIT": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-cek-04",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-e-1-transmission",
        "compliance/pci-dss-4": "pcidss-requirement-4-2-1",
        "compliance/vda-isa-5": "vda-isa-5-5-1-2",
    },
    "NET_SEG": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-ivs-03",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-e-1-transmission",
        "compliance/pci-dss-4": "pcidss-requirement-1-3-1",
        "compliance/vda-isa-5": "vda-isa-5-5-2-6",
    },
    "AUDIT_LOG": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-log-08",
        "compliance/dora": "dora-art-10",
        "compliance/hipaa": "hipaa-security-ss164-312-b-audit-controls",
        "compliance/pci-dss-4": "pcidss-requirement-10-2-1",
        "compliance/vda-isa-5": "vda-isa-5-5-2-4",
    },
    "AUTHENTICATOR": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-iam-02",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-d-person-or-entity-authentication",
        "compliance/pci-dss-4": "pcidss-requirement-8-3-1",
        "compliance/vda-isa-5": "vda-isa-5-4-1-3",
    },
    "AUTHZ": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-iam-04",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-a-1-access-control",
        "compliance/pci-dss-4": "pcidss-requirement-7-2-1",
        "compliance/vda-isa-5": "vda-isa-5-4-1-1",
    },
    "HARDENING": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-ccc-06",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "false",
        "compliance/pci-dss-4": "pcidss-requirement-2-2-1",
        "compliance/vda-isa-5": "vda-isa-5-5-2-1",
    },
    "PATCHING": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-tvm-03",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "false",
        "compliance/pci-dss-4": "pcidss-requirement-6-3-3",
        "compliance/vda-isa-5": "vda-isa-5-5-2-1",
    },
    "BACKUP": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "cloud-controls-matrix-4-bcr-08",
        "compliance/dora": "dora-art-12",
        "compliance/hipaa": "hipaa-security-ss164-308-a-7-ii-a-contingency-plan-data-backup-plan",
        "compliance/pci-dss-4": "false",
        "compliance/vda-isa-5": "false",
    },
    "INTEGRITY": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "false",
        "compliance/dora": "dora-art-9",
        "compliance/hipaa": "hipaa-security-ss164-312-c-1-integrity",
        "compliance/pci-dss-4": "false",
        "compliance/vda-isa-5": "false",
    },
    "ALL_FALSE": {
        "compliance/bsi-grundschutz-sys15": "false",
        "compliance/csa-cloud-controls-matrix-4": "false",
        "compliance/dora": "false",
        "compliance/hipaa": "false",
        "compliance/pci-dss-4": "false",
        "compliance/vda-isa-5": "false",
    },
}

# Canonical output order for compliance/* keys. Mirrors the order used in
# fully-tagged AWS/Azure parent checks.
CANONICAL_ORDER = [
    "compliance/bsi-grundschutz-sys15",
    "compliance/csa-cloud-controls-matrix-4",
    "compliance/dora",
    "compliance/hipaa",
    "compliance/iso-27001-2022",
    "compliance/nis-2",
    "compliance/nist-csf-1",
    "compliance/nist-csf-2",
    "compliance/nist-sp-800-53-rev5",
    "compliance/nist-sp-800-171",
    "compliance/pci-dss-4",
    "compliance/soc2-2017",
    "compliance/vda-isa-5",
]


def categorize(iso, csf2, sp53):
    """Map a (iso, csf2, sp53) signature to a 6-framework template category.

    Returns one of the keys in TEMPLATES. Falls back to ALL_FALSE for novel
    combinations — extend this function with new rules when that happens.
    """
    iso = iso.strip('"').strip("'")
    csf2 = csf2.strip('"').strip("'")
    sp53 = sp53.strip('"').strip("'")

    if iso == "false" and csf2 == "false" and sp53 == "false":
        return "ALL_FALSE"

    # Backup
    if iso in ("iso-27001-2022-a-8-13", "iso-27001-2022-a-8-14"):
        return "BACKUP"

    # Malware protection
    if iso == "iso-27001-2022-a-8-7":
        return "PATCHING"

    # Vulnerability management
    if iso == "iso-27001-2022-a-8-8":
        return "PATCHING"
    if iso == "iso-27001-2022-a-8-25":
        if sp53 == "nist-sp-800-53-rev5-sa-10":
            return "ENC_AT_REST"
        return "PATCHING"

    # Encryption (a-8-24)
    if iso == "iso-27001-2022-a-8-24":
        if csf2 == "nist-csf-2-pr-ds-01":
            if sp53 == "nist-sp-800-53-rev5-sc-7":
                return "ENC_IN_TRANSIT"
            return "ENC_AT_REST"
        if csf2 == "nist-csf-2-pr-ds-02":
            if sp53 == "nist-sp-800-53-rev5-sc-28":
                return "ENC_AT_REST"
            if sp53 in ("nist-sp-800-53-rev5-sc-12", "nist-sp-800-53-rev5-sc-17"):
                return "ENC_AT_REST"
            return "ENC_IN_TRANSIT"
        if csf2 in ("nist-csf-2-pr-aa-01", "nist-csf-2-pr-aa-02", "nist-csf-2-pr-aa-03"):
            return "AUTHENTICATOR"

    # Network segregation / boundary
    if iso in ("iso-27001-2022-a-8-22", "iso-27001-2022-a-8-23", "iso-27001-2022-a-8-12"):
        return "NET_SEG"

    # Networks security (a-8-20)
    if iso == "iso-27001-2022-a-8-20":
        if csf2 == "nist-csf-2-pr-ir-01":
            return "NET_SEG"
        if csf2 == "nist-csf-2-pr-ds-02":
            return "ENC_IN_TRANSIT"
        if csf2 == "nist-csf-2-pr-aa-05":
            if sp53 in ("nist-sp-800-53-rev5-ac-3", "nist-sp-800-53-rev5-ac-4"):
                return "AUTHZ"
            if sp53 == "nist-sp-800-53-rev5-cm-7":
                return "HARDENING"
            if sp53 == "nist-sp-800-53-rev5-sc-5":
                return "NET_SEG"
            if sp53 in ("nist-sp-800-53-rev5-sc-7", "nist-sp-800-53-rev5-sc-10"):
                return "NET_SEG"
        return "NET_SEG"

    # Logging / monitoring
    if iso in ("iso-27001-2022-a-8-15", "iso-27001-2022-a-8-16", "iso-27001-2022-a-8-17"):
        return "AUDIT_LOG"

    # Identity & access control (a-5-15/17/18, a-8-2/3)
    if iso in ("iso-27001-2022-a-5-15", "iso-27001-2022-a-5-17", "iso-27001-2022-a-5-18",
               "iso-27001-2022-a-8-2", "iso-27001-2022-a-8-3"):
        if csf2 == "nist-csf-2-pr-aa-03":
            return "AUTHZ"
        if csf2 == "nist-csf-2-pr-aa-01":
            if sp53 == "nist-sp-800-53-rev5-ia-5":
                return "AUTHENTICATOR"
            return "AUTHZ"
        return "AUTHZ"

    # MFA / authenticator (a-8-5)
    if iso == "iso-27001-2022-a-8-5":
        return "AUTHENTICATOR"

    # Concurrent session control
    if iso == "iso-27001-2022-a-8-6":
        return "AUTHZ"

    # Session timeout / formal access (a-8-1)
    if iso == "iso-27001-2022-a-8-1":
        return "AUTHENTICATOR"

    # Configuration / hardening (a-8-9)
    if iso == "iso-27001-2022-a-8-9":
        if csf2 == "nist-csf-2-pr-ds-01" and sp53 == "nist-sp-800-53-rev5-sc-28":
            return "ENC_AT_REST"
        if csf2 == "nist-csf-2-de-cm-01" and sp53 == "nist-sp-800-53-rev5-sc-8":
            return "ENC_IN_TRANSIT"
        if csf2 == "false" and sp53 == "nist-sp-800-53-rev5-cm-3":
            return "HARDENING"
        if csf2 in ("nist-csf-2-pr-ps-01", "nist-csf-2-pr-ps-05"):
            return "HARDENING"
        if csf2 == "nist-csf-2-pr-ds-02":
            return "HARDENING"
        if csf2 == "nist-csf-2-pr-aa-03":
            return "AUTHENTICATOR"
        if csf2 == "nist-csf-2-pr-aa-05":
            return "AUTHZ"
        if csf2 == "nist-csf-2-de-cm-09":
            return "AUDIT_LOG"
        if csf2 == "nist-csf-2-pr-ps-06":
            return "INTEGRITY"

    # Software installation control
    if iso == "iso-27001-2022-a-8-19":
        return "HARDENING"

    # Supplier-relationship access enforcement
    if iso == "iso-27001-2022-a-5-19":
        if csf2 == "nist-csf-2-gv-sc-07" and sp53 == "nist-sp-800-53-rev5-ac-3":
            return "AUTHZ"
        return "ALL_FALSE"

    # Supply chain risk, asset inventory, identity proofing — no fit in the
    # 6 derived frameworks per content/CLAUDE.md.
    if iso in ("iso-27001-2022-a-5-9", "iso-27001-2022-a-5-16",
               "iso-27001-2022-a-5-21", "iso-27001-2022-a-5-22"):
        return "ALL_FALSE"

    # Services security (a-8-21)
    if iso == "iso-27001-2022-a-8-21":
        if csf2 == "nist-csf-2-pr-aa-01":
            return "AUTHENTICATOR"

    return "ALL_FALSE"


def process_file(path, dry_run=False):
    """Apply backfill to one policy file. Returns (counts, byte_delta, parents_total)."""
    with open(path) as f:
        text = f.read()
    lines = text.split("\n")
    out_lines = []
    pol = os.path.basename(path).replace(".mql.yaml", "")
    counts = Counter()
    pending_uid = None
    parents_total = 0
    i = 0

    while i < len(lines):
        line = lines[i]
        m_uid = re.match(r"^  - uid: (\S+)$", line)
        if m_uid:
            pending_uid = m_uid.group(1) if m_uid.group(1) != pol else None

        if line == "    tags:" and pending_uid:
            tag_lines = []
            j = i + 1
            while j < len(lines) and re.match(r"^      \S", lines[j]):
                tag_lines.append(lines[j])
                j += 1

            existing = {}
            non_compliance = []
            for tl in tag_lines:
                m = re.match(r"^      (\S+):\s*(\S.*)$", tl)
                if not m:
                    non_compliance.append(tl)
                    continue
                key, val = m.group(1), m.group(2).strip()
                if key.startswith("compliance/"):
                    existing[key] = val
                else:
                    non_compliance.append(tl)

            if not existing:
                out_lines.append(line)
                i += 1
                continue

            parents_total += 1
            sig = (
                existing.get("compliance/iso-27001-2022", "false"),
                existing.get("compliance/nist-csf-2", "false"),
                existing.get("compliance/nist-sp-800-53-rev5", "false"),
            )
            cat = categorize(*sig)
            counts[cat] += 1

            for k, v in TEMPLATES[cat].items():
                if k not in existing:
                    existing[k] = v

            out_lines.append(line)
            for nc in non_compliance:
                out_lines.append(nc)
            for k in CANONICAL_ORDER:
                if k in existing:
                    out_lines.append(f"      {k}: {existing[k]}")
            i = j
            continue

        out_lines.append(line)
        i += 1

    new_text = "\n".join(out_lines)
    # Normalize legacy "false" (quoted) to false (unquoted) per content/CLAUDE.md.
    new_text = re.sub(
        r'^(\s+compliance/[a-z0-9.\-]+:) "false"$',
        r'\1 false',
        new_text,
        flags=re.MULTILINE,
    )

    byte_delta = len(new_text) - len(text)
    if not dry_run and new_text != text:
        with open(path, "w") as f:
            f.write(new_text)
    return counts, byte_delta, parents_total


def main():
    parser = argparse.ArgumentParser(
        description="Backfill the 6 derived compliance frameworks "
                    "(bsi-grundschutz-sys15, csa-cloud-controls-matrix-4, "
                    "dora, hipaa, pci-dss-4, vda-isa-5) on parent checks "
                    "that already carry the 7 hand-mapped frameworks."
    )
    parser.add_argument("paths", nargs="*", help="Policy file paths to process")
    parser.add_argument("--all", action="store_true",
                        help="Process every content/mondoo-*-security.mql.yaml file")
    parser.add_argument("--dry-run", action="store_true",
                        help="Report what would change without writing files")
    args = parser.parse_args()

    if args.all:
        paths = sorted(glob.glob(str(CONTENT_DIR / "mondoo-*-security.mql.yaml")))
    else:
        paths = args.paths

    if not paths:
        parser.error("provide PATH... or --all")

    grand_total = Counter()
    total_parents = 0
    total_delta = 0
    for path in paths:
        counts, byte_delta, parents = process_file(path, dry_run=args.dry_run)
        grand_total.update(counts)
        total_parents += parents
        total_delta += byte_delta
        prefix = "[dry-run] " if args.dry_run else ""
        cats = ", ".join(f"{k}={v}" for k, v in counts.most_common()) or "(none)"
        print(f"{prefix}{os.path.basename(path)}: {parents} parents, "
              f"{byte_delta:+d} bytes  [{cats}]")

    print(f"\nTotal: {total_parents} parents across {len(paths)} files, "
          f"{total_delta:+d} bytes")
    for cat, n in grand_total.most_common():
        print(f"  {cat}: {n}")


if __name__ == "__main__":
    main()
