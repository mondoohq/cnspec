# M365 Policy: Terraform Variants, Remediation Standardization, and Audit Sections

**Date:** 2026-05-16
**File under change:** `content/mondoo-m365-security.mql.yaml`
**Delivery:** One PR.

## Problem

The Mondoo Microsoft 365 Security policy (`mondoo-m365-security`, version 2.2.0) has 18
checks defined as top-level `queries:`. Three gaps exist relative to `content/CLAUDE.md`
authoring standards:

1. **No `audit:` sections.** Every check's `docs:` block must include `desc:`, `audit:`,
   and `remediation:`. All 18 checks are missing `audit:`.
2. **Inconsistent `remediation:` coverage.** 9 of 18 checks have a `terraform` remediation
   entry; 9 do not. There is no explicit, consistent target set for M365.
3. **No `variants:` blocks.** No check runs against Terraform assets, only the live
   `microsoft365` runtime.

## Goals

- Add an `audit:` section to all 18 checks.
- Standardize the `remediation:` list across all 18 checks.
- Add Terraform `variants:` to the checks where a real `azuread` analog exists.
- `cnspec policy lint content/mondoo-m365-security.mql.yaml` passes.
- `version:` bumped to 2.3.0.

## Non-goals

- No new checks (control objectives) are added.
- No Microsoft Graph API / CLI remediation type is added (explicitly out of scope).
- No change to runtime MQL of any check.
- No Intune / Exchange Online / SharePoint Online Terraform provider is adopted.

## Workstream A — `audit:` sections (all 18 checks)

Add an `audit:` field to every check's `docs:` block. Structure follows the repo pattern:

- Use H3 headers (`### Audit via Console`, `### Audit via PowerShell`) when both a console
  path and a PowerShell path exist; otherwise a single path with no header.
- Each path is a short numbered list ending with what a passing vs. failing result looks
  like.
- Console paths use the relevant vendor admin center: Microsoft Entra admin center
  (`entra.microsoft.com`), Microsoft 365 admin center, Exchange admin center, or SharePoint
  admin center — matched to the check's subject.
- PowerShell paths use Microsoft Graph PowerShell or Exchange Online PowerShell, matched to
  the check's subject (Exchange Online checks use Exchange Online PowerShell).
- **Never** reference `cnspec`, `mql`, or the Mondoo console.

The SPF check audits via DNS tooling (`dig` / `nslookup`) rather than PowerShell.

## Workstream B — standardize `remediation:`

Target remediation set for M365:

- `console` — admin-center click-through. Present on all 18 (already is).
- `powershell` — Graph PowerShell or Exchange Online PowerShell. Present on all checks
  except SPF, which uses `cli` (DNS tooling). Leave SPF as `cli`.
- `terraform` — only where a genuine resource exists (see matrix below). Otherwise add a
  `# No Terraform remediation:` comment as the last line inside the `remediation:` list,
  indented level with the `- id:` entries, explaining the technical limitation.

Order within each list: `console` → `powershell`/`cli` → `terraform`.

Existing `terraform` remediation entries that use a `null_resource` + `local-exec` Graph
API call (security-defaults, passwords-not-set-to-expire, third-party-apps) are **kept** —
a `local-exec` is still a documented Terraform path to apply the fix. They do not, however,
qualify for a Terraform *variant* (a `local-exec` cannot be statically scanned).

The "2–4 global admins" check gains a new `terraform` remediation entry using
`azuread_directory_role_assignment`.

## Workstream C — Terraform `variants:`

Only the 5 Conditional Access policy checks have a clean, statically-scannable `azuread`
analog. Each is converted to a `variants:` block:

- Parent query keeps `title`, `impact`, `tags` (including compliance tags), and `docs:`.
- `<uid>-microsoft365` — runtime child: current MQL, `filters: asset.platform == "microsoft365"`.
- `<uid>-terraform-hcl` — `terraform.resources` over `azuread_conditional_access_policy`.
- `<uid>-terraform-plan` — `terraform.plan.resourceChanges`.
- `<uid>-terraform-state` — `terraform.state.resources`.
- Each child carries `mondoo.com/filter-title` + `mondoo.com/filter-icon` tags. Compliance
  tags are **not** repeated on children.

The 13 non-variant checks each get a `# No Terraform variants:` comment on the line before
`- uid:`, stating the limitation.

