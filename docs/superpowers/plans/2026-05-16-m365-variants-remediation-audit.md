# M365 Policy: Variants, Remediation, and Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring all 18 checks in `content/mondoo-m365-security.mql.yaml` up to repo authoring standards by adding `audit:` sections, standardizing `remediation:`, and adding Terraform `variants:` to the 5 Conditional Access policy checks.

**Architecture:** Single-file edit. The 5 Conditional Access checks become `variants:` blocks (runtime `-microsoft365` child + `-terraform-hcl`/`-terraform-plan`/`-terraform-state` children querying `azuread_conditional_access_policy`). The other 13 checks gain `audit:` sections and `# No Terraform variants:` / `# No Terraform remediation:` comments. All 18 gain an `audit:` section.

**Tech Stack:** cnspec policy bundles (MQL YAML), `azuread` Terraform provider, `cnspec policy lint` for validation.

**Spec:** `docs/superpowers/specs/2026-05-16-m365-variants-remediation-audit-design.md`

---

## Reference material (read before starting)

### R1. Variant block skeleton

A check converted to variants keeps `title`/`impact`/`tags`/`docs:` on the parent, adds a `variants:` list, and gets one new top-level query per variant. Pattern (from `mondoo-gcp-security-memorystore-iam-auth-enabled`):

```yaml
  - uid: <PARENT-UID>
    title: <unchanged>
    impact: <unchanged>
    tags:
      <unchanged compliance tags>
    variants:
      - uid: <PARENT-UID>-microsoft365
        tags:
          mondoo.com/filter-title: Microsoft 365
          mondoo.com/filter-icon: microsoft
      - uid: <PARENT-UID>-terraform-hcl
        tags:
          mondoo.com/filter-title: Terraform HCL
          mondoo.com/filter-icon: terraform
      - uid: <PARENT-UID>-terraform-plan
        tags:
          mondoo.com/filter-title: Terraform Plan
          mondoo.com/filter-icon: terraform
      - uid: <PARENT-UID>-terraform-state
        tags:
          mondoo.com/filter-title: Terraform State
          mondoo.com/filter-icon: terraform
    docs:
      <unchanged desc/audit/remediation/refs>
```

