#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# The registry of every upstream the remediation and IaC-variant validators are
# pinned to, and the machinery to read a pin, ask upstream what is current, and
# rewrite the pin in place.
#
# Two scripts consume this module and neither owns a pin list of its own:
#
#   upstream/check.py   reports which pins are behind
#   upstream/bump.py    rewrites the ones that are
#
# The load-bearing rule is that a pin is always read from, and written back to,
# *the file that declares it*. There is no second copy of a version anywhere in
# here, so this module cannot drift out of sync with what CI actually installs.
# The same rule is why the download URL used to re-checksum a CLI tarball is
# extracted from the workflow's own `curl` line rather than restated below: a
# restated URL would silently start checksumming a different artifact than the
# one CI downloads.
#
# What is watched, and where it lives:
#
#   linter / cli  .github/workflows/validate-remediation.yaml
#                 `pipx install x==1.2.3`, `gem install x -v 1.2.3`,
#                 `tflint_version:`, and the `X_VERSION`/`X_SHA256` env pairs
#   tflint-ruleset
#   terraform-provider
#                 content/validation/remediation/code/terraform.py
#                 TFLINT_PLUGIN_MAP and PROVIDER_MAP
#   spec          content/validation/remediation/commands/openapi.py
#                 the `*_OPENAPI_SHA` commit pins
#   grammar       content/validation/data/*.json
#                 stamped in each file's `_meta`; refreshed by re-running the
#                 dump script against the new tool, never by hand
#
# Dependabot covers gomod and github-actions. Nothing else watches any of these.

from __future__ import annotations

import hashlib
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path
from typing import Callable

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))  # content/validation
from paths import DATA_DIR, DUMP_DIR, REPO_ROOT, VALIDATION_DIR  # noqa: E402

WORKFLOW = REPO_ROOT / ".github" / "workflows" / "validate-remediation.yaml"
# Read for the artifact download URLs only. Never written: an installation
# token cannot push a change under .github/workflows/, which is why the pins
# the bumper rewrites live in TOOL_PINS instead.
TOOL_PINS = VALIDATION_DIR / "upstream" / "tool-pins.env"
OPENAPI_MODULE = VALIDATION_DIR / "remediation" / "commands" / "openapi.py"
TERRAFORM_VALIDATOR = VALIDATION_DIR / "remediation" / "code" / "terraform.py"
API_SPECS_DUMP = VALIDATION_DIR / "upstream" / "dump" / "api_specs.py"
PROXMOX_DUMP = VALIDATION_DIR / "upstream" / "dump" / "proxmox.py"
CMD_DATA = DATA_DIR

USER_AGENT = "cnspec-validation-drift-check"
TIMEOUT = 30

# Roughly 45 api.github.com calls per run once the terraform providers and
# tflint rulesets are included. Unauthenticated that shares a 60/hour budget
# with everything else on the runner's IP, so a rate-limited run would report
# "could not reach upstream" for most pins and read as an all-clear. With a
# token the limit is 5,000/hour. Both workflows pass the one Actions provides.
GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN", "").strip()


# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------

# Every value these resolvers return is a version, a release tag, or a commit
# SHA: one short token. Anything else is not something a pin can be compared
# against, and a value carrying a newline would break both the markdown table
# and the `key=value` lines the bump workflow appends to $GITHUB_OUTPUT --
# where extra lines become extra outputs, which are interpolated straight into
# a pull request title and branch. Rejecting a malformed value as "unknown" is
# the fail-safe reading: an upstream we cannot parse stops the bump rather than
# feeding an upstream-controlled string into a pull request.
UPSTREAM_TOKEN = re.compile(r"[A-Za-z0-9][A-Za-z0-9._+-]{0,63}")


def as_token(value: object) -> str:
    text = str(value).strip()
    return text if UPSTREAM_TOKEN.fullmatch(text) else "unknown"


