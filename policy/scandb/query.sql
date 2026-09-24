-- Copyright Mondoo, Inc. 2024, 2026
-- SPDX-License-Identifier: BUSL-1.1

-- Metadata operations
-- name: InsertMetadata :exec
INSERT INTO metadata (key, value) VALUES (?, ?);

-- name: GetMetadata :many
SELECT key, value FROM metadata;

-- name: GetMetadataByKey :one
SELECT value FROM metadata WHERE key = ?;

-- Scores operations
-- name: InsertScore :exec
INSERT OR REPLACE INTO scores (
    qr_id, risk_score, type, value, weight, message, risk_factors, sources, checksum
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetScore :one
SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
FROM scores WHERE qr_id = ?;

-- name: StreamScores :many
SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
FROM scores ORDER BY qr_id;

-- Data operations
-- name: InsertData :exec
INSERT OR REPLACE INTO data (code_id, data, checksum) VALUES (?, ?, ?);

-- name: GetData :one
SELECT data FROM data WHERE code_id = ?;

-- name: StreamData :many
SELECT code_id, data FROM data ORDER BY code_id;

-- Risk factor operations
-- name: InsertRiskFactor :exec
INSERT OR REPLACE INTO scored_risk_factors (mrn, risk, is_toxic, is_detected, checksum)
VALUES (?, ?, ?, ?, ?);

-- name: GetRiskFactor :one
SELECT mrn, risk, is_toxic, is_detected
FROM scored_risk_factors WHERE mrn = ?;

-- name: StreamRiskFactors :many
SELECT mrn, risk, is_toxic, is_detected
FROM scored_risk_factors ORDER BY mrn;

-- Resource operations
-- name: InsertResource :exec
INSERT OR REPLACE INTO resources (name, id, data, checksum) VALUES (?, ?, ?, ?);

-- name: GetResource :one
SELECT data FROM resources WHERE name = ? AND id = ?;

-- name: StreamResources :many
SELECT name, id, data FROM resources ORDER BY name, id;

-- Asset operations (optional table, announced by its presence;
-- schema_version stays 1.0 - see SchemaVersion in scan_data_store.go)
-- name: InsertAsset :exec
INSERT OR REPLACE INTO asset (id, data) VALUES (0, ?);

-- name: GetAsset :one
SELECT data FROM asset WHERE id = 0;

-- Asset filter code_id operations (optional table, announced by its
-- presence; schema_version stays 1.0 - see SchemaVersion)
-- name: InsertAssetFilter :exec
INSERT OR REPLACE INTO asset_filters (code_id) VALUES (?);

-- name: StreamAssetFilters :many
SELECT code_id FROM asset_filters ORDER BY code_id;
