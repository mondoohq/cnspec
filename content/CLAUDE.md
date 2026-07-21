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
  - **Windows / macOS**: `gui`, `cli`, `ansible`, and `script` (PowerShell on Windows, bash on macOS)
  - **Linux**: `cli`, `script` (bash), `ansible`
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

- `<uid>-<cloud>` — runtime check (`asset.platform == 'gcp'`, `'aws'`, …)
- `<uid>-terraform-hcl` — `terraform.resources(...)` against HCL source
- `<uid>-terraform-plan` — `terraform.plan.resourceChanges` against `terraform plan` JSON
- `<uid>-terraform-state` — `terraform.state.resources` against `terraform.tfstate`

Reference patterns in this repo:

- GCP: `mondoo-gcp-security-memorystore-iam-auth-enabled` in `mondoo-gcp-security.mql.yaml`
- HCL nested-block fanout: `mondoo-gcp-security-cloud-sql-mysql-skip-show-database-enabled-terraform-*` (database_flags)
- Plan/state list-of-objects shape: `mondoo-gcp-security-cloud-storage-bucket-retention-policy-locked-terraform-*`

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

## Linting and testing

```bash
# Lint a single policy
cnspec policy lint content/mondoo-aws-security.mql.yaml

# Lint everything in this directory
cnspec policy lint ./content

# Test locally against a target
cnspec scan local -f content/your-policy.mql.yaml
```

`cnspec policy lint` must pass before committing any policy changes.

## Validating remediation CLI commands

The `content/validation/` directory contains tooling to verify that CLI commands in remediation sections use valid subcommands and flags.

```bash
# Validate all clouds
python3 content/validation/validate_remediation_commands.py

# Validate a specific cloud CLI
python3 content/validation/validate_remediation_commands.py aws
python3 content/validation/validate_remediation_commands.py azure
python3 content/validation/validate_remediation_commands.py oci
python3 content/validation/validate_remediation_commands.py gcp
python3 content/validation/validate_remediation_commands.py digitalocean
python3 content/validation/validate_remediation_commands.py nutanix

# Validate Vercel (runs BOTH the `vercel` CLI and the REST API checks)
python3 content/validation/validate_remediation_commands.py vercel

# Validate a Cobra-based CLI (kubectl / gh / glab / hcloud / databricks)
python3 content/validation/validate_remediation_commands.py kubernetes
python3 content/validation/validate_remediation_commands.py github
python3 content/validation/validate_remediation_commands.py gitlab
python3 content/validation/validate_remediation_commands.py hetzner
python3 content/validation/validate_remediation_commands.py databricks

# Validate a specific REST API (curl commands against a vendor API)
python3 content/validation/validate_remediation_commands.py cloudflare
python3 content/validation/validate_remediation_commands.py tailscale
python3 content/validation/validate_remediation_commands.py slack
python3 content/validation/validate_remediation_commands.py atlassian
python3 content/validation/validate_remediation_commands.py grafana
python3 content/validation/validate_remediation_commands.py mongodbatlas
```

