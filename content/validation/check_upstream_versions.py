#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Reports which of the remediation validators' pinned upstreams have moved.
#
# The validators are only as current as the things they check against: a linter
# release, a vendor CLI grammar, an OpenAPI spec. Every one of those is pinned —
# in a `run:` step of validate-remediation.yaml, in a `_SHA` constant in
# validators/openapi.py, or as a `_meta` version inside cmd_data/ — and none of
# them are watched by Dependabot, which covers gomod and github-actions only.
#
# This script reads each pin from the file that declares it (never a second
# copy, so it cannot itself go stale) and asks the upstream what the current
# version is. It is a report, not a gate: a bump is a judgement call, and a
# GitHub API blip must not fail a build.
#
# Usage:
#   python3 content/validation/check_upstream_versions.py
#   python3 content/validation/check_upstream_versions.py --format markdown
#   python3 content/validation/check_upstream_versions.py --exit-code   # 1 if behind

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
REPO_ROOT = SCRIPT_DIR.parent.parent
WORKFLOW = REPO_ROOT / ".github" / "workflows" / "validate-remediation.yaml"
OPENAPI_MODULE = SCRIPT_DIR / "validators" / "openapi.py"
CMD_DATA = SCRIPT_DIR / "cmd_data"

USER_AGENT = "cnspec-validation-drift-check"
TIMEOUT = 30

# This makes up to 17 api.github.com calls per run. Unauthenticated that shares
# a 60/hour budget with everything else on the runner's IP, so a rate-limited
# run would report "could not reach upstream" for most pins and look like
# everything is fine. With a token the limit is 5,000/hour. The workflow passes
# the one Actions already provides.
GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN", "").strip()


@dataclass
class Result:
    name: str
    kind: str
    pinned: str
    latest: str
    note: str = ""

    # Set for a pin whose upstream publishes no machine-readable version, so an
    # absent `latest` reads as "nothing to compare" rather than "network down".
    manual: bool = False

    @property
    def state(self) -> str:
        if self.manual:
            return "manual"
        if self.pinned == "unknown":
            return "unstamped"
        if self.latest == "unknown":
            return "unchecked"
        return "current" if self.pinned == self.latest else "behind"


def fetch_json(url: str) -> dict | list | None:
    headers = {"User-Agent": USER_AGENT}
    if GITHUB_TOKEN and "api.github.com" in url:
        headers["Authorization"] = f"Bearer {GITHUB_TOKEN}"
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            return json.load(resp)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        # Upstream metadata being unreachable is not a finding about our repo.
        return None


def latest_github_release(repo: str) -> str:
    data = fetch_json(f"https://api.github.com/repos/{repo}/releases/latest")
    if not isinstance(data, dict) or not data.get("tag_name"):
        return "unknown"
    return str(data["tag_name"]).lstrip("v")


def latest_pypi(pkg: str) -> str:
    data = fetch_json(f"https://pypi.org/pypi/{pkg}/json")
    if not isinstance(data, dict):
        return "unknown"
    return str(data.get("info", {}).get("version", "unknown"))


def latest_rubygem(gem: str) -> str:
    data = fetch_json(f"https://rubygems.org/api/v1/gems/{gem}.json")
    if not isinstance(data, dict):
        return "unknown"
    return str(data.get("version", "unknown"))


def latest_gitlab_release(project: str) -> str:
    """Latest release of a GitLab-hosted project. glab lives on gitlab.com and
    the GitHub mirror publishes no releases, so `releases/latest` there 404s."""
    data = fetch_json(f"https://gitlab.com/api/v4/projects/{project}/releases?per_page=1")
    if not isinstance(data, list) or not data:
        return "unknown"
    return str(data[0].get("tag_name", "unknown")).lstrip("v")


def latest_npm(pkg: str) -> str:
    data = fetch_json(f"https://registry.npmjs.org/{pkg}/latest")
    if not isinstance(data, dict):
        return "unknown"
    return str(data.get("version", "unknown"))


def head_commit_for_path(repo: str, path: str) -> str:
    data = fetch_json(f"https://api.github.com/repos/{repo}/commits?path={path}&per_page=1")
    if not isinstance(data, list) or not data:
        return "unknown"
    return str(data[0].get("sha", "unknown"))


# ---------------------------------------------------------------------------
# Pins declared in the workflow's `run:` steps
# ---------------------------------------------------------------------------

# name -> (env var or literal pattern in validate-remediation.yaml, upstream lookup)
WORKFLOW_TOOLS = [
    # (display name, regex capturing the pinned version, resolver)
    ("cfn-lint", r"pipx install cfn-lint==([0-9][^\s]*)", lambda: latest_pypi("cfn-lint")),
    ("ansible-lint", r"pipx install ansible-lint==([0-9][^\s]*)", lambda: latest_pypi("ansible-lint")),
    ("cookstyle", r"gem install cookstyle -v ([0-9][^\s]*)", lambda: latest_rubygem("cookstyle")),
    ("tflint", r"tflint_version:\s*v?([0-9][^\s]*)", lambda: latest_github_release("terraform-linters/tflint")),
    ("bicep", r"BICEP_VERSION:\s*\"?([0-9][^\"\s]*)", lambda: latest_github_release("Azure/bicep")),
    ("doctl", r"DOCTL_VERSION:\s*\"?([0-9][^\"\s]*)", lambda: latest_github_release("digitalocean/doctl")),
    ("glab", r"GLAB_VERSION:\s*\"?([0-9][^\"\s]*)", lambda: latest_gitlab_release("gitlab-org%2Fcli")),
    ("hcloud", r"HCLOUD_VERSION:\s*\"?([0-9][^\"\s]*)", lambda: latest_github_release("hetznercloud/cli")),
    ("databricks", r"DATABRICKS_VERSION:\s*\"?([0-9][^\"\s]*)", lambda: latest_github_release("databricks/cli")),
]