def is_prerelease(version: str) -> bool:
    """True for a version carrying a pre-release marker.

    Semver puts the marker after a hyphen (`2.0.0-beta2`); RubyGems puts it in
    a dotted segment (`0.27.0.beta4`). Both reduce to "something past the
    numeric prefix contains a letter", which is also exactly how
    `Gem::Version#prerelease?` decides. Every released version this module
    resolves today is digits and dots, so nothing legitimate trips it.
    """
    return bool(re.search(r"[A-Za-z]", version.strip().lstrip("vV")))


def stable_token(value: object) -> str:
    """`as_token`, but a pre-release is not a version a pin may be moved to.

    The registries disagree about whether "latest" hides a pre-release.
    rubygems.org's `gems/<name>.json`, npm's `latest` dist-tag, PyPI's
    `info.version` and GitHub's `releases/latest` all exclude one; the Terraform
    registry's `version` field does not. Rather than remember which is which at
    every call site, every version resolver ends here, and a pre-release comes
    back as "unknown" -- the same fail-safe `as_token` already applies to a
    value it cannot parse, and it reads in the report as "could not reach
    upstream" rather than as a bump someone should make.
    """
    token = as_token(value)
    return "unknown" if token != "unknown" and is_prerelease(token) else token


def newest_stable(versions: list[str]) -> str:
    """The highest non-pre-release entry of a registry's version list.

    Ordering is by numeric component rather than by the list's own order: the
    Terraform registry does not document a sort, and a backported patch
    released after a newer minor would otherwise read as the newest thing
    published and report a current pin as behind.
    """
    stable = [v for v in versions if v and not is_prerelease(v)]
    return max(stable, key=lambda v: tuple(int(n) for n in re.findall(r"\d+", v)), default="")


def fetch_json(url: str) -> dict | list | None:
    headers = {"User-Agent": USER_AGENT}
    # Compare the parsed hostname, never a substring of the URL: `"…" in url`
    # also matches a host that merely mentions api.github.com in its path or
    # query, which would hand the token to it. Every URL here is built from
    # constants in this file, so this is hardening rather than a live hole --
    # but a credential-scoping check should not depend on that staying true.
    if GITHUB_TOKEN and urllib.parse.urlparse(url).hostname == "api.github.com":
        headers["Authorization"] = f"Bearer {GITHUB_TOKEN}"
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            return json.load(resp)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        # Upstream metadata being unreachable is not a finding about our repo.
        return None


def fetch_bytes(url: str) -> bytes | None:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT * 4) as resp:
            return resp.read()
    except (urllib.error.URLError, TimeoutError, OSError):
        return None


def latest_github_release(repo: str) -> str:
    data = fetch_json(f"https://api.github.com/repos/{repo}/releases/latest")
    if not isinstance(data, dict) or not data.get("tag_name"):
        return "unknown"
    return stable_token(str(data["tag_name"]).lstrip("v"))


def latest_pypi(pkg: str) -> str:
    data = fetch_json(f"https://pypi.org/pypi/{pkg}/json")
    if not isinstance(data, dict):
        return "unknown"
    return stable_token(data.get("info", {}).get("version", "unknown"))


def latest_rubygem(gem: str) -> str:
    data = fetch_json(f"https://rubygems.org/api/v1/gems/{gem}.json")
    if not isinstance(data, dict):
        return "unknown"
    return stable_token(data.get("version", "unknown"))


def latest_gitlab_release(project: str) -> str:
    """Latest release of a GitLab-hosted project. glab lives on gitlab.com and
    the GitHub mirror publishes no releases, so `releases/latest` there 404s.

    GitLab has no "exclude pre-releases" flag on this endpoint and returns
    releases newest-first, so a page is fetched and the first stable tag taken
    rather than whatever happens to sit at the top.
    """
    data = fetch_json(f"https://gitlab.com/api/v4/projects/{project}/releases?per_page=20")
    if not isinstance(data, list):
        return "unknown"
    for release in data:
        tag = stable_token(str(release.get("tag_name", "")).lstrip("v"))
        if tag != "unknown":
            return tag
    return "unknown"


def latest_npm(pkg: str) -> str:
    data = fetch_json(f"https://registry.npmjs.org/{pkg}/latest")
    if not isinstance(data, dict):
        return "unknown"
    return stable_token(data.get("version", "unknown"))


