# 0009. Upstream integration tier

- **Status:** Proposed
- **Date:** 2026-09-22
- **Deciders:** @chris-rock
- **Consulted / Informed:**

## Context

[ADR-0008](0008-release-integration-testing.md) scoped the integration suite to
targets that need no credentials, and recorded the coverage that gave up: the
platform path was left untested. That decision stands for the tiers it covers.
This one adds the piece it deliberately left out, for a reason 0008 did not
weigh.

The client and the platform are released separately. A fleet routinely runs a
client that is ahead of the server it reports to, so "the client is newer than
the server" is the normal case rather than an edge case — and before a major
client release it is the specific question worth answering.

Nothing in the repo could answer it. An incognito scan skips the entire upstream
path: it does not register an asset, does not resolve the policies a space
assigns, does not ask the vulnerability service, and does not upload a report. A
release that broke any of those would leave every existing tier green.

The failure mode that makes this hard to test is that **cnspec does not fail
without a credential** — it logs `Switching to --incognito mode` and scans
happily. A tier that authenticates, but does not assert that it authenticated,
passes just as well with a missing or expired service account as with a working
one. That property, not the network access, is what the design has to address.

## Decision

An opt-in `upstream` tier in the existing suite, selected by a dedicated
variable and asserting that it actually reached the platform.

**A dedicated variable, not `MONDOO_CONFIG_PATH`.** The suite strips every
`MONDOO_*` from the child environment specifically so no tier can be pulled
upstream by a credential that happens to be present. That protection is kept;
`CNSPEC_IT_UPSTREAM_CONFIG` is read by the suite and passed through for this
tier only. Reaching the platform has to be asked for by name.

**Assert the negative.** The tier fails if stderr carries the incognito
fallback, if the asset MRN is not a space-scoped one, or if the upload did not
happen. A credential that is configured but unreadable is a failure, never a
skip.

**Skip when not asked for.** Unset, the tier skips with the variable name in the
message, so the credential-free tiers remain the default everywhere.

**Report both API versions.** `cnspec status` is run first and both versions are
logged, so the CI record states what the run proved compatible rather than
leaving it to be inferred.

## Security implications

This is the change ADR-0008 avoided, so the cost is stated plainly.

It introduces the first credential in this repo's CI: a Mondoo service account
in `MONDOO_INTEGRATION_SERVICE_ACCOUNT`. Handling:

- written to `$RUNNER_TEMP` at mode 0600, outside the checkout, so a later step
  that globs the workspace cannot pick it up;
- removed in an `always()` step, so a failing run does not leave it behind;
- never placed on a command line, where it would be visible in a process
  listing;
- absent on forks, where the job reports the absence and does not run.

The scoping protection is unchanged for every other tier: `MONDOO_*` is still
stripped from the child environment, and only this tier restores a credential.
A developer running the docker or local tier cannot reach the platform by
accident, which was the property the first scan attempted during 0008's work
violated.

The service account should belong to a space kept for this purpose. The tier
registers assets there on every run, so it writes to a real space — that is the
point of the test, and it is the reason the space should not be a customer's or
a production inventory.

## Performance implications

No effect on the other tiers or on PR latency: the tier is behind the same build
tag and skips unless the variable is set.

Measured against `v14.0.0-rc.10` and the production platform on 2026-09-22:
`cnspec status` 2.2s, the full authenticated scan 35.9s, 39s for the tier. It is
a separate CI job, so it runs concurrently with the others and does not extend
the suite's wall clock.

## Consequences

### Positive

- The question "does this client still work against the server in production?"
  has an answer that is produced by a run rather than by reasoning.
- The upstream path — registration, space policy resolution, the vulnerability
  service, upload — is covered for the first time.
- Both API versions land in the CI log, so the record is self-describing.

### Negative

- A credential now exists in CI, with the handling obligations above. This is a
  real widening of the repo's trust boundary and should be reviewed as one.
- The tier writes to a live space on every run, accumulating assets there.
  Nothing prunes them.
- It cannot run on forks, so a contributor's PR gets no signal from it.
- Being opt-in, it is skipped by default, which means it can quietly stop being
  exercised — the same dormancy risk 0008 accepted for the manual trigger.

### Follow-up

- Decide whether the space needs periodic pruning of accumulated test assets.
- Consider asserting the *reverse* pairing once a v14 server exists, so the
  suite covers a client behind its server as well as ahead of it.

## Alternatives considered

**Reuse `MONDOO_CONFIG_PATH`.** Rejected: it is exactly the variable the suite
strips, and honouring it would mean any tier could go upstream whenever a
developer happened to have a credential configured — silently, since an
authenticated scan looks like an incognito one until you read the asset MRN.

**Assert only that the scan exits 0.** Rejected: a scan with no credential also
exits 0. Without the incognito assertion the tier would pass while exercising
nothing, which is the failure this suite was built to avoid.

**A floor on the number of checks.** Rejected: the policies come from the space,
so the count is the space's configuration rather than a property of cnspec. The
tier asserts that checks ran and reached verdicts instead.

**Leave the platform untested and verify by hand before each release.** Rejected
as the status quo that made this necessary: a manual check is not recorded,
cannot be pointed at a release candidate reproducibly, and had not been done.

## References

- [ADR-0008](0008-release-integration-testing.md) — the suite this extends, and
  the scope decision it revisits
- `test/integration/README.md` — how to run the tier and what it asserts
