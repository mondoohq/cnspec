# content/CLAUDE.md

Authoring rules for the `*.mql.yaml` policies in this directory. Loads automatically when working under `content/`. Terse on purpose.

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

The parts you can't infer from reading a policy file:

- **`checks:` score, `queries:` don't.** A `queries:` entry collects data. It never passes or fails.
- **`filters:` is asset selection, not logic.** A filter picks *which assets a check applies to* (`asset.platform == "aws"`). Predicates (`field != empty`, `flag == true`, a threshold) belong in `mql:`. Lifting a predicate into `filters:` doesn't make the check stricter, it drops the failing assets from scoring, so the policy reports compliant on assets it never evaluated.
- **Multi-line `filters:` join with an explicit `&&`.** Multi-line `mql:` uses newline-as-AND. One exception, and it's a trap: a filter compiles as a single MQL snippet, so its lines also join with newline-as-AND. When a later line is itself an `||` chain (`platform == "terraform-hcl"` on one line, `resources.contains(a) || resources.contains(b)` on the next), adding the `&&` regroups it, because `&&` binds tighter: `platform && a || b` means `(platform && a) || b`, which matches any platform. MQL has no parens to restore the grouping, so leave those filters on separate lines.
- **A check's `mql:` can hold several top-level statements.** Each is scored as its own datapoint and the check passes only if all of them pass. It is *not* "last expression wins". Use it when you want each assertion to surface separately in scan output; collapse to one `&&`-joined expression if you want a single datapoint.
- **`summary:` is required, 130 chars max.** See below.

## Formatting requirements

