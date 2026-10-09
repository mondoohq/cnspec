# 0012. Diff-scoped IaC scanning for pull requests

- **Status:** Proposed
- **Date:** 2026-10-09
- **Deciders:** @chris-rock
- **Consulted / Informed:**

## Context

A pull request that changes one Terraform resource or one Kubernetes manifest
gets a scan of the whole directory. `cnspec scan terraform <dir>` and
`cnspec scan k8s <dir>` report every failing check in it, and
`--risk-threshold` gates the job on the worst score of the asset. A repository
with existing findings therefore fails every pull request until all of them are
fixed, whether or not the change touched them, and the findings the change did
introduce are mixed in with the ones it did not.

Teams want a pull request scan to show, and to fail on, the findings that
relate to the change. They also want the complete posture of the repository to
keep reaching Mondoo Platform.

cnspec already knows where many findings are. Resources declared with
`@context` carry a source path and line range (`llx.SourceContext`), and
`CodeBundle.FailingResourceContexts` collects them for the failing items of a
check. The SARIF, JUnit and HDF reporters use this today. What the source
locations cover was measured with cnspec 14.4.0:

- **Terraform HCL.** Scanning a two-file directory with
  `mondoo-aws-security`, 12 of the 13 failing SARIF results carried the line
  range of the failing `resource` block (`terraform.block.context`). The paths
  are relative to the scan argument (`tfdir/a.tf`), not to the repository.
- **Terraform plan and state.** No source positions. Terraform's plan and state
  JSON do not record them, and `terraform.block.context` returns an error for
  these assets.
- **Kubernetes manifests.** No finding carries a location, in SARIF or
  elsewhere. Discovery turns each workload into its own asset, and the shipped
  checks are scalar assertions on that asset
  (`k8s.deployment.runsPrivileged == false`), so the failing value is a boolean
  without a context. The asset carries `k8s.mondoo.com/kind`, `name` and
  `namespace` labels, but not the manifest it came from.
- **Kubernetes directory scans.** The resource context exists but is wrong for
  every file after the first. The manifests are concatenated into one stream,
  so in a directory holding `a.yaml` and `b.yaml`, the Deployment in `b.yaml`
  reports path `k8sdir` and lines 22-40; scanned alone it reports
  `k8sdir/b.yaml` and lines 1-19.

## Decision

Add a diff scope to `cnspec scan`. The scan itself is unchanged; the scope
decides which findings are shown and which gate the job.

### Scan everything, scope the output

cnspec parses and evaluates the full target as it does today. Terraform
resolves variables, modules and references across files, and a Kubernetes
object can depend on a namespace declared elsewhere, so evaluating only the
changed files would produce different, wrong verdicts. The diff is applied
after evaluation, to the report renderers and the exit code.

`policy.Report` is never modified. The report uploaded to Mondoo Platform is
the same full report a scan without a diff produces, so a pull request scan
cannot lower or raise the posture of the asset.

### Where the diff comes from

- `--diff-base <ref>`: cnspec computes `git merge-base <ref> HEAD` and diffs
  that commit against the working tree with `git diff --unified=0 -M`. The
  merge-base, rather than the tip of the base branch, keeps commits that landed
  on the base branch after the pull request forked from being attributed to the
  pull request. A merge-base answer is accepted only if it is a single commit
  hash; otherwise cnspec falls back to the ref with a warning.
- `--diff-base auto`: the base ref comes from the CI environment that
  `execruntime` already detects, for example `GITHUB_BASE_REF` on GitHub
  Actions or `CI_MERGE_REQUEST_DIFF_BASE_SHA` on GitLab. Outside a pull request
  there is no base, and the scan reports everything, with a notice.
- `--diff-file <path|->`: a unified diff supplied by the caller, for CI systems
  that do not provide the git history.

A base ref that is not present in the clone is an error that names the cause
(a shallow clone, for example `actions/checkout` without `fetch-depth: 0`). It
never degrades into a full or an empty result.

### What is in scope

Changed lines are the new-side lines of each hunk. Paths from the diff and from
source contexts are both made relative to the repository root
(`git rev-parse --show-toplevel`) before they are compared.

A failing check is in scope when one of its locations overlaps a changed line.
Its locations are, in order of preference:

1. The source contexts of its failing resources. The whole range counts, not
   the first line, so a changed attribute inside a resource block puts that
   block in scope.
2. The source location of the asset, when the failing value carries no context.
   This is the Kubernetes case: each discovered object is an asset, and its
   manifest path and line range identify it.

A failing check with neither location is out of scope. So are findings in
deleted files, and a pure rename has no changed lines.

### Output and exit code

Every report format omits out-of-scope findings and states how many it omitted
("N findings outside this change not shown"), in the human output and in the
machine formats (in SARIF, under `invocations[].properties`). Pre-existing
findings stay visible as a count, rather than disappearing.