def latest_terraform_provider(source: str) -> str:
    """Latest *released* version of a Terraform provider, e.g. `hashicorp/aws`.

    The registry's top-level `version` is the newest thing published, including
    a pre-release, and it is the one resolver here whose upstream works that
    way. `aliyun/alicloud` reads `2.0.0-beta2` there while its newest release is
    `1.289.0`, which turned a perfectly current `~> 1.288` pin into a BEHIND row
    proposing a `~> 2.0` constraint that no released provider satisfies. The
    same response carries the full `versions` list, so the newest stable entry
    costs no extra request.
    """
    data = fetch_json(f"https://registry.terraform.io/v1/providers/{source}")
    if not isinstance(data, dict):
        return "unknown"
    versions = data.get("versions")
    if isinstance(versions, list):
        newest = newest_stable([str(v) for v in versions])
        if newest:
            return stable_token(newest)
    return stable_token(data.get("version", "unknown"))


def latest_okta_spec_release() -> str:
    """Newest dated spec directory under the Okta spec repo's dist/.

    Okta ships one directory per release, named YYYY.MM.N, alongside a
    `current` alias and a `legacy-*` tree. Only the dated ones are candidates,
    and they sort numerically rather than lexically: 2026.07.10 is newer than
    2026.07.3, which a string sort gets backwards.
    """
    data = fetch_json(
        "https://api.github.com/repos/okta/okta-management-openapi-spec/contents/dist"
    )
    if not isinstance(data, list):
        return "unknown"
    dated = [e["name"] for e in data
             if e.get("type") == "dir" and re.fullmatch(r"\d{4}\.\d{2}\.\d+", e.get("name", ""))]
    if not dated:
        return "unknown"
    return max(dated, key=lambda n: tuple(int(p) for p in n.split(".")))


def head_commit_for_path(repo: str, path: str) -> str:
    quoted = urllib.parse.quote(path)
    data = fetch_json(f"https://api.github.com/repos/{repo}/commits?path={quoted}&per_page=1")
    if not isinstance(data, list) or not data:
        return "unknown"
    return as_token(data[0].get("sha", "unknown"))


# ---------------------------------------------------------------------------
# Pin
# ---------------------------------------------------------------------------

@dataclass
class Pin:
    name: str
    kind: str
    pinned: str
    latest: str
    note: str = ""

    # Set for a pin whose upstream publishes no machine-readable version, so an
    # absent `latest` reads as "nothing to compare" rather than "network down".
    manual: bool = False

    # Rewrites this pin to `latest` in whichever file declares it and returns
    # the paths it changed. None means the pin has no mechanical bump: a
    # checked-in grammar, for instance, is refreshed by re-running its dump
    # script against the new tool, which needs that tool installed.
    apply: Callable[[str], list[Path]] | None = None

    # Files a bump touches, for the PR body. Filled in by the discoverer.
    files: list[Path] = field(default_factory=list)

    @property
    def state(self) -> str:
        if self.manual:
            return "manual"
        if self.pinned == "unknown":
            return "unstamped"
        if self.latest == "unknown":
            return "unchecked"
        return "current" if self.pinned == self.latest else "behind"

    @property
    def slug(self) -> str:
        """Branch- and matrix-safe identifier."""
        return re.sub(r"[^a-z0-9]+", "-", self.name.lower()).strip("-")

    @property
    def automatable(self) -> bool:
        return self.apply is not None


def _sub_once(text: str, pattern: str, new_value: str, path: Path, flags: int = 0) -> str:
    """Replace group 1 of the first `pattern` match with `new_value`.

    Every rewrite in this module goes through here so that a pattern which
    stopped matching (upstream reformatted the file, someone moved the pin)
    raises instead of writing a file that looks bumped but is not.

    `flags` is for the line-anchored patterns in tool-pins.env, whose `^` means
    nothing without re.MULTILINE -- and which, read with the same flags they
    are matched with here, would otherwise find the pin and then fail to write
    it.
    """
    m = re.search(pattern, text, flags)
    if not m:
        raise ValueError(f"pin pattern no longer matches in {path}: {pattern}")
    start, end = m.span(1)
    return text[:start] + new_value + text[end:]


