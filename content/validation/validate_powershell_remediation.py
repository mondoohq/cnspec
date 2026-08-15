#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Validates PowerShell remediation and audit snippets.
#
# PowerShell is the largest block of remediation no validator read: 341
# remediation snippets plus 202 in `audit:` sections, across the Windows,
# vSphere, Azure, M365 and AI policies. shellcheck cannot help — the fence is
# ```powershell, not ```bash — and PSScriptAnalyzer's style rules are the wrong
# tool for a documentation snippet.
#
# Three things are checked, in descending order of certainty:
#
#   1. It parses. The real PowerShell parser (the same one the shell uses)
#      builds an AST; a parse error is unambiguous.
#   2. Commands that resolve on this machine get their PARAMETER NAMES checked
#      against the cmdlet's actual parameters. That is the PowerShell analogue
#      of what the aws/az grammar validators do, and it covers the built-in
#      cmdlets that make up the largest single group of invocations here
#      (New-ItemProperty, Set-ItemProperty, Get-CimInstance, …).
#   3. Commands that need a module we do not have (Az, Microsoft.Graph,
#      VMware.PowerCLI, ExchangeOnlineManagement, or a Windows-only module)
#      cannot have their parameters checked, so their NAME is checked for shape
#      instead: PowerShell requires Verb-Noun with an approved verb, and a
#      wrong verb is the most common way to misremember a cmdlet
#      (`Enable-MpPreference` for `Set-MpPreference`).
#
# Everything here runs against `pwsh`, which ships on the GitHub runner images,
# so the job needs no module installs and stays fast. Checking the parameters of
# the module cmdlets too would need those modules on the runner — Az and
# Microsoft.Graph are large and slow to install — and is better done later from
# a checked-in grammar, the way dump_azure_commands.py handles the Azure CLI.
#
# Usage:
#   python3 content/validation/validate_powershell_remediation.py
#   python3 content/validation/validate_powershell_remediation.py windows
#   python3 content/validation/validate_powershell_remediation.py --github-actions

import argparse
import json
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
CONTENT_DIR = SCRIPT_DIR / ".."
REPO_ROOT = SCRIPT_DIR.parent.parent.resolve()

# Fence language, not remediation id: the Windows convention puts PowerShell
# under `- id: script`, M365 uses `- id: powershell`, vSphere uses both, and
# `audit:` blocks use none of them. The ```powershell fence is what they share.
PS_FENCE_LANGUAGES = ("powershell", "ps1", "posh")

# Modules a snippet may legitimately require that will not be present here.
# The value is what to tell the reader when a command from it is unresolvable.
EXTERNAL_MODULES = {
    r"-Az[A-Z]": "Az (Azure PowerShell)",
    r"^(Connect|Disconnect)-AzAccount$": "Az (Azure PowerShell)",
    r"-Mg[A-Z]": "Microsoft.Graph",
    r"^(Connect|Disconnect)-MgGraph$": "Microsoft.Graph",
    r"^(Connect|Disconnect)-ExchangeOnline$": "ExchangeOnlineManagement",
    r"(Mailbox|OrganizationConfig|TransportRule|AntiPhish|SafeLinks|SafeAttachment|"
    r"RemoteDomain|DkimSigningConfig|AdminAuditLogConfig|HostedContentFilter|"
    r"HostedOutboundSpamFilter|MalwareFilter|AtpPolicy|OwaMailboxPolicy|CASMailbox)":
        "ExchangeOnlineManagement",
    r"-SPO[A-Z]": "Microsoft.Online.SharePoint.PowerShell",
    r"^(Connect|Disconnect)-VIServer$": "VMware.PowerCLI",
    r"-(VM|VMHost|VMHostService|VMHostFirewall|VMHostAccount|VMHostSnmp|"
    r"VMHostAuthentication|VMHostNtpServer|VMHostSysLogServer|VMHostProfile|"
    r"AdvancedSetting|VirtualPortGroup|VDPortgroup|VDSwitch|Datastore|Cluster|"
    r"HardDisk|CDDrive|FloppyDrive|UsbDevice|VTpm|VIPermission|VIRole|KmsCluster|"
    r"VsanClusterConfiguration|SsoLockoutPolicy|SsoAdminServer|View|Folder|"
    r"EsxCli|Snapshot)": "VMware.PowerCLI",
    r"-(BitLocker|BitLockerVolume|Tpm)": "BitLocker (Windows only)",
    r"-(MpPreference|MpComputerStatus|MpThreat|MpSignature)": "Defender (Windows only)",
    r"-(LocalUser|LocalGroup|LocalGroupMember)":
        "Microsoft.PowerShell.LocalAccounts (Windows only)",
    r"-(NetFirewall|NetAdapter|NetIPsec|NetConnection|SmbServer|SmbClient|DnsClient)":
        "Windows networking modules (Windows only)",
    r"-(WindowsFeature|WindowsOptionalFeature|WindowsCapability|AppxPackage|"
    r"ScheduledTask|ScheduledTaskAction|ScheduledTaskTrigger)": "Windows-only modules",
    r"-(SecureBootUEFI|Disk|Partition|Volume|PhysicalDisk|StorageSubSystem)":
        "Windows storage/firmware modules (Windows only)",
    r"-(ProcessMitigation|DAPolicyChange|WinEvent|WinUserLanguageList|"
    r"ComputerRestorePoint|ItemPropertyValue)": "Windows-only modules",
}

