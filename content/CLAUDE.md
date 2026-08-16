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

The parts whose behavior is not obvious from reading a policy file:

- **`checks:` score, `queries:` do not.** A `queries:` entry collects data and never passes or fails.
- **`filters:` is asset selection, not logic.** A filter decides *which assets a check applies to* (`asset.platform == "aws"`). Predicate logic — `field != empty`, `flag == true`, a threshold — belongs in `mql:`. Lifting a predicate into `filters:` does not make the check stricter; it silently drops the failing assets from scoring, so the policy reports compliant on assets it never evaluated. Multi-line `filters:` join with an explicit `&&`; multi-line `mql:` uses newline-as-AND. **One exception, and it is a trap:** a filter is compiled as a single MQL snippet, so its lines join with newline-as-AND the same way `mql:` does. When a later line is itself an `||` chain — `platform == "terraform-hcl"` on one line, `resources.contains(a) || resources.contains(b)` on the next — adding the `&&` regroups it, because `&&` binds tighter: `platform && a || b` means `(platform && a) || b`, which matches assets of any platform. MQL has no parentheses to restore the grouping, so leave those filters on separate lines.
- **Multi-statement check `mql:`.** A check's `mql:` block can contain multiple top-level statements. Each is scored as a separate datapoint and the check passes only if every datapoint passes — it is *not* "last expression wins". Use this when you want each assertion to surface independently in scan output; collapse to a single `&&`-joined expression only if you want one combined datapoint.
- **`summary:`** is required, ≤130 chars. See Formatting requirements.

## Formatting requirements

- Every policy must have a `summary:` field — the one-line description shown in policy listings and the marketplace. It is **required** and must be **130 characters or fewer**. Write it verb-first (`Secure`, `Enforce`, `Validate`, `Detect`, `Harden`) followed by the concrete scope, matching the existing policies. Do **not** use em-dashes (`—`, `–`) or `--` in the summary; restructure the sentence instead.
- All `desc` and `remediation` fields must be valid Markdown (rendered in the UI). Use proper headings, lists, code blocks, links.
- **The em-dash ban is not only for `summary:`.** No `—`, `–`, or `--` anywhere in policy prose — `desc`, `audit`, `remediation`. The same goes for parenthetical asides: do not bolt a clarification onto a sentence with `(...)`. In both cases restructure the sentence or drop the aside; do not trade one for the other.
- **Write about the check, not about the bundle.** Policy prose is read next to a single finding, so it must not name variant UIDs or the `variants:` mechanism. Say "the Terraform version of this check", never "the `-terraform-hcl` variant". Do not reference "this policy", the MQL query, or Mondoo tooling.
- **Containers are not patched.** Images are rebuilt and redeployed. Remediation prose for container checks says so, rather than describing an in-place package upgrade that would be lost on the next deploy.
- **No inline linter suppressions in remediation snippets.** A shipped snippet is example code an operator will paste. If shellcheck, cfn-lint, or PSScriptAnalyzer flags it, fix the snippet or add the rule to the validator's exclude list — never `# shellcheck disable=...` inside the fence.
- **Spelling exceptions go in `typos.toml`** under `[default.extend-words]`, which is case-insensitive, so one lowercase entry covers every casing. There is no `expect.txt`. If the flagged word is a deliberate misspelling in an example, reword the example instead of allowlisting it.
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
4. **If no control fits, tag it with the YAML boolean `false`** — unquoted, like `compliance/soc2-2017: false`. **Not** `"false"` (the string), not `false-fit`, not `n/a`, not omitting the key. The unquoted boolean is the established repo convention (`grep -rho 'compliance/[a-z0-9-]*: false' content/*.mql.yaml | wc -l` counts over 5,000) and is what downstream tooling expects. A missing mapping is strictly better than a wrong one — wrong mappings get caught in compliance audits and create trust debt.
5. **Cite the control you chose.** When you present tags to the user, include the control title and a short quote from the control description so the user can verify.

**The `<framework>` in the key must exactly match the framework's `uid:` field** — the value declared inside the framework YAML, *not* the file name. They are not always the same: `cnspec-enterprise-policies/frameworks/bsi-grundschutz-sys15.mql.yaml` declares `uid: bsi-sys-1-5`, so the tag is `compliance/bsi-sys-1-5`. A key that matches no real framework `uid` generates a framework map with a dangling `framework_owner`, which fails bundle migration in `cnspec-enterprise-policies` with `cannot find framework owner`. Likewise the `<control-uid>` value must be a `uid` that exists under that framework's `controls:`.