# ---------------------------------------------------------------------------
# Tool pins declared in upstream/tool-pins.env
# ---------------------------------------------------------------------------
#
# Read from the data file, not from the workflow that installs them, so that
# the weekly bumper can push the result: GitHub refuses a GitHub App
# installation token any push touching `.github/workflows/`, and these are the
# only pins a bot rewrites that ever lived there. `tool-pins.env` has the whole
# argument.
#
# The download URLs stay in the workflow and are read back out of it, which is
# what keeps the re-checksum honest -- the bytes hashed are the bytes CI
# fetches. That is a read of the workflow, never a write.

# Tools installed at a bare version string, with nothing to checksum.
# (display name, env var in tool-pins.env, resolver)
WORKFLOW_TOOLS = [
    ("cfn-lint", "CFN_LINT_VERSION", lambda: latest_pypi("cfn-lint")),
    ("ansible-lint", "ANSIBLE_LINT_VERSION", lambda: latest_pypi("ansible-lint")),
    ("cookstyle", "COOKSTYLE_VERSION", lambda: latest_rubygem("cookstyle")),
    ("tflint", "TFLINT_VERSION", lambda: latest_github_release("terraform-linters/tflint")),
    ("terraform", "TERRAFORM_VERSION", lambda: latest_github_release("hashicorp/terraform")),
]

# Tools downloaded as a release artifact and verified against a checksum.
# `tool-pins.env` declares each as an `X_VERSION` / `X_SHA256` pair; the
# artifact URL is read out of the workflow step that interpolates `X_VERSION`,
# so bumping one of these re-downloads exactly what CI will download and
# recomputes the digest from it.
# (display name, env var prefix, resolver)
WORKFLOW_CHECKSUMMED = [
    ("bicep", "BICEP", lambda: latest_github_release("Azure/bicep")),
    ("doctl", "DOCTL", lambda: latest_github_release("digitalocean/doctl")),
    ("glab", "GLAB", lambda: latest_gitlab_release("gitlab-org%2Fcli")),
    ("hcloud", "HCLOUD", lambda: latest_github_release("hetznercloud/cli")),
    ("databricks", "DATABRICKS", lambda: latest_github_release("databricks/cli")),
    ("stackit", "STACKIT", lambda: latest_github_release("stackitcloud/stackit-cli")),
]


def _pin_pattern(var: str) -> str:
    """Match `VAR=value` on its own line in tool-pins.env.

    Anchored per line so a mention of the name inside a comment cannot be
    rewritten in place of the pin, and so `BICEP_VERSION` cannot match inside
    some future `BICEP_VERSION_SUFFIX`.
    """
    return rf"^{var}=([^\s#]+)"


def _read_pin(text: str, var: str) -> str | None:
    m = re.search(_pin_pattern(var), text, re.MULTILINE)
    return m.group(1) if m else None


def _workflow_url_for(text: str, version_var: str) -> str | None:
    """The quoted download URL in the workflow that interpolates `version_var`.

    Reading the URL back out of the workflow is what keeps the re-checksum
    honest: the bytes hashed here are the bytes CI fetches, even if someone
    later switches a tool to a different asset name or host.
    """
    for m in re.finditer(r'"(https://[^"\s]+)"', text):
        url = m.group(1)
        if "${" + version_var + "}" in url:
            return url
    return None


