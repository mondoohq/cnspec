# content/CLAUDE.md

Policy authoring guidance for `*.mql.yaml` files in this directory. Loaded automatically when working under `content/`.

## Bundle structure

```yaml
policies:
  - uid: example-policy
    name: Example Policy
    version: 1.0.0
    summary: Secure the example service configuration and access controls
    groups:
      - title: Security Checks
        filters: asset.platform == "linux"
        checks:
          - uid: example-check
            title: Example Check
            impact: 80
            mql: |
              users.where(name == "root").list {
                shell != "/bin/bash"
              }
```

**Key concepts**:

- **uid**: Unique identifier for policies, checks, queries.
- **summary**: Required one-line policy description (≤130 chars). See Formatting requirements.
- **filters**: MQL expressions that determine applicability.
- **impact**: Risk score 0-100 for prioritization.
- **checks**: Scoring queries (pass/fail).
- **queries**: Data collection queries (no scoring).
- **Multi-statement check MQL**: A check's `mql:` block can contain multiple top-level statements. Each statement is scored as a separate datapoint and the check passes only if every datapoint passes — it is *not* "last expression wins". Use this pattern when you want each assertion to surface independently in scan output; collapse to a single `&&`-joined expression only if you want one combined datapoint.

## Formatting requirements

- Every policy must have a `summary:` field — the one-line description shown in policy listings and the marketplace. It is **required** and must be **130 characters or fewer**. Write it verb-first (`Secure`, `Enforce`, `Validate`, `Detect`, `Harden`) followed by the concrete scope, matching the existing policies. Do **not** use em-dashes (`—`, `–`) or `--` in the summary; restructure the sentence instead.
- All `desc` and `remediation` fields must be valid Markdown (rendered in the UI). Use proper headings, lists, code blocks, links.
- Check `title` fields must be 75 characters or fewer.
- Check `title` must match the action enforced by the `mql` query and described in `desc`. If the title says "Ensure X is enabled" the query must assert X is enabled and the description must explain X — don't let titles drift from what the check actually does (e.g., a title about "encryption at rest" paired with a query that inspects TLS settings).
- Every check's `docs:` block must include all three sections: `desc:`, `audit:`, and `remediation:`. None of these are optional — `desc` explains *what and why*, `audit` explains *how to verify manually*, and `remediation` explains *how to fix*.
- `audit:` instructions must use the **vendor's own tooling** — the cloud console or the vendor CLI (`aws`, `az`, `gcloud`, `oci`, `doctl`, `kubectl`, `gh`, …). **Never** reference Mondoo tools (`cnspec`, `mql`, the Mondoo console) in audit steps. Prefer the vendor CLI for an automated path; fall back to console click-through when no CLI exists. The point of `audit:` is to give an auditor a vendor-native way to reproduce the finding without trusting Mondoo's output.
- `remediation:` must include **every remediation method the target platform supports** — not just one or two. Use `- id: <method>` entries in the list. Required coverage by platform:
  - **AWS**: `console`, `cli`, `terraform`, `cloudformation`
  - **Azure**: `portal`, `cli`, `terraform`, `bicep` (Azure uses `portal` — the product name for the Azure web UI — instead of the generic `console` used by other clouds)
  - **GCP / OCI / DigitalOcean / Cloudflare / Hetzner / other clouds**: `console`, `cli`, `terraform`
  - **Windows / macOS**: `gui`, `cli`, `ansible`, and `script` (PowerShell on Windows, bash on macOS). In `mondoo-windows-security` also `chef`.
  - **Linux / FreeBSD**: `cli`, `script` (bash or sh), `ansible`. In `mondoo-linux-security` and `mondoo-freebsd-security` also `chef`.

  A `chef` entry is a Chef Infra recipe, matching the `chef` entries in the two Chef Infra policies. Add one wherever Chef Infra Client runs on the asset and the fix is a change to state on that host. Prefer a purpose-built resource over a shell-out: `registry_key`, `windows_security_policy`, `windows_audit_policy`, and `windows_user_privilege` cover almost all of the Windows policy; `sysctl`, `kernel_module`, `systemd_unit`, `user_ulimit`, and `sudo` cover most of the Linux one. On FreeBSD the `sysctl` resource writes `/etc/sysctl.d`, which the base system does not read, so kernel parameters go into `/etc/sysctl.conf` instead; the `service` provider there edits `/etc/rc.conf` directly, making `action :disable` the `sysrc` equivalent.
  - **Kubernetes**: `kubectl`, `manifest` (YAML), and where applicable `helm`
  - **Microsoft 365** (`mondoo-m365-security`): `console`, `powershell`, and `terraform` where a real resource exists.
    - `console` — the relevant Microsoft admin center (Microsoft Entra, Microsoft 365, Microsoft Defender, Exchange, SharePoint, or Intune). Always applies.
    - `powershell` — Microsoft Graph PowerShell for Entra/identity checks, Exchange Online PowerShell for Exchange checks, SharePoint Online Management Shell for SharePoint checks. Always applies, except where the control is a DNS record (SPF), which uses `cli` (`az network dns …`) instead.
    - `terraform` — **only** where the `azuread` provider (or, for DNS-record checks, a DNS provider such as `azurerm`) exposes a genuine resource for the setting. The Conditional Access checks (`azuread_conditional_access_policy`) and role assignments (`azuread_directory_role_assignment`) qualify. Exchange Online, SharePoint Online, Intune device configuration, the tenant authorization policy, the security-defaults toggle, and per-domain password validity have **no** Terraform resource — use a `# No Terraform remediation:` comment, never a `null_resource` + `local-exec` shell-out or an `azapi` block faking one.

  Omit a method only when it genuinely doesn't apply, and leave a YAML comment above the check explaining why (same pattern as the "No Terraform variants" comment above).