### Which frameworks a check has to resolve

Step 1 says "each framework the policy already tags", and in practice that is the same list nearly everywhere. Fourteen frameworks are applied to essentially every tagged check in `content/`, so a new check resolves all fourteen — to a control uid, or to `false`:

`bsi-sys-1-5`, `csa-cloud-controls-matrix-4`, `dora`, `hipaa`, `iso-27001-2022`, `nis-2`, `nist-csf-1`, `nist-csf-2`, `nist-sp-800-171`, `nist-sp-800-53-rev5`, `owasp-top-10-2025`, `pci-dss-4`, `soc2-2017`, `vda-isa-5`

Four more are **subject-scoped** and are added only when the check's subject matter is genuinely in scope, not as part of the standard sweep: `owasp-llm-top-10-2025` and `nist-ai-100-1` (AI/LLM checks), `owasp-asvs-5` (application security verification), `mitre-attack` (checks that map to a specific adversary technique).

Confirm the current list rather than trusting this one, since it moves as frameworks are added:

```bash
grep -rho "compliance/[a-z0-9-]*:" content/*.mql.yaml | sort | uniq -c | sort -rn
```

The frameworks in the standard set appear on nearly every tagged check; the subject-scoped ones appear on a few dozen to a few hundred. **A policy missing one of the standard fourteen entirely is drift, not a decision** — `mondoo-postgresql-security.mql.yaml` carries thirteen and no `owasp-top-10-2025` across all 29 of its checks, which is what an undocumented convention looks like after one policy is written without it.

**The control uid is not derived from the framework key.** Several frameworks name their controls with a prefix that does not match the key you tag them under, so a constructed uid compiles and maps to nothing:

| Tag key | A real control uid under it |
|---|---|
| `compliance/pci-dss-4` | `pcidss-requirement-10-2-1` (no hyphens in `pcidss`) |
| `compliance/soc2-2017` | `soc2-control-cc6-8-1` (`-control-` infix) |
| `compliance/csa-cloud-controls-matrix-4` | `cloud-controls-matrix-4-log-08` (drops `csa-`) |
| `compliance/nist-sp-800-171` | `nist-sp-800-171--3-4-8` (**double** hyphen) |
| `compliance/hipaa` | `hipaa-security-ss164-312-b-audit-controls` |

Read the control uid out of the framework YAML. Do not construct it from the framework name.

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

Every variant in the corpus satisfies its own remediation, so this is a flat assertion with no debt list to add to. A snippet that stops closing its check fails here.

Two shapes account for nearly every failure. A snippet that documents only the fixing resource never triggers a check whose `filters:` select on the resource being protected, so it has to declare that resource too. And a value the HCL parser cannot resolve statically, such as a `jsonencode` body containing a resource reference or a policy supplied through an `aws_iam_policy_document` data source, reads as absent rather than as the value it will become at apply time.

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

### Terraform variants read a different shape than the runtime check

The HCL, plan, and state variants of one check are **not** interchangeable, and the usual bug is assuming they are:

- **A default-true attribute must be asserted as `!= false`, not `== true`.** An attribute the author omitted is `null`, and `null == true` fails, so `== true` flags every correct config that relied on the default.
- **An absent block is not neutral.** Whether a missing Terraform block means pass or fail depends on the vendor's API default for that setting, which you have to look up. Two attributes in the same block can go opposite ways.
- **In plan and state JSON, nested blocks serialize as arrays**, and an omitted optional block is `[]`, not `null`. A missing block usually means the insecure default, so guard with `!= empty &&` rather than letting the empty list pass vacuously.
- **The HCL variant is often stricter than its plan/state siblings**, legitimately — HCL sees the author's intent, plan/state see a resolved value. Do not "unify" variants by copying one body into another; that silently weakens the strict one.
- Verify every attribute name against the provider's real schema (`terraform providers schema -json`) rather than from memory.

## MQL traps that produce a confidently wrong verdict

Most of these do not fail lint and do not error at scan time. They return a **verdict**, and the verdict is wrong — worse than a broken check, because nothing signals it. Read this section before writing a query, not after a reviewer questions one.

