# Integration tests

Runs a real cnspec binary against real targets — container images, the local
system, and a Kubernetes cluster — and asserts on the structured JSON report.

Most of it runs without credentials: no cloud tenants, no SaaS orgs, and every
scan incognito. The network is still used, deliberately — container images are
pulled and providers are downloaded from the production registry, and those
paths are part of what a release has to get right.

The one exception is the **upstream tier**, which authenticates against the
Mondoo Platform. It is opt-in and skips unless asked for; see below.

## Running it

```bash
make test/integration            # every tier
make test/integration/docker     # container images + the risk-threshold test
make test/integration/local      # the machine you are on
make test/integration/k8s        # needs a cluster, see below
make test/integration/formats    # the output format matrix
```

The suite is behind the `integration` build tag, so it never runs in
`go test ./...`. A package whose files are all tagged out does not appear in
`go list ./...` at all, which is what `make test/go/plain` iterates.

Tiers skip when their infrastructure is missing, with the reason and the command
that would fix it. A laptop with Docker runs the docker and local tiers and
skips kind.

For the k8s tier:

```bash
kind create cluster --name cnspec-integration
make test/integration/k8s
```

The suite refuses to scan a kubeconfig context whose name does not start with
`kind-`. `cnspec scan k8s` reads whatever context is current, and a developer
running this with a production context active would point a scan at it. Set
`CNSPEC_IT_K8S_CONTEXT=<name>` to override deliberately.

## Testing a release artifact

This is the reason the suite exists in this shape. `CNSPEC_BINARY` points it at
an already-built binary instead of building the working tree:

```bash
gh release download v14.0.0-rc.10 --repo mondoohq/cnspec \
  --pattern 'cnspec_14.0.0-rc.10_linux_amd64.tar.gz' --dir /tmp/rc
tar xzf /tmp/rc/cnspec_14.0.0-rc.10_linux_amd64.tar.gz -C /tmp/rc

# a cold provider dir, so provider resolution is exercised too
CNSPEC_BINARY=/tmp/rc/cnspec PROVIDERS_PATH=$(mktemp -d) make test/integration
```

The binary and its version are printed before the first test, so a CI log always
names what was exercised.

In CI, the `version` input of `.github/workflows/integration-tests.yaml` does
the same thing via `.github/scripts/resolve-cnspec-artifact.sh`.

## What each tier catches

| Tier | Covers |
|---|---|
| `docker` | provider download and resolution from the production registry; default policy resolution with no bundle; apk and dpkg platform detection; a repo bundle compiling and scoring against a real OS |
| `local` | the local connector and OS provider on whatever the runner is |
| `k8s` | the k8s provider, cluster discovery, and the only multi-asset scan in the suite — the one place an aggregation bug (assets discovered, results dropped or collapsed) is visible |
| `formats` | `-o <name>` wiring from the format registry through the output handler to stdout, for every format a scan supports |
| `upstream` | registration, policy resolution from the space, the vulnerability service, and report upload — everything incognito skips |

## The upstream tier

```bash
CNSPEC_IT_UPSTREAM_CONFIG=/path/to/serviceaccount.json make test/integration/upstream
```

Unset, it skips. It registers assets in whatever space the service account
belongs to.

**Not `MONDOO_CONFIG_PATH`.** The suite strips every `MONDOO_*` from the child
environment, so a credential that happens to be in the environment — a
developer's own, or a secret exported into a CI job for something else — cannot
pull a tier upstream that was meant to run incognito. Reaching the platform has
to be asked for by name, and only this tier passes it through.

### What it is for

The client and the platform ship separately, and a fleet routinely runs a client
ahead of the server it reports to, so "the client is newer" is the normal case
rather than the exception. Nothing else here covers it: incognito skips
registration, policy resolution from the space, the vulnerability service and
the upload entirely.

`TestUpstreamStatus` logs both API versions, so the CI log records what the run
actually proved compatible:

```
integration: version cnspec 14.0.0-rc.10
client API v14
server API v13 ✓ client ahead
```

### The assertion the tier turns on

Without a usable credential cnspec does not fail — it logs `Switching to
--incognito mode` and scans happily. Every other assertion in the tier would
then pass while testing none of the upstream path, which is the exact shape of a
test that reports success for something it stopped doing. So:

- stderr must not contain the incognito fallback;
- the asset MRN must match `//assets.<host>/spaces/<space>/assets/…`, because an
  incognito scan mints a local identifier instead;
- stderr must contain `uploaded scan data`, asserted there rather than on the
  report because stdout is written before the upload is attempted;