With a diff scope, `--risk-threshold` is compared against the highest risk
among in-scope failing checks, not against the asset score. Scan errors still
exit 1. Without a diff scope, every output and the exit code are unchanged.

SARIF fingerprints do not change with the scope, so a code scanning alert keeps
its identity between a pull request scan and a full scan of the default branch.

### Required provider changes (mql)

- The Kubernetes manifest connection indexes positions per file and records
  each object's own path, which fixes directory scans.
- Discovery records the manifest path and line range of each object on the
  asset it creates, so scalar checks have a location. As a side effect,
  Kubernetes findings gain a SARIF physical location, which they lack today.

### CI integration

The `cnspec` action in [mondoohq/actions](https://github.com/mondoohq/actions)
gains a `diff` input (`off` or `auto`, default `off`) that passes
`--diff-base auto`, and its documentation states that the checkout needs the
base branch history.

## Security implications

- **No new way to hide a failure from the platform.** The scope only affects
  local output and the exit code; the uploaded report is complete. A pull
  request scan cannot make a repository look compliant in Mondoo Platform.
- **The gate becomes narrower by design.** A job using `--diff-base` passes when
  the change introduces no in-scope failure, even when the repository has
  existing findings. That is the purpose of the feature, and the omitted count
  keeps the existing findings visible. A team that wants pull requests to fail
  on existing findings does not set `--diff-base`.
- **Indirect effects are not caught in this phase.** A change to a Terraform
  variable, a module input or a default can make an unchanged resource fail.
  The diff does not touch that resource, so the finding is out of scope and only
  counted. The same holds for every Terraform plan and state finding. This is
  the main limitation of this decision; see Follow-up.
- **Out-of-scope is never silent.** A finding without a location is counted as
  omitted, not dropped without a trace.
- **Untrusted input.** A ref from the CI environment or a flag reaches git as a
  single argument after validation against a ref pattern. git is executed
  directly, never through a shell. A supplied diff file is parsed as data.

## Performance implications

A diff-scoped scan costs the same as a full scan plus one `git merge-base` and
one `git diff`, which is negligible next to policy evaluation. There is no
speed-up from scanning less, because the full target is still evaluated; that
is the cost of correct verdicts across files. The scope filter does work
proportional to the number of failing resources, which the reporters already
iterate.

## Consequences

### Positive

- A pull request scan shows and gates on the findings the change relates to,
  for Terraform HCL and Kubernetes manifests.
- The platform keeps receiving full, unchanged reports.
- Kubernetes findings get file locations in every format, and Kubernetes
  directory scans report correct paths and lines, with or without a diff.
- Paths in reports become repository-relative under a diff scope, which code
  scanning integrations require to attach findings to files.

### Negative

- Findings caused by a change but located in unchanged code are not shown, and
  Terraform plan and state findings are never in scope (see Security
  implications).
- The scan of a pull request takes as long as a full scan.
- A check that fails without a resource context on a multi-resource asset (one
  of the 13 Terraform failures measured above) cannot be scoped and is always
  omitted under a diff. Making such checks name their failing resource is
  content work outside this decision.

### Follow-up

- A base-versus-head comparison as a second mode, in its own ADR: scan the
  merge-base and the head, match findings by check and a resource identity that
  does not depend on line numbers (the Terraform address, the Kubernetes
  kind/namespace/name), and report findings that are new. This covers indirect
  effects and Terraform plans, at the cost of two scans and a stable identity
  field on resource contexts.
- Annotations or a job summary for CI systems that do not ingest SARIF.

## Alternatives considered

- **Report every finding in a changed file.** Simplest, needs no line numbers.
  Rejected: the pre-existing findings in any file the change touches still fail
  the job, which is the problem being solved, just in fewer files.
- **Match on the first line of a finding.** Rejected: a Terraform resource or a
  Kubernetes object is a block, and the common change is an attribute inside
  it. The first line of the block is usually unchanged.
- **Scan only the changed files.** Faster. Rejected: Terraform evaluation
  needs the variables, modules and references in other files, and a scan
  without them returns different verdicts, not a subset of the right ones.
- **Filter the SARIF output in the CI action.** Needs no cnspec change.
  Rejected: it has to be rebuilt for each CI system and each format, the paths
  in the report are not repository-relative to match against, Kubernetes
  findings have no location to match, and the exit code would still be decided
  by the unscoped asset score.
- **Base-versus-head comparison as the first step.** Covers indirect effects.
  Deferred, not rejected: it needs a resource identity that survives line moves
  and doubles the scan time, while the resource-overlap scope works with the
  locations cnspec already produces.

## References

- `llx.SourceContext` and `CodeBundle.FailingResourceContexts` (mql
  `llx/source_context.go`)
- `cli/reporter/sarif.go`, `reports/reportdoc/query.go` (current consumers of
  source contexts)
- [SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