def check_tool_pins() -> list[Pin]:
    if not TOOL_PINS.exists():
        return []
    text = TOOL_PINS.read_text()
    workflow_text = WORKFLOW.read_text() if WORKFLOW.exists() else ""
    out: list[Pin] = []

    for name, var, resolver in WORKFLOW_TOOLS:
        pinned = _read_pin(text, var)
        if pinned is None:
            out.append(Pin(name, "linter", "unknown", "unknown",
                           f"{var} not found in tool-pins.env"))
            continue

        def make_apply(v: str) -> Callable[[str], list[Path]]:
            def apply(new_version: str) -> list[Path]:
                TOOL_PINS.write_text(
                    _sub_once(TOOL_PINS.read_text(), _pin_pattern(v), new_version,
                              TOOL_PINS, re.MULTILINE)
                )
                return [TOOL_PINS]
            return apply

        out.append(Pin(name, "linter", pinned, resolver(),
                       apply=make_apply(var), files=[TOOL_PINS]))

    for name, prefix, resolver in WORKFLOW_CHECKSUMMED:
        version_var, sha_var = f"{prefix}_VERSION", f"{prefix}_SHA256"
        pinned = _read_pin(text, version_var)
        if pinned is None:
            out.append(Pin(name, "cli", "unknown", "unknown",
                           f"{version_var} not found in tool-pins.env"))
            continue
        url_template = _workflow_url_for(workflow_text, version_var)
        note = "" if url_template else f"no download URL interpolating ${{{version_var}}}"

        # Every loop variable the closure needs is passed in explicitly: bound
        # by reference they would all resolve to the last tool in the list, and
        # a bump would then hash the wrong artifact into the right pin.
        def make_apply(vvar: str, svar: str, template: str) -> Callable[[str], list[Path]]:
            def apply(new_version: str) -> list[Path]:
                url = template.replace("${" + vvar + "}", new_version)
                blob = fetch_bytes(url)
                if not blob:
                    raise ValueError(f"could not download {url} to re-checksum")
                digest = hashlib.sha256(blob).hexdigest()
                body = TOOL_PINS.read_text()
                body = _sub_once(body, _pin_pattern(vvar), new_version, TOOL_PINS, re.MULTILINE)
                body = _sub_once(body, _pin_pattern(svar), digest, TOOL_PINS, re.MULTILINE)
                TOOL_PINS.write_text(body)
                return [TOOL_PINS]
            return apply

        out.append(Pin(
            name, "cli", pinned, resolver(), note,
            apply=(make_apply(version_var, sha_var, url_template)
                   if url_template else None),
            files=[TOOL_PINS],
        ))

    return out


def verify_tool_checksums() -> list[tuple[str, bool, str]]:
    """Re-derive each checksummed CLI's digest at the version it is *already*
    pinned to, and compare it against the digest in tool-pins.env.

    This is the one thing about the auto-bump that is not self-evident from a
    diff: everything else rewrites a string that a reviewer can eyeball, but a
    64-character digest is only meaningful if the bytes it came from are the
    bytes CI will fetch. Running it against the current pins turns that into a
    checkable claim -- a mismatch means either the URL shape moved or the pin
    was wrong all along, and both need a person before any bump is trusted.

    Returns (name, matches, detail) per tool.
    """
    if not TOOL_PINS.exists():
        return []
    text = TOOL_PINS.read_text()
    workflow_text = WORKFLOW.read_text() if WORKFLOW.exists() else ""
    out = []
    for name, prefix, _ in WORKFLOW_CHECKSUMMED:
        version_var, sha_var = f"{prefix}_VERSION", f"{prefix}_SHA256"
        pinned = _read_pin(text, version_var)
        recorded = _read_pin(text, sha_var)
        if not pinned or not recorded:
            out.append((name, False, f"{version_var}/{sha_var} not found in tool-pins.env"))
            continue
        template = _workflow_url_for(workflow_text, version_var)
        if not template:
            out.append((name, False, f"no download URL interpolating ${{{version_var}}}"))
            continue
        url = template.replace("${" + version_var + "}", pinned)
        blob = fetch_bytes(url)
        if blob is None:
            out.append((name, False, f"could not download {url}"))
            continue
        digest = hashlib.sha256(blob).hexdigest()
        out.append((
            name, digest == recorded,
            f"v{pinned} {url}" if digest == recorded
            else f"v{pinned} computed {digest}, pinned {recorded}",
        ))
    return out



# ---------------------------------------------------------------------------
# Terraform tooling declared in remediation/code/terraform.py
# ---------------------------------------------------------------------------

