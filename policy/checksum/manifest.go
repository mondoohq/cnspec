// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package checksum

// The manifest is the comparison artifact of the unchanged-scan
// short-circuit: a trimmed-down scandb — the same four tables, keyed
// identically, every payload column dropped — holding nothing but each row's
// key and its 8-byte checksum, plus a metadata table naming the producing
// algorithm and asset. The server stages one next to every processed scan;
// in client_compare the client pulls it back and diffs its own rows against
// it before deciding to upload. The schema is declared here, with the
// client, because both sides read and write the same file format — the
// server builds manifests today, the client reads them tomorrow.
//
// WITHOUT ROWID: the tables are pure key → checksum maps, so the declared
// primary key IS the clustering key — one B-tree per table, no hidden rowid,
// no shadow index. Smaller files, and single-descent lookups for the
// key-addressed diff queries (EXCEPT / JOIN USING(key)).
const ManifestSchema = `
CREATE TABLE data                (code_id TEXT PRIMARY KEY, checksum INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE scores              (qr_id   TEXT PRIMARY KEY, checksum INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE scored_risk_factors (mrn     TEXT PRIMARY KEY, checksum INTEGER NOT NULL) WITHOUT ROWID;
CREATE TABLE resources           (name TEXT NOT NULL, id TEXT NOT NULL, checksum INTEGER NOT NULL,
                                  PRIMARY KEY (name, id)) WITHOUT ROWID;
CREATE TABLE metadata            (key TEXT PRIMARY KEY, value TEXT NOT NULL);
`

// Manifest metadata keys.
const (
	// ManifestMetaAlgoVersion reports the AlgoVersion that produced every
	// checksum in the file. Checksums across versions never compare: a
	// reader whose own algorithm differs treats the manifest as absent.
	ManifestMetaAlgoVersion = "checksum_algo_version"
	// ManifestMetaAssetMrn names the asset the manifest describes.
	ManifestMetaAssetMrn = "asset_mrn"
)