- Verify CLI commands in remediation steps with the validator (see below) before committing.

## Impact scoring

`impact:` drives prioritization in scan output and dashboards. The rows below are **bands** — ranges of values, not single numbers. Pick the band that matches the check's risk, then choose a value within that range; any value inside the band is valid (`75`, for example, sits within the 70–79 band).

| Impact | When to use |
|--------|-------------|
| 90–100 | Direct path to data loss, account compromise, or full takeover. Public exposure of customer data, unauthenticated admin endpoints, plaintext secrets in shared storage, disabled audit logging on production. |
| 80–89 | High-confidence misconfiguration with realistic exploit chain. Encryption disabled on sensitive resources, overly permissive IAM, network-wide ingress on management ports, missing MFA on privileged identities. |
| 70–79 | Important hardening that meaningfully reduces blast radius. CMK encryption instead of vendor-managed keys, private endpoints over public, log retention/aggregation, disabling remote management shells and consoles left reachable by vendor defaults. |
| 60–69 | Recommended hardening with moderate risk reduction. Tag/label hygiene that gates other controls, non-default versions of managed services, password complexity above vendor defaults. |
| 30–59 | Best practices and informational. Defense-in-depth that rarely changes outcomes on its own (resource labeling, optional telemetry, naming conventions). |

Anchor to a sibling check in the same policy whenever possible — if you're adding an encryption-at-rest check next to five others at `impact: 70`, use 70 unless you can justify why this one differs. Cite a sibling UID in the PR description.

## UID and naming conventions

**Pattern**: `mondoo-<provider>-security-<resource>-<rule>`

- `<provider>` is the policy's cloud/platform — `aws`, `azure`, `gcp`, `oci`, `digitalocean`, `hetzner`, `linux`, `windows`, `macos`, `kubernetes`, `github`, `gitlab`, etc.
- `<resource>` is the service or object being checked — `eks-cluster`, `s3-bucket`, `cloud-sql-mysql`, `network-security-group`. Use the vendor's own naming where it exists; don't invent new terminology.
- `<rule>` describes the assertion in active voice — `cmks-in-kms`, `private-controlplane`, `logging-enabled`, `restrict-public-access`. Keep it short and concrete; avoid generic suffixes like `-misconfigured` or `-check`.

**Variant suffixes** (parent UID + suffix; see the Terraform variants section for context):

- `-<cloud>` — runtime variant (e.g., `-aws`, `-azure`, `-gcp`). Match the parent's cloud.
- `-terraform-hcl` / `-terraform-plan` / `-terraform-state` — Terraform asset variants.
- `-cloudformation` / `-bicep` — IaC variants for AWS / Azure where applicable.

The parent check carries `title`, `impact`, `tags`, and `docs:`; variants carry the platform-specific `mql:` and a `mondoo.com/filter-title` + `mondoo.com/filter-icon` tag pair. Don't repeat compliance tags on variants.

**Before adding a new UID**: grep the existing policy for the resource name to confirm there isn't already a check on the same control objective (`grep -i "<resource>-<rule>" content/mondoo-<provider>-security.mql.yaml`). Duplicate checks fragment compliance mappings and confuse scan output.

## `docs:` body structure

The three required sections follow a consistent shape across the repo. Match it so new checks are visually and structurally indistinguishable from the surrounding policy.

**`desc:`** — what the check enforces, then why it matters.