def check_workflow_tools() -> list[Result]:
    if not WORKFLOW.exists():
        return []
    text = WORKFLOW.read_text()
    out = []
    for name, pattern, resolver in WORKFLOW_TOOLS:
        m = re.search(pattern, text)
        pinned = m.group(1) if m else "unknown"
        note = "" if m else "no pin found in validate-remediation.yaml"
        out.append(Result(name, "linter", pinned, resolver() if m else "unknown", note))
    return out


# ---------------------------------------------------------------------------
# OpenAPI specs pinned to a commit SHA in validators/openapi.py
# ---------------------------------------------------------------------------

SPEC_PINS = [
    ("cloudflare spec", "CLOUDFLARE_OPENAPI_SHA", "cloudflare/api-schemas", "openapi.json"),
    ("slack spec", "SLACK_OPENAPI_SHA", "slackapi/slack-api-specs", "web-api/slack_web_openapi_v2.json"),
    ("grafana spec", "GRAFANA_OPENAPI_SHA", "grafana/grafana", "public/openapi3.json"),
    ("mongodbatlas spec", "MONGODBATLAS_OPENAPI_SHA", "mongodb/openapi", "openapi/v2.json"),
]


def check_spec_pins() -> list[Result]:
    if not OPENAPI_MODULE.exists():
        return []
    text = OPENAPI_MODULE.read_text()
    out = []
    for name, const, repo, path in SPEC_PINS:
        m = re.search(rf'{const}\s*=\s*"([0-9a-f]{{7,40}})"', text)
        if not m:
            out.append(Result(name, "spec", "unknown", "unknown", f"{const} not found"))
            continue
        pinned = m.group(1)
        head = head_commit_for_path(repo, path)
        # A differing SHA is not by itself proof of drift: some of these specs
        # live in repos whose default branch was rewritten, so the newest commit
        # touching the path can predate the pin. Ask GitHub how many commits
        # actually separate them and treat "none" as current.
        if head != "unknown" and head != pinned:
            cmp = fetch_json(f"https://api.github.com/repos/{repo}/compare/{pinned}...{head}")
            if isinstance(cmp, dict) and cmp.get("total_commits") == 0:
                head = pinned
        out.append(Result(
            name, "spec", pinned[:10],
            head[:10] if head != "unknown" else "unknown",
            f"{repo}/{path}",
        ))
    return out


# ---------------------------------------------------------------------------
# Checked-in grammars, versioned by their own `_meta`
# ---------------------------------------------------------------------------

def meta_of(filename: str) -> dict:
    path = CMD_DATA / filename
    if not path.exists():
        return {}
    try:
        data = json.loads(path.read_text())
    except json.JSONDecodeError:
        return {}
    meta = data.get("_meta") if isinstance(data, dict) else None
    return meta if isinstance(meta, dict) else {}


def check_grammars() -> list[Result]:
    out = []

    azure = meta_of("azure_commands.json").get("azure_cli_version", "unknown")
    out.append(Result(
        "azure CLI grammar", "grammar", azure, latest_pypi("azure-cli"),
        "regenerate with dump_azure_commands.py",
    ))

    vercel = meta_of("vercel_commands.json").get("vercel_version", "unknown")
    out.append(Result(
        "vercel CLI grammar", "grammar", vercel, latest_npm("vercel"),
        "regenerate with dump_vercel_commands.py",
    ))

    ncli = meta_of("ncli_commands.json").get("book", "unknown")
    out.append(Result(
        "nutanix ncli grammar", "grammar", ncli, "n/a",
        "pinned to an AOS doc book; bump NCLI_BOOK in dump_ncli_commands.py",
        manual=True,
    ))

    return out


# ---------------------------------------------------------------------------
# Output
# ---------------------------------------------------------------------------

STATE_MARK = {
    "behind": "BEHIND",
    "current": "ok",
    "unstamped": "NO VERSION RECORDED",
    "unchecked": "could not reach upstream",
    "manual": "no machine-readable upstream, review by hand",
}


def render_text(results: list[Result]) -> str:
    width = max(len(r.name) for r in results)
    lines = []
    for r in results:
        lines.append(
            f"{r.name:<{width}}  pinned={r.pinned:<12} upstream={r.latest:<12} "
            f"{STATE_MARK[r.state]}"
        )
    return "\n".join(lines)


def render_markdown(results: list[Result]) -> str:
    lines = [
        "| Pinned artifact | Kind | Pinned at | Upstream | State |",
        "| --- | --- | --- | --- | --- |",
    ]
    for r in results:
        lines.append(
            f"| {r.name} | {r.kind} | `{r.pinned}` | `{r.latest}` | {STATE_MARK[r.state]} |"
        )
    return "\n".join(lines)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Report which validator upstreams have moved past their pin"
    )
    parser.add_argument("--format", choices=["text", "markdown"], default="text")
    parser.add_argument(
        "--exit-code",
        action="store_true",
        help="exit 1 when any pin is behind (default: always exit 0, this is a report)",
    )
    args = parser.parse_args()

    results = check_workflow_tools() + check_spec_pins() + check_grammars()

    print(render_markdown(results) if args.format == "markdown" else render_text(results))

    behind = [r for r in results if r.state == "behind"]
    unstamped = [r for r in results if r.state == "unstamped"]
    print(
        f"\n{len(behind)} behind, {len(unstamped)} unstamped, {len(results)} checked",
        file=sys.stderr,
    )
    if args.exit_code and behind:
        sys.exit(1)


if __name__ == "__main__":
    main()
