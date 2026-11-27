# 0014. A reporter option on the findings upload helper

- **Status:** Proposed
- **Date:** 2026-11-27
- **Deciders:** @chris-rock
- **Consulted / Informed:** cnspec upload owners; owners of the producers that upload through `upload.UploadFindings`

## Context

cnspec builds every third-party findings upload in one place. `upload.Opts`
(`upload/findings.go:44-74`) carries `ConfigPath`, `ScopeMrn`, `HTTPClient`,
`CompleteKinds`, `Delta` and `ScannedComponents`. `findingsUploadRequest`
(`upload/findings.go:184-198`) copies them into an mql `fex.FindingsUploadRequest` and
sets `ImportStartedAt` unless `Delta` is set. `UploadFindings`
(`upload/findings.go:83-120`) marshals the request with `protojson` and sends it through
the signed-URL flow. Producers call the helper as a library. cnspec itself calls it from
`cnspec upload` (`apps/cnspec/cmd/upload.go:163`) and `cnspec vuln`
(`apps/cnspec/cmd/vuln.go:153`).

A full pull (with `ImportStartedAt`) lets the platform close, by absence, every finding
of the upload's source and kind on the assets the upload covers. `CompleteKinds` narrows
that by kind. Some producers run in two modes under one source. A code scanner, for
example, sends code findings in one run and data-flow findings in another. Such a
producer cannot say which mode a full pull was complete for, so one mode's full pull
closes the other mode's findings.

mql ADR 059 adds an optional `string reporter = 14` to `FindingsUploadRequest`. The
format is `<producer>:<mode>` (`^[a-z0-9-]+:[a-z0-9-]+$`, at most 64 bytes), and
`fex.ValidReporter(string) bool` checks it. With `ImportStartedAt` set, the platform
then closes by absence only the findings last reported under the same reporter. A
producer that uploads through cnspec can send the field only if `upload.Opts` carries
it. grep found no `Reporter` in `upload/` on `main` (`877c63d5`).

## Decision

1. **`Opts.Reporter string`.** `upload.Opts` gains one field, after `ScannedComponents`:

   ```go
   // Reporter names the producer mode this upload speaks for, as
   // "<producer>:<mode>" (for example "<tool>:code"). It is sent as
   // FindingsUploadRequest.reporter. With a full pull the platform closes by
   // absence only the findings last reported under the same reporter; a Delta
   // closes nothing whatever the reporter. Empty sends no reporter and keeps
   // today's behaviour. One reporter per upload; it is not free text.
   Reporter string
   ```

2. **Copied verbatim.** `findingsUploadRequest` sets `Reporter: opts.Reporter` on the
   request, and nothing else in the request changes. `CompleteKinds`, `Delta`,
   `ScannedComponents` and `ImportStartedAt` behave as today. The field is sent on both
   a full pull and a delta. The platform records a delta's reporter, and a delta still
   closes nothing.
3. **No value check in cnspec.** The platform decides which reporters a source accepts.
   It refuses an unknown or malformed reporter with `InvalidArgument`, and
   `UploadFindings` already returns that error to the caller. cnspec does not call
   `fex.ValidReporter` either. The helper stays a carrier, so a change of the format
   needs no cnspec release.
4. **No CLI flag.** `cnspec upload` and `cnspec vuln` send no reporter. Both upload in
   a single mode, and an empty reporter keeps their close behaviour unchanged.
5. **Dependency.** `go.mod` bumps `go.mondoo.com/mql` to the first release whose
   `fex.FindingsUploadRequest` has `Reporter` (field 14). The cnspec change merges only
   after that release, and it adds no `replace` directive.
6. **Test.** `upload/findings_test.go` gains `TestFindingsUploadRequestCarriesReporter`:

   | Case | Input | Expected |
   |---|---|---|
   | set | `Opts{Reporter: "tool:dspm"}`, one document | `req.Reporter == "tool:dspm"`; the `protojson.Marshal` output contains `"reporter":"tool:dspm"`; `ImportStartedAt` set |
   | unset | `Opts{}`, one document | `req.Reporter == ""`; the marshalled JSON has no `reporter` key, so the bytes equal today's for the same input |
   | delta | `Opts{Reporter: "tool:code", Delta: true}` | `req.Reporter == "tool:code"`; `ImportStartedAt` nil |
   | empty pull | `Opts{Reporter: "tool:code", ScannedComponents: [repo]}`, no documents | `req.Reporter == "tool:code"`; `ScannedComponents` carried as in `TestFindingsUploadRequestScannedComponents` |

   Each case catches a specific edit:
   - "set" fails if the copy in `findingsUploadRequest` is dropped.
   - "unset" fails if the field gets a default value.
   - "delta" fails if the reporter is tied to `ImportStartedAt`.

   The values are synthetic.

File-level changes:

