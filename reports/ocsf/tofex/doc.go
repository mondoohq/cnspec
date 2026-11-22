// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package tofex turns Open Cybersecurity Schema Framework findings into the
// Mondoo finding format, mql.fex.v1 (go.mondoo.com/mql/providers-sdk/v1/upstream/fex).
//
// It is the reverse of reports/ocsf/convert. convert knows cnspec and writes
// OCSF; tofex knows OCSF and fex and nothing else, so any producer that speaks
// OCSF, a runtime sensor or an importer reading a security data lake, maps onto
// fex the same way:
//
//	Detection Finding (2004)  one FindingExchange in CATEGORY_THREAT, with an
//	                          evidence per artifact of every evidences entry
//
// # Mapping
//
// Identity: finding_info.uid is the finding's id. The ref groups findings of the
// same detection: finding_info.analytic.uid, or the title when the finding names
// no analytic.
//
// Status follows the open/closed split the platform scores on, not the OCSF
// workflow: New and In Progress are AFFECTED; Resolved, Archived and Deleted are
// FIXED; Suppressed is NOT_AFFECTED. A dismissed finding has to close, and
// FALSE_POSITIVE and WONT_FIX would keep it open.
//
// Evidence: fex carries one artifact per evidence, OCSF a set of them per entry.
// Each artifact of an entry becomes an evidence of its own, in a fixed order
// (actor process, actor user, process, file, user, connection, query,
// container), and each carries the ATT&CK tactic and technique of the first
// attack and the finding's confidence. The source and destination endpoints and
// the connection info of an entry are one connection.
//
// What fex has no field for travels in the properties maps under keys prefixed
// with "ocsf/", named by the OCSF path: "ocsf/process.uid",
// "ocsf/connection_info.direction_id". Every evidence names the entry and the
// artifact it came from in "ocsf/source", e.g. "evidences.0.actor.process". The
// full ATT&CK lists are finding properties under "attack.mitre.org/tactics" and
// "attack.mitre.org/techniques".
//
// # Scope
//
// The package imports reports/ocsf and no other cnspec package, so it can move
// with the OCSF types if they are published as a module of their own
// (docs/adr/0005-ocsf-type-generation.md).
package tofex