- a credential that is configured but unreadable is a **failure**, never a skip.

Both failure modes are verified: pointing the variable at a missing file fails
with the path in the message, and pointing it at a readable file that is not a
service account fails on the incognito assertion rather than passing quietly.

No check-count floor here. The policies come from the space, so how many apply
is the space's business — what is asserted is that the engine ran them and they
produced real verdicts, with no check ending in an error.

### In CI

The `upstream` job reads the `MONDOO_INTEGRATION_SERVICE_ACCOUNT` secret, writes
it to `$RUNNER_TEMP` at 0600 (outside the checkout, so nothing that globs the
workspace picks it up), and removes it in an `always()` step. When the secret is
absent — a fork, or a repo that has not configured it — the job reports that and
does not run the tier, because a missing secret is a property of where the
workflow is running and not a result about cnspec.

## Two things that shape every assertion here

**The exit code is not a gate.** cnspec's default `--risk-threshold` is 101 and
the comparison is `100 - worstScore >= threshold`, which 100 can never satisfy.
A scan where every check fails, errors or is skipped still exits 0; only an
asset-level error changes that. So `set -e` around a scan proves only that the
process started and connected, which is why everything here asserts on the
report instead. `TestRiskThresholdExitCodes` pins the arithmetic on both sides.

**`-o json` does not emit a `policy.ReportCollection`.** `--json` and `-o json`
both resolve to `FormatJSONv2`, which goes through `ConvertToProto` and
`protojson.Marshal`, so the payload is `cli/reporter.Report`: assets, data,
errors, and per-check scores keyed by asset. Decoding it into
`policy.ReportCollection` appears to work because the `assets` key collides, and
then everything else comes back empty. Decode with **protojson**, not
`encoding/json`: protojson emits a field's JSON name (`platformName`) while the
generated struct tag carries the proto name (`platform_name`), so
`encoding/json` silently leaves that field empty and an assertion on it compares
against `""`. `TestDecodeReportNeedsProtojson` pins this.

## No check may error

Every scenario asserts that no check ended with status `error`. A check's
outcome is a verdict, pass or fail. `error` means it never reached one: the
target lacks the resource, the provider returned an error, or the query could
not run. Each is a defect to fix, in the check's scoping or in the provider,
never an expected property of the target. A check that cannot apply to a target
is filtered out of it and reports as skipped. Errored checks do not reach the
report's error map or the exit code, so nothing else here would see them; the
failure message lists every errored check.

## Why the assertions are floors, not equalities

| Asserted | Not asserted | Why |
|---|---|---|
| `platformName == "alpine"` | an exact OS version | patch releases move under a tag |
| `len(checks) >= 20` | `len(checks) == 64` | content releases add checks |
| a check UID prefix is present | a named check *passes* | whether a given check passes on alpine is a content fact, owned by `content/validation` |
| at least one `pass` **and** one `fail` | an overall score value | the score is a content-weighted number |

The last row is the one that matters most. A scan that connects, produces an
asset, scores every check as `skip` or `error` and exits 0 is indistinguishable
from a healthy one by exit code, by asset count, and by check count — so that is
the case `requireVerdicts` exists to catch.

`test/sbom/README.md` records why golden output was abandoned there — mutable
tags broke goldens that were still correct about cnspec. That lesson constrains
what is asserted, not how images are pinned, which is why images here use minor
tags (`alpine:3.20`) rather than digests.

Each floor carries the count observed when it was set, so drift is visible
without re-deriving it. To recalibrate, run the tier with `-v`, read the
`statuses: ...` line from the failure, and set the floor at roughly half.

## Why the suite tests itself

`assert_test.go` runs against synthetic reports and needs no daemon, cluster or
network. The assertions are the whole product here, and an assertion helper with
a bug is indistinguishable from no assertion at all. `TestVerdictCount` encodes
the case that matters most: a report where nothing reached a verdict.

## How long it takes

Measured against the `v14.0.0-rc.10` release artifact on 2026-09-22, on a
developer machine under variable load:

| CI job | Tiers | Cold providers | Warm |
|---|---|---|---|
| `docker-local` | docker (67s) + local (42s) + formats (17s) | 126s | ~98s |
| `k8s` | kind | 23s + cluster creation | 5s |

The two jobs run concurrently, so the suite is roughly **two minutes** of wall
clock and a pre-release run lands around four to five minutes once runner
startup, checkout and Go setup are counted. With the workflow's `version` input
set there is no compile at all; building from source adds ~6s on a warm Go
cache and several minutes on a cold one.

