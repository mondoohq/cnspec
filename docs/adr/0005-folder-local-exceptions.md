# ADR-0005: Folder-Local Exceptions

**Date:** 2026-08-22
**Status:** Proposed

## Context

An exception suppresses a check for a good reason: the finding is wrong, the risk is
knowingly accepted, a compensating control exists, or the check should not run at all.
cnspec already models this. A `PolicyGroup` whose type is `IGNORED`, `DISABLE` or
`OUT_OF_SCOPE_GROUP` carries the checks it applies to, a validity window, a
justification, authors and reviewers, and a review status
([cnspec_policy.proto](../../policy/cnspec_policy.proto)). `normalizeAction` turns the
group type into an action, and `isGroupMatching` already drops groups whose
`valid.until` has passed.

What is missing is a way for an exception to **travel with the code it excuses**.
Today an exception is either authored into a policy bundle — which puts infrastructure
decisions in the hands of whoever maintains the bundle — or held upstream and applied
when upstream resolves the asset's policy. Neither is visible to the person reading the
Terraform module, and neither is reviewed alongside the change that made the exception
necessary.

Teams scanning infrastructure-as-code want the same workflow they already have for
everything else in the repository: write the exception in a file, justify it in the pull
request, have it reviewed by the people who own the code, and have it expire.

Three constraints shape the design.

**Exceptions must be known before resolution.** They change which checks execute and how
results score, and both are decided when the policy is resolved. That rules out reading
them through MQL, because MQL queries run from the execution job that resolution has
already produced. The exceptions have to be in hand at connect time.

**When connected to an upstream, resolution belongs to the upstream.**
`LocalServices.ResolveAndUpdateJobs` ([policy/resolver.go](../../policy/resolver.go))
delegates wholesale to `s.Upstream` when one is configured. Nothing cnspec writes into
its local datalake reaches that resolution. Exceptions found locally must therefore be
sent to the upstream before it resolves, not applied afterwards.

**Only the provider knows where the scanned root really is.** The IaC providers put the
target in `asset.Connections[0].Options["path"]`, but that is not always a path on the
machine running cnspec. Assets discovered from a repository — the `terraform-hcl-git`
connection type, for instance — clone into a temporary directory created inside the
provider (`plugin.NewGitClone`), and cnspec never learns where it is. Path semantics also
vary per provider. Reading the file from cnspec would work for the simplest case and fail
for the case that matters most: a file committed to the repository being scanned.

## Decision

### 1. Exceptions are an entry in `mondoo.yml`, not a new file

No new file format and no new filename. `exceptions` becomes a new top-level entry in the
existing config schema, which is then readable at two scopes:

| Scope | Location | Found by |
|---|---|---|
| user | `~/.config/mondoo/mondoo.yml` | the existing config loader |
| context | the scanned path or repository root | the provider, at connect time |

This is a layered config, in the shape of `git config` global/local: one schema, several
scopes, the narrower one closer to the thing being scanned. It also leaves room for
folder-local properties and policy selection later without inventing a second file.

### 2. Providers find and expose the context config

Discovery is a provider responsibility, performed at connect time and returned with the
asset. The provider is the only component that knows its own root, including when that
root is a temporary clone.

The first pass covers two paths:

| Provider | Reads |
|---|---|
| `terraform` | `mondoo.yml` at the scanned path |
| `github` | `mondoo.yml` at the repository root |

**Root only. No cascading lookup**, no walking up through parent directories. A
consequence worth stating: the same Terraform module can be governed by a different file
depending on how it is reached — scanned directly, the root is the module directory;
discovered through a repository, the root is the repository. This is why reporting which
config governed an asset (below) is a requirement and not a nicety.

Each ingested config carries an **origin record** — provider, repository and ref where
applicable, and path — so that reports can name the file that governed a result.

### 3. A context config may only carry exceptions

The config schema includes credentials, endpoints and feature flags. A context config is
parsed for `exceptions` and nothing else.

