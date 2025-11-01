// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/v12/llx"
	"google.golang.org/protobuf/proto"
)

//go:embed scan_db.sql
var uploadSchema string

// UploadFileMetadata contains metadata from the upload file
type UploadFileMetadata struct {
	SchemaVersion   string `json:"schema_version"`
	AssetMrn        string `json:"asset_mrn"`
	UploadSessionId string `json:"upload_session_id"`
	CreatedAt       string `json:"created_at"`
}

type ScanDataStoreReader interface {
	StreamScores(ctx context.Context, callback func(*Score) error) error
	StreamData(ctx context.Context, callback func(string, *llx.Result) error) error
	StreamRisks(ctx context.Context, callback func(*ScoredRiskFactor) error) error

	// Reader methods for specific items
	GetScore(ctx context.Context, qrId string) (*Score, error)
	GetData(ctx context.Context, codeId string) (*llx.Result, error)
	GetRisk(ctx context.Context, mrn string) (*ScoredRiskFactor, error)
	Close() error
}

type ScanDataStoreWriter interface {
	WriteScores(ctx context.Context, scores []*Score) error
	WriteData(ctx context.Context, data []*llx.Result) error
	WriteRisk(ctx context.Context, risk *ScoredRiskFactor) error
	WriteResource(ctx context.Context, resource *llx.ResourceRecording) error
	Finalize() (string, error)
	Close() error
}

// ScanDataStore defines the interface for reading and writing scan data
type ScanDataStore interface {
	ScanDataStoreReader
	ScanDataStoreWriter
}

const SchemaVersion = "1.0"

// SqliteScanDataStore implements ScanDataStore using SQLite
type SqliteScanDataStore struct {
	db           *sql.DB
	scoresStmt   *sql.Stmt
	dataStmt     *sql.Stmt
	riskStmt     *sql.Stmt
	resourceStmt *sql.Stmt
	assetMrn     string
	filePath     string
	readOnly     bool
}

// NewSqliteScanDataStore creates a new SQLite-based scan data store for writing
func NewSqliteScanDataStore(filePath string, assetMrn string) (*SqliteScanDataStore, error) {
	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}

	store := &SqliteScanDataStore{
		db:       db,
		assetMrn: assetMrn,
		filePath: filePath,
		readOnly: false,
	}

	if err := store.initializeDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// NewSqliteScanDataStoreReader creates a new SQLite-based scan data store for reading
func NewSqliteScanDataStoreReader(filePath string) (*SqliteScanDataStore, error) {
	db, err := sql.Open("sqlite", filePath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}

	return &SqliteScanDataStore{
		db:       db,
		filePath: filePath,
		readOnly: true,
	}, nil
}

// initializeDatabase sets up the schema and metadata for write mode
func (s *SqliteScanDataStore) initializeDatabase() error {
	if s.readOnly {
		return fmt.Errorf("cannot initialize database in read-only mode")
	}

	// Execute the embedded schema
	if _, err := s.db.Exec(uploadSchema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert metadata
	if err := s.insertMetadata(); err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	// Prepare statements for bulk inserts
	var err error
	s.scoresStmt, err = s.db.Prepare(`
		INSERT OR REPLACE INTO scores (
			qr_id, risk_score, type, value, weight, message, risk_factors, sources
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare scores statement: %w", err)
	}

	s.dataStmt, err = s.db.Prepare(`
		INSERT OR REPLACE INTO data (code_id, data) VALUES (?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare data statement: %w", err)
	}

	s.riskStmt, err = s.db.Prepare(`
		INSERT OR REPLACE INTO scored_risk_factors (mrn, risk, is_toxic, is_detected) VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare risk statement: %w", err)
	}

	s.resourceStmt, err = s.db.Prepare(`
		INSERT OR REPLACE INTO resources (name, id, data) VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare resource statement: %w", err)
	}

	return nil
}

// insertMetadata adds metadata about the upload
func (s *SqliteScanDataStore) insertMetadata() error {
	metadata := map[string]string{
		"schema_version": SchemaVersion,
		"asset_mrn":      s.assetMrn,
		"created_at":     time.Now().Format(time.RFC3339),
	}

	stmt, err := s.db.Prepare("INSERT INTO metadata (key, value) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare metadata statement: %w", err)
	}
	defer stmt.Close()

	for key, value := range metadata {
		if _, err := stmt.Exec(key, value); err != nil {
			return fmt.Errorf("failed to insert metadata %s: %w", key, err)
		}
	}

	return nil
}

// WriteScores writes multiple scores efficiently
func (s *SqliteScanDataStore) WriteScores(ctx context.Context, scores []*Score) error {
	if s.readOnly {
		return fmt.Errorf("cannot write scores in read-only mode")
	}

	for _, score := range scores {
		if err := s.writeScore(s.scoresStmt, score); err != nil {
			return fmt.Errorf("failed to write score %s: %w", score.QrId, err)
		}
	}
	return nil
}

// WriteData writes multiple data results
func (s *SqliteScanDataStore) WriteData(ctx context.Context, data []*llx.Result) error {
	if s.readOnly {
		return fmt.Errorf("cannot write data in read-only mode")
	}

	for _, result := range data {
		codeId := result.CodeId
		resultData, err := proto.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result %s: %w", codeId, err)
		}

		if _, err := s.dataStmt.Exec(codeId, resultData); err != nil {
			return fmt.Errorf("failed to write data %s: %w", codeId, err)
		}
	}

	return nil
}