# Provider-supplied *dynamic* parameters. These are contributed by a PowerShell
# provider rather than declared on the cmdlet, so `Get-Command` cannot see them
# anywhere the provider is absent — `Set-ItemProperty -Type` is real on Windows
# (the Registry provider adds it) and invisible on Linux and macOS, which is
# where this validator runs. Without this list the CI runner reports a valid
# Windows snippet as broken.
DYNAMIC_PARAMETERS = {
    "set-itemproperty": {"type"},
    "get-itemproperty": {"type"},
    "new-itemproperty": {"type"},
    "get-childitem": {"type"},
}

# Parameters are only checked for commands from modules that ship with
# PowerShell itself. Anything else — a module that happens to be installed on
# one machine, or a native executable, whose `.Parameters` is empty — would make
# the result depend on the environment, and a gate that says something different
# on a laptop than on the runner is worse than no gate. Everything outside this
# set falls through to the name-shape check, which is environment-independent.
BUILTIN_MODULES = {
    "microsoft.powershell.core",
    "microsoft.powershell.management",
    "microsoft.powershell.utility",
    "microsoft.powershell.security",
    "microsoft.powershell.diagnostics",
    "microsoft.powershell.host",
    "microsoft.powershell.archive",
    "microsoft.wsman.management",
    "cimcmdlets",
}

# Native executables invoked from PowerShell. They are not cmdlets, so neither
# Verb-Noun nor Get-Command applies.
NATIVE_EXECUTABLES = {
    "auditpol", "secedit", "gpupdate", "gpresult", "winget", "mbr2gpt", "reg",
    "netsh", "sc", "wmic", "bcdedit", "dism", "wevtutil", "icacls", "cipher",
    "takeown", "cmd", "powershell", "pwsh", "net", "certutil", "fsutil",
    "diskpart", "schtasks", "shutdown", "slmgr", "manage-bde", "nltest",
    "klist", "whoami", "where", "findstr", "tasklist", "taskkill",
}

FAILURES: list[dict] = []


@dataclass
class Snippet:
    code: str
    line: int
    uid: str
    policy: Path
    section: str          # "remediation" or "audit"
    method: str           # remediation id, or "" for audit


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

def extract_snippets(content: str, filepath: Path) -> list[Snippet]:
    """Every ```powershell fence in a remediation entry or an audit block."""
    lines = content.split("\n")
    uid_positions: list[tuple[int, str]] = []
    for i, line in enumerate(lines):
        m = re.match(r"^  - uid:\s+(\S+)", line)
        if m:
            uid_positions.append((i + 1, m.group(1)))

    def uid_for(line_num: int) -> str:
        result = ""
        for pos, uid in uid_positions:
            if pos <= line_num:
                result = uid
            else:
                break
        return result

    lang_alt = "|".join(PS_FENCE_LANGUAGES)
    fence_re = re.compile(rf"```(?:{lang_alt})\s*\n(.*?)```", re.DOTALL | re.IGNORECASE)
    out: list[Snippet] = []

    def collect(body: str, body_start: int, section: str, method: str) -> None:
        for fence in fence_re.finditer(body):
            code = fence.group(1).rstrip()
            if not code.strip():
                continue
            offset = body_start + fence.start(1)
            out.append(Snippet(
                code=code,
                line=content[:offset].count("\n") + 1,
                uid=uid_for(content[:offset].count("\n") + 1),
                policy=filepath,
                section=section,
                method=method,
            ))

    rem_re = re.compile(
        r"- id: (\S+)\s*\n\s+desc: \|-?\s*\n(.*?)(?=\n\s+- id: |\n\s+refs:|\n  - uid: |\Z)",
        re.DOTALL,
    )
    for m in rem_re.finditer(content):
        collect(m.group(2), m.start(2), "remediation", m.group(1))

    audit_re = re.compile(r"\n(\s+)audit: \|-?\s*\n(.*?)(?=\n\1\S|\Z)", re.DOTALL)
    for m in audit_re.finditer(content):
        collect(m.group(2), m.start(2), "audit", "")

    return out