| Key group | user | context |
|---|---|---|
| `exceptions` | yes | **yes** |
| credentials (`mrn`, `scope_mrn`, `private_key`, `certificate`, `token`) | yes | ignored, **warn** |
| `api_endpoint`, `api_proxy` | yes | ignored, **warn** |
| `features`, `category`, `detect-cicd`, `labels`, `annotations` | yes | ignored |

Ignoring rather than rejecting is deliberate: someone who adds exceptions to a folder did
not thereby ask for the rest of their config to come along, and a copied config should
still produce a working scan. **Credentials warn** regardless, naming the file — silently
ignoring a private key would mean it is sitting in version control and we said nothing.

Two implementation rules make this safe:

- **A context config is parsed into its own typed struct and never merged into viper.**
  A context config is written by anyone who can open a pull request against the scanned
  repository. If it reached the global config, that person could redirect where results
  are sent or change compiler behaviour.
- **A context config cannot escalate its own scope.** No key inside the file widens what
  that file is permitted to do.

### 4. Format

```yaml
# mondoo.yml, at the scanned terraform path or the repository root
exceptions:
  - title: Central CloudTrail covers this      # optional
    checks:                                     # UID or MRN, several per entry
      - mondoo-terraform-aws-security-s3-bucket-logging
      - mondoo-terraform-aws-security-s3-bucket-log-target
    action: risk-accepted
    justification: >                            # required
      Access logging is handled centrally by the org-wide CloudTrail trail.
    valid_until: 2026-11-01                     # RFC3339 date
```

| Field | Required | Notes |
|---|---|---|
| `title` | no | Free text, carried through to upstream when set. Not an identifier. |
| `checks` | yes | Check UID, or MRN when precision is wanted. Several per entry. **No globs** in the first pass — you name what you mean. |
| `action` | yes | See below. |
| `justification` | **yes** | An exception without a stated reason cannot be defended when it comes up for renewal. |
| `valid_until` | no | RFC3339. Only meaningful for the scoring actions; on `disable` it has no effect, and lint warns. |

Deliberately absent:

- **No `paths`.** Tools whose ignore file is global need a path field to scope entries. A
  context config is already scoped to its folder.