Treat these as indicative and confirm them on the first CI run. They were taken
on a shared machine whose load average reached 349 during later measurements,
which is enough to distort any of them — `scan local` is the most sensitive,
since it scans the whole host, and it was observed taking anywhere from 42s to
over 5 minutes purely on contention. The docker and kind tiers are the stable
numbers.

## Why serial, and why only one retry

Scenarios run serially (`-parallel 1`, no `t.Parallel()`), and this was measured
rather than assumed. On the docker tier:

| | Cold providers | Warm |
|---|---|---|
| serial | **67s** | **85s** |
| `-parallel 5` | 86s | 111s |

Parallel is 25–30% slower both ways. A scan is CPU and container-daemon bound
rather than waiting on IO, so five at once only adds contention — and on a cold
directory they also duplicate the provider downloads. The correctness argument
points the same way: each scenario is its own cnspec process, so the in-process
hazards documented in `content/validation/scans/main_test.go` (the
`providers.ListAll()` cache-warm race, the Coordinator lock-ordering deadlock)
do not apply, but parallel processes would race to install the same provider
into a shared `PROVIDERS_PATH`, whose failure mode is a corrupted provider
directory that reads as a cnspec bug and does not reproduce.

Parallelism that does pay off is already in place: the docker/local and kind
tiers are separate CI jobs and run at the same time.

`docker pull` is retried three times with backoff. **A scan is never retried.**
That is the deliberate opposite of `scanSnippet` in
`content/validation/scans/remediation_closes_check_test.go`, which retries
because in-process concurrency kills provider subprocesses — a pressure this
suite does not create. An intermittently failing scan is the signal this suite
exists to produce; retrying it would hide exactly what we want to see.

## Environment isolation

Every cnspec invocation runs with all `MONDOO_*` variables removed and
`MONDOO_CONFIG_PATH` pointed at a file that does not exist.

This is correctness, not hygiene. The config loader reads
`MONDOO_CONFIG_BASE64`, then `$MONDOO_CONFIG_PATH`, then
`~/.config/mondoo/mondoo.yml`, and an `inventory.yml` next to that config is
auto-discovered. Both were observed while building this suite: the first scan
attempted here failed with `could not initialize client authentication` because
it had picked up a local service account and inventory, having never been asked
to. Without the scrub, a developer's laptop and a CI runner run different scans.

## Skips are failures in CI

`CNSPEC_IT_REQUIRE_ALL=1` turns a tier skip into a failure. The workflow sets it
on every job. A suite that can quietly skip a tier reports success for the thing
it stopped testing, and a skip is much easier to introduce by accident than a
failure is.

## Providers are not cached in CI

`EnsureProvider` returns early when a provider is already present, so a restored
cache pins provider versions and the run stops exercising what it is here for:
resolving and downloading providers from the production registry with the binary
under test. `content-iac-tests.yaml` caches with a daily-rotating key because it
runs on every PR; this runs a handful of times per release, so ~60s is not worth
the loss of signal.

## Promoting to a nightly

The workflow is `workflow_dispatch` only, which means it runs when someone
remembers to run it. Promoting it to a nightly is adding a `schedule:` block:

```yaml
on:
  schedule:
    - cron: "17 5 * * *"
  workflow_dispatch:
```

Nothing else changes. The job conditions are written as `inputs.tiers != '...'`
rather than `== '...'` precisely so a scheduled run, where inputs are empty,
executes every tier.

Before adding a pull-request trigger, move the images behind a pull-through
cache: Docker Hub's anonymous limit is per-IP and shared across the runner fleet.

## Known gaps

**`-o csv` is advertised but not implemented for scan reports.** It is in the
reporter's format map, so `cnspec scan --help` lists it, but writing a scan
report in it falls through to the default branch and the command exits 1 with
`unknown reporter type`. CSV works only for vulnerability reports.
`TestOutputFormatCSVIsNotImplemented` asserts the current behaviour so that the
day it is implemented, that test fails and someone moves it into the matrix.

**Live-credential connectors are not covered.** `aws`, `azure`, `gcp`, `ms365`,
`okta`, `github`, `gitlab` and `tls` all need credentials and standing tenants,
which this tier deliberately does without: a suite whose result depends on
somebody else's SaaS org is red for reasons its readers cannot act on. Their
static IaC equivalents are covered by `content/validation/scans`. Covering the
live API paths is a separate decision with a separate cost — tenant ownership,
credential rotation, a named owner — and worth making on its own terms.