func (s *SqliteScanDataStore) WriteResource(ctx context.Context, resource *llx.ResourceRecording) error {
	if s.readOnly {
		return fmt.Errorf("cannot write resource in read-only mode")
	}

	resourceData, err := proto.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource %s/%s: %w", resource.Resource, resource.Id, err)
	}

	if _, err := s.resourceStmt.Exec(resource.Resource, resource.Id, resourceData); err != nil {
		return fmt.Errorf("failed to write resource %s/%s: %w", resource.Resource, resource.Id, err)
	}
	return nil
}

// WriteRisk writes a single risk factor
func (s *SqliteScanDataStore) WriteRisk(ctx context.Context, risk *ScoredRiskFactor) error {
	if s.readOnly {
		return fmt.Errorf("cannot write risk in read-only mode")
	}

	_, err := s.riskStmt.Exec(risk.Mrn, risk.Risk, risk.IsToxic, risk.IsDetected)
	if err != nil {
		return fmt.Errorf("failed to write risk %s: %w", risk.Mrn, err)
	}

	return nil
}

// writeScore writes a single score using the provided statement
func (s *SqliteScanDataStore) writeScore(stmt *sql.Stmt, score *Score) error {
	var riskFactors, sources []byte
	var err error

	if score.RiskFactors != nil {
		riskFactors, err = proto.Marshal(score.RiskFactors)
		if err != nil {
			return fmt.Errorf("failed to marshal risk factors: %w", err)
		}
	}

	if score.Sources != nil {
		sources, err = proto.Marshal(score.Sources)
		if err != nil {
			return fmt.Errorf("failed to marshal sources: %w", err)
		}
	}

	_, err = stmt.Exec(
		score.QrId,
		score.RiskScore,
		score.Type,
		score.Value,
		score.Weight,
		score.Message,
		riskFactors,
		sources,
	)

	return err
}

// GetMetadata retrieves and parses metadata from the upload file
func (s *SqliteScanDataStore) GetMetadata() (*UploadFileMetadata, error) {
	rows, err := s.db.Query("SELECT key, value FROM metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}
	defer rows.Close()

	rawMetadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		rawMetadata[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metadata rows: %w", err)
	}

	// Parse into structured metadata
	metadata := &UploadFileMetadata{
		SchemaVersion:   rawMetadata["schema_version"],
		AssetMrn:        rawMetadata["asset_mrn"],
		UploadSessionId: rawMetadata["upload_session_id"],
		CreatedAt:       rawMetadata["created_at"],
	}

	return metadata, nil
}

// StreamScores reads all scores with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamScores(ctx context.Context, callback func(*Score) error) error {
	rows, err := s.db.Query(`
		SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
		FROM scores ORDER BY qr_id
	`)
	if err != nil {
		return fmt.Errorf("failed to query scores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		score, err := s.scanScore(rows)
		if err != nil {
			return fmt.Errorf("failed to scan score: %w", err)
		}

		if err := callback(score); err != nil {
			return fmt.Errorf("callback error for score %s: %w", score.QrId, err)
		}
	}

	return rows.Err()
}

// StreamData reads all data with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamData(ctx context.Context, callback func(string, *llx.Result) error) error {
	rows, err := s.db.Query("SELECT code_id, data FROM data ORDER BY code_id")
	if err != nil {
		return fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var codeId string
		var data []byte

		if err := rows.Scan(&codeId, &data); err != nil {
			return fmt.Errorf("failed to scan data row: %w", err)
		}

		result := &llx.Result{}
		if err := proto.Unmarshal(data, result); err != nil {
			return fmt.Errorf("failed to unmarshal result %s: %w", codeId, err)
		}

		if err := callback(codeId, result); err != nil {
			return fmt.Errorf("callback error for data %s: %w", codeId, err)
		}
	}

	return rows.Err()
}

// GetScore retrieves a specific score by QR ID
func (s *SqliteScanDataStore) GetScore(ctx context.Context, qrId string) (*Score, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT qr_id, risk_score, type, value, weight, message, risk_factors, sources
		FROM scores WHERE qr_id = ?
	`, qrId)

	score, err := s.scanScore(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("score not found: %s", qrId)
		}
		return nil, fmt.Errorf("failed to scan score: %w", err)
	}

	return score, nil
}

// GetData retrieves a specific data result by code ID
func (s *SqliteScanDataStore) GetData(ctx context.Context, codeId string) (*llx.Result, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT data FROM data WHERE code_id = ?
	`, codeId).Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data not found: %s", codeId)
		}
		return nil, fmt.Errorf("failed to scan data: %w", err)
	}

	result := &llx.Result{}
	if err := proto.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result %s: %w", codeId, err)
	}

	return result, nil
}

