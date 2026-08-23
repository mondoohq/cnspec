// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ocsf holds the subset of the Open Cybersecurity Schema Framework that
// cnspec emits, plus the two writers for it: newline-delimited JSON and Apache
// Parquet.
//
// # Generated types
//
// The event types in types.gen.go and the identifiers in enums.gen.go are
// generated from the compiled OCSF schemas in schemas/ and the attribute list in
// gen.yaml. Attribute names, Go types, optionality, enum values, captions and
// documentation all come from the schema, so none of them can drift from it. To
// add or remove an attribute, edit gen.yaml and re-run:
//
//	go generate ./reports/ocsf/...
//
// The generated set is deliberately a subset. Generating all of OCSF does not
// work here: the object graph is recursive, so deriving a Parquet schema from a
// full type set does not terminate, and a complete compliance_finding is over
// 7,000 Parquet columns against 24 for the subset cnspec populates.
//
// # Shared by both writers
//
// Every field carries a json and a parquet tag. Two rules keep one struct
// serving both formats:
//
//   - No any/interface{} fields. The Parquet schema is derived from the Go types
//     by reflection and cannot type an empty interface, so free-form values (MQL
//     results, assessments) are JSON-encoded into strings or into the unmapped
//     map.
//   - Optional columns are written as null when the Go value is its zero value,
//     which lines up with omitempty on the JSON side.
//
// The types cover the union of the supported OCSF versions (see version.go).
// Fields that only exist in a newer version are left empty when an older version
// is selected; in Parquet they show up as an all-null column, which Glue, Athena
// and every other reader handle.
//
// # Scope: what this package deliberately does not contain
//
// Nothing about cnspec. This package imports no cnspec package at all, and that
// is a property to preserve rather than an accident of how it grew. It is what
// keeps publishing it as go.mondoo.com/ocsf, or upstreaming the generator into
// the OCSF project, a mechanical move; ADR-0005 keeps both open, and one
// policy.Score in here would close them.
//
// So the translation from a cnspec scan lives one directory down, in
// reports/ocsf/convert: which class a check becomes, how a score becomes a
// severity and a status, what travels in unmapped. Read the split as a direction
// of knowledge - this package knows OCSF and nothing about cnspec, convert knows
// both. Anything that has to look at a policy.Score, a policy.ReportCollection or
// an inventory.Asset belongs there, however OCSF-shaped its output is.
//
// The one thing that does live here and is not a struct field is the vocabulary
// a caller selects from: ParseVersion and ParseFindingClasses. Which OCSF
// versions and which event classes exist is a fact about OCSF, and closing both
// sets here is what stops a value cnspec cannot actually emit reaching
// metadata.version or a class router.
//
// The generator is internal for the same practical reason the package is not yet
// published: it has one consumer, so a public API and a release process would buy
// nothing today. Its boundary is kept clean (spec and schemas in, Go out) so that
// changes if a second one appears. See docs/adr/0005-ocsf-type-generation.md.
//
// # Conformance
//
// Correctness is proven against the schema rather than asserted: every event of
// every supported version is validated with the OCSF project's own validator in
// reports/ocsf/convert/validate_test.go.
package ocsf

//go:generate go run ./internal/gen
