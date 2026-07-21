#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Regenerates content/validation/cmd_data/alicloud_commands.json — the
# checked-in command database used by the `alicloud` remediation validator.
#
# The aliyun CLI has no local, machine-readable command tree we can walk
# (unlike doctl/kubectl's Cobra completion, or the botocore models bundled
# with the AWS CLI). Its command surface is Alibaba Cloud OpenAPI: every RPC
# product is `aliyun <product> <ApiAction> --<Param> <value>`. So this script
# sources the command set two ways:
#
#   1. RPC/ROA products (ecs, ram, kms, ...): the public OpenAPI metadata
#      service at api.aliyun.com. `products.json` maps a product code to its
#      default version; `<product>/<version>/api-docs.json` lists every API
#      action and its parameters (including nested object/array properties,
#      which the CLI accepts as flat flags).
#
#   2. `aliyun oss`: NOT an OpenAPI RPC product. The oss subcommand is an
#      embedded ossutil, whose commands (bucket-encryption, logging, ...) come
#      from the aliyun-cli source tree, not the OSS OpenAPI. We read the
#      command names out of aliyun/aliyun-cli's oss/lib/*.go `name:` fields.
#
# Run this when adding aliyun commands for a new product/action, or to refresh
# against upstream API changes. Requires network access; no CLI install and no
# credentials needed.

import json
import re
import sys
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

META = "https://api.aliyun.com/meta/v1"
CLI_RAW = "https://raw.githubusercontent.com/aliyun/aliyun-cli/master"
CLI_API = "https://api.github.com/repos/aliyun/aliyun-cli/contents"
CMD_DATA = Path(__file__).parent / "cmd_data" / "alicloud_commands.json"

# CLI product names referenced by the Alibaba policy. Keep in sync with the
# products that appear in `aliyun <product> ...` remediation/audit commands.
PRODUCTS = [
    "ecs", "ram", "kms", "vpc", "rds", "polardb", "nas", "dds", "slb",
    "actiontrail", "ddoscoo", "cloudfw", "cs", "r-kvstore", "waf-openapi",
]


def _get(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": "cnspec-validation"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.read()
    except Exception as e:
        raise RuntimeError(f"failed to fetch {url}: {e}") from e


def _get_json(url: str):
    return json.loads(_get(url))


def collect_param_names(node, out: set) -> None:
    """Recursively collect every parameter/property name in an API's parameter
    tree. The aliyun CLI accepts nested object properties and array item
    properties as flat `--Name` flags, so a lenient recursive name set matches
    real-world usage without enforcing exact nesting."""
    if isinstance(node, dict):
        # OpenAPI-meta parameter entries carry a top-level "name".
        name = node.get("name")
        if isinstance(name, str) and name:
            out.add(name)
        # Object properties are keyed by their parameter name.
        props = node.get("properties")
        if isinstance(props, dict):
            for k, v in props.items():
                out.add(k)
                collect_param_names(v, out)
        for key in ("schema", "items"):
            if key in node:
                collect_param_names(node[key], out)
        if "parameters" in node:
            collect_param_names(node["parameters"], out)
    elif isinstance(node, list):
        for item in node:
            collect_param_names(item, out)


def fetch_product(cli_name: str, code: str, version: str) -> dict:
    url = f"{META}/products/{code}/versions/{version}/api-docs.json"
    doc = _get_json(url)
    style = (doc.get("info") or {}).get("style", "RPC")
    actions = {}
    for action, spec in (doc.get("apis") or {}).items():
        names: set = set()
        collect_param_names(spec.get("parameters", []), names)
        actions[action] = sorted(names)
    print(f"  {cli_name:14} {code}/{version}  style={style}  actions={len(actions)}", file=sys.stderr)
    return {"style": style, "actions": actions}


def fetch_oss_subcommands() -> list:
    """Read the embedded-ossutil command names from aliyun-cli's oss/lib/*.go."""
    listing = _get_json(f"{CLI_API}/oss/lib")
    go_files = [
        f["name"] for f in listing
        if f["name"].endswith(".go") and not f["name"].endswith("_test.go")
    ]
    name_re = re.compile(r'name:\s*"([a-z0-9][a-z0-9-]*)"')

    def one(fn: str):
        try:
            text = _get(f"{CLI_RAW}/oss/lib/{fn}").decode("utf-8", "replace")
        except Exception:
            return []
        # findall (not search): a single .go file may register more than one
        # command, and we want every `name:` it defines, not just the first.
        return name_re.findall(text)

    names = set()
    with ThreadPoolExecutor(max_workers=16) as pool:
        for found in pool.map(one, go_files):
            names.update(found)
    print(f"  oss (ossutil)  subcommands={len(names)}", file=sys.stderr)
    return sorted(names)


def main() -> None:
    print("Fetching Alibaba Cloud OpenAPI product metadata...", file=sys.stderr)
    products_meta = _get_json(f"{META}/products.json")
    by_lc = {p["code"].lower(): p for p in products_meta}

    products: dict = {}
    for cli_name in PRODUCTS:
        meta = by_lc.get(cli_name)
        if not meta:
            print(f"  WARNING: no OpenAPI metadata product for '{cli_name}'", file=sys.stderr)
            continue
        version = meta.get("defaultVersion")
        products[cli_name] = fetch_product(cli_name, meta["code"], version)

    print("Fetching aliyun oss (ossutil) subcommands...", file=sys.stderr)
    oss_subcommands = fetch_oss_subcommands()

    out = {
        "_comment": "Generated by dump_alicloud_commands.py from api.aliyun.com "
                    "OpenAPI metadata and aliyun/aliyun-cli oss source. Do not hand-edit.",
        "products": products,
        "oss_subcommands": oss_subcommands,
    }
    CMD_DATA.parent.mkdir(parents=True, exist_ok=True)
    CMD_DATA.write_text(json.dumps(out, indent=1, sort_keys=True) + "\n")
    print(f"Wrote {CMD_DATA} ({CMD_DATA.stat().st_size} bytes)", file=sys.stderr)


if __name__ == "__main__":
    main()
