// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package hdf converts a cnspec scan into the OASIS Heimdall Data Format
// (OHDF, formerly HDF).
//
// OHDF is the normalized security result schema of the MITRE Security Automation
// Framework, and the same schema InSpec emits as `exec-json`, so everything that
// reads InSpec results - Heimdall, the SAF CLI, the SAF threshold gates - reads a
// cnspec scan unchanged.
//
// A document describes exactly one asset, because that is all its consumers can
// read: Heimdall and the SAF CLI resolve a document down to a single root profile
// and tally only that one, so findings parked in a second profile are dropped
// without a word. A scan of several assets is written as a file each.
//
//	platform      the asset the document covers
//	profiles[0]   its only profile: controls = the asset's checks and advisories,
//	              groups = the policies they came from
//	control       a check: id, title, desc, impact, refs, tags, MQL, results
//	result        the outcome of that check on that asset
//	passthrough   asset metadata and scores that OHDF itself has no place for
//
// The check documentation (description, audit steps, remediation, references,
// compliance mappings) comes from the same helpers the JUnit and SARIF reporters
// use, so all three surface identical content.
//
// It is a package of its own, under reports/ rather than cli/, so that the format
// and the CLI plumbing can move independently: cli/reporter decides that `-o hdf`
// was asked for and where the bytes go - fifteen lines of it - and everything
// about what those bytes say lives here. Nothing in here is terminal-facing. The
// dependency runs one way - cli/reporter imports hdf, never the reverse - which is
// why the check documentation both of them read comes from reports/reportdoc
// rather than from cli/reporter itself.
package hdf