```markdown
This check ensures that <resource> is configured to <enforced behavior>. By default <vendor default>, but <why the safer setting matters>.

**Why this matters**

- **<benefit 1>:** <one sentence>.
- **<benefit 2>:** <one sentence>.
- **<benefit 3>:** <one sentence>.

**Risk mitigation**

- **<mitigation 1>:** <one sentence>.
- **<mitigation 2>:** <one sentence>.
```

Lead with the assertion (one paragraph), then bullet the benefits. Don't restate the title verbatim. Don't reference the MQL query, variant UIDs, or "this policy" — the reader sees the check in isolation.

**`audit:`** — vendor-native verification steps. Use H3 headers (`### Audit via Console`, `### Audit via CLI`) when both a console path and a CLI path exist. Each path is a short numbered list ending with what a passing vs. failing result looks like. Never reference `cnspec`, `mql`, or the Mondoo console (see the formatting rule above).

**`remediation:`** — a list of `- id: <method>` entries, one per supported management surface (see the remediation-coverage rule above). Each entry's `desc:` follows the same shape:

```markdown
To <restate the fix in active voice> using <method>:

1. <step>
2. <step>
3. <step>

```<lang>
<example code or command>
```
```

Order the list consistently: `console`/`portal` → `cli` → `terraform` → `cloudformation`/`bicep` for clouds; `gui` → `cli` → `script` → `ansible` for OS targets. Reviewers scan in this order — keep it predictable.

## Compliance tags (`compliance/<framework>: <control-uid>`)

**Never copy compliance tags from a neighboring check.** The nearby check was mapped for a different control objective; reusing its tags propagates a wrong mapping and misleads auditors. Two checks that both "relate to identity" can map to different controls.

**A new check is not done until it carries compliance tags.** Any check added to a policy whose existing checks have `compliance/*` tags must ship with its own verified tags (completing the process below) — or with the user's explicit approval to skip them. Do **not** open a PR that adds untagged checks next to tagged siblings and merely note the omission in the PR body; that ships an inconsistent, half-finished policy. An empty `find` for the `cnspec-enterprise-policies` repo means **ask the user where the clone lives** — it does not authorize proceeding without tags.

When adding or changing compliance tags, follow this process for **each** framework the policy already tags:

1. **Read the authoritative control text.** Open the framework definition in `cnspec-enterprise-policies/frameworks/<framework>.mql.yaml` (e.g., `iso-27001-2022.mql.yaml`, `soc2-2017.mql.yaml`, `nist-sp-800-53-rev5.mql.yaml`). Each control has a `uid`, `title`, and usually `docs.desc`. Ask the user where their clone lives if you don't already know; if the files aren't available, stop and tell the user — do not guess.
2. **State in one sentence what the check actually enforces.** If the check is about identity proofing, say so; if it's about encryption-at-rest, say so. Do not let the check's *title* mislead you — read the MQL.
3. **Find the single best-matching control** by scanning control titles and descriptions for language that covers the enforced behavior. Strict fit only: MFA, password policy, and session-timeout controls are *not* acceptable stand-ins for identity-proofing, encryption, network-isolation, etc.
4. **If no control fits, tag it with the YAML boolean `false`** — unquoted, like `compliance/soc2-2017: false`. **Not** `"false"` (the string), not `false-fit`, not `n/a`, not omitting the key. The unquoted boolean is the established repo convention (grep `compliance/.*: false` for ~150+ examples) and is what downstream tooling expects. A missing mapping is strictly better than a wrong one — wrong mappings get caught in compliance audits and create trust debt.
5. **Cite the control you chose.** When you present tags to the user, include the control title and a short quote from the control description so the user can verify.

**The `<framework>` in the key must exactly match the framework's `uid:` field** — the value declared inside the framework YAML, *not* the file name. They are not always the same: `cnspec-enterprise-policies/frameworks/bsi-grundschutz-sys15.mql.yaml` declares `uid: bsi-sys-1-5`, so the tag is `compliance/bsi-sys-1-5`. A key that matches no real framework `uid` generates a framework map with a dangling `framework_owner`, which fails bundle migration in `cnspec-enterprise-policies` with `cannot find framework owner`. Likewise the `<control-uid>` value must be a `uid` that exists under that framework's `controls:`.

Known high-value anchors (verify before using):

