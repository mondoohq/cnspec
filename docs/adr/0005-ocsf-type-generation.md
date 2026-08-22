# ADR-0005: OCSF Type Generation

**Date:** 2026-08-22
**Status:** Accepted

## Context

`cnspec scan -o ocsf-json` and `-o ocsf-parquet` emit Open Cybersecurity Schema Framework
events for security data lakes. Three event classes are produced: Compliance Finding (2003)
per check, Vulnerability Finding (2002) per advisory, and Device Inventory Info (5001) per
asset.

Something has to define those events as Go types. Three sources were available.

**A third-party OCSF type library.** Several exist and all generate the full schema:

| library | state |
|---|---|
| `telophasehq/go-ocsf` | MIT, generated per version (1.4.0, 1.5.0) with `json` + `parquet` tags |
| `hstern/go-ocsf` | Apache-2.0, generated for 1.8.0, includes a `Validate` method |
| `valllabh/ocsf-schema-golang` | proto-generated, last released 2024 (OCSF 1.0/1.1 era) |
| `grokify/ocsf-lake`, smithy, synqly | partial or vendor-specific type sets |

Full type sets do not survive the Parquet path, which is not a matter of taste:

- `parquet.SchemaOf(hstern/go-ocsf.ComplianceFinding{})` **does not terminate**. The OCSF
  object graph is recursive (a process has a parent process), and schema derivation recurses
  with it until the goroutine stack is exhausted.
- `parquet.SchemaOf(telophasehq/go-ocsf/v1_5_0.ComplianceFinding{})` completes, because that
  project breaks the cycles, but yields **7,708 leaf columns**; its vulnerability finding is
  3,595 and its inventory info 2,177. The subset cnspec populates is 24.
- Those tags only parse under `parquet-go` v0.25.0. On the current v0.32.0 they fail
  (`timestamp_millis` was replaced by `timestamp(millisecond)`), so adopting them pins the
  Parquet library backwards for the whole module.
- Version coverage does not line up either: the libraries offer 1.4/1.5 or 1.8, while cnspec
  emits 1.3.0 (the highest version Amazon Security Lake accepts for custom sources) and 1.9.0
  (current).

Telophase is the closest fit and is instructive for a different reason: its own Parquet writer
does not use its generated types. It writes from a hand-curated `arrow.Field` list per class.
The library that solved this problem before us solved it the same way.

**The OCSF project's own libraries.** `ocsf/ocsf-toolkit` (Go) loads a compiled schema and
validates, enriches and processes events as `map[string]any`; it has no types and no codegen,
and codegen is not on its roadmap. `ocsf/ocsf-lib-py` loads, compiles, validates, diffs and
checks compatibility, and explicitly does not generate domain classes. Neither the Go nor the
Python standard library fills this role.

**Hand-written structs.** The original implementation. Validating it against the official
compiled schema found five conformance defects that reading the code had not: `device` and
`cloud` used without declaring the profiles they belong to, a `cpu_architecture` attribute
that does not exist in 1.3, an attribute deprecated in 1.9, a vulnerability object that named
neither a CVE nor an advisory, and enum siblings that repeated the caption `"Other"` instead
of carrying cnspec's own value. Hand-written types drift from the schema, and the drift is
invisible until a lake rejects the events.

## Decision

Generate the types from the compiled OCSF schema, from a spec that selects what to emit, and
keep the generator internal to cnspec.

- `cli/reporter/ocsf/gen.yaml` lists the classes, attributes and objects cnspec emits.
- `cli/reporter/ocsf/schemas/schema-<version>.json.gz` holds the compiled schema per supported
  version, produced by the OCSF Schema Compiler and checked in.
- `cli/reporter/ocsf/internal/gen` reads both and emits `types.gen.go` and `enums.gen.go`:
  attribute names, Go types, optionality, enum constants, captions, doc comments, and a
  constructor per class that fills in the classification attributes.
- The same schemas are the input to `TestOcsfSchemaValidation`, which validates every emitted
  event with `ocsf/ocsf-toolkit`. Generation and verification share one source of truth.

Selection is the load-bearing part. "Compile the schema to Go and use all of it" is what the
existing libraries do and is exactly what does not work here; the spec is what keeps the
Parquet schema at 24 columns and the object graph acyclic.

The generator stays at `cli/reporter/ocsf/internal/gen`. It has one consumer, and a public
module would mean a published API, a release process and cross-repo version bumps for no
present benefit.

## Alternatives considered

- **Adopt a third-party type library.** Rejected on the measurements above: non-terminating or
  7,708-column Parquet schemas, a backwards pin on `parquet-go`, and no overlap with the OCSF
  versions cnspec targets.
- **Use `ocsf-toolkit` at runtime for enrichment** (enum siblings and observables) instead of
  generating. Measured against real output: 0 enum siblings added, because cnspec already sets
  them, and 3 observables of which two are documentation URLs. It would cost ~1 MB of embedded
  schema, 54–111 ms to parse at startup, a temp file (the loader takes a path, not a reader),
  and would not reach the Parquet path, which needs structs rather than maps.
- **Publish `go.mondoo.com/ocsf`, or upstream the generator into `ocsf-toolkit`.** Both remain
  open. The package boundary is deliberately clean — spec and schemas in, Go out — so either is
  mechanical once a second consumer exists. Upstreaming is the better of the two if the
  generator matures: the OCSF project would maintain it, and the ecosystem has no Go codegen
  path today.

## Consequences

- Adding or removing an attribute is an edit to `gen.yaml` plus
  `go generate ./cli/reporter/ocsf/...`; nobody writes an OCSF struct field by hand.
- Generation fails when a listed attribute exists in no supported version, and annotates the
  ones that exist in only some. That is the `cpu_architecture` defect caught at generate time
  rather than at validation time.
- `TestGeneratedTypesAreCurrent` re-runs the generator into a temporary directory and diffs it
  against the committed files, so the spec cannot drift from the generated output.
- Supporting a new OCSF version means adding its compiled schema, extending
  `ocsf.SupportedVersions` and `versions` in the generator, and regenerating. The validator
  then reports which attributes that version deprecated.
- The two compiled schemas add ~1 MB of gzipped fixtures to the repository. They are build- and
  test-time inputs only and are not embedded in the binary.
- cnspec carries `parquet-go` as a runtime dependency and `ocsf-toolkit` as a test-only one
  (no non-test package imports it, so it is never linked into the binary).

## References

- `cli/reporter/ocsf/doc.go` — package documentation and the generation contract
- `cli/reporter/ocsf/schemas/README.md` — how the compiled schemas are produced
- [OCSF Schema](https://github.com/ocsf/ocsf-schema) ·
  [Schema Compiler](https://pypi.org/project/ocsf-schema-compiler/) ·
  [ocsf-toolkit](https://github.com/ocsf/ocsf-toolkit) ·
  [ocsf-lib-py](https://github.com/ocsf/ocsf-lib-py)
- [Security Lake custom source requirements](https://docs.aws.amazon.com/security-lake/latest/userguide/custom-sources.html)
