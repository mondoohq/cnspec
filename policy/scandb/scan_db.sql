-- Copyright (c) Mondoo, Inc.
-- SPDX-License-Identifier: BUSL-1.1

-- SQLite schema for cnspec policy upload files
-- Optimized for fast writes on client side
-- Version: 1.0

-- Metadata table for versioning and asset information
CREATE TABLE metadata (
    key TEXT PRIMARY KEY NOT NULL,
    value TEXT NOT NULL
);

-- Insert schema version and asset information
-- These will be populated by the client:
-- INSERT INTO metadata (key, value) VALUES ('schema_version', '1');
-- INSERT INTO metadata (key, value) VALUES ('asset_mrn', ?);
-- INSERT INTO metadata (key, value) VALUES ('upload_session_id', ?);
-- INSERT INTO metadata (key, value) VALUES ('created_at', ?); -- RFC3339 timestamp

-- Scores table - optimized for bulk inserts with upsert capability
-- qr_id is the primary key for uniqueness and upsert operations
CREATE TABLE scores (
    qr_id TEXT PRIMARY KEY,          -- Score.qr_id (unique identifier)
    risk_score INTEGER NOT NULL,      -- Score.risk_score
    type INTEGER NOT NULL,            -- Score.type
    value INTEGER NOT NULL,           -- Score.value
    weight INTEGER NOT NULL,          -- Score.weight  
    message TEXT,                     -- Score.message
    risk_factors BLOB,               -- protobuf encoded ScoredRiskFactors
    sources BLOB                     -- protobuf encoded Sources
);

-- Data table - stores LLX results with upsert capability
-- code_id is the primary key for uniqueness and upsert operations
CREATE TABLE data (
    code_id TEXT PRIMARY KEY,        -- llx.Result.CodeID (query checksum, unique identifier)
    data BLOB NOT NULL               -- protobuf encoded llx.Result
);

-- ScoredRiskFactor table - stores individual risk factors for scores
-- mrn is the primary key for uniqueness and upsert operations
CREATE TABLE scored_risk_factors (
    mrn TEXT PRIMARY KEY,            -- ScoredRiskFactor.mrn (unique identifier)
    risk REAL NOT NULL,              -- ScoredRiskFactor.risk (float32)
    is_toxic BOOLEAN NOT NULL,       -- ScoredRiskFactor.is_toxic
    is_detected BOOLEAN NOT NULL     -- ScoredRiskFactor.is_detected
);

CREATE TABLE resources (
    name TEXT NOT NULL,              -- The resource name
    id TEXT NOT NULL,                -- The resource ID
    data BLOB NOT NULL,              -- protobuf encoded llx.ResourceRecording
    PRIMARY KEY (name, id)           -- Composite primary key for uniqueness
);

-- Primary key indexes are automatically created for scores(qr_id) and data(code_id)
-- No additional indexes needed since we're using the primary keys for lookups

-- Pragma settings for maximum write performance
-- These should be set by the client before writing:
PRAGMA synchronous = OFF;          -- Disable fsync for speed (risk of corruption on crash)
PRAGMA journal_mode = MEMORY;      -- Keep journal in memory
PRAGMA temp_store = MEMORY;        -- Keep temp tables in memory
PRAGMA cache_size = 10000;         -- Large cache for better performance
PRAGMA locking_mode = EXCLUSIVE;   -- Exclusive locking for single writer
