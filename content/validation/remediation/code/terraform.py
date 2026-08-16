#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates Terraform HCL code blocks found in remediation sections of cnspec
# policies by running tflint and `terraform validate` against each snippet.
#
# tflint checks style and the security rules its provider rulesets ship. It does
# not resolve a snippet against the provider's own schema, so it cannot see an
# argument the resource does not accept, a required argument left out, or a value
# assigned to a computed attribute. Only aws, azurerm and google have rulesets at
# all, which left the other seventeen providers checked for syntax alone.
# `terraform validate` closes that gap for every provider in PROVIDER_MAP.
#
# Usage:
#   python3 content/validation/remediation/code/terraform.py                # validate all
#   python3 content/validation/remediation/code/terraform.py aws            # validate AWS only
#   python3 content/validation/remediation/code/terraform.py azure          # validate Azure only
#   python3 content/validation/remediation/code/terraform.py gcp            # validate GCP only
#   python3 content/validation/remediation/code/terraform.py --github-actions  # emit GH annotations

import concurrent.futures
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import time
from dataclasses import dataclass, field
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))  # content/validation
from paths import CONTENT_DIR, REPO_ROOT  # noqa: E402

TARGETS = {
    "aws": [CONTENT_DIR / "mondoo-aws-security.mql.yaml"],
    "azure": [CONTENT_DIR / "mondoo-azure-security.mql.yaml"],
    "gcp": [CONTENT_DIR / "mondoo-gcp-security.mql.yaml"],
    "oci": [CONTENT_DIR / "mondoo-oci-security.mql.yaml"],
    "github": [
        CONTENT_DIR / "mondoo-github-security.mql.yaml",
        CONTENT_DIR / "mondoo-github-best-practices.mql.yaml",
    ],
    "gitlab": [CONTENT_DIR / "mondoo-gitlab-security.mql.yaml"],
    "okta": [CONTENT_DIR / "mondoo-okta-security.mql.yaml"],
    "m365": [CONTENT_DIR / "mondoo-m365-security.mql.yaml"],
    # Only aws/azurerm/google have tflint rulesets; the rest are checked by
    # the terraform preset's syntax and declaration rules alone.
    "alibaba": [CONTENT_DIR / "mondoo-alibaba-security.mql.yaml"],
    "cloudflare": [CONTENT_DIR / "mondoo-cloudflare-security.mql.yaml"],
    "databricks": [CONTENT_DIR / "mondoo-databricks-security.mql.yaml"],
    "digitalocean": [CONTENT_DIR / "mondoo-digitalocean-security.mql.yaml"],
    "dns": [CONTENT_DIR / "mondoo-dns-security.mql.yaml"],
    "email": [CONTENT_DIR / "mondoo-email-security.mql.yaml"],
    "hetzner": [CONTENT_DIR / "mondoo-hetzner-security.mql.yaml"],
    "openstack": [CONTENT_DIR / "mondoo-openstack-security.mql.yaml"],
    "portainer": [CONTENT_DIR / "mondoo-portainer-security.mql.yaml"],
    "snowflake": [CONTENT_DIR / "mondoo-snowflake-security.mql.yaml"],
    "stackit": [CONTENT_DIR / "mondoo-stackit-security.mql.yaml"],
    "tailscale": [CONTENT_DIR / "mondoo-tailscale-security.mql.yaml"],
    "unifi": [CONTENT_DIR / "mondoo-unifi-security.mql.yaml"],
    "vercel": [CONTENT_DIR / "mondoo-vercel-security.mql.yaml"],
    "vsphere": [
        CONTENT_DIR / "mondoo-vmware-vsphere.mql.yaml",
        CONTENT_DIR / "mondoo-vmware-vsphere-esxi.mql.yaml",
    ],
}

# Map resource prefix -> (provider source, version constraint)
PROVIDER_MAP = {
    "aws": ("hashicorp/aws", "~> 6.0"),
    "azurerm": ("hashicorp/azurerm", "~> 5.0"),
    "azuread": ("hashicorp/azuread", "~> 3.0"),
    "azapi": ("azure/azapi", "~> 2.0"),
    "google": ("hashicorp/google", "~> 7.0"),
    "google-beta": ("hashicorp/google-beta", "~> 7.0"),
    "oci": ("oracle/oci", "~> 8.0"),
    "github": ("integrations/github", "~> 6.0"),
    "gitlab": ("gitlabhq/gitlab", "~> 19.0"),
    "okta": ("okta/okta", "~> 6.0"),
    "null": ("hashicorp/null", "~> 3.0"),
    "time": ("hashicorp/time", "~> 0.14"),
    # Providers for the policies added to TARGETS above. A prefix missing
    # from this map produces an empty `required_providers` block, which the
    # terraform preset's terraform_required_providers rule fails on every
    # resource — so a new target needs an entry here to be checkable at all.
    # `~> 2.0` matched no released version: the registry carries 2.0.0-beta1 and
    # 2.0.0-beta2 and nothing else in that line, so terraform could not resolve
    # the provider at all. tflint never noticed because it does not resolve
    # versions. Current stable is 1.288.0.
    "alicloud": ("aliyun/alicloud", "~> 1.288"),
    "cloudflare": ("cloudflare/cloudflare", "~> 5.0"),
    "databricks": ("databricks/databricks", "~> 1.0"),
    "digitalocean": ("digitalocean/digitalocean", "~> 2.0"),
    "hcloud": ("hetznercloud/hcloud", "~> 1.0"),
    "openstack": ("terraform-provider-openstack/openstack", "~> 3.0"),
    "portainer": ("portainer/portainer", "~> 1.0"),
    "snowflake": ("snowflakedb/snowflake", "~> 2.0"),
    "stackit": ("stackitcloud/stackit", "~> 0.111"),
    "tailscale": ("tailscale/tailscale", "~> 0.29"),
    "unifi": ("ubiquiti-community/unifi", "~> 0.55"),
    "vercel": ("vercel/vercel", "~> 5.0"),
    "vsphere": ("hashicorp/vsphere", "~> 2.0"),
}