**Check the field exists before you write the query.** The installed provider schema is what lint resolves against: `~/.config/mondoo/providers/<name>/<name>.resources.json`. The source of truth is `providers/<name>/resources/<name>.lr` in the [mql repo](https://github.com/mondoohq/mql). Prove the query end to end with `cnquery run <provider> -c '<mql>'`, and read the verdict as `[ok]` / `[failed]` — a check asserting `x == false` prints `[ok] value: false` when it passes, so "value: false" in the output is not a failure.

### There is no parenthesized grouping

`(` is not a valid operand anywhere in MQL. This one *is* a compile error, but it is listed here because the usual response to it is wrong:

```
(a == 1) || (a == 2)          → expected operand, got token "("
[1,2].all((_ == 1) || _ == 2) → expected closing ')', got '('
[1,2].all(_ == 1 || (_ == 2)) → expected operand, got token "("
```

Rely on `&&` binding tighter than `||`, so `a || b && c` already parses as `a || (b && c)`, or split the assertion into separate `.any()` / `.all()` calls. Never add parentheses "for clarity" — and decline review suggestions that ask for them.

### A guard chain is not a skipped check

The dominant shape in this directory is a **guard chain**: some `||`-joined guards that exempt an asset, then the assertion as the final `&&` conjunct.

```coffee
aws.batch.jobDefinition.status != "ACTIVE" ||            # guard: not in scope
aws.batch.jobDefinition.container == null ||             # guard: nothing to check
aws.batch.jobDefinition.container.jobRole == null ||     # guard: no role attached
aws.batch.jobDefinition.container.jobRole.inlinePolicyDetails.length == 0 &&
aws.batch.jobDefinition.container.jobRole.attachedPolicies.all(…)
```

Because `&&` binds tighter, that parses as `guard || guard || guard || (D && E)`, which is the intended semantics. Reviewers — human and automated — repeatedly misread the `D && E` tail as "E only runs when D is true, so the check is skipped and silently passes." **That inference is wrong, and it is the single most-filed false positive on this repo.**

Short-circuit evaluation decides *what gets evaluated*, never *what the verdict is*. In a disjunction, a false conjunct makes the whole expression false, so the check **fails**:

| `jobRole` present | inline policies exist (`D`) | attached clean (`E`) | verdict |
|---|---|---|---|
| yes | **yes** (`D` false) | yes | **`[failed]`** — the inline policy *is* the violation |
| yes | no | no (`E` false) | `[failed]` |
| yes | no | yes | `[ok]` |
| no | — | — | `[ok]` via the guard |

There is no input for which the guard-chain form passes and a "fully parenthesized" form would fail. Before claiming a precedence bug, build that table and name the row that differs. If no row differs, there is no bug.

Two corollaries:

- The suggested fix is usually **not expressible** — MQL rejects `(`, so the grouping has to come from precedence (see the section above).
- A guard chain and a pure `&&` chain differ in what an **absent** field means, not in precedence. `role == null ||` *passes* an asset with no role; a pure `&&` chain *fails* it. That is a deliberate authoring decision about absent data, so do not "unify" the two shapes.

### Do not verify a literal by eye in a rendered diff

Character-level claims about string literals — a colon count in an ARN, a missing path segment, a truncated prefix — are not reliable from diff output, where proportional rendering and syntax highlighting distort spacing. One PR collected five findings against `arn:aws:iam::aws:policy/ReadOnlyAccess`, each proposing a different "correct" form and three retracting themselves mid-comment. The ARN was canonical throughout.

Resolve the literal against the system that owns it, and quote the result:

```bash
aws iam get-policy --policy-arn arn:aws:iam::aws:policy/ReadOnlyAccess --query 'Policy.Arn' --output text
# arn:aws:iam::aws:policy/ReadOnlyAccess
```

AWS-managed policy ARNs have an **empty account field**, so `iam` is followed by exactly two colons. Same rule for any literal with an authoritative oracle: `cfn-lint` for CloudFormation resource types and property names, `az`/`gcloud`/`aws` for CLI grammar, the provider schema for field paths.

### Comparison against an unresolved field is asymmetric

A field the provider never populated is `null`, and null does not compare like a value. Against a missing map key (`m = {"a": 1}`):

| Written as | Verdict when the field is absent |
|---|---|
| `m["b"] == "x"` | **fails** |
| `m["b"] != "x"` | **passes** |
| `m["b"] != ""` | **passes** — `!= ""` is not a non-empty test |
| `m["b"] != empty` | **fails** — this is the null-safe guard |

The trap is the second row. A check phrased in the negative — `setting != "insecure"`, `mode != "off"` — **passes on every asset where the field never resolved**. It should be inconclusive; it reports compliant. Assert presence first:

```coffee
setting != empty && setting != "insecure"
```

Prefer `!= empty` over `!= ""` for the same reason: `"" == empty` is true, but so is `null != ""`.

### `.all()` and `.none()` treat null and empty differently

An absent HCL or map key is `null`, **not** an empty list, and the two go opposite ways:

```coffee
[1,2].where(_ > 5).all(_ == 99)   # empty list  → [ok] value: true   (vacuous pass)
m["missing"].all(_ == 1)          # null        → [failed] actual: _
m["missing"] == empty || m["missing"].all(_ == 1)   # [ok] true      (null-safe form)
```

So `blocks.where(type == 'x').all(y)` and `values['x'].all(y)` are **not** equivalent rewrites: the first is vacuously true when nothing matches, the second fails outright. Do not swap one for the other — the vacuous pass and the hard fail are both wrong answers for "the block is absent", and which one you want depends on the vendor default (see the Terraform section above).

### A dotted path that is also a resource name is not a field read

The compiler extends the resource path greedily, and the longest matching resource name wins unconditionally over a field on a shorter one — even when the longer resource cannot stand on its own. `azure.subscription.aksService.cluster.autoUpgradeProfile.upgradeChannel` builds a bare `…cluster.autoUpgradeProfile` resource with no id and no fields; the cluster's accessor never runs and every field reads `null`. Combined with the asymmetry above, the check returns a confident wrong answer rather than an error.

Suspect it when the value is a sub-object (a profile, config, or settings block) **and** the full path appears as a resource in its own right in `cnspec providers resources <provider> --json`. Confirm by running the query: the log line is `provider returned no data and no error for a field … id=` with an **empty `id=`**. Fix by reaching the value through an accessor whose path is not a resource name (`azure.subscription.aks.cluster.…`) or by binding a block to the parent:

```coffee
azure.subscription.aks.cluster {
  autoUpgradeProfile.upgradeChannel != "none"
}
```

Not Azure-specific: Cloudflare (`cloudflare.zone.settings.*`), GCP (`gcp.project.gkeService.cluster.networkPolicy.*`), AWS (`aws.emr.cluster.encryptionConfiguration.*`), vSphere, and Arista all have resources shaped this way. Cloudflare adds a second failure mode — a 401/403 degrades to an empty list rather than an error, so an unauthorized scan passes vacuously.

### Smaller ones that still flip a verdict

- **`files.find` regex matches the whole path**, not the basename. A pattern without a leading `.*` matches nothing, and the `.all()` wrapped around it then passes vacuously. There are 17 `files.find` regex usages in `content/` today; check yours against a real path before trusting it.
- **Flatten with `.flat`, not `.flatten`** — and guard the nested key first.
- **`map[string][]string` is filtered, never indexed.** Use `keys` / `values` / `where(key …)` / `flat`. Indexing a missing key returns `null`, which collides with a legitimately empty list.
- **`terraform.resources()` takes a positional type argument** — `terraform.resources("aws_s3_bucket")`. The named form does not compile.
- **`parse.int` fails inside variant queries.** Use date arithmetic or a string match instead. For port ranges there is no string→int conversion at all, so match with an anchored regex.
- **GCP dict fields omit defaults.** `protoToDict` drops `false`, `0`, and `""` and camelCases the keys, so assert presence rather than `== false`.

## Validation and testing

**[`validation/README.md`](validation/README.md) is the definitive reference** for every check that runs against this directory: what each one proves, when CI runs it, and how to run it yourself. What follows is the short version.

```bash
make test/content              # lint + bundle scans + compliance mappings
make test/content/lint         # cnspec policy lint (run this first, and fix it first)
make test/content/iac          # every IaC fixture suite — slow; scope it instead, see below
make test/content/remediation  # the remediation code-block linters
make test/content/commands     # the remediation CLI and API validators
```

`cnspec policy lint` must pass before committing any policy change. To lint or scan one policy:

```bash
cnspec policy lint content/mondoo-aws-security.mql.yaml
cnspec scan local -f content/your-policy.mql.yaml
```

**Do not run the whole IaC suite to test one check.** Subtests are named `<policy>/<check-uid>/<pass|fail>/<scenario>`, so a `-run` pattern with a **trailing slash** scopes it to a single check in a couple of seconds:

```bash
go test -tags iac_variants ./content/validation/scans \
  -run 'TestTerraformVariants/mondoo-aws-security/mondoo-aws-security-s3-bucket-encryption-terraform-hcl/'
```

The trailing slash is what makes it work; without it the pattern matches the suite name and runs everything under it.

Five things gate a content change, and each fails differently:

| What | Target | Fails when |
|---|---|---|
| Lint | `make test/content/lint` | a check does not compile against the provider schema |
| IaC fixtures | `make test/content/iac/<type>` | a check reaches the wrong verdict, or is silently **skipped** |
| Fixture coverage | `make test/content/iac/coverage` | a variant ships without pass+fail fixtures |
| Closed loop | `make test/content/iac/remediation` | the documented fix does not make the check pass |
| Remediation lint | `make test/content/remediation`, `make test/content/commands` | a snippet is malformed, or names a CLI flag or API endpoint that does not exist |

Three traps worth knowing before you hit them:

- **A validator only sees the policies in its `TARGETS`.** When a policy gains its first `terraform`/`ansible`/`powershell`/`chef` remediation, add it to that validator's `TARGETS` in the same change — otherwise it ships unlinted and CI stays green. Terraform needs a `PROVIDER_MAP` entry too. (`bash.py` is the exception: it globs every policy, deliberately.)
- **A skipped check is a fixture bug, not a pass.** A variant whose filter never matches anything looks identical to one that passes, in every report, forever.
- **A stale `KNOWN_BUG.md` marker fails the build.** Adding a check means deleting its markers in the same change.

**Never hand-edit** the checked-in CLI grammars and OpenAPI specs in `validation/data/`; re-run the relevant script in `validation/upstream/dump/` instead.

### A new policy is not covered until it is registered

A new *check* inherits the coverage its policy already has. A new *policy* inherits nothing. Every validator above is allowlist-driven except `bash.py` and the compliance suites, so a brand-new `*.mql.yaml` is not reported as unvalidated; it is simply never visited. The bundle merges with its variants untested, its HCL unlinted and its CLI commands unverified, and every gate stays green.

This is not hypothetical. The four SaaS policies added in PR #3338 were invisible to the remediation validators. Registering them with `terraform.py` in a follow-up commit failed **8 of their 11** HCL snippets against the real provider schemas, and hand-checking their CLI snippets found an entire invented `hcp vault` / `hcp consul` command surface and two `neonctl` flags that do not exist.

Wire the policy up in the **same change** that adds it. [`validation/README.md`](validation/README.md#adding-a-policy-what-to-register) is the definitive list, with what breaks in each case; in short, the policy has to be named in:

- `content/README.md`, the user-facing catalog, always.
- `tfVariantPolicies` in `validation/scans/iac_variants_test.go`, if it has any IaC variant, plus that file's `extraProviders` if its runtime variant needs a provider beyond the base six.
- `TARGETS` in each `validation/remediation/code/<language>.py` whose method it ships, and `PROVIDER_MAP` as well for Terraform.
- a registry under `validation/remediation/commands/`, if it ships `id: cli` / `id: api` blocks or `audit:` steps that invoke a CLI or REST call.
- `typos.toml`, if the vendor's terminology trips the spell checker.

Adding a policy for a platform with **no** non-interactive surface is a legitimate outcome, not a reason to skip a row. Record it as a comment on the check explaining what the vendor does not expose, so the next pass does not re-derive it, and still register the policy with the validators it *can* use so anything added later is checked from the start.

### Groups in a policy with variants carry no platform filter

This one is an authoring decision, not a registration step, and it is invisible until the fixtures exist.

A group filter is evaluated **before** the check's own filter, so a group carrying `filters: asset.platform == "<api-platform>"` means a Terraform asset never reaches the variant underneath it. Every fixture then reports as *skipped* rather than pass or fail, which is the outcome this whole suite exists to catch.

A policy with variants therefore leaves its groups unfiltered and lets each variant's own `filters:` select its asset, the way `mondoo-tailscale-security` and `mondoo-snowflake-security` do. Convert the groups in the same change that adds the first variant.

Reference links for MQL resources, built-in functions, and the authoring guide are in the repository-root [`CLAUDE.md`](../CLAUDE.md), which loads alongside this file.
