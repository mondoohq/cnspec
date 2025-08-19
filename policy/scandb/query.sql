-- Copyright (c) Mondoo, Inc.
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
    qr_id, risk_score, type, value, weight, message, risk_factors, sources
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetScore :one
SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
FROM scores WHERE qr_id = ?;

-- name: StreamScores :many
SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
FROM scores ORDER BY qr_id;

-- Data operations
-- name: InsertData :exec
INSERT OR REPLACE INTO data (code_id, data) VALUES (?, ?);

-- name: GetData :one
SELECT data FROM data WHERE code_id = ?;

-- name: StreamData :many
SELECT code_id, data FROM data ORDER BY code_id;

-- Risk factor operations
-- name: InsertRiskFactor :exec
INSERT OR REPLACE INTO scored_risk_factors (mrn, risk, is_toxic, is_detected)
VALUES (?, ?, ?, ?);

-- name: GetRiskFactor :one
SELECT mrn, risk, is_toxic, is_detected
FROM scored_risk_factors WHERE mrn = ?;

-- name: StreamRiskFactors :many
SELECT mrn, risk, is_toxic, is_detected
FROM scored_risk_factors ORDER BY mrn;

-- Resource operations
-- name: InsertResource :exec
INSERT OR REPLACE INTO resources (name, id, data) VALUES (?, ?, ?);

-- name: GetResource :one
SELECT data FROM resources WHERE name = ? AND id = ?;

-- name: StreamResources :many
SELECT name, id, data FROM resources ORDER BY name, id;
