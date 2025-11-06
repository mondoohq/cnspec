// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:generate go tool sqlc generate ./

package scandb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnspec/v12/policy"
	"go.mondoo.com/cnspec/v12/policy/scandb/sqlc"
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
	StreamScores(ctx context.Context, callback func(*policy.Score) error) error
	StreamData(ctx context.Context, callback func(string, *llx.Result) error) error
	StreamRisks(ctx context.Context, callback func(*policy.ScoredRiskFactor) error) error

	// Reader methods for specific items
	GetScore(ctx context.Context, qrId string) (*policy.Score, error)
	GetData(ctx context.Context, codeId string) (*llx.Result, error)
	GetRisk(ctx context.Context, mrn string) (*policy.ScoredRiskFactor, error)
	Close() error
}

type ScanDataStoreWriter interface {
	WriteScores(ctx context.Context, scores []*policy.Score) error
	WriteData(ctx context.Context, data []*llx.Result) error
	WriteRisk(ctx context.Context, risk *policy.ScoredRiskFactor) error
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

// SqliteScanDataStore implements ScanDataStore using SQLite with sqlc-generated queries
type SqliteScanDataStore struct {
	sqlDB    *sql.DB
	queries  *sqlc.Queries
	assetMrn string
	filePath string
	readOnly bool
}

// NewSqliteScanDataStore creates a new SQLite-based scan data store for writing
func NewSqliteScanDataStore(filePath string, assetMrn string) (*SqliteScanDataStore, error) {
	sqlDB, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}

	queries, err := sqlc.Prepare(context.Background(), sqlDB)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to prepare queries: %w", err)
	}

	store := &SqliteScanDataStore{
		sqlDB:    sqlDB,
		queries:  queries,
		assetMrn: assetMrn,
		filePath: filePath,
		readOnly: false,
	}

	if err := store.initializeDatabase(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// NewSqliteScanDataStoreReader creates a new SQLite-based scan data store for reading
func NewSqliteScanDataStoreReader(filePath string) (*SqliteScanDataStore, error) {
	sqlDB, err := sql.Open("sqlite", filePath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}

	queries := sqlc.New(sqlDB)

	return &SqliteScanDataStore{
		sqlDB:    sqlDB,
		queries:  queries,
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
	if _, err := s.sqlDB.Exec(uploadSchema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert metadata
	if err := s.insertMetadata(); err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}

// insertMetadata adds metadata about the upload
func (s *SqliteScanDataStore) insertMetadata() error {
	ctx := context.Background()
	metadata := map[string]string{
		"schema_version": SchemaVersion,
		"asset_mrn":      s.assetMrn,
		"created_at":     time.Now().Format(time.RFC3339),
	}

	for key, value := range metadata {
		if err := s.queries.InsertMetadata(ctx, sqlc.InsertMetadataParams{
			Key:   key,
			Value: value,
		}); err != nil {
			return fmt.Errorf("failed to insert metadata %s: %w", key, err)
		}
	}

	return nil
}

// WriteScores writes multiple scores efficiently
func (s *SqliteScanDataStore) WriteScores(ctx context.Context, scores []*policy.Score) error {
	if s.readOnly {
		return fmt.Errorf("cannot write scores in read-only mode")
	}

	for _, score := range scores {
		if err := s.writeScore(ctx, score); err != nil {
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

		if err := s.queries.InsertData(ctx, sqlc.InsertDataParams{
			CodeID: codeId,
			Data:   resultData,
		}); err != nil {
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

	if err := s.queries.InsertResource(ctx, sqlc.InsertResourceParams{
		Name: resource.Resource,
		ID:   resource.Id,
		Data: resourceData,
	}); err != nil {
		return fmt.Errorf("failed to write resource %s/%s: %w", resource.Resource, resource.Id, err)
	}
	return nil
}

// WriteRisk writes a single risk factor
func (s *SqliteScanDataStore) WriteRisk(ctx context.Context, risk *policy.ScoredRiskFactor) error {
	if s.readOnly {
		return fmt.Errorf("cannot write risk in read-only mode")
	}

	if err := s.queries.InsertRiskFactor(ctx, sqlc.InsertRiskFactorParams{
		Mrn:        risk.Mrn,
		Risk:       float64(risk.Risk),
		IsToxic:    risk.IsToxic,
		IsDetected: risk.IsDetected,
	}); err != nil {
		return fmt.Errorf("failed to write risk %s: %w", risk.Mrn, err)
	}

	return nil
}

// writeScore writes a single score
func (s *SqliteScanDataStore) writeScore(ctx context.Context, score *policy.Score) error {
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

	message := sql.NullString{
		String: score.Message,
		Valid:  score.Message != "",
	}

	return s.queries.InsertScore(ctx, sqlc.InsertScoreParams{
		QrID:        score.QrId,
		RiskScore:   int64(score.RiskScore),
		Type:        int64(score.Type),
		Value:       int64(score.Value),
		Weight:      int64(score.Weight),
		Message:     message,
		RiskFactors: riskFactors,
		Sources:     sources,
	})
}

// GetMetadata retrieves and parses metadata from the upload file
func (s *SqliteScanDataStore) GetMetadata() (*UploadFileMetadata, error) {
	metadataRows, err := s.queries.GetMetadata(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}

	rawMetadata := make(map[string]string)
	for _, row := range metadataRows {
		rawMetadata[row.Key] = row.Value
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
func (s *SqliteScanDataStore) StreamScores(ctx context.Context, callback func(*policy.Score) error) error {
	scores, err := s.queries.StreamScores(ctx)
	if err != nil {
		return fmt.Errorf("failed to query scores: %w", err)
	}

	for _, scoreRow := range scores {
		score, err := s.convertScore(&scoreRow)
		if err != nil {
			return fmt.Errorf("failed to convert score: %w", err)
		}

		if err := callback(score); err != nil {
			return fmt.Errorf("callback error for score %s: %w", score.QrId, err)
		}
	}

	return nil
}

// StreamData reads all data with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamData(ctx context.Context, callback func(string, *llx.Result) error) error {
	dataRows, err := s.queries.StreamData(ctx)
	if err != nil {
		return fmt.Errorf("failed to query data: %w", err)
	}

	for _, row := range dataRows {
		result := &llx.Result{}
		if err := proto.Unmarshal(row.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal result %s: %w", row.CodeID, err)
		}

		if err := callback(row.CodeID, result); err != nil {
			return fmt.Errorf("callback error for data %s: %w", row.CodeID, err)
		}
	}

	return nil
}

// GetScore retrieves a specific score by QR ID
func (s *SqliteScanDataStore) GetScore(ctx context.Context, qrId string) (*policy.Score, error) {
	scoreRow, err := s.queries.GetScore(ctx, qrId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("score not found: %s", qrId)
		}
		return nil, fmt.Errorf("failed to get score: %w", err)
	}

	return s.convertScore(&scoreRow)
}

// GetData retrieves a specific data result by code ID
func (s *SqliteScanDataStore) GetData(ctx context.Context, codeId string) (*llx.Result, error) {
	data, err := s.queries.GetData(ctx, codeId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data not found: %s", codeId)
		}
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	result := &llx.Result{}
	if err := proto.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result %s: %w", codeId, err)
	}

	return result, nil
}

// GetRisk retrieves a specific risk factor by MRN
func (s *SqliteScanDataStore) GetRisk(ctx context.Context, mrn string) (*policy.ScoredRiskFactor, error) {
	riskRow, err := s.queries.GetRiskFactor(ctx, mrn)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, policy.ErrRiskNotFound
		}
		return nil, fmt.Errorf("failed to get risk: %w", err)
	}

	return &policy.ScoredRiskFactor{
		Mrn:        riskRow.Mrn,
		Risk:       float32(riskRow.Risk),
		IsToxic:    riskRow.IsToxic,
		IsDetected: riskRow.IsDetected,
	}, nil
}

// StreamRisks reads all risk factors with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamRisks(ctx context.Context, callback func(*policy.ScoredRiskFactor) error) error {
	riskRows, err := s.queries.StreamRiskFactors(ctx)
	if err != nil {
		return fmt.Errorf("failed to query risk factors: %w", err)
	}

	for _, row := range riskRows {
		risk := &policy.ScoredRiskFactor{
			Mrn:        row.Mrn,
			Risk:       float32(row.Risk),
			IsToxic:    row.IsToxic,
			IsDetected: row.IsDetected,
		}

		if err := callback(risk); err != nil {
			return fmt.Errorf("callback error for risk %s: %w", risk.Mrn, err)
		}
	}

	return nil
}

// convertScore converts a sqlc-generated Score to a policy.Score
func (s *SqliteScanDataStore) convertScore(scoreRow *sqlc.Score) (*policy.Score, error) {
	score := &policy.Score{
		QrId:            scoreRow.QrID,
		RiskScore:       uint32(scoreRow.RiskScore),
		Type:            uint32(scoreRow.Type),
		Value:           uint32(scoreRow.Value),
		Weight:          uint32(scoreRow.Weight),
		ScoreCompletion: 100,
		DataCompletion:  100,
	}

	if scoreRow.Message.Valid {
		score.Message = scoreRow.Message.String
	}

	// Unmarshal protobuf fields
	if len(scoreRow.RiskFactors) > 0 {
		score.RiskFactors = &policy.ScoredRiskFactors{}
		if err := proto.Unmarshal(scoreRow.RiskFactors, score.RiskFactors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal risk factors: %w", err)
		}
	}

	if len(scoreRow.Sources) > 0 {
		score.Sources = &policy.Sources{}
		if err := proto.Unmarshal(scoreRow.Sources, score.Sources); err != nil {
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

	// Validate table structure exists - check all required tables in a single query
	requiredTables := []string{"metadata", "scores", "data", "scored_risk_factors", "resources"}
	rows, err := s.sqlDB.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table'
	`)
	if err != nil {
		return fmt.Errorf("failed to check tables: %w", err)
	}
	defer rows.Close()

	foundTables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		foundTables[tableName] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating table rows: %w", err)
	}

	// Check that all required tables were found
	for _, table := range requiredTables {
		if !foundTables[table] {
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

	// Close prepared statements (sqlc manages these internally)
	if err := s.queries.Close(); err != nil {
		return "", fmt.Errorf("failed to close queries: %w", err)
	}

	// Optimize the database
	if _, err := s.sqlDB.Exec("VACUUM"); err != nil {
		return "", fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Close the current connection
	if err := s.sqlDB.Close(); err != nil {
		return "", fmt.Errorf("failed to close write connection: %w", err)
	}

	// Reopen as read-only
	sqlDB, err := sql.Open("sqlite", s.filePath+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("failed to reopen database as read-only: %w", err)
	}

	// Update the store to read-only mode
	s.sqlDB = sqlDB
	s.queries = sqlc.New(sqlDB)
	s.readOnly = true

	return s.filePath, nil
}

// Close closes the database connections without optimization
func (s *SqliteScanDataStore) Close() error {
	if s.queries != nil {
		if err := s.queries.Close(); err != nil {
			return err
		}
	}
	return s.sqlDB.Close()
}
