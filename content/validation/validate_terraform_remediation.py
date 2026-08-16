#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates Terraform HCL code blocks found in remediation sections of cnspec
# policies by running tflint against each snippet.
#
# Usage:
#   python3 validate_terraform_remediation.py                # validate all
#   python3 validate_terraform_remediation.py aws            # validate AWS only
#   python3 validate_terraform_remediation.py azure          # validate Azure only
#   python3 validate_terraform_remediation.py gcp            # validate GCP only
#   python3 validate_terraform_remediation.py --github-actions  # emit GH annotations

import concurrent.futures
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass, field
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent

TARGETS = {
    "aws": [SCRIPT_DIR / ".." / "mondoo-aws-security.mql.yaml"],
    "azure": [SCRIPT_DIR / ".." / "mondoo-azure-security.mql.yaml"],
    "gcp": [SCRIPT_DIR / ".." / "mondoo-gcp-security.mql.yaml"],
    "oci": [SCRIPT_DIR / ".." / "mondoo-oci-security.mql.yaml"],
    "github": [
        SCRIPT_DIR / ".." / "mondoo-github-security.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-github-best-practices.mql.yaml",
    ],
    "gitlab": [SCRIPT_DIR / ".." / "mondoo-gitlab-security.mql.yaml"],
    "okta": [SCRIPT_DIR / ".." / "mondoo-okta-security.mql.yaml"],
    "m365": [SCRIPT_DIR / ".." / "mondoo-m365-security.mql.yaml"],
    # Only aws/azurerm/google have tflint rulesets; the rest are checked by
    # the terraform preset's syntax and declaration rules alone.
    "alibaba": [SCRIPT_DIR / ".." / "mondoo-alibaba-security.mql.yaml"],
    "cloudflare": [SCRIPT_DIR / ".." / "mondoo-cloudflare-security.mql.yaml"],
    "databricks": [SCRIPT_DIR / ".." / "mondoo-databricks-security.mql.yaml"],
    "digitalocean": [SCRIPT_DIR / ".." / "mondoo-digitalocean-security.mql.yaml"],
    "dns": [SCRIPT_DIR / ".." / "mondoo-dns-security.mql.yaml"],
    "email": [SCRIPT_DIR / ".." / "mondoo-email-security.mql.yaml"],
    "hetzner": [SCRIPT_DIR / ".." / "mondoo-hetzner-security.mql.yaml"],
    "openstack": [SCRIPT_DIR / ".." / "mondoo-openstack-security.mql.yaml"],
    "portainer": [SCRIPT_DIR / ".." / "mondoo-portainer-security.mql.yaml"],
    "snowflake": [SCRIPT_DIR / ".." / "mondoo-snowflake-security.mql.yaml"],
    "stackit": [SCRIPT_DIR / ".." / "mondoo-stackit-security.mql.yaml"],
    "tailscale": [SCRIPT_DIR / ".." / "mondoo-tailscale-security.mql.yaml"],
    "unifi": [SCRIPT_DIR / ".." / "mondoo-unifi-security.mql.yaml"],
    "vercel": [SCRIPT_DIR / ".." / "mondoo-vercel-security.mql.yaml"],
    "vsphere": [
        SCRIPT_DIR / ".." / "mondoo-vmware-vsphere.mql.yaml",
        SCRIPT_DIR / ".." / "mondoo-vmware-vsphere-esxi.mql.yaml",
    ],
}

# Map resource prefix -> (provider source, version constraint)
PROVIDER_MAP = {
    "aws": ("hashicorp/aws", "~> 6.0"),
    "azurerm": ("hashicorp/azurerm", "~> 5.0"),
    "azuread": ("hashicorp/azuread", "~> 3.0"),
    "azapi": ("azure/azapi", "~> 2.0"),
    "google": ("hashicorp/google", "~> 7.0"),
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
    "alicloud": ("aliyun/alicloud", "~> 2.0"),
    "cloudflare": ("cloudflare/cloudflare", "~> 5.0"),
    "databricks": ("databricks/databricks", "~> 1.0"),
    "digitalocean": ("digitalocean/digitalocean", "~> 2.0"),
    "hcloud": ("hetznercloud/hcloud", "~> 1.0"),
    "openstack": ("terraform-provider-openstack/openstack", "~> 3.0"),
    "portainer": ("portainer/portainer", "~> 1.0"),
    "snowflake": ("snowflakedb/snowflake", "~> 2.0"),
    "stackit": ("stackitcloud/stackit", "~> 0.111"),
    "tailscale": ("tailscale/tailscale", "~> 0.29"),
    "unifi": ("paultyng/unifi", "~> 0.41"),
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
    """Detect provider prefixes from resource/data block type names."""
    prefixes = set()
    for m in re.finditer(
        r'(?:resource|data)\s+"([a-z][a-z0-9]*)_', hcl_code
    ):
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

    for p in sorted(providers):
        if p in PROVIDER_MAP:
            extra = PROVIDER_EXTRA_CONFIG.get(p, "")
            parts.append(f'provider "{p}" {{\n{extra}}}\n\n')

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
        return block, result.success, result.issues


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

    resolved = filepath.resolve()
    try:
        policy_relpath = str(resolved.relative_to(Path.cwd()))
    except ValueError:
        # Running from a subdirectory — use path relative to repo root
        policy_relpath = str(resolved.relative_to(SCRIPT_DIR.resolve().parent.parent))
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