| File | Change |
|---|---|
| `upload/findings.go` | `Opts.Reporter`; `findingsUploadRequest` copies it |
| `upload/findings_test.go` | `TestFindingsUploadRequestCarriesReporter` |
| `go.mod`, `go.sum` | `go.mondoo.com/mql` at the release with field 14 |

## Security implications

- **Threat model:** none new. The field is a short label inside a request the caller
  already writes and the platform already authorizes. A caller that sets another mode's
  reporter could close that mode's findings, and only in its own space. The same caller
  can already close every finding of its source today, with a full pull that has no
  reporter. The field therefore narrows what a full pull closes and never widens it.
- **Data handling:** a label of at most 64 bytes in the format above, with no
  credential, path or finding content. It is written to the same signed-URL object as
  the rest of the request and kept as long as the platform keeps that object.
- **Authentication and authorization:** unchanged. `UploadFindings` uses the same
  service account credentials and space scope as today. The platform checks the
  reporter against the source.
- **Supply chain:** nothing added. The mql bump is a new version of an existing
  dependency.
- **Residual risk:** the platform refuses a mistyped reporter at upload time, not
  cnspec locally. The producer sees it as a failed upload in its run.

## Performance implications

- **Expected impact:** none measurable. One string field of at most 64 bytes is copied
  once per upload.
- **Evidence:** an estimate from the request shape, not a measurement. An upload of
  10,000 findings is several megabytes of JSON, so 64 bytes is less than one part in
  10^4.
- **Scale assumptions:** this holds for any upload size, since the field does not grow
  with the number of findings.
- **Regression budget:** none consumed. There is no new request, retry or timeout.

## Consequences

### Positive

- A producer that uploads through cnspec can run two modes under one source. A full
  pull of one mode no longer closes the other mode's findings.
- A request without the field is byte-identical to today's, so existing callers and
  `cnspec upload` and `cnspec vuln` are unaffected.

### Negative

- Two releases are needed, in order: an mql release with the field, then a cnspec
  release, before a producer can set it.
- Only the platform catches a malformed reporter, after the upload round trip.
- A platform whose ingest predates the field does not act on it. A producer sets
  `Reporter` only once the platform accepts the field.

### Follow-up

- **Implementation, once an mql release carries field 14.** The change touches three
  places:
  - `go.mod` and `go.sum`: bump `go.mondoo.com/mql` to that release.
  - `upload/findings.go`: add `Reporter string` to `Opts` after `ScannedComponents`,
    with the comment in decision 1. Add `Reporter: opts.Reporter` to the literal in
    `findingsUploadRequest`.
  - `upload/findings_test.go`: add `TestFindingsUploadRequestCarriesReporter` with the
    four cases of decision 6, modelled on `TestFindingsUploadRequest` and
    `TestFindingsUploadRequestScannedComponents`.

  `UploadFindings`, `hasUploadContent` and the CLI commands do not change. Done when
  `go test ./upload/ -run TestFindingsUploadRequest` and `go build ./...` pass with no
  `replace` directive.
- Producers then set `Reporter` on each call. A code scanner, for example, sets one
  value on its code upload and its empty-scan upload, another on its data-flow mode's
  uploads, and none on a dependency VEX upload.
- A `--reporter` flag on `cnspec upload` is a separate, additive change, made only if a
  CLI user ever needs a multi-mode upload.

## Alternatives considered

### Option A - Check the format in cnspec with `fex.ValidReporter`

`UploadFindings` would return an error before the upload for a malformed value. Not
chosen, for two reasons:
- The platform decides the accepted values per source. A local check catches only
  format errors, and the platform must still refuse the rest.
- It would tie a cnspec release to any future change of the format.

A caller that wants to fail early can call `fex.ValidReporter` itself.

### Option B - Derive the reporter from `CompleteKinds` or the source

A full pull for `FINDING` alone could imply one mode. Not chosen, for two reasons:
- Both modes of a code scanner send FEX findings (`FINDING_KIND_FINDING`), so the kind
  does not tell them apart.
- A source name per mode would split one product's findings across two sources and
  break their deduplication on the platform.

### Option C - A `--reporter` flag on `cnspec upload` in the same change

Not chosen now. Neither cnspec command uploads more than one mode, and a flag is a
public interface that is hard to remove. It stays a follow-up for when a user needs it.

## References

- cnspec `upload/findings.go:44-74` (`Opts`), `:83-120` (`UploadFindings`), `:184-198`
  (`findingsUploadRequest`); `upload/findings_test.go:104` (`TestFindingsUploadRequest`),
  `:128` (`TestFindingsUploadRequestScannedComponents`);
  `apps/cnspec/cmd/upload.go:163`; `apps/cnspec/cmd/vuln.go:153` (commit `877c63d5`)
- mql ADR 059, a reporter on findings uploads (mondoohq/mql#12838), and its
  implementation (mondoohq/mql#12839)
- mql `providers-sdk/v1/upstream/fex/fex.proto` (`FindingsUploadRequest`)