The validator scans each `aws`/`az`/`oci`/`gcloud`/`doctl`/`ncli`/`vercel`/`kubectl`/`gh`/`glab`/`hcloud`/`databricks` CLI command — and `curl` calls against the registered vendor API hosts — in ```` ```bash ```` code blocks within `id: cli` remediation sections. For the REST API and Cobra CLI targets, ```` ```bash ```` blocks in `audit:` sections are validated too (these products' verification paths use the same CLI/API surface, and the new validators have no backlog of unvalidated audit blocks). Output shows `[PASS]` or `[FAIL]` with the check UID and the offending command.

The `vercel` target is the only one that runs a **CLI validator and a REST API validator together** (both keyed `vercel`): the Vercel policy fixes some settings with the `vercel` CLI (`- id: cli`) and others with `curl` against the Vercel REST API (`- id: api`). Both remediation ids and the `audit:` blocks are validated.

The code lives in the `content/validation/validators/` package — one module per validator family (`aws.py`, `azure.py`, …, `openapi.py` for all REST APIs), shared helpers in `common.py` — with `validate_remediation_commands.py` as the entry-point shim.

The `azure` target validates **both** `mondoo-azure-security.mql.yaml` and `mondoo-m365-security.mql.yaml`, because the M365 policy's `id: cli` remediations also use the Azure CLI (`az`).

**How the validator sources command data:**

For `aws`, `oci`, `gcp`, and `digitalocean`, the database is built **in-memory** at validation time by introspecting the locally-installed CLI. The relevant CLI must be on PATH:

- **aws**: introspects botocore service models bundled with the AWS CLI v2
- **oci**: walks the Click command tree from the `oci_cli` Python package
- **gcp**: reads the Google Cloud SDK's static completion tree
- **digitalocean**: walks the `doctl --help` Cobra tree breadth-first (parallelized; ~1s for the full ~475-command tree)
- **kubernetes / github / gitlab / hetzner / databricks**: walk the CLI's hidden Cobra `__complete` command (`kubectl`/`gh`/`glab`/`hcloud`/`databricks` must be on PATH), which returns machine-readable subcommand and flag candidates regardless of each CLI's custom help layout. Valid flags are the union of flag completions and flag lines parsed from `--help` (hcloud filters its flag completions to required-only). The walk runs with cloud credentials stripped (`KUBECONFIG=/dev/null`, tokens unset, `DATABRICKS_CONFIG_FILE=/dev/null` so the databricks CLI loads no profile) so positional-value completions stay empty, and only tab-described candidates count as subcommands (kubectl's `rollout restart` statically completes resource types as bare names). Each CLI is an entry in the `COBRA_CLIS` registry in `validators/cobra.py`.

The REST API targets (**cloudflare**, **tailscale**, **slack**, **atlassian**, **grafana**, **vercel**, **mongodbatlas**) need no CLI: each provider is an entry in the `API_PROVIDERS` registry in `validators/openapi.py` that maps an API host to the vendor's OpenAPI (or Swagger 2.0) spec. The validator verifies each curl call's path + HTTP method, plus the `--data` JSON payload against the operation's `requestBody` schema: field names, types, enums, and required properties. Angle-bracket (`<account-name>`) and environment-variable (`$ORG_ID`) placeholders act as wildcards. Known spec-vs-docs divergences are listed per provider under `body_exemptions`; Cloudflare additionally narrows the generic `/zones/{zone_id}/settings/{setting_id}` schema to the per-setting component via its `path_hook`. Two more per-provider options exist for API-first products: `strip_api_version` normalizes a leading `/vN` URL segment on both the curl path and the spec (Vercel versions every path and keeps several versions live, but the spec documents one version per operation), and `path_exemptions` allowlists endpoints the API serves but omits from its published spec.

API specs are sourced two ways:

- **Pinned download** (cloudflare, slack, grafana, mongodbatlas): the spec lives in a git repo, so it's downloaded at validation time from a raw URL pinned to a commit SHA (cached under `~/.cache/cnspec-validation/`). Bump the `*_OPENAPI_SHA` constant in `validators/openapi.py` to refresh.
- **Checked-in dump** (tailscale, atlassian, vercel): the vendor serves the spec from a live, unversioned endpoint, so it's checked into `cmd_data/` and refreshed with `python3 content/validation/dump_api_specs.py`. The Vercel spec is stored minified (~2.9 MiB vs. ~9.5 MiB pretty-printed).

If a required CLI is missing, the validator prints actionable install hints and exits non-zero.

**azure** and **vercel** are the exceptions among the CLI targets: they use a checked-in command grammar (`cmd_data/azure_commands.json`, `cmd_data/vercel_commands.json`) rather than live introspection. Azure CLI metadata is too slow to refresh every run; `vercel` is a Node.js CLI with no completion surface (not Cobra, no botocore/Click tree), so `dump_vercel_commands.py` parses its `--help` tree once — scoped to the command groups the policy uses — and checks the result in.

**Regenerate checked-in command/spec data**:

```bash
python3 content/validation/dump_azure_commands.py   # when the Azure CLI version changes
python3 content/validation/dump_ncli_commands.py    # when bumping the pinned AOS release
python3 content/validation/dump_vercel_commands.py  # when bumping the pinned vercel CLI version
python3 content/validation/dump_api_specs.py        # Tailscale + Atlassian + Vercel API specs
```

**Never hand-edit** the files in `cmd_data/`.

## Resources

- [MQL Documentation](https://mondoo.com/docs/mql)
- [MQL Built-in Functions](https://mondoo.com/docs/mql/functions)
- [MQL Resources by Provider](https://mondoo.com/docs/mql/resources) ([AWS](https://mondoo.com/docs/mql/resources/aws), [Azure](https://mondoo.com/docs/mql/resources/azure), [GCP](https://mondoo.com/docs/mql/resources/gcp), [Core](https://mondoo.com/docs/mql/resources/core))
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro)