- **No declared authors or approvers.** Both are derived; see [§8](#8-provenance).
- **No per-file expiry policy.** The upstream scope may enforce one, and duplicating that
  in a file the scope does not control would be theatre.
- **No exception identifier.** An entry naming N checks is N check exceptions; identity
  comes from the scope, the check and the action.

### 5. Action vocabulary and its effect

`action` names why the exception exists. The vocabulary matches the exception actions an
upstream exposes, so that an entry in a file and an exception held upstream are the same
object rather than two models to translate between.

| `action` | `GroupType` | Effect |
|---|---|---|
| `disable` | `DISABLE` | The check does not execute. |
| `risk-accepted` | `IGNORED` | Runs, does not score. |
| `false-positive` | `IGNORED` | Runs, does not score. |
| `workaround` | `IGNORED` | Runs, does not score. |

The three scoring actions collapse to the same effect on purpose. The distinction between
them is *why*, and that is metadata carried alongside the finding — not a different score.
Treating a false positive as, say, out-of-scope would change the scoring denominator and
silently disagree with how the same exception is counted elsewhere.

**Out-of-scope is not available for checks.** Declaring something out of scope is a
statement about whether a *control* applies, and it is expressed against controls, not
against the checks that evidence them. Control-level exceptions are not part of this
first pass; lint rejects `out-of-scope` in a file with that reason.

Because MQL check UIDs are what people write and MRNs are what an upstream exception
refers to, cnspec translates UID to MRN before submitting, drops checks the upstream does
not know, and warns — one stale entry must never sink the whole submission.

### 6. How exceptions take effect

The resolver is not modified.

**Without an upstream**, cnspec applies the exceptions itself: after the resolved policy
is built and before `executor.ExecuteResolvedPolicy`
([policy/scan/local_scanner.go](../../policy/scan/local_scanner.go)), set the `ChildJobs`
impact for each excepted check according to the table above, and for `disable` drop the
query from the `ExecutionJob` so it genuinely does not run.

This is the same representation `buildExceptionSet`
([cli/reporter/print_compact.go](../../cli/reporter/print_compact.go)) already reads, so
the CLI's exception output, the JSON reporter and SARIF pick the result up with no
further work.

The resolved policy is **not** re-checksummed after this mutation. Changing impacts does
not change reporting-job UUIDs, so result storage keys stay valid, but
`GraphExecutionChecksum` must keep matching what was cached.

**With an upstream**, cnspec sends the exceptions for the asset *before* the asset's
policy is resolved. The upstream decides what to do with them — apply them, hold them for
review, reject them — and then returns the resolved policy as it always does, now
accounting for whatever it accepted. cnspec does not mutate an upstream-resolved policy.
Both the local report and the upstream record derive from the same resolution, so they
agree by construction rather than by convention.

### 7. Syncing without flooding the upstream

Sending every exception on every scan is wasteful, and against an upstream whose
exception API only creates would actively accumulate duplicates. Three measures, in
order of how much they buy:

**Only authoritative runs sync.** A local scan of a feature branch applies exceptions and
reports them, but does not submit them. Only a run cnspec considers authoritative — CI,
on the default branch — submits. `execruntime.Detect()` already reports the environment
and the ref. This removes most of the volume before any protocol work, and it stops a
developer's local experiment from becoming a shared exception.

**Batch, don't drip.** One submission per scope carrying the whole exception set, not one
call per entry.

**Short-circuit on a checksum.** cnspec computes a checksum over the normalized set and
sends it; when the upstream already holds that set, nothing is written and no rescoring
is triggered.

Scope granularity matters here too: context exceptions are asset-scoped and belong with
the per-asset resolve, while user-scope exceptions apply across the whole configured
scope and must be submitted once per run rather than once per asset.

### 8. Provenance

Authors and approvers are never declared in the file.

**Authors are derived** — from `git blame` on the entry where a checkout is available,
falling back to the CI identity that `execruntime.Detect()` already captures and
[apps/cnspec/cmd/scan.go](../../apps/cnspec/cmd/scan.go) already applies as asset labels.

**Approvers cannot be declared.** At the moment an entry is written, in a pull request,
the approval has not happened yet; any approver named in the file is a prediction written
by the person requesting the exception. The merge is the approval, and the review record
lives in the forge. Where an upstream requires approval before an exception applies, that
mechanism governs — cnspec surfaces its outcome rather than reimplementing it.

### 9. Expired exceptions

Expired entries stay in the file. Deleting them destroys the audit trail, and rewriting
the file on every expiry churns version control for no one's benefit.

They do not apply — `isGroupMatching` already enforces `valid.until`. The scan reports
them explicitly, so a check that starts failing again is explained rather than
mysterious, and warns about entries that are close to expiring. `cnspec exceptions prune`
can rewrite the file without them.

cnspec edits the file and stops there. Committing, branching and opening a pull request
are the user's workflow and cnspec does not touch version control.

## What this requires from an upstream

cnspec keeps its upstream interface neutral, so this is stated as a contract rather than
an integration:

1. **Accept a set of exceptions for a scope, before resolution**, and reflect the ones it
   accepts in the resolved policy it returns.
2. **Be idempotent.** Submitting an unchanged set repeatedly must not accumulate
   duplicates. Either upsert against a stable key, or accept a whole-set submission with
   a checksum and diff internally. Without this, cnspec cannot submit on a schedule and
   the sync in §7 has to be gated much more aggressively.
3. **Report per-entry outcome**, so cnspec can tell the user that some entries were not
   accepted. If the response carries only a resolved policy, cnspec can still derive
   which exceptions took effect from the reporting-job impacts, but not why the others
   did not.
4. **Allow the scan identity to submit exceptions.** Submitting is a distinct permission
   from scanning.

An upstream that offers none of this still works: cnspec applies exceptions locally and
reports them, and simply does not sync.

## Alternatives considered

**A dedicated file — `mondooignore.yaml` or `.mondoo/exceptions.yaml`.** Rejected.
"Ignore" is the wrong word for a justified, expiring, auditable exception, and a
dedicated name cannot grow when folder-local properties or policy selection arrive; we
would ship a second file. A `.mondoo/` directory would let CODEOWNERS target the
exceptions path specifically, which is a real loss — but users can put `mondoo.yml` under
CODEOWNERS themselves.

**cnspec reads the file from the connection's `path` option.** Rejected as the primary
mechanism. It works for a local scan and fails for the case the feature exists for, a
file committed to a repository that a provider clones internally. It also puts
per-provider path semantics in cnspec.

**Reading the config through MQL.** Rejected on timing: MQL queries execute from the
execution job that resolution already produced, and exceptions must be known before
resolution.

**Applying local exceptions by mutating an upstream-resolved policy.** Rejected as the
primary path. The local report would reflect the exception and the upstream record would
not, and the two would disagree silently. Local mutation is used only when there is no
upstream.

**Carrying exception decisions on `ResolvedPolicy`.** Rejected. It is a checksummed,
cacheable artifact keyed by execution checksum; per-scan, per-asset status does not
belong in it.

**A separate reason and effect field.** Rejected as redundant for the first pass: each
action already implies its effect, and a file that can state both invites entries whose
two halves disagree.

## Consequences

**Positive**

- Exceptions are reviewed where the code is reviewed, by the people who own it.
- One config schema rather than a new file format, with room for folder-local properties
  and policy selection later.
- Reporting comes almost free: the local application produces exactly the impacts
  `buildExceptionSet` already reads.
- Expiry is enforced by machinery that already exists.

**Negative**

- The same module can be governed by different files depending on how it is scanned,
  until cascading lookup lands.
- A `mondoo.yml` in a repository is untrusted input, and the allowlist and the
  parsed-separately rule are load-bearing rather than defensive extras.
- Exception behaviour differs between a connected and a disconnected scan: connected, the
  upstream decides what applies; disconnected, everything in the file applies. That is
  correct but it needs to be explained in the docs.
- The vocabulary must stay aligned with whatever an upstream exposes. Divergence would
  show up as a file that lints locally and is refused on submission.

**Neutral**

- Provider work is per-provider. Two are covered here; the rest arrive as discovery
  needs them.

## Out of scope

Deferred deliberately, so the first pass stays small:

- Other providers — cloudformation, helm, kustomize, bicep, ansible, k8s, os, gitlab.
- Cascading config lookup through parent directories.
- Globs on check identifiers.
- Exceptions for controls, CVEs, advisories and third-party findings.
- Folder-local properties and policy selection.
- Inline source comments, e.g. a `#mondoo:ignore` marker in the HCL itself.
- Pulling exceptions held upstream back into the repository file.
- Reporting CODEOWNERS-derived approval.

## References

- [policy/cnspec_policy.proto](../../policy/cnspec_policy.proto) — `GroupType`,
  `PolicyGroup`, `Validity`, `ReviewStatus`
- [policy/resolved_policy_builder.go](../../policy/resolved_policy_builder.go) —
  `normalizeAction`, `isGroupMatching`
- [policy/resolver.go](../../policy/resolver.go) — `ResolveAndUpdateJobs` and the
  upstream delegation
- [policy/scan/local_scanner.go](../../policy/scan/local_scanner.go) — `runPolicy`, the
  seam for local application
- [cli/reporter/print_compact.go](../../cli/reporter/print_compact.go) —
  `buildExceptionSet`
- `mql/cli/config` — the `mondoo.yml` schema and loader
- `mql/providers-sdk/v1/plugin` — the connect-time channel for the context config
- `mql/cli/execruntime` — CI identity and ref detection
- [ADR-0001](0001-scan-parallelization-pipeline.md) — scan pipeline the connect-time
  config ingestion has to fit into
