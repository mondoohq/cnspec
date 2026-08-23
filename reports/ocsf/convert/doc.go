// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package convert turns a cnspec scan into Open Cybersecurity Schema Framework
// events, so a scan can land in Amazon Security Lake or a SIEM without a
// converter in between.
//
// It carries the same content as the SARIF report, re-cut along the event classes
// a security data lake indexes on:
//
//	Compliance Finding (2003)     one per check per asset, plus one per asset
//	                              that failed to scan at all
//	Detection Finding (2004)      the same checks in the class Splunk Enterprise
//	                              Security models on, selected by Options.Findings
//	Vulnerability Finding (2002)  one per advisory per asset, plus one per
//	                              affected package no advisory accounts for
//	Device Inventory Info (5001)  one per asset, carrying the platform details
//	                              and (with Options.IncludeData) the results of
//	                              data-only queries
//
// Every event is self-describing: class_uid says what it is, metadata.version
// says which OCSF version it follows.
//
// Convert writes every class into one newline-delimited stream, which is what a
// SIEM ingesting a single file expects. ConvertToDir writes a file per class,
// which is what Amazon Security Lake requires of a custom source and what lets a
// Glue crawler derive one schema per table. Both stream: the converter produces
// one asset at a time and the writer encodes and drops each set, so a fleet scan
// does not have to fit in memory.
//
// # Why this is not the parent package
//
// reports/ocsf is the schema: the generated event types and the two writers for
// them, and it imports no cnspec package anywhere. That is deliberate, and
// docs/adr/0005-ocsf-type-generation.md keeps publishing it as go.mondoo.com/ocsf
// open. Putting the mapping there would put policy.Score, policy.ReportCollection
// and cnspec's remediation rules inside a package whose whole value is that it
// depends on nothing of ours — which is the one change that would make it
// unpublishable.
//
// So the split is by direction of knowledge rather than by subject: the parent
// knows OCSF and nothing about cnspec, this package knows both and is what
// translates. The dependency runs one way, and cli/reporter sits above both,
// deciding that `-o ocsf-json` was asked for and where the bytes go — about
// fifteen lines of it.
//
// The check documentation this package reads (description, MQL, remediation,
// references, compliance mappings) comes from reports/reportdoc for the same
// reason it does in reports/hdf: it is shared with the other formats, and a
// shared package underneath all of them is what keeps that from being an import
// cycle.
package convert
