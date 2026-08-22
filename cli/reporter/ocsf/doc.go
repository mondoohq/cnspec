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
//	go generate ./cli/reporter/ocsf/...
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
// # Conformance
//
// Correctness is proven against the schema rather than asserted: every event of
// every supported version is validated with the OCSF project's own validator in
// cli/reporter/ocsf_validate_test.go.
package ocsf

//go:generate go run ./internal/gen