### Known inspection-surface difference (accepted)

The 5 runtime CA checks query Microsoft Secure Score
(`microsoft.security.latestSecureScores.controlScores.where(controlName == '...')`), while
the Terraform variants inspect `azuread_conditional_access_policy` resources directly. This
is the standard runtime-vs-source variant model (runtime inspects deployed state, Terraform
inspects declarative source) — the same control objective via different surfaces. It is
intentionally not a literal translation of the runtime MQL.

## Per-check applicability matrix

| Check (uid suffix) | Subject | TF variant | TF remediation |
|---|---|---|---|
| enable-azure-ad-identity-protection-sign-in-risk-policies | CA policy | ✅ hcl/plan/state | ✅ (exists) |
| enable-azure-ad-identity-protection-user-risk-policies | CA policy | ✅ hcl/plan/state | ✅ (exists) |
| enable-conditional-access-policies-to-block-legacy-authentication | CA policy | ✅ hcl/plan/state | ✅ (exists) |
| ensure-multifactor-authentication-...-administrative-roles | CA policy | ✅ hcl/plan/state | ✅ (exists) |
| ensure-multifactor-authentication-...-all-roles | CA policy | ✅ hcl/plan/state | ✅ (exists) |
| ensure-security-defaults-is-disabled-on-azure-active-directory | tenant policy | ❌ no azuread resource (Graph-only toggle) | ✅ keep `null_resource` |
| ensure-that-between-two-and-four-global-admins-are-designated | role assignments | ❌ count over live assignments; HCL misses portal-created ones | ➕ add `azuread_directory_role_assignment` |
| ensure-that-mobile-device-encryption-is-enabled-... | Intune config | ❌ Intune, no azuread provider | ❌ `# No Terraform remediation:` |
| ensure-that-mobile-devices-require-a-minimum-password-length-... | Intune config | ❌ Intune, no azuread provider | ❌ `# No Terraform remediation:` |
| ensure-that-ms-365-passwords-are-not-set-to-expire | per-domain policy | ❌ no azuread resource (Graph-only) | ✅ keep `null_resource` |
| ensure-that-spf-records-are-published-... | live DNS records | ❌ TF analog spans 3 DNS providers, operational | ✅ (exists, multi-cloud DNS) |
| ensure-third-party-integrated-applications-are-not-allowed | tenant authorizationPolicy | ❌ no azuread resource | ✅ keep `null_resource` |
| ensure-dkim-signing-enabled-for-all-exchange-domains | Exchange Online | ❌ Exchange Online, no TF provider | ❌ `# No Terraform remediation:` |
| ensure-safe-links-policies-configured | Exchange Online | ❌ Exchange Online, no TF provider | ❌ `# No Terraform remediation:` |
| ensure-safe-attachments-policies-configured | Exchange Online | ❌ Exchange Online, no TF provider | ❌ `# No Terraform remediation:` |
| ensure-anti-phishing-policies-enabled | Exchange Online | ❌ Exchange Online, no TF provider | ❌ `# No Terraform remediation:` |
| ensure-sharepoint-external-sharing-restricted | SharePoint Online | ❌ SharePoint Online, no TF provider | ❌ `# No Terraform remediation:` |
| ensure-transport-rules-enforce-tls | Exchange Online | ❌ Exchange Online, no TF provider | ❌ `# No Terraform remediation:` |

Result: **5 checks gain Terraform variants**, **13 get `# No Terraform variants:` comments**.

## Validation

- `cnspec policy lint content/mondoo-m365-security.mql.yaml` must pass.
- The remediation CLI validator (`content/validation/validate_remediation_commands.py`)
  does not cover M365 (no `az`/`aws`/`gcloud` M365 surface); not applicable here.
- Manually confirm the `azuread_conditional_access_policy` HCL/plan/state MQL shapes against
  the reference variant patterns in `content/CLAUDE.md` (GCP examples).

## Risks

- The CA-policy HCL variant MQL must traverse nested blocks (`conditions`,
  `grant_controls`, `state`); plan/state shapes differ. Each of the 5 needs its own tested
  MQL. Mitigation: model on the GCP nested-block reference checks named in `content/CLAUDE.md`.
- Compliance tags must stay on the parent only when converting to `variants:`. Mitigation:
  explicit check during implementation.