# tflint provider plugins (only for providers that have rulesets)
TFLINT_PLUGIN_MAP = {
    "aws": ("github.com/terraform-linters/tflint-ruleset-aws", "0.48.0"),
    "azurerm": ("github.com/terraform-linters/tflint-ruleset-azurerm", "0.32.0"),
    "google": ("github.com/terraform-linters/tflint-ruleset-google", "0.39.0"),
}

# Ruleset rules that do not apply to documentation snippets, keyed by plugin.
# A remediation example demonstrates the one setting its check is about and is
# meant to be copied into a reader's own configuration, not applied as written,
# so operational-completeness rules only produce noise here. prevent_destroy
# would be worse than noise: following it hands the reader a resource they
# cannot delete until they find and remove the lifecycle block. Neither rule
# touches the security control being demonstrated. Both ship enabled-by-default
# in tflint-ruleset-azurerm 0.29.0+.
DISABLED_RULES = {
    "azurerm": (
        "azurerm_resources_missing_prevent_destroy",
        "azurerm_app_service_missing_auto_heal_setting",
    ),
}

# Providers that need extra config in their provider block
PROVIDER_EXTRA_CONFIG = {
    "azurerm": "  features {}\n",
}

# `tflint --init` reaches GitHub for every ruleset plugin. Retry before calling
# a plugin set unusable, with a growing pause so a rate-limited or briefly
# unreachable release download recovers instead of failing the whole run.
INIT_ATTEMPTS = 3
INIT_RETRY_DELAY = 3  # seconds, multiplied by the attempt number

FAILURES: list[dict] = []


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------

@dataclass
class HclBlock:
    code: str
    line: int
    uid: str
    file: Path


@dataclass
class TflintResult:
    success: bool
    issues: list[str] = field(default_factory=list)


# `terraform validate` diagnostics that describe the documentation convention
# rather than a defect. A remediation snippet shows the one setting its check is
# about and is meant to be copied into a reader's own configuration, so it names
# surrounding resources it does not declare. A single `- id: terraform` may also
# offer two alternative configurations of the same resource in separate fences,
# which extract_hcl_blocks concatenates, producing a duplicate that exists only
# in the harness. Everything else terraform reports is a real defect in the
# snippet: an argument the resource does not accept, a required argument left
# out, a value assigned to a computed attribute, or a dependency cycle.
TF_VALIDATE_IGNORED_SUMMARY_PREFIXES = (
    "Reference to undeclared resource",
    "Reference to undeclared input variable",
    "Reference to undeclared module",
    "Reference to undeclared local value",
    "Duplicate resource ",
    "Duplicate data ",
)

# Diagnostics that are only noise when the harness caused them. Blanket-ignoring
# these summaries hides real defects: a type error on a value the snippet really
# does get wrong reads identically to one on a value this file substituted, and
# a missing required attribute inside a block surfaces as a type error too. Each
# is therefore ignored only on evidence that the harness produced it.
TF_VALIDATE_CONDITIONAL_IGNORES = {
    # Raised where neutralize_dangling_refs put a string in place of a
    # reference, so only the lines carrying that substitution are excused.
    "Incorrect attribute value type": lambda d: "placeholder" in _diag_code(d),
    "Invalid value for input variable": lambda d: "placeholder" in _diag_code(d),
    # `file("./cert.pem")` and friends: the snippet names material the reader
    # supplies, which is not shipped with the policy and never will be.
    "Invalid function argument": lambda d: "no file exists at" in (d.get("detail") or ""),
}


def _diag_code(diag: dict) -> str:
    """The source line a diagnostic points at, as terraform reports it."""
    return (diag.get("snippet") or {}).get("code", "")