def sanitize(code: str) -> str:
    """Replace <placeholder> tokens so they do not read as comparison operators.

    PowerShell has no `<x>` syntax, so an unsubstituted placeholder is a parse
    error that says nothing about the snippet.
    """
    code = re.sub(r'"<[^"<>\n]{1,60}>"', '"placeholder"', code)
    code = re.sub(r"<[^<>\n]{1,60}>", "placeholder", code)
    return code


# ---------------------------------------------------------------------------
# PowerShell-side analysis
# ---------------------------------------------------------------------------

# Parses each snippet, and reports for every command whether it resolves here,
# what its valid parameters are, and which parameters the snippet used.
ANALYZE_PS = r"""
$ErrorActionPreference = 'Stop'
# The payload arrives as a file path rather than on stdin. Reading stdin from a
# -Command script is version-dependent — on the 7.6 runner it came back empty
# while it worked locally, and pwsh still exited 0, so the failure surfaced as
# an unexplained JSON decode error rather than as anything diagnosable.
$items = Get-Content -Raw -LiteralPath $args[0] | ConvertFrom-Json
$verbs = (Get-Verb).Verb
$out = New-Object System.Collections.ArrayList
foreach ($it in $items) {
  $tokens = $null; $errors = $null
  $ast = [System.Management.Automation.Language.Parser]::ParseInput(
           $it.code, [ref]$tokens, [ref]$errors)
  $cmds = New-Object System.Collections.ArrayList
  if ($ast) {
    $found = $ast.FindAll({ param($n)
      $n -is [System.Management.Automation.Language.CommandAst] }, $true)
    foreach ($c in $found) {
      $name = $c.GetCommandName()
      if (-not $name) { continue }
      $params = New-Object System.Collections.ArrayList
      foreach ($el in $c.CommandElements) {
        if ($el -is [System.Management.Automation.Language.CommandParameterAst]) {
          [void]$params.Add($el.ParameterName)
        }
      }
      $resolved = Get-Command -Name $name -ErrorAction SilentlyContinue |
                  Select-Object -First 1
      $valid = @()
      if ($resolved -and $resolved.Parameters) { $valid = @($resolved.Parameters.Keys) }
      $module = ''
      if ($resolved) {
        $module = $resolved.ModuleName
        # An alias resolves to the command it points at; that is where the
        # parameters live (gc -> Get-Content).
        if ($resolved.CommandType -eq 'Alias' -and $resolved.ResolvedCommand) {
          $module = $resolved.ResolvedCommand.ModuleName
          if ($resolved.ResolvedCommand.Parameters) {
            $valid = @($resolved.ResolvedCommand.Parameters.Keys)
          }
        }
      }
      $verb = $null
      if ($name -match '^([A-Za-z]+)-') { $verb = $matches[1] }
      [void]$cmds.Add([pscustomobject]@{
        name       = $name
        params     = @($params)
        resolved   = [bool]$resolved
        module     = "$module"
        valid      = @($valid)
        verb       = $verb
        verbKnown  = ($verb -ne $null -and $verbs -contains $verb)
      })
    }
  }
  [void]$out.Add([pscustomobject]@{
    i        = $it.i
    errors   = @($errors | ForEach-Object { $_.Message })
    commands = @($cmds)
  })
}
$out | ConvertTo-Json -Depth 8 -Compress
"""


def analyze(snippets: list[Snippet]) -> list[dict]:
    payload = json.dumps([{"i": i, "code": sanitize(s.code)}
                          for i, s in enumerate(snippets)])
    with tempfile.TemporaryDirectory(prefix="psvalidate_") as tmp:
        payload_path = Path(tmp) / "snippets.json"
        payload_path.write_text(payload)
        script_path = Path(tmp) / "analyze.ps1"
        script_path.write_text(ANALYZE_PS)
        result = subprocess.run(
            ["pwsh", "-NoProfile", "-NonInteractive", "-File",
             str(script_path), str(payload_path)],
            capture_output=True, text=True, timeout=900,
        )
    if result.returncode != 0 or not result.stdout.strip():
        print(
            "Error: the PowerShell analysis step produced no usable output "
            f"(exit {result.returncode}).\n"
            f"stderr:\n{result.stderr[:2000]}\n"
            f"stdout:\n{result.stdout[:500]}",
            file=sys.stderr,
        )
        sys.exit(1)
    data = json.loads(result.stdout)
    return data if isinstance(data, list) else [data]


def external_module_for(name: str) -> str | None:
    for pattern, module in EXTERNAL_MODULES.items():
        if re.search(pattern, name):
            return module
    return None