The parent **loses** its top-level `mql:` field — the MQL moves to the `-microsoft365` child. Child queries are appended as new top-level `- uid:` entries immediately after the parent, each carrying only `filters:` + `mql:` (no `docs:`, no `tags:` beyond what's in the `variants:` list).

The group reference in `groups:` still points at the **parent** UID — no change to `groups:`.

### R2. Variant child query MQL (the 5 Conditional Access checks)

`azuread_conditional_access_policy` HCL shape: top-level `display_name`, `state`; nested blocks `conditions { client_app_types, sign_in_risk_levels, user_risk_levels; users { included_users, included_roles }; applications { included_applications } }` and `grant_controls { built_in_controls }`.

The runtime checks query Microsoft Secure Score; the Terraform variants assert that **at least one enabled CA policy** declares the corresponding condition + grant. Each `-terraform-hcl`/`-plan`/`-state` query below is a starting point — **Step "verify MQL" in each task lints and adjusts shapes** if the parser or provider data disagrees.

The MQL parser rejects a parenthesized clause as the first token inside `.all(`/`.any(`. Use `&&`/`||` precedence, not leading parens.

### R3. `audit:` section template

Add `audit:` to each check's `docs:` block, between `desc:` and `remediation:`. Use this shape:

```yaml
      audit: |
        ### Audit via Console

        1. <step>
        2. <step>

        A passing tenant shows <PASS>. A failing tenant shows <FAIL>.

        ### Audit via PowerShell

        1. <step>

        ```powershell
        <cmdlet>
        ```

        A passing tenant returns <PASS>. A failing tenant returns <FAIL>.
```

When only one path exists (SPF: DNS only), drop the H3 headers and write a single path. Never reference `cnspec`, `mql`, or the Mondoo console.

### R4. `audit:` parameters per check

| # | uid suffix | Console path | PowerShell / CLI | Pass condition |
|---|---|---|---|---|
| 1 | enable-azure-ad-identity-protection-sign-in-risk-policies | Microsoft Entra admin center (`entra.microsoft.com`) → Protection → Conditional Access → Policies | `Connect-MgGraph -Scopes "Policy.Read.All"` then `Get-MgIdentityConditionalAccessPolicy | Where-Object {$_.State -eq "enabled" -and $_.Conditions.SignInRiskLevels}` | ≥1 enabled policy with sign-in risk levels and an MFA grant |
| 2 | enable-azure-ad-identity-protection-user-risk-policies | Entra admin center → Protection → Conditional Access → Policies | `Get-MgIdentityConditionalAccessPolicy | Where-Object {$_.State -eq "enabled" -and $_.Conditions.UserRiskLevels}` | ≥1 enabled policy with user risk levels and an MFA grant |
| 3 | enable-conditional-access-policies-to-block-legacy-authentication | Entra admin center → Protection → Conditional Access → Policies | `Get-MgIdentityConditionalAccessPolicy | Where-Object {$_.State -eq "enabled" -and $_.GrantControls.BuiltInControls -contains "block"}` | ≥1 enabled policy blocking legacy auth client app types |
| 4 | ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles | Entra admin center → Protection → Conditional Access → Policies | `Get-MgIdentityConditionalAccessPolicy | Where-Object {$_.Conditions.Users.IncludeRoles}` | ≥1 enabled policy requiring MFA scoped to admin directory roles |
| 5 | ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles | Entra admin center → Protection → Conditional Access → Policies | `Get-MgIdentityConditionalAccessPolicy | Where-Object {$_.Conditions.Users.IncludeUsers -contains "All"}` | ≥1 enabled policy requiring MFA for all users |
| 6 | ensure-security-defaults-is-disabled-on-azure-active-directory | Entra admin center → Identity → Overview → Properties → Manage security defaults | `Get-MgPolicyIdentitySecurityDefaultEnforcementPolicy` | `IsEnabled` is `False` |
| 7 | ensure-that-between-two-and-four-global-admins-are-designated | Entra admin center → Roles & admins → Roles → Global Administrator → Assignments | `Get-MgDirectoryRole -Filter "displayName eq 'Global Administrator'"` then `Get-MgDirectoryRoleMember` | between 2 and 4 members |
| 8 | ensure-that-mobile-device-encryption-is-enabled-... | Microsoft Intune admin center (`intune.microsoft.com`) → Devices → Configuration | `Connect-MgGraph -Scopes "DeviceManagementConfiguration.Read.All"` then `Get-MgDeviceManagementDeviceConfiguration` | Android profiles set `storageRequireDeviceEncryption` true |
| 9 | ensure-that-mobile-devices-require-a-minimum-password-length-... | Intune admin center → Devices → Configuration | `Get-MgDeviceManagementDeviceConfiguration` | password minimum length ≥ 8 on device profiles |
| 10 | ensure-that-ms-365-passwords-are-not-set-to-expire | Microsoft 365 admin center → Settings → Org settings → Security & privacy → Password expiration policy | `Get-MgDomain` | `PasswordValidityPeriodInDays` is `2147483647` for every domain |
| 11 | ensure-that-spf-records-are-published-for-all-exchange-domains | (DNS only — no console path) | `dig +short TXT <domain>` or `nslookup -type=TXT <domain>` | a `v=spf1` TXT record exists for every Exchange domain |
| 12 | ensure-third-party-integrated-applications-are-not-allowed | Entra admin center → Identity → Users → User settings | `Get-MgPolicyAuthorizationPolicy` | `DefaultUserRolePermissions.AllowedToCreateApps` is `False` |
| 13 | ensure-dkim-signing-enabled-for-all-exchange-domains | Microsoft Defender portal (`security.microsoft.com`) → Email & collaboration → Policies & rules → Threat policies → Email authentication settings → DKIM | Exchange Online PowerShell: `Connect-ExchangeOnline` then `Get-DkimSigningConfig` | `Enabled` is `True` for every domain |
| 14 | ensure-safe-links-policies-configured | Defender portal → Threat policies → Safe Links | `Connect-ExchangeOnline` then `Get-SafeLinksPolicy` | ≥1 Safe Links policy exists |
| 15 | ensure-safe-attachments-policies-configured | Defender portal → Threat policies → Safe Attachments | `Connect-ExchangeOnline` then `Get-SafeAttachmentPolicy` | ≥1 Safe Attachments policy exists |
| 16 | ensure-anti-phishing-policies-enabled | Defender portal → Threat policies → Anti-phishing | `Connect-ExchangeOnline` then `Get-AntiPhishPolicy` | ≥1 anti-phishing policy exists |
| 17 | ensure-sharepoint-external-sharing-restricted | SharePoint admin center → Policies → Sharing | SharePoint Online PowerShell: `Connect-SPOService` then `Get-SPOTenant | Select SharingCapability` | `SharingCapability` is not `ExternalUserAndGuestSharing` |
| 18 | ensure-transport-rules-enforce-tls | Exchange admin center → Mail flow → Rules | `Connect-ExchangeOnline` then `Get-TransportRule` | ≥1 enabled rule with `RouteMessageOutboundRequireTls` true |

### R5. `# No Terraform` comment formats

`# No Terraform variants:` goes on the line directly above the check's `- uid:`:

```yaml
        # No Terraform variants: <reason>.
  - uid: mondoo-m365-security-...
```

`# No Terraform remediation:` goes as the last line **inside** the `remediation:` list, indented level with the `- id:` entries:

```yaml
      remediation:
        - id: console
          desc: |
            ...
        - id: powershell
          desc: |
            ...
        # No Terraform remediation: <reason>.
```

### R6. Per-check `# No Terraform variants:` reasons (13 checks)

| uid suffix | variants comment reason | remediation comment? |
|---|---|---|
| ensure-security-defaults-is-disabled-on-azure-active-directory | the security defaults toggle has no `azuread` Terraform resource and can only be changed through the Graph API or portal | no — keeps existing `null_resource` terraform remediation |
| ensure-that-between-two-and-four-global-admins-are-designated | the check counts live Global Administrator assignments, which HCL cannot enumerate because portal-created assignments are invisible to Terraform source | no — gains a new `azuread_directory_role_assignment` terraform remediation (Task 7) |
| ensure-that-mobile-device-encryption-is-enabled-... | Intune device configuration profiles have no resource in the `azuread` provider | yes — `# No Terraform remediation: Intune device configuration has no resource in the azuread Terraform provider.` |
| ensure-that-mobile-devices-require-a-minimum-password-length-... | Intune device configuration profiles have no resource in the `azuread` provider | yes — same reason as above |
| ensure-that-ms-365-passwords-are-not-set-to-expire | per-domain password validity has no `azuread` Terraform resource and is set only through the Graph API | no — keeps existing `null_resource` terraform remediation |
| ensure-that-spf-records-are-published-for-all-exchange-domains | the check inspects live published DNS records; the Terraform analog depends on which DNS provider hosts each domain and cannot be expressed as a single M365 resource | no — keeps existing multi-provider DNS terraform remediation |
| ensure-third-party-integrated-applications-are-not-allowed | the tenant authorization policy has no resource in the `azuread` provider | no — keeps existing `null_resource` terraform remediation |
| ensure-dkim-signing-enabled-for-all-exchange-domains | Exchange Online configuration has no Terraform provider | yes — `# No Terraform remediation: Exchange Online configuration has no Terraform provider.` |
| ensure-safe-links-policies-configured | Exchange Online Safe Links policies have no Terraform provider | yes — same reason |
| ensure-safe-attachments-policies-configured | Exchange Online Safe Attachments policies have no Terraform provider | yes — same reason |
| ensure-anti-phishing-policies-enabled | Exchange Online anti-phishing policies have no Terraform provider | yes — same reason |
| ensure-transport-rules-enforce-tls | Exchange Online transport rules have no Terraform provider | yes — same reason |
| ensure-sharepoint-external-sharing-restricted | SharePoint Online tenant settings have no Terraform provider | yes — `# No Terraform remediation: SharePoint Online tenant settings have no Terraform provider.` |

---

## Task 1: Bump policy version

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml:6`

- [ ] **Step 1: Change the version**

Change line 6 from `version: 2.2.0` to:

```yaml
    version: 2.3.0
```

- [ ] **Step 2: Lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":bookmark: Bump M365 security policy to 2.3.0"
```

---

## Task 2: Sign-in risk policy — variants + audit

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml` — check `mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies`

- [ ] **Step 1: Add the `audit:` section**

Insert an `audit:` block after `desc:` and before `remediation:` in this check's `docs:`, using template R3 and row 1 of table R4:

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Protection** > **Conditional Access** > **Policies**.
        3. Look for a policy that, under **Conditions**, sets **Sign-in risk** to one or more levels and, under **Grant**, requires multifactor authentication.

        A passing tenant has at least one enabled policy that targets sign-in risk and requires MFA. A failing tenant has no such policy, or the policy is in **Report-only** or **Off** state.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and list Conditional Access policies:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgIdentityConditionalAccessPolicy |
          Where-Object { $_.State -eq "enabled" -and $_.Conditions.SignInRiskLevels }
        ```

        A passing tenant returns at least one policy. A failing tenant returns nothing.
```

- [ ] **Step 2: Convert the check to a `variants:` block**

Remove the top-level `mql:` line from the parent check. Add a `variants:` block between `tags:` and `docs:` following skeleton R1, with `<PARENT-UID>` = `mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies`.

- [ ] **Step 3: Append the four child queries**

Immediately after the parent check's closing `refs:` block, add four new top-level queries:

```yaml
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies-microsoft365
    filters: |
      asset.platform == "microsoft365"
    mql: |
      microsoft.security.latestSecureScores.controlScores.where(controlName == 'SigninRiskPolicy').all(_['score'] == 7)
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies-terraform-hcl
    filters: |
      asset.platform == 'terraform-hcl' && terraform.resources.contains(nameLabel == 'azuread_conditional_access_policy')
    mql: |
      terraform.resources('azuread_conditional_access_policy').any(
        arguments['state'] == 'enabled' &&
        blocks.where(type == 'conditions').any(arguments['sign_in_risk_levels'] != empty) &&
        blocks.where(type == 'grant_controls') != empty
      )
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies-terraform-plan
    filters: |
      asset.platform == 'terraform-plan' && terraform.plan.resourceChanges.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.plan.resourceChanges.where(type == 'azuread_conditional_access_policy').any(
        change.after['state'] == 'enabled' &&
        change.after['conditions'].any(_['sign_in_risk_levels'] != empty) &&
        change.after['grant_controls'] != empty
      )
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-sign-in-risk-policies-terraform-state
    filters: |
      asset.platform == 'terraform-state' && terraform.state.resources.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.state.resources.where(type == 'azuread_conditional_access_policy').any(
        values['state'] == 'enabled' &&
        values['conditions'].any(_['sign_in_risk_levels'] != empty) &&
        values['grant_controls'] != empty
      )
```

- [ ] **Step 4: Verify MQL and lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors. If the linter reports a parse error on a child `mql:`, the most likely cause is the leading-paren parser quirk (R2) or a list/map accessor mismatch — adjust `blocks.where(...)` / `change.after[...]` / `values[...]` shapes and re-lint. Do not proceed until lint passes.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":sparkles: Add Terraform variants and audit to M365 sign-in risk check"
```

---

## Task 3: User risk policy — variants + audit

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml` — check `mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies`

- [ ] **Step 1: Add the `audit:` section**

Insert `audit:` after `desc:`, before `remediation:`, using template R3 and row 2 of R4:

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Protection** > **Conditional Access** > **Policies**.
        3. Look for a policy that, under **Conditions**, sets **User risk** to one or more levels and, under **Grant**, requires multifactor authentication.

        A passing tenant has at least one enabled policy that targets user risk and requires MFA. A failing tenant has no such policy, or the policy is in **Report-only** or **Off** state.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and list Conditional Access policies:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgIdentityConditionalAccessPolicy |
          Where-Object { $_.State -eq "enabled" -and $_.Conditions.UserRiskLevels }
        ```

        A passing tenant returns at least one policy. A failing tenant returns nothing.
```

- [ ] **Step 2: Convert to a `variants:` block**

Remove the parent `mql:`; add a `variants:` block per R1 with `<PARENT-UID>` = `mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies`.

- [ ] **Step 3: Append the four child queries**

After the parent's `refs:`, add:

```yaml
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies-microsoft365
    filters: |
      asset.platform == "microsoft365"
    mql: |
      microsoft.security.latestSecureScores.controlScores.where(controlName == 'UserRiskPolicy').all(_['score'] == 7)
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies-terraform-hcl
    filters: |
      asset.platform == 'terraform-hcl' && terraform.resources.contains(nameLabel == 'azuread_conditional_access_policy')
    mql: |
      terraform.resources('azuread_conditional_access_policy').any(
        arguments['state'] == 'enabled' &&
        blocks.where(type == 'conditions').any(arguments['user_risk_levels'] != empty) &&
        blocks.where(type == 'grant_controls') != empty
      )
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies-terraform-plan
    filters: |
      asset.platform == 'terraform-plan' && terraform.plan.resourceChanges.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.plan.resourceChanges.where(type == 'azuread_conditional_access_policy').any(
        change.after['state'] == 'enabled' &&
        change.after['conditions'].any(_['user_risk_levels'] != empty) &&
        change.after['grant_controls'] != empty
      )
  - uid: mondoo-m365-security-enable-azure-ad-identity-protection-user-risk-policies-terraform-state
    filters: |
      asset.platform == 'terraform-state' && terraform.state.resources.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.state.resources.where(type == 'azuread_conditional_access_policy').any(
        values['state'] == 'enabled' &&
        values['conditions'].any(_['user_risk_levels'] != empty) &&
        values['grant_controls'] != empty
      )
```

- [ ] **Step 4: Verify MQL and lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors. Adjust shapes per R2 guidance if a child query fails to parse.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":sparkles: Add Terraform variants and audit to M365 user risk check"
```

---

## Task 4: Block legacy authentication — variants + audit

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml` — check `mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication`

- [ ] **Step 1: Add the `audit:` section**

Insert `audit:` after `desc:`, before `remediation:`, using R3 and row 3 of R4:

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Protection** > **Conditional Access** > **Policies**.
        3. Look for an enabled policy whose **Conditions** > **Client apps** targets the legacy authentication clients (Exchange ActiveSync and other clients) and whose **Grant** control is set to **Block access**.

        A passing tenant has at least one enabled policy that blocks legacy authentication. A failing tenant has no such policy, or it is in **Report-only** or **Off** state.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and list Conditional Access policies:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgIdentityConditionalAccessPolicy |
          Where-Object { $_.State -eq "enabled" -and $_.GrantControls.BuiltInControls -contains "block" }
        ```

        A passing tenant returns at least one policy that blocks legacy authentication client app types. A failing tenant returns nothing.
```

- [ ] **Step 2: Convert to a `variants:` block**

Remove the parent `mql:`; add a `variants:` block per R1 with `<PARENT-UID>` = `mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication`.

- [ ] **Step 3: Append the four child queries**

After the parent's `refs:` (or `remediation:` if no `refs:`), add:

```yaml
  - uid: mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication-microsoft365
    filters: |
      asset.platform == "microsoft365"
    mql: |
      microsoft.security.latestSecureScores.controlScores.where(controlName == 'BlockLegacyAuthentication').all(_['score'] == 8)
  - uid: mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication-terraform-hcl
    filters: |
      asset.platform == 'terraform-hcl' && terraform.resources.contains(nameLabel == 'azuread_conditional_access_policy')
    mql: |
      terraform.resources('azuread_conditional_access_policy').any(
        arguments['state'] == 'enabled' &&
        blocks.where(type == 'conditions').any(arguments['client_app_types'] != empty) &&
        blocks.where(type == 'grant_controls').any(arguments['built_in_controls'].contains('block'))
      )
  - uid: mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication-terraform-plan
    filters: |
      asset.platform == 'terraform-plan' && terraform.plan.resourceChanges.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.plan.resourceChanges.where(type == 'azuread_conditional_access_policy').any(
        change.after['state'] == 'enabled' &&
        change.after['conditions'].any(_['client_app_types'] != empty) &&
        change.after['grant_controls'].any(_['built_in_controls'].contains('block'))
      )
  - uid: mondoo-m365-security-enable-conditional-access-policies-to-block-legacy-authentication-terraform-state
    filters: |
      asset.platform == 'terraform-state' && terraform.state.resources.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.state.resources.where(type == 'azuread_conditional_access_policy').any(
        values['state'] == 'enabled' &&
        values['conditions'].any(_['client_app_types'] != empty) &&
        values['grant_controls'].any(_['built_in_controls'].contains('block'))
      )
```

- [ ] **Step 4: Verify MQL and lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors. Adjust shapes per R2 if needed.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":sparkles: Add Terraform variants and audit to M365 block-legacy-auth check"
```

---

## Task 5: MFA for administrative roles — variants + audit

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml` — check `mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles`

- [ ] **Step 1: Add the `audit:` section**

Insert `audit:` after `desc:`, before `remediation:`, using R3 and row 4 of R4:

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Protection** > **Conditional Access** > **Policies**.
        3. Look for an enabled policy whose **Users** assignment targets administrative directory roles and whose **Grant** control requires multifactor authentication.

        A passing tenant has at least one enabled policy that requires MFA for administrators. A failing tenant has no such policy, or it is in **Report-only** or **Off** state.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and list Conditional Access policies:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgIdentityConditionalAccessPolicy |
          Where-Object { $_.State -eq "enabled" -and $_.Conditions.Users.IncludeRoles }
        ```

        A passing tenant returns at least one policy scoped to administrative roles. A failing tenant returns nothing.
```

- [ ] **Step 2: Convert to a `variants:` block**

Remove the parent `mql:`; add a `variants:` block per R1 with `<PARENT-UID>` = `mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles`.

- [ ] **Step 3: Append the four child queries**

After the parent's `refs:` (or `remediation:` if no `refs:`), add:

```yaml
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles-microsoft365
    filters: |
      asset.platform == "microsoft365"
    mql: |
      microsoft.security.latestSecureScores.controlScores.where(controlName == 'AdminMFAV2').all(_['score'] == 10)
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles-terraform-hcl
    filters: |
      asset.platform == 'terraform-hcl' && terraform.resources.contains(nameLabel == 'azuread_conditional_access_policy')
    mql: |
      terraform.resources('azuread_conditional_access_policy').any(
        arguments['state'] == 'enabled' &&
        blocks.where(type == 'conditions').any(blocks.where(type == 'users').any(arguments['included_roles'] != empty)) &&
        blocks.where(type == 'grant_controls').any(arguments['built_in_controls'].contains('mfa'))
      )
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles-terraform-plan
    filters: |
      asset.platform == 'terraform-plan' && terraform.plan.resourceChanges.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.plan.resourceChanges.where(type == 'azuread_conditional_access_policy').any(
        change.after['state'] == 'enabled' &&
        change.after['conditions'].any(_['users'].any(_['included_roles'] != empty)) &&
        change.after['grant_controls'].any(_['built_in_controls'].contains('mfa'))
      )
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-administrative-roles-terraform-state
    filters: |
      asset.platform == 'terraform-state' && terraform.state.resources.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.state.resources.where(type == 'azuread_conditional_access_policy').any(
        values['state'] == 'enabled' &&
        values['conditions'].any(_['users'].any(_['included_roles'] != empty)) &&
        values['grant_controls'].any(_['built_in_controls'].contains('mfa'))
      )
```

- [ ] **Step 4: Verify MQL and lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors. The doubly-nested `blocks.where(...)` is the riskiest MQL in this plan — if it fails to parse or scan, fall back to asserting only `blocks.where(type == 'conditions') != empty` plus the `grant_controls` mfa clause, and note the relaxation in the commit message.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":sparkles: Add Terraform variants and audit to M365 admin MFA check"
```

---

## Task 6: MFA for all users — variants + audit

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml` — check `mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles`

- [ ] **Step 1: Add the `audit:` section**

Insert `audit:` after `desc:`, before `remediation:`, using R3 and row 5 of R4:

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Protection** > **Conditional Access** > **Policies**.
        3. Look for an enabled policy whose **Users** assignment includes **All users** and whose **Grant** control requires multifactor authentication.

        A passing tenant has at least one enabled policy that requires MFA for all users. A failing tenant has no such policy, or it is in **Report-only** or **Off** state.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and list Conditional Access policies:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgIdentityConditionalAccessPolicy |
          Where-Object { $_.State -eq "enabled" -and $_.Conditions.Users.IncludeUsers -contains "All" }
        ```

        A passing tenant returns at least one policy scoped to all users. A failing tenant returns nothing.
```

- [ ] **Step 2: Convert to a `variants:` block**

Remove the parent `mql:`; add a `variants:` block per R1 with `<PARENT-UID>` = `mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles`.

- [ ] **Step 3: Append the four child queries**

After the parent's `refs:` (or `remediation:` if no `refs:`), add:

```yaml
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles-microsoft365
    filters: |
      asset.platform == "microsoft365"
    mql: |
      microsoft.security.latestSecureScores.controlScores.where(controlName == 'MFARegistrationV2').all(_['score'] == 9)
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles-terraform-hcl
    filters: |
      asset.platform == 'terraform-hcl' && terraform.resources.contains(nameLabel == 'azuread_conditional_access_policy')
    mql: |
      terraform.resources('azuread_conditional_access_policy').any(
        arguments['state'] == 'enabled' &&
        blocks.where(type == 'conditions').any(blocks.where(type == 'users').any(arguments['included_users'].contains('All'))) &&
        blocks.where(type == 'grant_controls').any(arguments['built_in_controls'].contains('mfa'))
      )
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles-terraform-plan
    filters: |
      asset.platform == 'terraform-plan' && terraform.plan.resourceChanges.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.plan.resourceChanges.where(type == 'azuread_conditional_access_policy').any(
        change.after['state'] == 'enabled' &&
        change.after['conditions'].any(_['users'].any(_['included_users'].contains('All'))) &&
        change.after['grant_controls'].any(_['built_in_controls'].contains('mfa'))
      )
  - uid: mondoo-m365-security-ensure-multifactor-authentication-is-enabled-for-all-users-in-all-roles-terraform-state
    filters: |
      asset.platform == 'terraform-state' && terraform.state.resources.contains(type == 'azuread_conditional_access_policy')
    mql: |
      terraform.state.resources.where(type == 'azuread_conditional_access_policy').any(
        values['state'] == 'enabled' &&
        values['conditions'].any(_['users'].any(_['included_users'].contains('All'))) &&
        values['grant_controls'].any(_['built_in_controls'].contains('mfa'))
      )
```

- [ ] **Step 4: Verify MQL and lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors. Same doubly-nested-block fallback as Task 5 Step 4 applies.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":sparkles: Add Terraform variants and audit to M365 all-users MFA check"
```

---

## Task 7: Identity, Authentication & Application non-variant checks — audit + remediation

Covers 4 checks: `ensure-security-defaults-is-disabled-on-azure-active-directory`, `ensure-that-between-two-and-four-global-admins-are-designated`, `ensure-third-party-integrated-applications-are-not-allowed`, `ensure-that-ms-365-passwords-are-not-set-to-expire`.

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml`

- [ ] **Step 1: Add `audit:` sections**

For each of the 4 checks, insert an `audit:` block after `desc:`, before `remediation:`, built from template R3 and rows 6, 7, 12, 10 of table R4 respectively. Each gets an `### Audit via Console` path and an `### Audit via PowerShell` path. Example for `ensure-security-defaults-is-disabled-on-azure-active-directory` (row 6):

```yaml
      audit: |
        ### Audit via Console

        1. Sign in to the Microsoft Entra admin center at `https://entra.microsoft.com`.
        2. Go to **Identity** > **Overview** > **Properties**.
        3. Select **Manage security defaults** and check the **Security defaults** toggle.

        A passing tenant has security defaults set to **Disabled** (with Conditional Access providing equivalent or stronger controls). A failing tenant has security defaults **Enabled**.

        ### Audit via PowerShell

        1. Connect to Microsoft Graph and read the security defaults policy:

        ```powershell
        Connect-MgGraph -Scopes "Policy.Read.All"
        Get-MgPolicyIdentitySecurityDefaultEnforcementPolicy | Select-Object IsEnabled
        ```

        A passing tenant returns `IsEnabled : False`. A failing tenant returns `IsEnabled : True`.
```

Write the other 3 the same way from their R4 rows.

- [ ] **Step 2: Add the `# No Terraform variants:` comments**

Add a `# No Terraform variants:` comment line directly above the `- uid:` of all 4 checks, using the reasons from table R6. Format per R5.

- [ ] **Step 3: Add the `azuread_directory_role_assignment` terraform remediation to the global-admins check**

In `ensure-that-between-two-and-four-global-admins-are-designated`, add a `- id: terraform` entry to the `remediation:` list after `- id: powershell`:

```yaml
        - id: terraform
          desc: |
            **Using Terraform (azuread provider)**

            Declare each Global Administrator assignment explicitly so the count is reviewable in source control:

            ```hcl
            data "azuread_directory_roles" "all" {}

            locals {
              global_admin_template_id = "62e90394-69f5-4237-9190-012177145e10"
            }

            resource "azuread_directory_role_assignment" "global_admins" {
              for_each = toset([
                azuread_user.admin_one.object_id,
                azuread_user.admin_two.object_id,
              ])
              role_id             = local.global_admin_template_id
              principal_object_id = each.value
            }
            ```

            For more details, see [azuread_directory_role_assignment](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs/resources/directory_role_assignment) in the Terraform Registry.
```

- [ ] **Step 4: Confirm remediation comments where applicable**

The other 3 checks (`security-defaults`, `third-party-apps`, `passwords-not-set-to-expire`) already carry a `terraform` remediation entry — leave them unchanged, do **not** add a `# No Terraform remediation:` comment to them (R6 column 3 = "no").

- [ ] **Step 5: Lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":memo: Add audit sections and Terraform notes to M365 identity checks"
```

---

## Task 8: Mobile Device Security checks — audit + remediation

Covers 2 checks: `ensure-that-mobile-device-encryption-is-enabled-to-prevent-unauthorized-access-to-mobile-data`, `ensure-that-mobile-devices-require-a-minimum-password-length-to-prevent-brute-force-attacks`.

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml`

- [ ] **Step 1: Add `audit:` sections**

For both checks, insert an `audit:` block after `desc:`, before `remediation:`, from template R3 and rows 8 and 9 of R4. Both get an `### Audit via Console` (Intune admin center) and an `### Audit via PowerShell` (`Get-MgDeviceManagementDeviceConfiguration`) path.

- [ ] **Step 2: Add the `# No Terraform variants:` comments**

Add a `# No Terraform variants:` comment above each check's `- uid:` per R5/R6: reason — *Intune device configuration profiles have no resource in the `azuread` provider*.

- [ ] **Step 3: Add the `# No Terraform remediation:` comments**

Both checks have only `console` + `powershell` remediation. Add as the last line inside each `remediation:` list, indented level with `- id:`:

```yaml
        # No Terraform remediation: Intune device configuration has no resource in the azuread Terraform provider.
```

- [ ] **Step 4: Lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":memo: Add audit sections and Terraform notes to M365 mobile device checks"
```

---

## Task 9: Email Security checks — audit + remediation

Covers 6 checks: `ensure-that-spf-records-are-published-for-all-exchange-domains`, `ensure-dkim-signing-enabled-for-all-exchange-domains`, `ensure-safe-links-policies-configured`, `ensure-safe-attachments-policies-configured`, `ensure-anti-phishing-policies-enabled`, `ensure-transport-rules-enforce-tls`.

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml`

- [ ] **Step 1: Add `audit:` sections**

Insert an `audit:` block after `desc:`, before `remediation:`, for each of the 6 checks, from template R3 and rows 11, 13, 14, 15, 16, 18 of R4.

- For SPF (row 11): **single path, no H3 headers** — DNS only. Example:

```yaml
      audit: |
        Verify that each Exchange domain publishes an SPF record by querying its DNS TXT records:

        ```bash
        dig +short TXT example.com
        ```

        A passing domain returns a TXT record beginning with `v=spf1`. A failing domain returns no `v=spf1` record.
```

- For DKIM, Safe Links, Safe Attachments, anti-phishing, transport rules: two paths — `### Audit via Console` (Microsoft Defender portal, or Exchange admin center for transport rules) and `### Audit via PowerShell` (Exchange Online PowerShell — `Connect-ExchangeOnline` then the cmdlet from R4).

- [ ] **Step 2: Add the `# No Terraform variants:` comments**

Add a `# No Terraform variants:` comment above each of the 6 checks' `- uid:` per R5/R6 (SPF and the five Exchange Online checks each have their own reason in R6).

- [ ] **Step 3: Add the `# No Terraform remediation:` comments**

For the 5 Exchange Online checks (DKIM, Safe Links, Safe Attachments, anti-phishing, transport rules), add as the last line inside each `remediation:` list:

```yaml
        # No Terraform remediation: Exchange Online configuration has no Terraform provider.
```

Do **not** add a remediation comment to the SPF check — it keeps its existing multi-provider DNS `terraform` remediation entry.

- [ ] **Step 4: Lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":memo: Add audit sections and Terraform notes to M365 email security checks"
```

---

## Task 10: SharePoint check — audit + remediation

Covers 1 check: `ensure-sharepoint-external-sharing-restricted`.

**Files:**
- Modify: `content/mondoo-m365-security.mql.yaml`

- [ ] **Step 1: Add the `audit:` section**

Insert an `audit:` block after `desc:`, before `remediation:`, from template R3 and row 17 of R4 — `### Audit via Console` (SharePoint admin center → Policies → Sharing) and `### Audit via PowerShell` (`Connect-SPOService` then `Get-SPOTenant | Select SharingCapability`).

- [ ] **Step 2: Add the `# No Terraform variants:` comment**

Add above the check's `- uid:` per R5/R6: reason — *SharePoint Online tenant settings have no Terraform provider*.

- [ ] **Step 3: Add the `# No Terraform remediation:` comment**

Add as the last line inside the `remediation:` list:

```yaml
        # No Terraform remediation: SharePoint Online tenant settings have no Terraform provider.
```

- [ ] **Step 4: Lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":memo: Add audit section and Terraform note to M365 SharePoint check"
```

---

## Task 11: Final validation

**Files:**
- None modified — verification only.

- [ ] **Step 1: Full lint**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml`
Expected: no errors, no warnings.

- [ ] **Step 2: Confirm every check has all three docs sections**

Run:

```bash
python3 -c "
import yaml
d = yaml.safe_load(open('content/mondoo-m365-security.mql.yaml'))
bad = []
for q in d.get('queries', []):
    docs = q.get('docs')
    if docs is None:
        continue  # variant child queries have no docs
    for sec in ('desc', 'audit', 'remediation'):
        if sec not in docs:
            bad.append((q['uid'], sec))
print('MISSING:', bad if bad else 'none')
"
```

Expected: `MISSING: none`.

- [ ] **Step 3: Confirm variant structure**

Run:

```bash
grep -c "variants:" content/mondoo-m365-security.mql.yaml
grep -c "filter-icon: terraform" content/mondoo-m365-security.mql.yaml
```

Expected: `5` variant blocks; `15` terraform filter-icon tags (3 per variant check).

- [ ] **Step 4: Confirm no orphaned group references**

Run: `cnspec policy lint content/mondoo-m365-security.mql.yaml` again and confirm the `groups:` block still resolves — every `- uid:` under `groups:` must match a parent query UID (not a variant child). The linter fails if a referenced UID is missing.

- [ ] **Step 5: Smoke-test the bundle loads**

Run: `cnspec bundle lint content/mondoo-m365-security.mql.yaml` (if `bundle lint` is unavailable, the `policy lint` in Step 1 is sufficient).
Expected: success.

- [ ] **Step 6: Final commit (if any verification fix was needed)**

Only if Steps 1–5 surfaced a fix:

```bash
git add content/mondoo-m365-security.mql.yaml
git commit -m ":bug: Fix M365 policy validation issues"
```

---

## Done criteria

- All 18 parent checks have `desc:`, `audit:`, and `remediation:` in `docs:`.
- 5 Conditional Access checks have `variants:` blocks with `-microsoft365` + 3 Terraform children.
- 13 checks have `# No Terraform variants:` comments.
- 8 checks (mobile ×2, Exchange Online ×5, SharePoint ×1) have `# No Terraform remediation:` comments.
- The global-admins check has a new `azuread_directory_role_assignment` Terraform remediation.
- `version:` is `2.3.0`.
- `cnspec policy lint content/mondoo-m365-security.mql.yaml` passes.