def terraform_available() -> bool:
    """Whether the terraform binary is on PATH."""
    return shutil.which("terraform") is not None


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def extract_hcl_blocks(content: str, filepath: Path) -> list[HclBlock]:
    """Extract HCL code blocks from terraform remediation sections."""
    lines = content.split("\n")
    uid_positions: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = re.match(r"^  - uid:\s+(\S+)", line)
        if m:
            uid_positions.append((i + 1, m.group(1)))

    def find_uid_for_line(line_num: int) -> str:
        result = ""
        for pos, uid in uid_positions:
            if pos <= line_num:
                result = uid
            else:
                break
        return result

    pattern = re.compile(
        r"- id: terraform\s*\n\s+desc: \|-?\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    blocks = []
    for match in pattern.finditer(content):
        desc_block = match.group(1)
        desc_start = match.start(1)
        tf_line = content[: match.start()].count("\n") + 1
        uid = find_uid_for_line(tf_line)

        # A single terraform remediation desc may split one logical config
        # across multiple ```hcl``` fences interleaved with prose. Concatenate
        # them so tflint sees the complete configuration, not fragments where
        # a `data` source in one fence appears "unused" in isolation.
        fences = []
        first_line = None
        for fence in re.finditer(r"```hcl\s*\n(.*?)```", desc_block, re.DOTALL):
            block = fence.group(1).strip()
            if not block:
                continue
            if first_line is None:
                code_offset = desc_start + fence.start(1)
                first_line = content[:code_offset].count("\n") + 1
            fences.append(block)

        if fences:
            blocks.append(HclBlock(
                code="\n\n".join(fences), line=first_line, uid=uid, file=filepath,
            ))
    return blocks


# ---------------------------------------------------------------------------
# Snippet processing
# ---------------------------------------------------------------------------

def detect_providers(hcl_code: str) -> set[str]:
    """Detect provider local names a snippet needs.

    Most come from the resource/data block type name, whose prefix is the
    provider. A resource can also name a provider explicitly with
    `provider = google-beta`, which is the only way to reach a beta-only
    resource type; that local name has to be declared too, or the generated
    configuration references a provider it never required.
    """
    prefixes = set()
    for m in re.finditer(
        r'(?:resource|data)\s+"([a-z][a-z0-9]*)_', hcl_code
    ):
        prefixes.add(m.group(1))
    # Both spellings occur: the modern unquoted reference and the legacy quoted
    # string form, which Terraform still parses.
    for m in re.finditer(r'^\s*provider\s*=\s*"?([a-z][a-z0-9-]*)"?\s*$', hcl_code, re.M):
        prefixes.add(m.group(1))
    return prefixes


def sanitize_snippet(hcl_code: str) -> str:
    """Clean up HCL snippet for tflint validation."""
    lines = hcl_code.split("\n")
    cleaned = []
    for line in lines:
        # Replace bare ellipsis lines with empty lines
        if re.match(r"^\s*\.\.\.\s*$", line):
            cleaned.append("")
            continue
        # Replace <placeholder> tokens with valid values.
        # If already inside quotes like "<foo>", replace just the angle
        # bracket token to avoid producing ""placeholder"".
        line = re.sub(r'"<[a-zA-Z][a-zA-Z0-9_-]*>"', '"placeholder"', line)
        line = re.sub(r"<[a-zA-Z][a-zA-Z0-9_-]*>", '"placeholder"', line)
        cleaned.append(line)
    return "\n".join(cleaned)


def extract_variables(hcl_code: str) -> set[str]:
    """Find var.xxx references that need placeholder variable blocks."""
    return set(re.findall(r"\bvar\.([a-zA-Z_][a-zA-Z0-9_]*)", hcl_code))


def neutralize_dangling_refs(hcl_code: str) -> str:
    """Replace references to resources the snippet does not declare.

    Terraform abandons a resource body as soon as one of its references cannot
    be resolved, so a snippet naming a surrounding resource it does not declare,
    which is the documentation convention here, never reaches schema validation
    and its arguments go unchecked. Declaring empty stubs does not help: the
    stub's own missing-argument errors come from an earlier validation phase and
    preempt the snippet's. Substituting a literal leaves the reference resolved
    and the body checkable.

    The substitute is always a string, so an attribute expecting a number or a
    bool now reports a type error. That class is ignored rather than fixed with
    a schema lookup, because the defects worth catching here are an argument the
    resource does not accept, a required argument left out, and a value assigned
    to a computed attribute, none of which depend on the substituted type.
    """
    # The traversal tail is matched as repeated `.attr` or `[index]` groups.
    # A single character class spanning both would consume the closing bracket
    # of the list the reference sits in, as in
    # `droplet_ids = [digitalocean_droplet.example.id]`, leaving unbalanced HCL
    # that fails to parse before any schema is ever consulted.
    # `depends_on` takes resource references, not values, so a substituted
    # string is itself an error there. It carries no schema surface, so the
    # validate copy drops it rather than neutralizing inside it.
    hcl_code = re.sub(
        r"^\s*depends_on\s*=\s*\[[^\]]*\]\s*$", "", hcl_code, flags=re.M | re.S
    )

    declared = {
        f"{rtype}.{name}"
        for rtype, name in re.findall(
            r'^\s*resource\s+"([^"]+)"\s+"([^"]+)"', hcl_code, re.M
        )
    }
    declared_data = {
        f"{dtype}.{name}"
        for dtype, name in re.findall(
            r'^\s*data\s+"([^"]+)"\s+"([^"]+)"', hcl_code, re.M
        )
    }

    def apply_outside_strings(text: str, sub) -> str:
        """Run a substitution everywhere except inside string literal text.

        A path like `file("${path.module}/keys/etl_service.pub")` contains
        `etl_service.pub`, which looks exactly like a resource reference and is
        not one. Interpolations are still processed, because a reference inside
        `${...}` is a real reference and has to resolve like any other.
        """
        out_parts = []
        pos = 0
        for sm in re.finditer(r'"(?:[^"\\]|\\.)*"', text):
            out_parts.append(sub(text[pos : sm.start()]))
            literal = sm.group(0)
            # Only the `${...}` segments inside the literal are code.
            out_parts.append(
                re.sub(
                    r"\$\{[^{}]*\}",
                    lambda im: sub(im.group(0)),
                    literal,
                )
            )
            pos = sm.end()
        out_parts.append(sub(text[pos:]))
        return "".join(out_parts)

    def replace_data(match: re.Match) -> str:
        key = f"{match.group(1)}.{match.group(2)}"
        return match.group(0) if key in declared_data else '"placeholder"'

    # `data.<type>.<name>.<attr>` first: the managed-resource pattern below
    # would otherwise match the `<type>.<name>.` inside it.
    out = apply_outside_strings(
        hcl_code,
        lambda s: re.sub(
            r"\bdata\.([a-z][a-z0-9]*(?:_[a-z0-9]+)+)\.([a-zA-Z_][\w-]*)(?:\.[\w*-]+|\[[^\]]*\])*",
            replace_data,
            s,
        ),
    )

    def replace_resource(match: re.Match) -> str:
        key = f"{match.group(1)}.{match.group(2)}"
        return match.group(0) if key in declared else '"placeholder"'

    # A managed resource type always carries an underscore, which is what keeps
    # var.x, local.x, each.value, count.index and self.x out of this. The
    # lookbehind keeps it out of a `data.<type>.<name>` traversal that the pass
    # above deliberately left alone: without it, a data source the snippet does
    # declare gets its type matched as if it were a managed resource, producing
    # `data."placeholder"`, which is not even parseable.
    return apply_outside_strings(
        out,
        lambda s: re.sub(
            r"(?<!data\.)\b([a-z][a-z0-9]*(?:_[a-z0-9]+)+)\.([a-zA-Z_][\w-]*)(?:\.[\w*-]+|\[[^\]]*\])*",
            replace_resource,
            s,
        ),
    )


def generate_wrapper(hcl_code: str, providers: set[str]) -> str:
    """Wrap an HCL snippet in a complete Terraform configuration."""
    parts = ['terraform {\n  required_version = ">= 1.0"\n  required_providers {\n']
    for p in sorted(providers):
        if p in PROVIDER_MAP:
            source, version = PROVIDER_MAP[p]
            parts.append(f'    {p} = {{\n')
            parts.append(f'      source  = "{source}"\n')
            parts.append(f'      version = "{version}"\n')
            parts.append('    }\n')
    parts.append('  }\n}\n\n')

    # A snippet may configure the provider itself, commonly to turn on a
    # preview feature its resources need. Emitting a second default block then
    # fails the whole run with "a default provider configuration was already
    # given", and deleting the snippet's block would hand the reader a
    # configuration that does not apply.
    self_configured = set(
        re.findall(r'^\s*provider\s+"([a-z][a-z0-9-]*)"\s*{', hcl_code, re.M)
    )
    for p in sorted(providers):
        if p in PROVIDER_MAP and p not in self_configured:
            extra = PROVIDER_EXTRA_CONFIG.get(p, "")
            parts.append(f'provider "{p}" {{\n{extra}}}\n\n')

    # A snippet may route a resource at an aliased provider configuration
    # (`provider = databricks.mws`). Without the alias declared, terraform
    # reports the missing configuration instead of checking the resource.
    for prov, alias in sorted(set(re.findall(
        r"^\s*provider\s*=\s*\"?([a-z][a-z0-9-]*)\.([a-zA-Z_][\w-]*)\"?\s*$",
        hcl_code, re.M,
    ))):
        if prov in providers and prov in PROVIDER_MAP:
            extra = PROVIDER_EXTRA_CONFIG.get(prov, "")
            parts.append(
                f'provider "{prov}" {{\n  alias = "{alias}"\n{extra}}}\n\n'
            )

    variables = extract_variables(hcl_code)
    for v in sorted(variables):
        parts.append(f'variable "{v}" {{\n  type    = string\n  default = "placeholder"\n}}\n\n')

    parts.append(hcl_code)
    parts.append("\n")
    return "".join(parts)


def write_tflint_config(tmp_dir: Path, providers: set[str]) -> None:
    """Write a .tflint.hcl config file with relevant plugins."""
    lines = [
        'config {\n',
        '  call_module_type = "none"\n',
        '}\n\n',
        'plugin "terraform" {\n',
        '  enabled = true\n',
        '  preset  = "recommended"\n',
        '}\n',
    ]
    for p in sorted(providers):
        if p in TFLINT_PLUGIN_MAP:
            source, version = TFLINT_PLUGIN_MAP[p]
            lines.append(f'\nplugin "{p}" {{\n')
            lines.append(f'  enabled = true\n')
            lines.append(f'  version = "{version}"\n')
            lines.append(f'  source  = "{source}"\n')
            lines.append('}\n')
            # Rule blocks only resolve for a plugin that is actually loaded --
            # naming a rule from an absent ruleset fails the whole run with
            # "Failed to check rule config; Rule not found". So each ruleset's
            # opt-outs are emitted with the plugin, not unconditionally.
            for rule in DISABLED_RULES.get(p, ()):
                lines.append(f'\nrule "{rule}" {{\n')
                lines.append('  enabled = false\n')
                lines.append('}\n')

    (tmp_dir / ".tflint.hcl").write_text("".join(lines))


# ---------------------------------------------------------------------------
# tflint execution
# ---------------------------------------------------------------------------

def init_tflint(tmp_dir: Path, plugin_cache: Path) -> tuple[bool, str]:
    """Run tflint --init to download plugins.

    Returns (success, diagnostics). Ruleset plugins are downloaded from GitHub
    releases, so this is the one step in the validator that depends on the
    network: it fails transiently on a dropped connection or a GitHub API hiccup
    even when every pin is correct. Each attempt is retried before the plugin
    set is declared unusable, and the last attempt's output is returned so a
    genuine failure (a version that no longer exists, a bad checksum) is
    diagnosable from the log rather than being reported as bare "init failed".
    """
    env = {**dict(os.environ), "TFLINT_PLUGIN_DIR": str(plugin_cache)}
    diagnostics = ""
    for attempt in range(INIT_ATTEMPTS):
        if attempt:
            time.sleep(INIT_RETRY_DELAY * attempt)
        try:
            result = subprocess.run(
                ["tflint", "--init"],
                cwd=tmp_dir,
                capture_output=True,
                text=True,
                timeout=120,
                env=env,
            )
        except subprocess.TimeoutExpired:
            diagnostics = "tflint --init timed out after 120s"
            continue
        if result.returncode == 0:
            return True, ""
        diagnostics = (result.stderr or result.stdout).strip()
    return False, diagnostics


def run_tflint(tmp_dir: Path, plugin_cache: Path) -> TflintResult:
    """Run tflint on a temp directory and return structured results."""
    env = {**dict(os.environ), "TFLINT_PLUGIN_DIR": str(plugin_cache)}
    result = subprocess.run(
        ["tflint", "--format=json", "--minimum-failure-severity=warning"],
        cwd=tmp_dir,
        capture_output=True,
        text=True,
        timeout=60,
        env=env,
    )

    if result.returncode == 0:
        return TflintResult(success=True)

    issues = []
    try:
        data = json.loads(result.stdout)
        for issue in data.get("issues", []):
            msg = issue.get("message", "unknown error")
            rule = issue.get("rule", {}).get("name", "")
            # Filter out false positives from our placeholder variable values
            if '"placeholder"' in msg:
                continue
            if rule:
                msg = f"{rule}: {msg}"
            issues.append(msg)
        for err in data.get("errors", []):
            msg = err.get("message", "unknown error")
            # Filter out noise from incomplete snippets
            if "Failed to check ruleset" in msg:
                continue
            issues.append(msg)
    except (json.JSONDecodeError, KeyError):
        stderr = result.stderr.strip()
        if stderr:
            issues.append(stderr)

    if not issues:
        return TflintResult(success=True)

    return TflintResult(success=False, issues=issues)


# `terraform validate` forks the provider plugin to read its schema. Eight
# workers each forking the AWS provider, several times over while other jobs run
# on the same machine, exhausts the process/file-descriptor budget and the run
# fails with "failed to instantiate provider ... to obtain schema" on snippets
# that are perfectly fine. Capping the terraform invocations keeps tflint at
# full width while making the schema pass deterministic.
_TERRAFORM_SLOTS = threading.BoundedSemaphore(4)


def first_error_line(stderr: str, stdout: str) -> str:
    """Pull something readable out of terraform's boxed CLI output.

    Terraform draws diagnostics inside a box, so the last line of stderr is a
    box-drawing character and taking it reports nothing at all. Prefer the line
    carrying the error text.
    """
    text = (stderr or stdout or "").replace("\x1b", "")
    lines = [
        re.sub(r"^[\s│╷╵|]*", "", ln).strip()
        for ln in text.splitlines()
    ]
    lines = [ln for ln in lines if ln and not set(ln) <= set("─-_=")]
    for ln in lines:
        if ln.startswith("Error:"):
            idx = lines.index(ln)
            return " ".join(lines[idx : idx + 3])[:300]
    return (lines[-1] if lines else "no diagnostic output")[:300]


def run_terraform_validate(tmp_dir: Path, mirror_dir: Path) -> list[str]:
    """Resolve a snippet against the real provider schemas.

    Returns the diagnostics worth reporting, empty when the snippet is clean.
    """
    if not mirror_dir.exists() or not any(mirror_dir.iterdir()):
        return []

    env = {**dict(os.environ), "TF_IN_AUTOMATION": "1", "TF_INPUT": "0"}
    with _TERRAFORM_SLOTS:
        return _terraform_validate_locked(tmp_dir, mirror_dir, env)


def _terraform_validate_locked(
    tmp_dir: Path, mirror_dir: Path, env: dict[str, str]
) -> list[str]:
    try:
        init = subprocess.run(
            [
                "terraform", "init", "-backend=false", "-input=false",
                f"-plugin-dir={mirror_dir}",
            ],
            cwd=tmp_dir, capture_output=True, text=True, timeout=300, env=env,
        )
    except subprocess.TimeoutExpired:
        return ["terraform init timed out after 300s"]
    if init.returncode != 0:
        # A provider missing from the mirror is an environment problem, not a
        # defect in the snippet, so say so rather than blaming the snippet.
        return ["terraform init failed: " + first_error_line(init.stderr, init.stdout)]

    try:
        result = subprocess.run(
            ["terraform", "validate", "-json"],
            cwd=tmp_dir, capture_output=True, text=True, timeout=120, env=env,
        )
    except subprocess.TimeoutExpired:
        return ["terraform validate timed out after 120s"]

    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        stderr = result.stderr.strip()
        return [f"terraform validate: {stderr[:200]}"] if stderr else []

    if data.get("valid"):
        return []

    issues = []
    for diag in data.get("diagnostics", []):
        if diag.get("severity") != "error":
            continue
        summary = diag.get("summary", "")
        if summary.startswith(TF_VALIDATE_IGNORED_SUMMARY_PREFIXES):
            continue
        conditional = TF_VALIDATE_CONDITIONAL_IGNORES.get(summary)
        if conditional is not None and conditional(diag):
            continue
        detail = " ".join((diag.get("detail") or "").split())
        issues.append(f"terraform validate: {summary}: {detail}"[:300])
    return issues


# ---------------------------------------------------------------------------
# Validation orchestration
# ---------------------------------------------------------------------------

def truncate_snippet(code: str, max_len: int = 100) -> str:
    """Show first line of HCL snippet, truncated."""
    first_line = code.split("\n")[0].strip()
    if len(first_line) > max_len:
        first_line = first_line[: max_len - 3] + "..."
    return first_line


def validate_block(
    block: HclBlock, plugin_cache: Path
) -> tuple[HclBlock, bool, list[str]]:
    """Validate a single HCL block. Returns (block, success, issues)."""
    providers = detect_providers(block.code)
    if not providers:
        # No resource/data blocks — likely a snippet showing just a block
        # attribute. Skip gracefully.
        return block, True, []

    sanitized = sanitize_snippet(block.code)
    wrapper = generate_wrapper(sanitized, providers)

    with tempfile.TemporaryDirectory(prefix="tflint_") as tmp:
        tmp_path = Path(tmp)
        (tmp_path / "main.tf").write_text(wrapper)
        write_tflint_config(tmp_path, providers)
        result = run_tflint(tmp_path, plugin_cache)
        issues = list(result.issues)

        # tflint runs first because it is the cheaper of the two and its
        # findings are the more specific. terraform validate runs regardless of
        # the tflint verdict so a snippet with a style warning still gets its
        # schema checked.
        # tflint has already seen the snippet as written. terraform validate
        # gets a copy with dangling references neutralized, in its own
        # directory so the tflint run is not affected by the substitution.
        if terraform_available():
            tf_dir = tmp_path / "tfvalidate"
            tf_dir.mkdir()
            (tf_dir / "main.tf").write_text(
                generate_wrapper(neutralize_dangling_refs(sanitized), providers)
            )
            issues.extend(
                run_terraform_validate(tf_dir, mirror_path(plugin_cache))
            )

        return block, not issues, issues


def plugins_for(providers: frozenset[str] | set[str]) -> frozenset[str]:
    """The ruleset plugins a provider set needs. Most providers have none."""
    return frozenset(p for p in providers if p in TFLINT_PLUGIN_MAP)


def init_plugins_for_providers(
    provider_sets: set[frozenset[str]], plugin_cache: Path
) -> set[frozenset[str]]:
    """Pre-initialize tflint plugins for all needed provider combinations.

    Returns the plugin sets that failed initialization. Both the "already done"
    and the "gave up" bookkeeping are keyed by the *plugin* set, not the
    provider set that asked for it: only providers with a ruleset reach the
    generated config, so `{azurerm}` and `{azurerm, azapi}` install exactly the
    same plugin and must share a verdict. Keyed by provider set instead, one
    combination's transient failure would blame whichever snippets happened to
    use it while the identical install succeeded for the rest.
    """
    initialized: set[frozenset[str]] = set()
    failed: set[frozenset[str]] = set()
    for providers in provider_sets:
        plugins_needed = plugins_for(providers)
        if not plugins_needed or plugins_needed in initialized:
            continue
        if plugins_needed in failed:
            continue

        with tempfile.TemporaryDirectory(prefix="tflint_init_") as tmp:
            tmp_path = Path(tmp)
            write_tflint_config(tmp_path, providers)
            ok, diagnostics = init_tflint(tmp_path, plugin_cache)
            if ok:
                initialized.add(plugins_needed)
            else:
                key = ",".join(sorted(plugins_needed))
                print(
                    f"Warning: tflint --init failed for plugins: {key} "
                    f"(after {INIT_ATTEMPTS} attempts)",
                    file=sys.stderr,
                )
                if diagnostics:
                    print(diagnostics, file=sys.stderr)
                failed.add(plugins_needed)
    return failed


def mirror_path(plugin_cache: Path) -> Path:
    """Where the provider mirror lives.

    CI and a local full run both benefit from keeping the mirror across
    invocations: building it is the single most expensive step and its contents
    depend only on PROVIDER_MAP. Point CNSPEC_TF_MIRROR at a directory to reuse
    one, otherwise it is built inside the run's own scratch space and discarded.
    """
    override = os.environ.get("CNSPEC_TF_MIRROR")
    return Path(override) if override else plugin_cache / "tf-mirror"


def build_provider_mirror(
    provider_sets: set[frozenset[str]], mirror_dir: Path
) -> None:
    """Populate a read-only filesystem mirror of every provider we need.

    Terraform's plugin cache is not safe for concurrent use: two workers running
    `terraform init` against the same TF_PLUGIN_CACHE_DIR race on the same path
    and leave a half-written binary, which surfaces as "exec format error" or a
    checksum that does not match the lock file. `terraform providers mirror`
    builds the layout once, serially, and the workers then init with
    `-plugin-dir`, which only ever reads. That also keeps every worker off the
    network.
    """
    if not terraform_available():
        return
    mirror_dir.mkdir(parents=True, exist_ok=True)
    env = {**dict(os.environ), "TF_IN_AUTOMATION": "1", "TF_INPUT": "0"}

    wanted = frozenset(
        p for providers in provider_sets for p in providers if p in PROVIDER_MAP
    )
    if not wanted:
        return

    # A mirror handed to us by CNSPEC_TF_MIRROR is already complete for every
    # provider in PROVIDER_MAP, so rebuilding it per target, or per agent
    # sharing a checkout, is pure cost.
    if all(
        (mirror_dir / "registry.terraform.io" / PROVIDER_MAP[p][0]).exists()
        for p in wanted
    ):
        return

    # One provider at a time. `terraform providers mirror` is all or nothing, so
    # a single unresolvable pin, a constraint matching no released version for
    # instance, would otherwise take every other provider down with it and the
    # schema pass would silently degrade to nothing.
    for prov in sorted(wanted):
        source, constraint = PROVIDER_MAP[prov]
        if (mirror_dir / "registry.terraform.io" / source).exists():
            continue
        with tempfile.TemporaryDirectory(prefix="tfmirror_") as tmp:
            tmp_path = Path(tmp)
            (tmp_path / "main.tf").write_text(generate_wrapper("", {prov}))
            try:
                result = subprocess.run(
                    ["terraform", "providers", "mirror", str(mirror_dir)],
                    cwd=tmp_path, capture_output=True, text=True,
                    timeout=1800, env=env,
                )
            except subprocess.TimeoutExpired:
                print(
                    f"Warning: mirroring {source} {constraint} timed out; "
                    f"its snippets will not be schema checked",
                    file=sys.stderr,
                )
                continue
            if result.returncode != 0:
                print(
                    f"Warning: cannot mirror {source} {constraint}, so its "
                    f"snippets will not be schema checked: "
                    + first_error_line(result.stderr, result.stdout),
                    file=sys.stderr,
                )
    # Exec every provider binary once, serially, before the workers start. The
    # first launch of a freshly written binary is slow enough to exceed
    # terraform's plugin-start timeout, on macOS because Gatekeeper assesses it,
    # and a worker that hits that reports "failed to instantiate provider ... to
    # obtain schema" against a snippet that is perfectly fine. Paying it once
    # here makes the pass deterministic.
    with tempfile.TemporaryDirectory(prefix="tfwarm_") as tmp:
        tmp_path = Path(tmp)
        (tmp_path / "main.tf").write_text(generate_wrapper("", set(wanted)))
        env = {**dict(os.environ), "TF_IN_AUTOMATION": "1", "TF_INPUT": "0"}
        try:
            subprocess.run(
                [
                    "terraform", "init", "-backend=false", "-input=false",
                    f"-plugin-dir={mirror_dir}",
                ],
                cwd=tmp_path, capture_output=True, text=True, timeout=600, env=env,
            )
            subprocess.run(
                ["terraform", "providers", "schema", "-json"],
                cwd=tmp_path, capture_output=True, text=True, timeout=900, env=env,
            )
        except subprocess.TimeoutExpired:
            print(
                "Warning: provider warm-up timed out; the schema pass may report "
                "spurious plugin-start failures",
                file=sys.stderr,
            )


def validate_policy_file(
    filepath: Path, plugin_cache: Path, workers: int
) -> tuple[int, int]:
    """Validate all terraform blocks in a policy file."""
    if not filepath.exists():
        print(f"Warning: Policy file not found: {filepath}", file=sys.stderr)
        return 0, 0

    content = filepath.read_text()
    blocks = extract_hcl_blocks(content, filepath)

    if not blocks:
        return 0, 0

    # Pre-init plugins
    provider_sets = set()
    for b in blocks:
        providers = detect_providers(b.code)
        if providers:
            provider_sets.add(frozenset(providers))
    failed_plugins = init_plugins_for_providers(provider_sets, plugin_cache)
    build_provider_mirror(provider_sets, mirror_path(plugin_cache))

    resolved = filepath.resolve()
    try:
        policy_relpath = str(resolved.relative_to(Path.cwd()))
    except ValueError:
        # Running from a subdirectory — use path relative to repo root
        policy_relpath = str(resolved.relative_to(REPO_ROOT))
    pass_count = 0
    fail_count = 0

    def process(block: HclBlock):
        plugins = plugins_for(detect_providers(block.code))
        if plugins in failed_plugins:
            key = ",".join(sorted(plugins))
            return block, False, [
                f"tflint plugin init failed for required ruleset: {key}"
            ]
        return validate_block(block, plugin_cache)

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {pool.submit(process, b): b for b in blocks}
        # Collect results in original order
        results = []
        for future in concurrent.futures.as_completed(futures):
            results.append(future.result())

    # Sort by line number for stable output
    results.sort(key=lambda r: r[0].line)

    for block, success, issues in results:
        snippet = truncate_snippet(block.code)
        if success:
            print(f"[PASS] {block.uid}")
            print(f"       {snippet}")
            pass_count += 1
        else:
            print(f"[FAIL] {block.uid}")
            print(f"       {snippet}")
            for issue in issues:
                print(f"       {issue}")
            fail_count += 1
            FAILURES.append({
                "file": policy_relpath,
                "line": block.line,
                "uid": block.uid,
                "snippet": snippet,
                "errors": issues,
            })

    return pass_count, fail_count


# ---------------------------------------------------------------------------
# GitHub Actions annotations
# ---------------------------------------------------------------------------

def emit_github_annotations() -> None:
    """Print GitHub Actions workflow commands for each failure."""
    for r in FAILURES:
        msg = "; ".join(r["errors"]) + f" — {r['snippet']}"
        title = f"Terraform HCL validation ({r['uid']})"
        msg = msg.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
        title = (
            title.replace("%", "%25")
            .replace("\r", "%0D")
            .replace("\n", "%0A")
            .replace(",", "%2C")
            .replace("::", "%3A%3A")
        )
        print(
            f"::error file={r['file']},line={r['line']},title={title}::{msg}"
        )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = sys.argv[1:]
    github_actions = False
    workers = 8
    target = "all"

    positional = []
    i = 0
    while i < len(args):
        if args[i] == "--github-actions":
            github_actions = True
        elif args[i] == "--workers" and i + 1 < len(args):
            workers = int(args[i + 1])
            i += 1
        else:
            positional.append(args[i])
        i += 1

    if positional:
        target = positional[0]

    valid_targets = ["all"] + list(TARGETS.keys())
    if target not in valid_targets:
        print(
            f"Unknown target: {target}\n"
            f"Usage: {sys.argv[0]} [{'|'.join(valid_targets)}] "
            f"[--github-actions] [--workers N]",
            file=sys.stderr,
        )
        sys.exit(2)

    if not shutil.which("tflint"):
        print(
            "Error: tflint not found in PATH.\n"
            "Install from https://github.com/terraform-linters/tflint",
            file=sys.stderr,
        )
        sys.exit(1)

    total_pass = 0
    total_fail = 0

    with tempfile.TemporaryDirectory(prefix="tflint_cache_") as cache:
        plugin_cache = Path(cache)

        targets_to_run = (
            TARGETS.keys() if target == "all" else [target]
        )

        for t in targets_to_run:
            for filepath in TARGETS[t]:
                p, f = validate_policy_file(filepath, plugin_cache, workers)
                total_pass += p
                total_fail += f

    if github_actions:
        emit_github_annotations()

    print(f"\n{total_pass} passed, {total_fail} failed", file=sys.stderr)
    sys.exit(1 if total_fail > 0 else 0)


if __name__ == "__main__":
    main()
