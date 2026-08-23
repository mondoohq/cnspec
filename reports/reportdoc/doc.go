// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package reportdoc reads the facts a report is written from out of a scan: what a
// check documents about itself (description, MQL, audit steps, remediation,
// references, compliance mappings), how severe its result was, which packages an
// advisory affects, and what to call the asset it all ran against.
//
// It exists because every output format needs the same answers and has to give the
// same ones. When the JUnit body, the SARIF rule help and the OHDF control each
// derived remediation on their own, they disagreed on which variant applied to the
// platform being scanned - a Terraform scan showed Kubernetes remediation in one
// format and not in another.
//
// Nothing here knows about an output format. That is what lets every format
// package depend on it without depending on each other: the OHDF and OCSF
// converters live in packages of their own precisely because they need these
// helpers, and a shared package underneath all of them is what keeps that from
// being an import cycle. It sits under reports/ rather than under cli/ for the
// same reason they do - none of it is terminal-facing, and reports/reportdoc is
// reachable from both reports/... and cli/reporter, which an internal package
// under either would not be.
package reportdoc