def _parse_pair_map(text: str, map_name: str) -> dict[str, tuple[str, str]]:
    """Parse a `NAME = { "key": ("source", "value"), ... }` literal out of source.

    Parsed rather than imported: importing the validator would pull in its
    module-level setup for no reason, and parsing keeps the "read the pin from
    the file that declares it" property that makes this module unable to go
    stale. Anything added to the map is watched automatically.
    """
    m = re.search(rf"^{map_name}\s*=\s*\{{(.*?)^\}}", text, re.DOTALL | re.MULTILINE)
    if not m:
        return {}
    return {
        key: (source, value)
        for key, source, value in re.findall(
            r'"([^"]+)":\s*\("([^"]+)",\s*"([^"]+)"\)', m.group(1)
        )
    }


def _entry_pattern(key: str, source: str) -> str:
    """Regex capturing the version of one `"key": ("source", "version")` entry."""
    return rf'"{re.escape(key)}":\s*\(\s*"{re.escape(source)}"\s*,\s*"([^"]+)"'


def constraint_for(version: str) -> str:
    """The `~>` constraint this repo writes when a provider outgrows its pin.

    House style is the floor of the release's own line: `~> 6.0` for 6.60.0,
    `~> 0.14` for a 0.14.x. It is only consulted for a provider whose current
    constraint can no longer resolve the newest release, so what it produces is
    always the *next* line rather than a restatement of the current one.
    """
    parts = version.split(".")
    if len(parts) < 2:
        return f"~> {version}"
    major, minor = parts[0], parts[1]
    return f"~> 0.{minor}" if major == "0" else f"~> {major}.0"


def constraint_admits(constraint: str, version: str) -> bool:
    """Whether a `~>` constraint already resolves to `version`.

    This is the question the report needs answered, and it is not the same as
    "is the constraint spelled the way `constraint_for` would spell it today".
    Terraform's pessimistic operator lets only the rightmost component the
    constraint names increment, so `~> 6.0` covers all of 6.x, `~> 1.288` covers
    every 1.x from 1.288 up, and `~> 1.0.4` stops at 1.1.0.

    `alicloud` is why the distinction matters. Its `~> 1.288` is a deliberate
    pin with a comment in `terraform.py` explaining it: the 2.x line is
    beta-only, and `~> 2.0` resolved to nothing at all. Comparing constraint
    text reported that pin behind every week and had the auto-bump workflow
    offering to reinstate the exact constraint someone had already removed for
    cause. Asking whether the pin resolves 1.289.0 answers "no action needed".

    The same rule means a 0.x provider reports current until its 1.0: `~> 0.14`
    genuinely does resolve 0.15, so rewriting it to `~> 0.15` changes which
    provider Terraform selects not at all.
    """
    m = re.fullmatch(r"~>\s*(\d+(?:\.\d+)*)", constraint.strip())
    if not m:
        return False
    bound = [int(p) for p in m.group(1).split(".")]
    actual = [int(p) for p in re.findall(r"\d+", version)]
    if not actual:
        return False

    # Everything left of the rightmost named component is frozen, so the
    # exclusive ceiling is that neighbour incremented. A one-component
    # constraint freezes nothing and has no ceiling.
    ceiling = bound[:-2] + [bound[-2] + 1] if len(bound) > 1 else None
    width = max(len(bound), len(actual), len(ceiling or []))

    def pad(v: list[int]) -> list[int]:
        return v + [0] * (width - len(v))

    if pad(actual) < pad(bound):
        return False
    return ceiling is None or pad(actual) < pad(ceiling)