- Identity proofing / email verification: `iso-27001-2022-a-5-16` (Identity management), `nist-csf-2-pr-aa-02` ("Identities are proofed and bound to credentials"), `nist-sp-800-53-rev5-ia-12` (Identity Proofing). No direct equivalent in NIST CSF 1.x, NIST 800-171 rev2, NIS2 Article 21(2), or SOC 2 2017.
- Authenticator / MFA strength: `iso-27001-2022-a-8-5`, `nist-csf-2-pr-aa-03`, `nist-sp-800-53-rev5-ia-2`, `soc2-control-cc6-1-4`. Do **not** reuse these for identity-proofing checks.

## Terraform variants and remediation for cloud policies

When you add or modify a check in a cloud policy (`mondoo-aws-security`, `mondoo-azure-security`, `mondoo-gcp-security`, `mondoo-oci-security`, `mondoo-hetzner-security`, `mondoo-digitalocean-security`, etc.), **two things** must ship together:

1. A **variants:** block so the check runs against the live cloud runtime *and* Terraform HCL/plan/state assets.
2. A **`- id: terraform`** entry in the `remediation:` list with HCL example code that fixes the underlying issue.

Both ride along with every new or modified check — don't ship one without the other.

### Variants

Convert single-platform checks to a `variants:` block with up to four children:

- `<uid>-<cloud>` — runtime check (`asset.platform == 'aws'`, `'azure'`, `'gcp-project'`, `'oci'`, …)
- `<uid>-terraform-hcl` — `terraform.resources(...)` against HCL source
- `<uid>-terraform-plan` — `terraform.plan.resourceChanges` against `terraform plan` JSON
- `<uid>-terraform-state` — `terraform.state.resources` against `terraform.tfstate`

**The runtime platform name must come from the provider's catalog, not from the cloud's short name.** A filter naming a platform the provider never emits compiles, lints, and passes CI — it just never matches an asset, so the check silently never runs. Check the name against `providers/<cloud>/connection/platforms.go` (or `providers/<cloud>/resources/platforms.go`) in the mql repo, or the `Platforms` array in the installed `~/.config/mondoo/providers/<cloud>/<cloud>.json`. AWS, Azure, and OCI do have an account-level platform named for the cloud (`aws`, `azure`, `oci`); **GCP does not** — use `gcp-project`, `gcp-org`, `gcp-folder`, or a per-resource platform such as `gcp-storage-bucket`.

Reference patterns in this repo:

- GCP: `mondoo-gcp-security-memorystore-iam-auth-enabled` in `mondoo-gcp-security.mql.yaml`
- HCL nested-block fanout: `mondoo-gcp-security-cloud-sql-mysql-skip-show-database-enabled-terraform-*` (database_flags)
- Plan/state list-of-objects shape: `mondoo-gcp-security-cloud-storage-bucket-retention-policy-locked-terraform-*`

### Fixtures for new IaC variants

Every `-terraform-hcl`, `-cloudformation`, and `-bicep` variant ships with pass **and** fail fixtures under `content/validation/scans/fixtures/iac-variants/<policy>/<variant-uid>/{pass,fail}/<scenario>/`.

**Coverage is 100% and the coverage gate holds it there** — a variant added without fixtures fails CI. There is no debt budget to grow. Where a variant asserts exactly what its own `filters:` require, no failing input exists; record that with a `fail/IMPOSSIBLE.md` marker explaining why, and it counts as covered. That marker is the only sanctioned way to ship a variant without a real fail fixture.

```bash
make test/content/iac/coverage
```

### The remediation has to satisfy the check

The closed-loop suite scans each IaC variant's **own remediation snippet** and requires the check that recommends it to pass. This is the question the linters cannot answer: cfn-lint, tflint and `bicep build` prove a snippet is well-formed, not that it is right. A snippet can name every property correctly and still demonstrate the exact misconfiguration the check forbids.

```bash
make test/content/iac/remediation
```

A variant whose remediation does not satisfy it yet is listed in `content/validation/scans/remediation-budget.json` with a reason. The list may only shrink: an entry that starts passing fails the test, so it is removed together with the fix, the same contract as a `KNOWN_BUG.md` marker.

See [validation/README.md](validation/README.md) for how the snippet is materialized and what each of the three failure modes means.

### Terraform remediation

Every parent check that has Terraform variants must also document how to fix the issue in Terraform. Add an `- id: terraform` entry to the `remediation:` list alongside the existing `id: console`, `id: cli`, `id: cloudformation`, `id: bicep` entries. The block holds a short Markdown intro and a fenced ```hcl``` example that resolves the violation.

Reference: `mondoo-aws-security-eks-cluster-cmks-in-kms` in `mondoo-aws-security.mql.yaml` shows the canonical structure (variants block + remediation list with `id: terraform` HCL alongside CLI/console/CloudFormation).

### When you can't write a variant or remediation, leave a YAML comment

