# 0008. Release integration testing

- **Status:** Proposed
- **Date:** 2026-09-22
- **Deciders:** @chris-rock
- **Consulted / Informed:**

## Context

Before a release we need to know that the cnspec binary we are about to ship
still scans real targets end to end. Nothing in the repo answered that.
`content/validation/` calls the scanner in process against file fixtures, and
`test/providers/` runs the binary but asserts only that its output is non-empty.

Two properties of cnspec decide what such a suite has to look like.

**A scan's exit code says very little.** The default `--risk-threshold` is 101
and the exit comparison is `100 - worstScore >= threshold`, which 100 can never
satisfy. A scan where every check fails, errors or is skipped exits 0; only an
asset-level error changes that. Wrapping a scan in `set -e` therefore proves
that the process started and connected, and nothing about what it found.

**A scan can succeed and still report nothing useful.** If a provider schema
moves, checks come back `error`; if a filter stops matching, they come back
`skip`. Either way there is an asset, a full set of scores, and exit 0. Asset
count and check count look normal. The only signal that separates this from a
healthy scan is whether any check reached a `pass` or `fail` verdict.

**A pre-release suite has to test the artifact, not the source.** Packaging,
ldflags and the production build tag are part of what ships, and a suite that
can only build the working tree cannot answer whether a given release candidate
works.

## Decision

A suite in this repo, at `test/integration/`, behind the `integration` build
tag, that runs a real cnspec binary against Docker images, the local system and
a kind cluster, and asserts on the structured JSON report.

Five choices carry the decision:

**Assert on the report, not the exit code.** Assertions are floors and ratios —
check counts above a floor, errored checks below a ratio, at least one `pass`
and one `fail` — never exact scores or named verdicts, which belong to
`content/validation` and move when a base image updates.

**`CNSPEC_BINARY` selects the artifact.** Unset, the suite builds the working
tree; set, it exercises a downloaded release. The CI workflow's `version` input
drives it, so a release candidate is checked as the bytes that will ship.

**Local targets only, no credentials.** No Mondoo Platform service account, no
cloud tenants. Every scan is incognito, and the suite strips `MONDOO_*` from
every child process so it cannot silently become an authenticated scan.

**Serial execution.** Measured, not assumed: on the docker tier, serial 67s vs
`-parallel 5` 86s cold, and 85s vs 111s warm.

**Manual trigger.** `workflow_dispatch` with a `version` input.

## Security implications

The suite runs no credentialed scans. It strips every `MONDOO_*` variable from
the environment of each cnspec invocation and points `MONDOO_CONFIG_PATH` at a
file that does not exist, so a developer's service account — or a secret
exported into a CI job — cannot turn a test run into an authenticated platform
scan against a real space. This is not theoretical: the first scan attempted
while building the suite failed because it had picked up a local service account
and an auto-discovered `inventory.yml` and retargeted itself, unasked.

The workflow requests `permissions: contents: read` and uses only the default
`GITHUB_TOKEN`, to download release assets from this repository. It adds no new
secret and no new trust boundary. It does reach the public network to pull
container images and to download providers from the production registry, which
is the same trust boundary any customer install has.

The kind tier refuses to scan a kubeconfig context whose name does not begin
with `kind-`, so a developer running the suite with a production context active
cannot accidentally point a scan at it.

## Performance implications

No effect on PR latency: the package is behind a build tag, so it does not
appear in `go list ./...` and cannot be picked up by `make test/go/plain-ci`.

Measured against the `v14.0.0-rc.10` artifact on 2026-09-22, with a cold
provider directory: docker + local + formats 126s, kind 23s plus cluster
creation. The two run as concurrent CI jobs, so the suite is about two minutes
of wall clock and a pre-release run is four to five minutes including runner
setup.

Providers are deliberately not cached in CI. `EnsureProvider` returns early when
a provider is present, so a restored cache would pin provider versions and the
run would stop exercising resolution against the production registry — which is
one of the things a release has to get right. The cost is roughly 60s per job.

## Consequences

### Positive

- A release candidate can be tested as the artifact that will ship.
- The assertions distinguish a healthy scan from one that connected and did
  nothing — a regression class that exit codes, asset counts and check counts
  all report as success.
- The suite tests its own assertion logic (`assert_test.go`) against synthetic
  reports, with no infrastructure, so a bug in a helper is visible.
- Two defects were found while building it and are now pinned by tests: `-o csv`
  is advertised in `--help` but always fails for scan reports, and a recovered
  provider panic turns checks into `error` while the process still exits 0.

### Negative

- **The live-credential connectors are not covered.** `aws`, `azure`, `gcp`,
  `ms365`, `okta`, `github`, `gitlab` and `tls` need credentials and standing
  tenants. Their static IaC equivalents are covered by `content/validation`, but
  the live API paths are covered by nothing here, and this record should not
  imply otherwise.
- **A manual trigger runs when someone remembers to run it.** Accepted for now;
  promoting to a nightly is adding a `schedule:` block, and the job conditions
  are written as `!=` rather than `==` so that a scheduled run, where inputs are
  empty, executes every tier.
- Floors need recalibrating when content or base images drift enough.

### Follow-up

- Decide whether a credentialed tier is worth its cost — tenant ownership,
  credential rotation, a named owner.
- A pull-request trigger needs a pull-through image cache first; Docker Hub's
  anonymous limit is per-IP and shared across the runner fleet.

## Alternatives considered

**Extend `test/providers/`.** It already builds the binary, scans a container
and decodes JSON, and it runs untagged on every push. Rejected: adding image
pulls, a kind cluster and a format matrix would put Docker Hub rate limits and
cluster availability on the critical path of every PR. A suite that is red for
reasons unrelated to the change under review stops being read.

**Golden-file output comparison.** Rejected on the evidence in
`test/sbom/README.md`, which records the same approach failing there: goldens
were recorded on one architecture, tags are mutable so an upstream security
update broke goldens that were still correct about cnspec, and a real parsing
regression looked exactly like either of those.

**Include live-tenant tiers.** Rejected for this suite: a result that depends on
somebody else's SaaS org fails for reasons its readers cannot act on, and a
release gate has to be believable to be useful. Worth deciding on its own terms,
with its own owner.

**Parallel scenarios.** Rejected on measurement — 25–30% slower both cold and
warm, because a scan is CPU and container-daemon bound rather than IO-bound, and
parallel processes on a cold provider directory duplicate downloads. The
parallelism that pays is already there: the two tiers are separate CI jobs.

**A nightly cron from day one.** Rejected in favour of a manual trigger while
the suite settles. Recorded because it is the decision most worth revisiting:
a suite that only runs on request runs less often than one that does not.

## References

- `test/integration/README.md` — how to run it, and what each tier catches
- `test/sbom/README.md` — why golden output was abandoned for live containers
- `content/validation/README.md` — the in-process content suites this complements
- `content/validation/scans/main_test.go` — the in-process provider hazards that
  do *not* apply to a subprocess-per-scenario design