def check_terraform_pins() -> list[Pin]:
    if not TERRAFORM_VALIDATOR.exists():
        return []
    text = TERRAFORM_VALIDATOR.read_text()
    out: list[Pin] = []

    def make_apply(pattern: str) -> Callable[[str], list[Path]]:
        def apply(new_value: str) -> list[Path]:
            TERRAFORM_VALIDATOR.write_text(
                _sub_once(TERRAFORM_VALIDATOR.read_text(), pattern, new_value, TERRAFORM_VALIDATOR)
            )
            return [TERRAFORM_VALIDATOR]
        return apply

    # tflint ruleset plugins are pinned exactly, so pin and upstream compare
    # directly. A ruleset bump is the one that most often turns up new findings
    # in existing snippets: rulesets ship new rules enabled by default.
    for key, (source, version) in _parse_pair_map(text, "TFLINT_PLUGIN_MAP").items():
        repo = source.removeprefix("github.com/")
        out.append(Pin(
            f"tflint-ruleset-{key}", "tflint-ruleset", version,
            latest_github_release(repo), source,
            apply=make_apply(_entry_pattern(key, source)),
            files=[TERRAFORM_VALIDATOR],
        ))

    # Provider entries hold a `~>` constraint, not a version, so a pin is behind
    # when the newest release is one the constraint cannot resolve -- not when
    # the constraint is merely spelled differently to what this repo would write
    # from scratch. hashicorp/aws at 6.60.0 leaves `~> 6.0` alone and makes
    # `~> 5.0` behind; aliyun/alicloud at 1.289.0 leaves the deliberate
    # `~> 1.288` alone, which comparing constraint text did not.
    for key, (source, constraint) in _parse_pair_map(text, "PROVIDER_MAP").items():
        version = latest_terraform_provider(source)
        if version == "unknown":
            latest = "unknown"
        elif constraint_admits(constraint, version):
            latest = constraint
        else:
            latest = constraint_for(version)
        out.append(Pin(
            f"terraform-provider-{key}", "terraform-provider", constraint, latest,
            f"{source} {version}",
            apply=make_apply(_entry_pattern(key, source)),
            files=[TERRAFORM_VALIDATOR],
        ))

    return out


# ---------------------------------------------------------------------------
# OpenAPI specs pinned to a commit SHA in remediation/commands/openapi.py
# ---------------------------------------------------------------------------

SPEC_PINS = [
    ("cloudflare spec", "CLOUDFLARE_OPENAPI_SHA", "cloudflare/api-schemas", "openapi.json"),
    ("slack spec", "SLACK_OPENAPI_SHA", "slackapi/slack-api-specs", "web-api/slack_web_openapi_v2.json"),
    ("grafana spec", "GRAFANA_OPENAPI_SHA", "grafana/grafana", "public/openapi3.json"),
    ("mongodbatlas spec", "MONGODBATLAS_OPENAPI_SHA", "mongodb/openapi", "openapi/v2.json"),
]


