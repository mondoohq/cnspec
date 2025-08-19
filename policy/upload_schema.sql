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

-- Scores table - optimized for bulk inserts
-- No indexes on write-heavy columns for maximum insert speed
CREATE TABLE scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    qr_id TEXT NOT NULL,             -- Score.qr_id
    risk_score INTEGER NOT NULL,      -- Score.risk_score
    type INTEGER NOT NULL,            -- Score.type
    value INTEGER NOT NULL,           -- Score.value
    weight INTEGER NOT NULL,          -- Score.weight  
    message TEXT,                     -- Score.message
    risk_factors BLOB,               -- protobuf encoded ScoredRiskFactors
    sources BLOB                     -- protobuf encoded Sources
);

-- Data table - stores LLX results
-- Optimized for bulk inserts with minimal indexing
CREATE TABLE data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code_id TEXT NOT NULL,           -- llx.Result.CodeID (query checksum)
    data BLOB NOT NULL               -- protobuf encoded llx.Result
);

-- Create indexes for efficient lookups
-- These are created when the file is finalized
CREATE INDEX IF NOT EXISTS idx_scores_qr_id ON scores(qr_id);
CREATE INDEX IF NOT EXISTS idx_data_code_id ON data(code_id);

-- Pragma settings for maximum write performance
-- These should be set by the client before writing:
PRAGMA synchronous = OFF;          -- Disable fsync for speed (risk of corruption on crash)
PRAGMA journal_mode = MEMORY;      -- Keep journal in memory
PRAGMA temp_store = MEMORY;        -- Keep temp tables in memory
PRAGMA cache_size = 10000;         -- Large cache for better performance
PRAGMA locking_mode = EXCLUSIVE;   -- Exclusive locking for single writer