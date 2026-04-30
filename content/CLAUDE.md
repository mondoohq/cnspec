# content/CLAUDE.md

Policy authoring guidance for `*.mql.yaml` files in this directory. Loaded automatically when working under `content/`.

## Bundle structure

```yaml
policies:
  - uid: example-policy
    name: Example Policy
    version: 1.0.0
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
- **filters**: MQL expressions that determine applicability.
- **impact**: Risk score 0-100 for prioritization.
- **checks**: Scoring queries (pass/fail).
- **queries**: Data collection queries (no scoring).
- **Multi-statement check MQL**: A check's `mql:` block can contain multiple top-level statements. Each statement is scored as a separate datapoint and the check passes only if every datapoint passes — it is *not* "last expression wins". Use this pattern when you want each assertion to surface independently in scan output; collapse to a single `&&`-joined expression only if you want one combined datapoint.

## Formatting requirements

- All `desc` and `remediation` fields must be valid Markdown (rendered in the UI). Use proper headings, lists, code blocks, links.
- Check `title` fields must be 75 characters or fewer.
- Verify CLI commands in remediation steps with the validator (see below) before committing.

## Compliance tags (`compliance/<framework>: <control-uid>`)

**Never copy compliance tags from a neighboring check.** The nearby check was mapped for a different control objective; reusing its tags propagates a wrong mapping and misleads auditors. Two checks that both "relate to identity" can map to different controls.

When adding or changing compliance tags, follow this process for **each** framework the policy already tags:

1. **Read the authoritative control text.** Open the framework definition in `cnspec-enterprise-policies/frameworks/<framework>.mql.yaml` (e.g., `iso-27001-2022.mql.yaml`, `soc2-2017.mql.yaml`, `nist-sp-800-53-rev5.mql.yaml`). Each control has a `uid`, `title`, and usually `docs.desc`. Ask the user where their clone lives if you don't already know; if the files aren't available, stop and tell the user — do not guess.
2. **State in one sentence what the check actually enforces.** If the check is about identity proofing, say so; if it's about encryption-at-rest, say so. Do not let the check's *title* mislead you — read the MQL.
3. **Find the single best-matching control** by scanning control titles and descriptions for language that covers the enforced behavior. Strict fit only: MFA, password policy, and session-timeout controls are *not* acceptable stand-ins for identity-proofing, encryption, network-isolation, etc.
4. **If no control fits, tag it with the YAML boolean `false`** — unquoted, like `compliance/soc2-2017: false`. **Not** `"false"` (the string), not `false-fit`, not `n/a`, not omitting the key. The unquoted boolean is the established repo convention (grep `compliance/.*: false` for ~150+ examples) and is what downstream tooling expects. A missing mapping is strictly better than a wrong one — wrong mappings get caught in compliance audits and create trust debt.
5. **Cite the control you chose.** When you present tags to the user, include the control title and a short quote from the control description so the user can verify.

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

Comment formats (inserted on the line before `- uid:`):

```yaml
# No Terraform variants: <one-sentence reason>. <Optional: when this could be revisited>.
- uid: mondoo-<cloud>-security-...
```

```yaml
# No Terraform remediation: <one-sentence reason>. <Optional: when this could be revisited>.
- uid: mondoo-<cloud>-security-...
```

When neither variants nor remediation are possible (the usual case — if you can't write a variant, you usually can't write Terraform remediation either), include both comments. The comment must explain the technical limitation, not just say "skip".

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

For MQL resources available per provider, see [MQL resources documentation](https://mondoo.com/docs/mql/resources/).

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

# Validate a specific cloud
python3 content/validation/validate_remediation_commands.py aws
python3 content/validation/validate_remediation_commands.py azure
python3 content/validation/validate_remediation_commands.py oci
python3 content/validation/validate_remediation_commands.py gcp
python3 content/validation/validate_remediation_commands.py digitalocean
python3 content/validation/validate_remediation_commands.py cloudflare
```

The validator scans each `aws`/`az`/`oci`/`gcloud`/`doctl` CLI command — and `curl` calls against `api.cloudflare.com` — in ```` ```bash ```` code blocks within `id: cli` remediation sections. Output shows `[PASS]` or `[FAIL]` with the check UID and the offending command.

**How the validator sources command data:**

For `aws`, `oci`, `gcp`, and `digitalocean`, the database is built **in-memory** at validation time by introspecting the locally-installed CLI. The relevant CLI must be on PATH:

- **aws**: introspects botocore service models bundled with the AWS CLI v2
- **oci**: walks the Click command tree from the `oci_cli` Python package
- **gcp**: reads the Google Cloud SDK's static completion tree
- **digitalocean**: walks the `doctl --help` Cobra tree breadth-first (parallelized; ~1s for the full ~475-command tree)
- **cloudflare**: downloads Cloudflare's published OpenAPI spec from [`cloudflare/api-schemas`](https://github.com/cloudflare/api-schemas) at a pinned commit (no Cloudflare CLI required — the validator scans `curl` calls against `api.cloudflare.com/client/v4` and verifies each path + HTTP method against the spec). Bump `CLOUDFLARE_OPENAPI_SHA` in `validate_remediation_commands.py` when refreshing the spec.

If a required CLI is missing, the validator prints actionable install hints and exits non-zero.

**azure** is the exception: it uses a checked-in `content/validation/cmd_data/azure_commands.json` because refreshing Azure CLI metadata is slow enough that doing it on every run would significantly extend CI.

**Regenerate Azure command data** (when the Azure CLI version changes):

```bash
python3 content/validation/dump_azure_commands.py
```

**Never hand-edit** `azure_commands.json`.

## Resources

- [MQL Documentation](https://mondoo.com/docs/mql/)
- [MQL Built-in Functions](https://mondoo.com/docs/mql/functions)
- [MQL Resources by Provider](https://mondoo.com/docs/mql/resources/) ([AWS](https://mondoo.com/docs/mql/resources/aws-pack/), [Azure](https://mondoo.com/docs/mql/resources/azure-pack/), [GCP](https://mondoo.com/docs/mql/resources/gcp-pack/), [Core](https://mondoo.com/docs/mql/resources/core-pack/))
- [Policy Authoring Guide](https://mondoo.com/docs/cnspec/write-policies/write-intro/)