def check_spec_pins() -> list[Pin]:
    if not OPENAPI_MODULE.exists():
        return []
    text = OPENAPI_MODULE.read_text()
    out: list[Pin] = []
    for name, const, repo, path in SPEC_PINS:
        pattern = rf'{const}\s*=\s*"([0-9a-f]{{7,40}})"'
        m = re.search(pattern, text)
        if not m:
            out.append(Pin(name, "spec", "unknown", "unknown", f"{const} not found"))
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

        def make_apply(pat: str, full_sha: str) -> Callable[[str], list[Path]]:
            # The report abbreviates SHAs to 10 characters for the table; the
            # file must get the full one it was pinned with.
            def apply(_new: str) -> list[Path]:
                OPENAPI_MODULE.write_text(
                    _sub_once(OPENAPI_MODULE.read_text(), pat, full_sha, OPENAPI_MODULE)
                )
                return [OPENAPI_MODULE]
            return apply

        out.append(Pin(
            name, "spec", pinned[:10],
            head[:10] if head != "unknown" else "unknown",
            f"{repo}/{path}",
            apply=make_apply(pattern, head) if head not in ("unknown", pinned) else None,
            files=[OPENAPI_MODULE],
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


def check_grammars() -> list[Pin]:
    # No `apply` on any of these: the pin is a stamp describing which tool
    # produced the checked-in JSON, so the only correct way to move it is to
    # install that tool at the new version and re-run its dump script. That
    # needs the tool itself, so it happens in a dedicated workflow job rather
    # than in-process here. Never hand-edit the files in data/.
    return [
        Pin(
            "azure CLI grammar", "grammar",
            meta_of("azure_commands.json").get("azure_cli_version", "unknown"),
            latest_pypi("azure-cli"),
            "regenerate with upstream/dump/azure.py",
            files=[CMD_DATA / "azure_commands.json"],
        ),
        Pin(
            "vercel CLI grammar", "grammar",
            meta_of("vercel_commands.json").get("vercel_version", "unknown"),
            latest_npm("vercel"),
            "regenerate with upstream/dump/vercel.py",
            files=[CMD_DATA / "vercel_commands.json"],
        ),
        Pin(
            "nutanix ncli grammar", "grammar",
            meta_of("ncli_commands.json").get("book", "unknown"), "n/a",
            "pinned to an AOS doc book; bump NCLI_BOOK in upstream/dump/ncli.py",
            manual=True,
            files=[CMD_DATA / "ncli_commands.json"],
        ),
    ]


def sync_dump_script_pins() -> list[str]:
    """Point a dump script's own version constant at the grammar it produced.

    A dump script sometimes names the tool version it expects, so a human
    running it locally gets told to install the right one. That constant is the
    only place in this scheme where a version is written down twice -- the
    other copy is the `_meta` stamp in the JSON the script emits -- so after an
    automated regeneration it has to be pulled back into line, or the script
    warns about a mismatch with itself forever.

    Only vercel has one: `upstream/dump/azure.py` reads the installed CLI's
    version at run time, and `NCLI_BOOK` is itself the source of truth.
    """
    changed = []
    version = meta_of("vercel_commands.json").get("vercel_version")
    script = DUMP_DIR / "vercel.py"
    if version and script.exists():
        text = script.read_text()
        pattern = r'VERCEL_VERSION = "([^"]+)"'
        if re.search(pattern, text) and not re.search(rf'VERCEL_VERSION = "{re.escape(version)}"', text):
            script.write_text(_sub_once(text, pattern, version, script))
            changed.append(f"upstream/dump/vercel.py VERCEL_VERSION -> {version}")
    return changed


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def check_spec_dumps() -> list[Pin]:
    """Specs that are pinnable upstream but published only as YAML.

    The raw-URL specs in check_spec_pins() are downloaded at validation time,
    so their pin lives in openapi.py. These are converted to JSON and checked
    into data/ instead, because the validators are stdlib-only and cannot
    parse YAML — so the pin lives in the dump script, and the checked-in file
    carries a copy in its `_meta` recording which revision produced it.

    No `apply`, for the same reason as the grammars: moving the pin means
    re-running the dump against the new revision, which needs PyYAML and the
    network, so it happens in a workflow job rather than in-process here.
    """
    return [
        Pin(
            "okta API spec", "spec-dump",
            meta_of("okta_openapi.json").get("pin", "unknown"),
            latest_okta_spec_release(),
            "bump OKTA_SPEC_VERSION in upstream/dump/api_specs.py and re-run it",
            files=[API_SPECS_DUMP, CMD_DATA / "okta_openapi.json"],
        ),
        Pin(
            "portainer API spec", "spec-dump",
            meta_of("portainer_openapi.json").get("pin", "unknown")[:10],
            head_commit_for_path("portainer/portainer", "api/docs/openapi.yaml")[:10],
            "bump PORTAINER_SPEC_SHA in upstream/dump/api_specs.py and re-run it",
            files=[API_SPECS_DUMP, CMD_DATA / "portainer_openapi.json"],
        ),
        Pin(
            "proxmox API schema", "spec-dump",
            meta_of("proxmox_api.json").get("pin", "unknown")[:10],
            head_commit_for_path("proxmox/pve-docs", "api-viewer/apidata.js")[:10],
            "bump PVE_DOCS_SHA in upstream/dump/proxmox.py and re-run it",
            files=[PROXMOX_DUMP, CMD_DATA / "proxmox_api.json"],
        ),
    ]


def discover() -> list[Pin]:
    """Every watched pin, with its current value and what upstream has."""
    return (
        check_tool_pins()
        + check_terraform_pins()
        + check_spec_pins()
        + check_spec_dumps()
        + check_grammars()
    )