If the runtime check has no Terraform analog, **leave a YAML comment above the parent check explaining why** so future passes don't re-investigate. Common reasons:

- The runtime check evaluates operational telemetry (job state, latest execution status, observed traffic) that has no configuration analog.
- The cloud resource is managed only via SDK / CLI / console and has no Terraform resource (e.g., short-lived imperative API calls like Vertex AI custom jobs).
- The runtime check depends on cross-resource correlation (e.g., "every cluster has a backup plan that points at it") that the runtime check itself does not yet implement correctly — in which case fix the runtime first.
- The runtime check inspects a field whose Terraform analog is a different feature (don't paper over the mismatch with a vacuous variant).

The `# No Terraform variants:` comment goes on the line before `- uid:`:

```yaml
# No Terraform variants: <one-sentence reason>. <Optional: when this could be revisited>.
- uid: mondoo-<cloud>-security-...
```

The `# No Terraform remediation:` comment goes **inside the `remediation:` list**, as its last line, indented at the same level as the `- id:` entries — where an `- id: terraform` entry would otherwise sit:

```yaml
remediation:
  - id: console
    desc: ...
  # No Terraform remediation: <one-sentence reason>. <Optional: when this could be revisited>.
```

When neither variants nor remediation are possible (the usual case — if you can't write a variant, you usually can't write Terraform remediation either), include both comments. Each comment must explain the technical limitation, not just say "skip".

### MQL parser quirk for Terraform variants

The parser rejects `.all((expr) || ...)` — a parenthesized clause as the first token inside `.all(`. Rely on `&&` binding tighter than `||` instead of writing leading parentheses.

## MQL syntax cheatsheet

```coffee
# Resource access
users.where(name == "root")

# Filtering and assertions
sshd.config.params["PermitRootLogin"] == "no"

# List operations
processes.list { name pid }

# Relationships
files("/etc").where(name == /\.conf$/)
```

For MQL resources available per provider, see [MQL resources documentation](https://mondoo.com/docs/mql/resources).

## Validation and testing

**[`validation/README.md`](validation/README.md) is the definitive reference** for every check that runs against this directory: what each one proves, when CI runs it, and how to run it yourself. What follows is the short version.

```bash
make test/content              # lint + bundle scans + compliance mappings
make test/content/lint         # cnspec policy lint (run this first, and fix it first)
make test/content/iac          # the IaC fixture suites — slow, run when you touch a variant
make test/content/remediation  # the remediation code-block linters
make test/content/commands     # the remediation CLI and API validators
```

`cnspec policy lint` must pass before committing any policy change. To lint or scan one policy:

```bash
cnspec policy lint content/mondoo-aws-security.mql.yaml
cnspec scan local -f content/your-policy.mql.yaml
```

Five things gate a content change, and each fails differently:

| What | Target | Fails when |
|---|---|---|
| Lint | `make test/content/lint` | a check does not compile against the provider schema |
| IaC fixtures | `make test/content/iac/<type>` | a check reaches the wrong verdict, or is silently **skipped** |
| Fixture coverage | `make test/content/iac/coverage` | a variant ships without pass+fail fixtures |
| Closed loop | `make test/content/iac/remediation` | the documented fix does not make the check pass |
| Remediation lint | `make test/content/remediation`, `make test/content/commands` | a snippet is malformed, or names a CLI flag or API endpoint that does not exist |

Three traps worth knowing before you hit them:

- **A validator only sees the policies in its `TARGETS`.** When a policy gains its first `terraform`/`ansible`/`bash`/`chef` remediation, add it to that validator's `TARGETS` in the same change — otherwise it ships unlinted and CI stays green. Terraform needs a `PROVIDER_MAP` entry too.
- **A skipped check is a fixture bug, not a pass.** A variant whose filter never matches anything looks identical to one that passes, in every report, forever.
- **A stale `KNOWN_BUG.md` marker fails the build.** Adding a check means deleting its markers in the same change.

**Never hand-edit** the checked-in CLI grammars and OpenAPI specs in `validation/data/`; re-run the relevant script in `validation/upstream/dump/` instead.

## Resources

- [MQL Documentation](https://mondoo.com/docs/mql)
- [MQL Built-in Functions](https://mondoo.com/docs/mql/functions)
- [MQL Resources by Provider](https://mondoo.com/docs/mql/resources) ([AWS](https://mondoo.com/docs/mql/resources/aws), [Azure](https://mondoo.com/docs/mql/resources/azure), [GCP](https://mondoo.com/docs/mql/resources/gcp), [Core](https://mondoo.com/docs/mql/resources/core))
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro)