// GetRisk retrieves a specific risk factor by MRN
func (s *SqliteScanDataStore) GetRisk(ctx context.Context, mrn string) (*ScoredRiskFactor, error) {
	var risk ScoredRiskFactor
	err := s.db.QueryRowContext(ctx, `
		SELECT mrn, risk, is_toxic, is_detected
		FROM scored_risk_factors WHERE mrn = ?
	`, mrn).Scan(&risk.Mrn, &risk.Risk, &risk.IsToxic, &risk.IsDetected)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRiskNotFound
		}
		return nil, fmt.Errorf("failed to scan risk: %w", err)
	}

	return &risk, nil
}

// StreamRisk reads all risk factors with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamRisks(ctx context.Context, callback func(*ScoredRiskFactor) error) error {
	rows, err := s.db.Query(`
		SELECT mrn, risk, is_toxic, is_detected
		FROM scored_risk_factors ORDER BY mrn
	`)
	if err != nil {
		return fmt.Errorf("failed to query risk factors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var risk ScoredRiskFactor
		if err := rows.Scan(&risk.Mrn, &risk.Risk, &risk.IsToxic, &risk.IsDetected); err != nil {
			return fmt.Errorf("failed to scan risk: %w", err)
		}

		if err := callback(&risk); err != nil {
			return fmt.Errorf("callback error for risk %s: %w", risk.Mrn, err)
		}
	}

	return rows.Err()
}

// scanScore scans a score row into a Score struct
func (s *SqliteScanDataStore) scanScore(row interface{ Scan(dest ...any) error }) (*Score, error) {
	score := &Score{
		ScoreCompletion: 100,
		DataCompletion:  100,
	}
	var riskFactors, sources []byte

	err := row.Scan(
		&score.QrId,
		&score.RiskScore,
		&score.Type,
		&score.Value,
		&score.Weight,
		&score.Message,
		&riskFactors,
		&sources,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal protobuf fields
	if len(riskFactors) > 0 {
		score.RiskFactors = &ScoredRiskFactors{}
		if err := proto.Unmarshal(riskFactors, score.RiskFactors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal risk factors: %w", err)
		}
	}

	if len(sources) > 0 {
		score.Sources = &Sources{}
		if err := proto.Unmarshal(sources, score.Sources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sources: %w", err)
		}
	}

	return score, nil
}

// ValidateFile performs basic validation on the upload file
func (s *SqliteScanDataStore) ValidateFile() error {
	metadata, err := s.GetMetadata()
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Check schema version
	if metadata.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version: %s (expected %s)",
			metadata.SchemaVersion, SchemaVersion)
	}

	// Check required fields
	if metadata.AssetMrn == "" {
		return fmt.Errorf("missing asset_mrn in metadata")
	}

	// Validate table structure exists
	tables := []string{"metadata", "scores", "data", "scored_risk_factors", "resources"}
	for _, table := range tables {
		var count int
		err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if count == 0 {
			return fmt.Errorf("missing required table: %s", table)
		}
	}

	return nil
}

// Finalize optimizes the database and converts to read-only mode
// Returns the database file path. No-op if already in read-only mode.
func (s *SqliteScanDataStore) Finalize() (string, error) {
	if s.readOnly {
		return s.filePath, nil
	}

	// Close prepared statements
	if s.scoresStmt != nil {
		s.scoresStmt.Close()
		s.scoresStmt = nil
	}
	if s.dataStmt != nil {
		s.dataStmt.Close()
		s.dataStmt = nil
	}
	if s.riskStmt != nil {
		s.riskStmt.Close()
		s.riskStmt = nil
	}

	// Optimize the database
	if _, err := s.db.Exec("VACUUM"); err != nil {
		return "", fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Close the current connection
	if err := s.db.Close(); err != nil {
		return "", fmt.Errorf("failed to close write connection: %w", err)
	}

	// Reopen as read-only
	db, err := sql.Open("sqlite", s.filePath+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("failed to reopen database as read-only: %w", err)
	}

	// Update the store to read-only mode
	s.db = db
	s.readOnly = true

	return s.filePath, nil
}

// Close closes the database connections without optimization
func (s *SqliteScanDataStore) Close() error {
	if s.scoresStmt != nil {
		s.scoresStmt.Close()
	}
	if s.dataStmt != nil {
		s.dataStmt.Close()
	}
	if s.riskStmt != nil {
		s.riskStmt.Close()
	}

	return s.db.Close()
}