- **`summary:`** is required on every policy, 130 chars or fewer. It's the one-liner shown in policy listings and the marketplace. Verb-first (`Secure`, `Enforce`, `Validate`, `Detect`, `Harden`) plus the concrete scope, matching existing policies.
- **No `—`, `–`, or `--` anywhere in policy prose.** Not in `summary`, `desc`, `audit`, or `remediation`. Same for parenthetical asides: don't bolt a clarification onto a sentence with `(...)`. Restructure the sentence or drop the aside. Don't trade one for the other.
- **`desc` and `remediation` must be valid Markdown** (they render in the UI): headings, lists, code blocks, links.
- **Write about the check, not the bundle.** Policy prose sits next to a single finding, so it must not name variant UIDs or the `variants:` mechanism. Say "the Terraform version of this check", never "the `-terraform-hcl` variant". Don't reference "this policy", the MQL query, or Mondoo tooling.
- **Containers aren't patched.** Images get rebuilt and redeployed. Container remediation says that, rather than describing an in-place package upgrade that the next deploy wipes.
- **No inline linter suppressions in remediation snippets.** A shipped snippet is example code someone will paste. If shellcheck, cfn-lint, or PSScriptAnalyzer flags it, fix the snippet or add the rule to the validator's exclude list. Never `# shellcheck disable=...` inside the fence.
- **Spelling exceptions go in `typos.toml`** under `[default.extend-words]`, which is case-insensitive, so one lowercase entry covers every casing. There is no `expect.txt`. If the flagged word is a deliberate misspelling in an example, reword the example instead of allowlisting it.
- **`title` must match what the `mql` asserts and what `desc` explains.** "Ensure X is enabled" means the query asserts X is enabled. Titles drift: a title about encryption at rest paired with a query inspecting TLS is the classic case.
- **Every check's `docs:` needs all three of `desc:`, `audit:`, `remediation:`.** None optional. `desc` = what and why, `audit` = how to verify by hand, `remediation` = how to fix.
- **`audit:` uses the vendor's own tooling**, the cloud console or vendor CLI (`aws`, `az`, `gcloud`, `oci`, `doctl`, `kubectl`, `gh`, ...). Never `cnspec`, `mql`, or the Mondoo console. Prefer the CLI; fall back to console click-through where no CLI exists. The point is to let an auditor reproduce the finding without trusting Mondoo's output.
- **`remediation:` covers every method the platform supports**, not one or two. Use `- id: <method>` entries:
  - **AWS**: `console`, `cli`, `terraform`, `cloudformation`
  - **Azure**: `portal`, `cli`, `terraform`, `bicep` (Azure's web UI is called the portal, so `portal`, not `console`)
  - **GCP / OCI / DigitalOcean / Cloudflare / Hetzner / other clouds**: `console`, `cli`, `terraform`
  - **Windows / macOS**: `gui`, `cli`, `ansible`, `script` (PowerShell on Windows, bash on macOS). Plus `chef` in `mondoo-windows-security`.
  - **Linux / FreeBSD**: `cli`, `script` (bash or sh), `ansible`. Plus `chef` in `mondoo-linux-security` and `mondoo-freebsd-security`.
  - **Kubernetes**: `kubectl`, `manifest` (YAML), and `helm` where applicable
  - **Microsoft 365** (`mondoo-m365-security`): `console`, `powershell`, and `terraform` where a real resource exists.
    - `console` is the relevant Microsoft admin center (Entra, Microsoft 365, Defender, Exchange, SharePoint, Intune). Always applies.
    - `powershell` is Microsoft Graph PowerShell for Entra/identity, Exchange Online PowerShell for Exchange, SharePoint Online Management Shell for SharePoint. Always applies, except DNS-record controls (SPF), which use `cli` (`az network dns ...`).
    - `terraform` only where the `azuread` provider (or a DNS provider such as `azurerm` for DNS records) has a genuine resource. Conditional Access (`azuread_conditional_access_policy`) and role assignments (`azuread_directory_role_assignment`) qualify. Exchange Online, SharePoint Online, Intune device config, the tenant authorization policy, the security-defaults toggle, and per-domain password validity have none: use a `# No Terraform remediation:` comment, never a `null_resource` + `local-exec` shell-out or an `azapi` block faking one.

  A `chef` entry is a Chef Infra recipe, matching the ones in the two Chef Infra policies. Add one wherever Chef Infra Client runs on the asset and the fix changes state on that host. Prefer a purpose-built resource over a shell-out: `registry_key`, `windows_security_policy`, `windows_audit_policy`, and `windows_user_privilege` cover most of the Windows policy; `sysctl`, `kernel_module`, `systemd_unit`, `user_ulimit`, and `sudo` cover most of the Linux one. On FreeBSD the `sysctl` resource writes `/etc/sysctl.d`, which the base system doesn't read, so kernel parameters go in `/etc/sysctl.conf`; the `service` provider there edits `/etc/rc.conf` directly, making `action :disable` the `sysrc` equivalent.

  Omit a method only when it genuinely doesn't apply, with a YAML comment above the check saying why.
- **Verify CLI commands in remediation with the validator** before committing (see Validation).

## Impact scoring

`impact:` drives prioritization in scan output and dashboards. The rows are **bands**, not single values. Pick the band that matches the risk, then any value inside it (`75` is a fine 70-79).

| Impact | When to use |
|--------|-------------|
| 90-100 | Direct path to data loss, account compromise, or full takeover. Public exposure of customer data, unauthenticated admin endpoints, plaintext secrets in shared storage, disabled audit logging on production. |
| 80-89 | High-confidence misconfiguration with a realistic exploit chain. Encryption disabled on sensitive resources, overly permissive IAM, network-wide ingress on management ports, missing MFA on privileged identities. |
| 70-79 | Hardening that meaningfully reduces blast radius. CMK instead of vendor-managed keys, private endpoints over public, log retention/aggregation, disabling remote management shells left reachable by vendor defaults. |
| 60-69 | Recommended hardening, moderate risk reduction. Tag/label hygiene that gates other controls, non-default versions of managed services, password complexity above vendor defaults. |
| 30-59 | Best practices and informational. Defense in depth that rarely changes outcomes alone: resource labeling, optional telemetry, naming conventions. |

Anchor to a sibling check in the same policy where you can. Adding an encryption-at-rest check next to five others at `impact: 70`? Use 70 unless you can say why this one differs, and cite the sibling UID in the PR description.

## UID and naming

Pattern: `mondoo-<provider>-security-<resource>-<rule>`

- `<provider>` is the policy's cloud or platform: `aws`, `azure`, `gcp`, `oci`, `digitalocean`, `hetzner`, `linux`, `windows`, `macos`, `kubernetes`, `github`, `gitlab`, ...
- `<resource>` is the service or object: `eks-cluster`, `s3-bucket`, `cloud-sql-mysql`, `network-security-group`. Use the vendor's naming, don't invent terminology.
- `<rule>` describes the assertion in active voice: `cmks-in-kms`, `private-controlplane`, `logging-enabled`, `restrict-public-access`. Short and concrete, no `-misconfigured` or `-check` suffixes.

Variant suffixes (parent UID + suffix, see Terraform variants below):

- `-<cloud>` for the runtime variant (`-aws`, `-azure`, `-gcp`), matching the parent's cloud
- `-terraform-hcl` / `-terraform-plan` / `-terraform-state`
- `-cloudformation` / `-bicep` for AWS / Azure where applicable

The parent carries `title`, `impact`, `tags`, and `docs:`. Variants carry the platform-specific `mql:` plus a `mondoo.com/filter-title` and `mondoo.com/filter-icon` tag pair. Don't repeat compliance tags on variants.

Before adding a UID, grep for an existing check on the same control objective: `grep -i "<resource>-<rule>" content/mondoo-<provider>-security.mql.yaml`. Duplicates fragment compliance mappings and confuse scan output.

## `docs:` body structure

Match the existing shape so new checks are indistinguishable from their neighbours.

**`desc:`** is what the check verifies, then why it matters.

```markdown
This check verifies that <capability> is <required state>.

<What that capability is and what it does, a sentence or two.> <Where the reader administers it, and the value the check requires.>

**Why this matters**

<What an attacker concretely does without it, or what concretely breaks. Two to four short paragraphs.>

<When this legitimately shouldn't be applied, or what to pair it with.>
```

Four rules, in priority order.

**1. The first sentence says what is being verified.** It starts with the literal string `This check` and names the capability a reader would recognize as a security property, not the resource, the config key, or a paraphrase of the title. Someone who stops after one sentence should know what was assessed.

**2. The capability leads; the setting is how it's measured.** The setting is the instrument, never the subject:

```markdown
This check verifies that Address Space Layout Randomization is fully enabled.

ASLR places a program's stack, heap, shared libraries, and executable at different memory addresses every time it runs, so an attacker cannot know in advance where anything is. The check reads the kernel parameter `kernel.randomize_va_space`, which must be `2`. A value of `1` randomizes everything except the `brk` heap, and `0` turns randomization off entirely.
```

Not "This check verifies that `kernel.randomize_va_space` is set to 2", which says what the scanner did rather than what's true about the system.

**3. Name the setting where the reader administers it, not where the scanner reads it.** On an OS these are the same: `kernel.randomize_va_space` *is* the sysctl, `/etc/ssh/sshd_config` *is* the file. On SaaS, cloud, and API-driven policies they diverge, and naming the MQL accessor is useless to the reader:

| Wrong | Right |
|---|---|
| the organization's `twoFactorRequirementEnabled` setting | GitHub calls this **Require two-factor authentication for everyone in your organization**, under Organization Settings, Authentication security |
| `github.organization.defaultRepositoryPermission` must be `read` | GitHub calls this **Base permissions**, under Organization Settings, Member privileges. It offers None, Read, Write, and Admin |
| the response header names from a GET request | the server must send `X-Content-Type-Options: nosniff` |

Backticks are still right for anything the reader types or sees: a filename, an HTTP header, a Terraform argument, a manifest field like `securityContext.readOnlyRootFilesystem`, a literal value. What never appears is the provider path the query walks.

**4. Say what an attacker does, not that risk is reduced.** "Reduces the attack surface", "improves the security posture", and "adheres to the principle of least functionality" are true of every check in the repo, so they tell the reader nothing. Name the mechanism, and name a real technique or advisory where you're confident it's accurate.

Also: don't close with a paragraph restating the opener. Don't write "compliance with security standards may be compromised" without naming the control. Don't reference the MQL query, variant UIDs, or "this policy", because the reader sees the check in isolation.

Use exactly one `**Why this matters**` heading. Older policies carry a second `**Risk mitigation**` heading. Don't add it to new checks, and drop it when you rewrite one. Roughly 2,000 descriptions still have it and will converge as policies are revisited.

**`audit:`** is vendor-native verification steps. Use H3 headers (`### Audit via Console`, `### Audit via CLI`) when both a console and a CLI path exist. Each path is a short numbered list ending with what pass and fail look like. Never reference `cnspec`, `mql`, or the Mondoo console.

The audit command must observe what the query observes. A vendor command that reports the *effective* config contradicts a check that reads a *file*, and the auditor concludes the scanner is wrong. Worked example, `sshd -T`: `sshd.config.params`, `.ciphers`, `.macs`, and `.kexs` parse `/etc/ssh/sshd_config` and its `Include` files and never shell out. So for a directive whose OpenSSH default is already secure (`X11Forwarding`, `IgnoreRhosts`, `HostbasedAuthentication`, `PermitRootLogin`, `PermitEmptyPasswords`, `PermitUserEnvironment`, `LogLevel`), `sshd -T` prints the passing value on a host where the directive is absent and the check fails. Grep the config file instead, and say in the audit text that an absent directive fails.

**`remediation:`** is a list of `- id: <method>` entries, one per supported management surface. Each entry's `desc:` follows this shape:

```markdown
To <restate the fix in active voice> using <method>:

1. <step>
2. <step>
3. <step>

```<lang>
<example code or command>
```
```

Order the list the same way every time: `console`/`portal` -> `cli` -> `terraform` -> `cloudformation`/`bicep` for clouds; `gui` -> `cli` -> `script` -> `ansible` for OS targets. Reviewers read in that order.

## Compliance tags (`compliance/<framework>: <control-uid>`)

**Never copy compliance tags from a neighbouring check.** That check was mapped for a different control objective; reusing its tags propagates a wrong mapping and misleads auditors. Two checks that both "relate to identity" can map to different controls.

**A new check isn't done until it carries compliance tags.** A check added to a policy whose existing checks have `compliance/*` tags ships with its own verified tags, or with the user's explicit approval to skip them. Don't open a PR that adds untagged checks next to tagged siblings and note the omission in the PR body; that ships a half-finished policy. An empty `find` for the `cnspec-enterprise-policies` repo means **ask the user where the clone lives**, it does not authorize proceeding without tags.

For **each** framework the policy already tags:

1. **Read the control text.** Open `cnspec-enterprise-policies/frameworks/<framework>.mql.yaml` (`iso-27001-2022.mql.yaml`, `soc2-2017.mql.yaml`, `nist-sp-800-53-rev5.mql.yaml`, ...). Each control has a `uid`, `title`, and usually `docs.desc`. Ask the user where their clone is if you don't know. If the files aren't available, stop and say so. Don't guess.
2. **Say in one sentence what the check actually enforces.** Identity proofing? Encryption at rest? Read the MQL; don't let the title mislead you.
3. **Find the single best-matching control** by scanning control titles and descriptions. Strict fit only: MFA, password policy, and session-timeout controls are not stand-ins for identity proofing, encryption, network isolation, etc.
4. **If no control fits, tag the YAML boolean `false`**, unquoted: `compliance/soc2-2017: false`. Not `"false"`, not `false-fit`, not `n/a`, not omitting the key. The unquoted boolean is the repo convention (`grep -rho 'compliance/[a-z0-9-]*: false' content/*.mql.yaml | wc -l` counts over 5,000) and is what downstream tooling expects. A missing mapping beats a wrong one; wrong mappings surface in compliance audits and create trust debt.
5. **Cite the control you picked.** Include the control title and a short quote from its description so the user can verify.

**The `<framework>` key must match the framework's `uid:` field**, the value declared inside the framework YAML, not the file name. They differ: `frameworks/bsi-grundschutz-sys15.mql.yaml` declares `uid: bsi-sys-1-5`, so the tag is `compliance/bsi-sys-1-5`. A key matching no real framework `uid` generates a framework map with a dangling `framework_owner`, which fails bundle migration in `cnspec-enterprise-policies` with `cannot find framework owner`. The `<control-uid>` likewise has to exist under that framework's `controls:`.

### Which frameworks a check has to resolve

In practice the list is the same nearly everywhere. Fourteen frameworks are applied to essentially every tagged check in `content/`, so a new check resolves all fourteen, to a control uid or to `false`:

`bsi-sys-1-5`, `csa-cloud-controls-matrix-4`, `dora`, `hipaa`, `iso-27001-2022`, `nis-2`, `nist-csf-1`, `nist-csf-2`, `nist-sp-800-171`, `nist-sp-800-53-rev5`, `owasp-top-10-2025`, `pci-dss-4`, `soc2-2017`, `vda-isa-5`

Four more are subject-scoped and only added when the subject matter is genuinely in scope, not as part of the standard sweep: `owasp-llm-top-10-2025` and `nist-ai-100-1` (AI/LLM), `owasp-asvs-5` (appsec verification), `mitre-attack` (checks mapping to a specific adversary technique).

The list moves as frameworks are added, so confirm it rather than trusting this one:

```bash
grep -rho "compliance/[a-z0-9-]*:" content/*.mql.yaml | sort | uniq -c | sort -rn
```

The standard set appears on nearly every tagged check; subject-scoped ones on a few dozen to a few hundred. A policy missing one of the standard fourteen entirely is drift, not a decision. The fourteen currently report identical counts, so a framework whose count falls out of line with the other thirteen points at the policy that skipped it.

**The control uid is not derived from the framework key.** Several frameworks prefix their controls differently from the key you tag them under, so a constructed uid compiles and maps to nothing:

| Tag key | A real control uid under it |
|---|---|
| `compliance/pci-dss-4` | `pcidss-requirement-10-2-1` (no hyphens in `pcidss`) |
| `compliance/soc2-2017` | `soc2-control-cc6-8-1` (`-control-` infix) |
| `compliance/csa-cloud-controls-matrix-4` | `cloud-controls-matrix-4-log-08` (drops `csa-`) |
| `compliance/nist-sp-800-171` | `nist-sp-800-171--3-4-8` (**double** hyphen) |
| `compliance/hipaa` | `hipaa-security-ss164-312-b-audit-controls` |

Read the control uid out of the framework YAML. Don't construct it from the framework name.

### OWASP Top 10 parity (enforced by test)

The security policies map to OWASP Top 10:2025 (application) and, for AI/LLM assets, OWASP Top 10:2025 for LLM Applications. `content/validation/compliance/owasp_mapping_test.go` enforces a parity invariant: a check carrying `compliance/pci-dss-4:` (the marker for a framework-mapped check) must also carry `compliance/owasp-top-10-2025:`, and vice versa. A policy is fully mapped or not mapped.

So if you add a check to a policy whose siblings are framework-mapped, add its `compliance/owasp-top-10-2025:` tag in the same PR. Omit it and `main` goes red on the next run, because the new check inherits a `pci-dss-4` tag from the mapping convention without its OWASP partner. Pick the category from what the check enforces:

- `a01` broken access control (anonymous/bypass, exposure, least privilege)
- `a02` security misconfiguration (surface area, dangerous defaults)
- `a04` cryptographic failures (encryption in transit and at rest, password storage)
- `a07` authentication failures (login/identity, password policy, MFA)
- `a08` software/data integrity
- `a09` logging and monitoring failures

`a03` (supply chain) applies to build-time checks. `a05` (injection), `a06` (insecure design), and `a10` (mishandling exceptional conditions) belong to source-level SAST, not posture scanning: never map a cnspec posture check to them, and the test rejects all three. Tag `false` (unquoted) only where no framework control fits. The OWASP tag isn't optional for a mapped check.

## Terraform variants and remediation for cloud policies

Add or modify a check in a cloud policy (`mondoo-aws-security`, `mondoo-azure-security`, `mondoo-gcp-security`, `mondoo-oci-security`, `mondoo-hetzner-security`, `mondoo-digitalocean-security`, ...) and two things ship together:

1. A `variants:` block so the check runs against the live cloud runtime *and* Terraform HCL/plan/state assets.
2. A `- id: terraform` entry in `remediation:` with HCL that fixes the issue.

Don't ship one without the other.

### Variants

Up to four children:

- `<uid>-<cloud>` — runtime (`asset.platform == 'aws'`, `'azure'`, `'gcp-project'`, `'oci'`, ...)
- `<uid>-terraform-hcl` — `terraform.resources(...)` against HCL source
- `<uid>-terraform-plan` — `terraform.plan.resourceChanges` against `terraform plan` JSON
- `<uid>-terraform-state` — `terraform.state.resources` against `terraform.tfstate`

**The runtime platform name comes from the provider's catalog, not the cloud's short name.** A filter naming a platform the provider never emits compiles, lints, and passes CI. It just never matches an asset, so the check silently never runs. Check the name against `providers/<cloud>/connection/platforms.go` (or `providers/<cloud>/resources/platforms.go`) in the mql repo, or the `Platforms` array in the installed `~/.config/mondoo/providers/<cloud>/<cloud>.json`. AWS, Azure, and OCI have an account-level platform named for the cloud (`aws`, `azure`, `oci`). **GCP does not**: use `gcp-project`, `gcp-org`, `gcp-folder`, or a per-resource platform such as `gcp-storage-bucket`.

Reference patterns here:

- GCP: `mondoo-gcp-security-memorystore-iam-auth-enabled` in `mondoo-gcp-security.mql.yaml`
- HCL nested-block fanout: `mondoo-gcp-security-cloud-sql-mysql-skip-show-database-enabled-terraform-*` (database_flags)
- Plan/state list-of-objects shape: `mondoo-gcp-security-cloud-storage-bucket-retention-policy-locked-terraform-*`

### Fixtures for new IaC variants

Every `-terraform-hcl`, `-cloudformation`, and `-bicep` variant ships pass **and** fail fixtures under `content/validation/scans/fixtures/iac-variants/<policy>/<variant-uid>/{pass,fail}/<scenario>/`.

Coverage is 100% and the gate holds it there: a variant without fixtures fails CI, and there's no debt budget to grow. Where a variant asserts exactly what its own `filters:` require, no failing input exists; record that with a `fail/IMPOSSIBLE.md` marker explaining why and it counts as covered. That marker is the only sanctioned way to ship a variant without a real fail fixture.

```bash
make test/content/iac/coverage
```

### The remediation has to satisfy the check

The closed-loop suite scans each IaC variant's own remediation snippet and requires the check that recommends it to pass. Linters can't answer that: cfn-lint, tflint, and `bicep build` prove a snippet is well-formed, not that it's right. A snippet can name every property correctly and still demonstrate the exact misconfiguration the check forbids.

```bash
make test/content/iac/remediation
```

Every variant in the corpus satisfies its own remediation, so this is a flat assertion with no debt list. A snippet that stops closing its check fails here.

Two shapes cause nearly every failure. A snippet documenting only the fixing resource never triggers a check whose `filters:` select on the resource being protected, so it has to declare that resource too. And a value the HCL parser can't resolve statically (a `jsonencode` body containing a resource reference, or a policy supplied through an `aws_iam_policy_document` data source) reads as absent rather than as what it becomes at apply time.

[validation/README.md](validation/README.md) covers how the snippet is materialized and what each of the three failure modes means.

### Terraform remediation

Every parent check with Terraform variants documents the Terraform fix too. Add `- id: terraform` to `remediation:` alongside `id: console`, `id: cli`, `id: cloudformation`, `id: bicep`. The block holds a short Markdown intro and a fenced ```hcl``` example that resolves the violation.

Canonical structure: `mondoo-aws-security-eks-cluster-cmks-in-kms` in `mondoo-aws-security.mql.yaml`.

### When you can't write a variant or remediation, leave a YAML comment

So the next pass doesn't re-investigate. Common reasons:

- The runtime check reads operational telemetry (job state, latest execution status, observed traffic) with no configuration analog.
- The resource is managed only via SDK/CLI/console and has no Terraform resource (short-lived imperative API calls like Vertex AI custom jobs).
- The runtime check depends on cross-resource correlation ("every cluster has a backup plan pointing at it") that the runtime check itself doesn't implement correctly yet. Fix the runtime first.
- The runtime check inspects a field whose Terraform analog is a different feature. Don't paper over that with a vacuous variant.

`# No Terraform variants:` goes on the line before `- uid:`:

```yaml
# No Terraform variants: <one-sentence reason>. <Optional: when this could be revisited>.
- uid: mondoo-<cloud>-security-...
```

`# No Terraform remediation:` goes inside the `remediation:` list as its last line, indented level with the `- id:` entries, where an `- id: terraform` would sit:

```yaml
remediation:
  - id: console
    desc: ...
  # No Terraform remediation: <one-sentence reason>. <Optional: when this could be revisited>.
```

When neither is possible (the usual case, since no variant usually means no Terraform remediation either), include both comments. Each explains the technical limitation, not just "skip".

### Terraform variants read a different shape than the runtime check

HCL, plan, and state variants of one check are not interchangeable. The usual bug is assuming they are:

- **A default-true attribute is asserted as `!= false`, not `== true`.** An omitted attribute is `null`, and `null == true` fails, so `== true` flags every correct config that relied on the default.
- **An absent block is not neutral.** Whether a missing block means pass or fail depends on the vendor's API default for that setting, which you have to look up. Two attributes in the same block can go opposite ways.
- **In plan and state JSON, nested blocks serialize as arrays**, and an omitted optional block is `[]`, not `null`. A missing block usually means the insecure default, so guard with `!= empty &&` rather than letting the empty list pass vacuously.
- **The HCL variant is often legitimately stricter than plan/state**, because HCL sees author intent and plan/state see a resolved value. Don't "unify" variants by copying one body into another; that silently weakens the strict one.
- **Verify every attribute name against the provider's real schema** (`terraform providers schema -json`), not from memory.

## MQL traps that produce a confidently wrong verdict

Most of these don't fail lint and don't error at scan time. They return a verdict and the verdict is wrong, which is worse than a broken check because nothing signals it. Read this before writing a query.

Check the field exists first. The installed provider schema is what lint resolves against: `~/.config/mondoo/providers/<name>/<name>.resources.json`. Source of truth: `providers/<name>/resources/<name>.lr` in the [mql repo](https://github.com/mondoohq/mql). Prove the query end to end with `cnquery run <provider> -c '<mql>'` and read the verdict as `[ok]` / `[failed]`. A check asserting `x == false` prints `[ok] value: false` when it passes, so "value: false" in the output is not a failure.

### There is no parenthesized grouping

`(` is not a valid operand anywhere in MQL. This one *is* a compile error, listed here because the usual response to it is wrong:

```
(a == 1) || (a == 2)          -> expected operand, got token "("
[1,2].all((_ == 1) || _ == 2) -> expected closing ')', got '('
[1,2].all(_ == 1 || (_ == 2)) -> expected operand, got token "("
```

Rely on `&&` binding tighter than `||`, so `a || b && c` already parses as `a || (b && c)`, or split the assertion into separate `.any()` / `.all()` calls. Never add parens "for clarity", and decline review suggestions asking for them.

### A guard chain is not a skipped check

The dominant shape in this directory: `||`-joined guards that exempt an asset, then the assertion as the final `&&` conjunct.

```coffee
aws.batch.jobDefinition.status != "ACTIVE" ||            # guard: not in scope
aws.batch.jobDefinition.container == null ||             # guard: nothing to check
aws.batch.jobDefinition.container.jobRole == null ||     # guard: no role attached
aws.batch.jobDefinition.container.jobRole.inlinePolicyDetails.length == 0 &&
aws.batch.jobDefinition.container.jobRole.attachedPolicies.all(...)
```

`&&` binds tighter, so that parses as `guard || guard || guard || (D && E)`, which is the intended semantics. Reviewers, human and automated, keep misreading the `D && E` tail as "E only runs when D is true, so the check is skipped and silently passes". That's wrong, and it's the most-filed false positive on this repo: short-circuiting decides what gets evaluated, never what the verdict is. In a disjunction a false conjunct makes the whole expression false, so the check fails:

| `jobRole` present | inline policies exist (`D`) | attached clean (`E`) | verdict |
|---|---|---|---|
| yes | **yes** (`D` false) | yes | **`[failed]`**, the inline policy *is* the violation |
| yes | no | no (`E` false) | `[failed]` |
| yes | no | yes | `[ok]` |
| no | - | - | `[ok]` via the guard |

No input makes the guard-chain form pass where a "fully parenthesized" form would fail. Before claiming a precedence bug, build that table and name the differing row. No differing row, no bug.

Two corollaries:

- The suggested fix usually isn't expressible, since MQL rejects `(` and the grouping has to come from precedence.
- A guard chain and a pure `&&` chain differ in what an **absent** field means, not in precedence. `role == null ||` *passes* an asset with no role; a pure `&&` chain *fails* it. That's a deliberate decision about absent data, so don't unify the two shapes.

### Don't verify a literal by eye in a rendered diff

Character-level claims about string literals (a colon count in an ARN, a missing path segment, a truncated prefix) aren't reliable from diff output, where proportional rendering and syntax highlighting distort spacing. One PR filed five findings against `arn:aws:iam::aws:policy/ReadOnlyAccess`, each proposing a different "correct" form, three retracting themselves mid-comment. The ARN was canonical throughout.

Resolve the literal against the system that owns it and quote the result:

```bash
aws iam get-policy --policy-arn arn:aws:iam::aws:policy/ReadOnlyAccess --query 'Policy.Arn' --output text
# arn:aws:iam::aws:policy/ReadOnlyAccess
```

AWS-managed policy ARNs have an empty account field, so `iam` is followed by exactly two colons. Same rule for any literal with an authoritative oracle: `cfn-lint` for CloudFormation resource types and property names, `az`/`gcloud`/`aws` for CLI grammar, the provider schema for field paths.

### Comparison against an unresolved field is asymmetric

A field the provider never populated is `null`, and null doesn't compare like a value. Against a missing map key (`m = {"a": 1}`):

| Written as | Verdict when the field is absent |
|---|---|
| `m["b"] == "x"` | **fails** |
| `m["b"] != "x"` | **passes** |
| `m["b"] != ""` | **passes**, so `!= ""` is not a non-empty test |
| `m["b"] != empty` | **fails**, this is the null-safe guard |

The trap is the second row. A check phrased in the negative (`setting != "insecure"`, `mode != "off"`) passes on every asset where the field never resolved. It should be inconclusive; it reports compliant. Assert presence first:

```coffee
setting != empty && setting != "insecure"
```

Prefer `!= empty` over `!= ""` for the same reason: `"" == empty` is true, but so is `null != ""`.

### `.all()` and `.none()` treat null and empty differently

An absent HCL or map key is `null`, not an empty list, and the two go opposite ways:

```coffee
[1,2].where(_ > 5).all(_ == 99)   # empty list  -> [ok] value: true   (vacuous pass)
m["missing"].all(_ == 1)          # null        -> [failed] actual: _
m["missing"] == empty || m["missing"].all(_ == 1)   # [ok] true      (null-safe form)
```

So `blocks.where(type == 'x').all(y)` and `values['x'].all(y)` are not equivalent rewrites: the first is vacuously true when nothing matches, the second fails outright. Don't swap them. The vacuous pass and the hard fail are both wrong answers for "the block is absent", and which one you want depends on the vendor default (see Terraform variants above).

### A dotted path that is also a resource name is not a field read

The compiler extends the resource path greedily, and the longest matching resource name wins unconditionally over a field on a shorter one, even when the longer resource can't stand on its own. `azure.subscription.aksService.cluster.autoUpgradeProfile.upgradeChannel` builds a bare `...cluster.autoUpgradeProfile` resource with no id and no fields; the cluster's accessor never runs and every field reads `null`. With the asymmetry above, the check returns a confident wrong answer instead of an error.

Suspect it when the value is a sub-object (a profile, config, or settings block) **and** the full path appears as a resource in its own right in `cnspec providers resources <provider> --json`. Confirm by running the query: the log line is `provider returned no data and no error for a field ... id=` with an empty `id=`. Fix by reaching the value through an accessor whose path isn't a resource name (`azure.subscription.aks.cluster....`) or by binding a block to the parent:

```coffee
azure.subscription.aks.cluster {
  autoUpgradeProfile.upgradeChannel != "none"
}
```

Not Azure-specific: Cloudflare (`cloudflare.zone.settings.*`), GCP (`gcp.project.gkeService.cluster.networkPolicy.*`), AWS (`aws.emr.cluster.encryptionConfiguration.*`), vSphere, and Arista all have resources shaped this way. Cloudflare adds a second failure mode: a 401/403 degrades to an empty list rather than an error, so an unauthorized scan passes vacuously.

### Smaller ones that still flip a verdict

- **`files.find` regex matches the whole path**, not the basename. A pattern without a leading `.*` matches nothing, and the `.all()` around it then passes vacuously. There are 17 `files.find` regex usages in `content/` today; check yours against a real path.
- **Flatten with `.flat`, not `.flatten`**, and guard the nested key first.
- **`map[string][]string` is filtered, never indexed.** Use `keys` / `values` / `where(key ...)` / `flat`. Indexing a missing key returns `null`, which collides with a legitimately empty list.
- **`terraform.resources()` takes a positional type argument**: `terraform.resources("aws_s3_bucket")`. The named form doesn't compile.
- **`parse.int` fails inside variant queries.** Use date arithmetic or a string match. Port ranges have no string-to-int conversion at all, so match with an anchored regex.
- **GCP dict fields omit defaults.** `protoToDict` drops `false`, `0`, and `""` and camelCases keys, so assert presence rather than `== false`.
- **`.one()` means exactly one** and is almost never what a presence check wants. `files.one(name.in(["dependabot.yaml", "dependabot.yml"]))` fails a repo holding both. Use `.any()` for "at least one exists"; reserve `.one()` for genuine uniqueness.
- **`.any()` fails on an empty list**, which is usually the verdict you want. It's the right operation for "something must exist", precisely because `.all()` and `.none()` pass vacuously there. Verified: `[1,2,3].where(_ > 99).any(_ > 0)` fails, while the same filter with `.all()` returns `[ok] value: true`.

## Validation and testing

[`validation/README.md`](validation/README.md) is the reference for every gate that runs against this directory: what each proves, when CI runs it, how to scope a run to a single check, what a new policy or check has to be registered with. This file owns the authoring rules; that one owns the gates.

```bash
make test/content              # lint + bundle scans + compliance mappings
make test/content/lint         # cnspec policy lint (run first, fix first)
make test/content/iac          # every IaC fixture suite, slow; scope it, see the README
make test/content/remediation  # remediation code-block linters
make test/content/commands     # remediation CLI and API validators
```

`cnspec policy lint` has to pass before committing a policy change. One policy at a time:

```bash
cnspec policy lint content/mondoo-aws-security.mql.yaml
cnspec scan local -f content/your-policy.mql.yaml
```

Three consequences of how the gates are wired, all invisible while they're wrong:

- **A skipped check is a fixture bug, not a pass.** A variant whose filter never matches anything is indistinguishable from one that passes, in every report, forever. Group filters are evaluated before the check's own filter, so a policy with variants leaves its groups unfiltered. See [Group filters and variants do not mix](validation/README.md#group-filters-and-variants-do-not-mix).
- **A new policy is visited by nothing until it's registered.** Most validators are allowlist-driven, so an unregistered `*.mql.yaml` ships with its variants untested and its remediation unverified, every gate green. Wire it up in the same change: [Adding a policy: what to register](validation/README.md#adding-a-policy-what-to-register).
- **A stale `KNOWN_BUG.md` marker fails the build.** Fixing a check means deleting its markers in the same change.

Never hand-edit the checked-in CLI grammars and OpenAPI specs in `validation/data/`. Re-run the relevant script in `validation/upstream/dump/`.

MQL resource links, built-in functions, and the authoring guide are in the repo-root [`CLAUDE.md`](../CLAUDE.md), which loads alongside this file.