def check_snippet(s: Snippet, analysis: dict) -> list[str]:
    problems: list[str] = []

    for message in analysis.get("errors") or []:
        problems.append(f"parse error: {message}")
    if problems:
        # Everything below reads the AST, which a parse error already invalidated.
        return problems

    for cmd in analysis.get("commands") or []:
        name = cmd["name"]
        if name.lower() in NATIVE_EXECUTABLES or "\\" in name or "/" in name:
            continue
        if cmd["resolved"] and (cmd.get("module") or "").lower() in BUILTIN_MODULES:
            valid = {p.lower() for p in cmd["valid"]}
            valid |= DYNAMIC_PARAMETERS.get(name.lower(), set())
            for param in cmd["params"]:
                # PowerShell accepts unambiguous prefixes of a parameter name.
                if param.lower() in valid:
                    continue
                if any(v.startswith(param.lower()) for v in valid):
                    continue
                problems.append(f"{name}: no parameter -{param}")
            continue
        if external_module_for(name):
            # Parameters cannot be checked without the module; the name still has
            # to be a well-formed cmdlet, and a wrong verb is the usual mistake.
            if not cmd["verb"]:
                problems.append(f"{name}: not Verb-Noun and not a known executable")
            elif not cmd["verbKnown"]:
                problems.append(
                    f"{name}: '{cmd['verb']}' is not an approved PowerShell verb "
                    f"(see Get-Verb)"
                )
            continue
        if not cmd["verb"]:
            problems.append(f"{name}: unknown command, and not Verb-Noun")
        elif not cmd["verbKnown"]:
            problems.append(
                f"{name}: '{cmd['verb']}' is not an approved PowerShell verb "
                f"(see Get-Verb)"
            )
    return problems


# ---------------------------------------------------------------------------
# Targets and main
# ---------------------------------------------------------------------------

TARGETS = {
    "windows": [
        CONTENT_DIR / "mondoo-windows-security.mql.yaml",
        CONTENT_DIR / "mondoo-windows-workstation-security.mql.yaml",
        CONTENT_DIR / "mondoo-windows-11-compatibility.mql.yaml",
    ],
    "vmware": [
        CONTENT_DIR / "mondoo-vmware-vsphere.mql.yaml",
        CONTENT_DIR / "mondoo-vmware-vsphere-esxi.mql.yaml",
    ],
    "azure": [CONTENT_DIR / "mondoo-azure-security.mql.yaml"],
    "m365": [CONTENT_DIR / "mondoo-m365-security.mql.yaml"],
}


def all_policy_files() -> list[Path]:
    """Every policy in content/ — a PowerShell fence anywhere is in scope, so
    there is no allowlist to forget to update."""
    return sorted(CONTENT_DIR.glob("*.mql.yaml"))


def emit_github_annotations() -> None:
    for r in FAILURES:
        msg = "; ".join(r["errors"]).replace("%", "%25").replace("\n", "%0A").replace("\r", "%0D")
        title = f"PowerShell validation ({r['uid']})".replace(",", "%2C").replace("::", "%3A%3A")
        print(f"::error file={r['file']},line={r['line']},title={title}::{msg}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Validate PowerShell remediation and audit snippets"
    )
    parser.add_argument("target", nargs="?", default="all", choices=["all", *TARGETS])
    parser.add_argument("--github-actions", action="store_true")
    args = parser.parse_args()

    if not shutil.which("pwsh"):
        print(
            "Error: pwsh not found in PATH.\n"
            "  macOS (Homebrew): brew install --cask powershell\n"
            "  Linux:            https://learn.microsoft.com/powershell/scripting/install/installing-powershell-on-linux",
            file=sys.stderr,
        )
        sys.exit(1)

    files = all_policy_files() if args.target == "all" else TARGETS[args.target]

    snippets: list[Snippet] = []
    for path in files:
        if path.exists():
            snippets.extend(extract_snippets(path.read_text(), path))

    if not snippets:
        print("No PowerShell snippets found", file=sys.stderr)
        return

    results = analyze(snippets)
    by_index = {r["i"]: r for r in results}

    passed = failed = 0
    for i, s in enumerate(snippets):
        problems = check_snippet(s, by_index.get(i, {}))
        method = f"/{s.method}" if s.method else ""
        label = f"{s.uid} ({s.section}{method})"
        if problems:
            failed += 1
            print(f"[FAIL] {label}")
            for p in problems:
                print(f"       {p}")
            FAILURES.append({
                "file": str(s.policy.resolve().relative_to(REPO_ROOT)),
                "line": s.line,
                "uid": s.uid,
                "errors": problems,
            })
        else:
            passed += 1
            print(f"[PASS] {label}")

    if args.github_actions:
        emit_github_annotations()

    print(f"\n{passed} passed, {failed} failed", file=sys.stderr)
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
