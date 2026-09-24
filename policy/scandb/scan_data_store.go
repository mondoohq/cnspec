// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:generate go tool sqlc generate ./

package scandb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/policy/checksum"
	"go.mondoo.com/cnspec/policy/scandb/sqlc"
	"go.mondoo.com/mql/llx"
	llxchecksum "go.mondoo.com/mql/llx/checksum"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
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
	ClientVersion   string `json:"client_version"`
	ClientBuild     string `json:"client_build"`
}

type ScanDataStoreReader interface {
	StreamScores(ctx context.Context, callback func(*policy.Score) error) error
	StreamData(ctx context.Context, callback func(string, *llx.Result) error) error
	StreamRisks(ctx context.Context, callback func(*policy.ScoredRiskFactor) error) error
	StreamResources(ctx context.Context, callback func(*llx.ResourceRecording) error) error

	// Reader methods for specific items
	GetScore(ctx context.Context, qrId string) (*policy.Score, error)
	GetData(ctx context.Context, codeId string) (*llx.Result, error)
	GetRisk(ctx context.Context, mrn string) (*policy.ScoredRiskFactor, error)
	GetResource(ctx context.Context, resource string, id string) (*llx.ResourceRecording, error)
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

// SchemaVersion is the scandb schema written by this client. The per-row
// `checksum` columns and checksum metadata keys (see checksum.go) are
// deliberately NOT a version bump: they are additive and invisible to older
// readers (explicit-column queries everywhere), while today's server ingest
// validates the version with an exact match — a bump would make every
// upload from this client unprocessable by servers that haven't shipped
// checksum support. Checksum presence is announced by the
// checksum_algo_version metadata key instead.
const SchemaVersion = "1.0"

// readOnlyDSN builds a DSN that SQLite itself enforces as read-only. The
// file: URI form is required: with a plain-path DSN the driver ignores the
// query string entirely and opens READWRITE|CREATE — silently creating
// missing files and letting raw SQL write through a handle the code
// believes is read-only. The checksum ownership contract (checksums are
// written by the scan owner, never a consumer) rests on this being real.
// ToSlash keeps the URI valid on Windows paths.
func readOnlyDSN(filePath string) string {
	return "file:" + filepath.ToSlash(filePath) + "?mode=ro"
}

// SqliteScanDataStore implements ScanDataStore using SQLite with sqlc-generated queries
type SqliteScanDataStore struct {
	sqlDB    *sql.DB
	queries  *sqlc.Queries
	assetMrn string
	filePath string
	readOnly bool

	// computeChecksums makes every Write* hash its row inline (see
	// WithWriteTimeChecksums). checksumStats tracks that activity.
	computeChecksums bool
	checksumStats    writeChecksumCounters
}

// writeChecksumCounters accumulates write-time checksum activity. Atomic:
// writes may arrive from concurrent executor callbacks.
type writeChecksumCounters struct {
	data, scores, risks, resources atomic.Int64
	errors                         atomic.Int64
	nanos                          atomic.Int64
}

// WriteChecksumStats is a snapshot of a store's write-time checksum
// activity. Counts are hash operations (an upserted row counts once per
// write); Duration is pure accumulated hashing time.
type WriteChecksumStats struct {
	Counts   ChecksumCounts
	Errors   int64
	Duration time.Duration
}

// WriteChecksumStats snapshots the write-time checksum counters.
func (s *SqliteScanDataStore) WriteChecksumStats() WriteChecksumStats {
	return WriteChecksumStats{
		Counts: ChecksumCounts{
			Data:      int(s.checksumStats.data.Load()),
			Scores:    int(s.checksumStats.scores.Load()),
			Risks:     int(s.checksumStats.risks.Load()),
			Resources: int(s.checksumStats.resources.Load()),
		},
		Errors:   s.checksumStats.errors.Load(),
		Duration: time.Duration(s.checksumStats.nanos.Load()),
	}
}

// StoreOption configures a SqliteScanDataStore at construction.
type StoreOption func(*SqliteScanDataStore)

// WithWriteTimeChecksums makes every Write* compute the row's content
// checksum inline and store it with the row — no second pass over the
// database. Correct under upserts by construction (INSERT OR REPLACE
// rewrites the row's checksum with the row) and bit-identical to the
// post-pass recompute (pinned by TestWriteTimeChecksumParity; the
// nil-vs-empty normalizations in policy/checksum and mql's llx/checksum
// are what make in-memory and round-tripped rows hash the same).
//
// Checksum work can never fail a scan: a hash error is counted and
// logged, the row is stored with checksum 0, and Finalize then skips the
// checksum_algo_version stamp so a partially checksummed file is never
// announced as checksummed — consumers treat it exactly like a
// pre-checksum file (full upload), fail-open.
func WithWriteTimeChecksums() StoreOption {
	return func(s *SqliteScanDataStore) { s.computeChecksums = true }
}

// rowChecksum runs one write-time hash, never failing the write: on error
// it counts, logs, and returns 0 (the schema default, meaning "no
// checksum"). Timing is accumulated so the true hashing cost is reported
// per scan.
func (s *SqliteScanDataStore) rowChecksum(kind, key string, counter *atomic.Int64, hash func() (uint64, error)) int64 {
	if !s.computeChecksums {
		return 0
	}
	start := time.Now()
	sum, err := hash()
	s.checksumStats.nanos.Add(time.Since(start).Nanoseconds())
	if err != nil {
		s.checksumStats.errors.Add(1)
		log.Warn().Err(err).Str("kind", kind).Str("key", key).
			Msg("failed to compute scan content checksum; row stored without one")
		return 0
	}
	counter.Add(1)
	// SQLite INTEGER is int64; store the uint64 bit pattern.
	return int64(sum)
}

// StampChecksums announces the checksum algorithm version in the metadata
// table — the marker consumers key on. The scan owner calls it exactly
// once, when the scan's writes are complete: for cnspec that is the sqlite
// datalake, right after the scan and before any Finalize/upload, which is
// the one seam both paths share (an uploaded file and a kept
// --output-scan-db file that never sees Finalize are stamped by the same
// call). Finalize deliberately does NOT stamp — a store user who writes
// checksums owns the stamp too.
//
// It refuses to stamp a file with any hash failure: partially checksummed
// must never masquerade as checksummed — the unstamped file is consumed
// exactly like a pre-checksum one (full upload, fail-open). Best-effort by
// contract: checksum work never fails a scan, so problems are logged,
// never returned.
func (s *SqliteScanDataStore) StampChecksums() {
	if !s.computeChecksums || s.readOnly {
		return
	}
	if errs := s.checksumStats.errors.Load(); errs > 0 {
		log.Warn().Int64("failures", errs).
			Msg("scan content checksums incomplete; leaving file unstamped (uploads in full)")
		return
	}
	if _, err := s.sqlDB.Exec(
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`,
		MetaChecksumAlgoVersion, llxchecksum.AlgoVersion); err != nil {
		log.Warn().Err(err).
			Msg("failed to stamp checksum algo version; leaving file unstamped (uploads in full)")
	}
}

// NewSqliteScanDataStore creates a new SQLite-based scan data store for writing
func NewSqliteScanDataStore(filePath string, assetMrn string, opts ...StoreOption) (*SqliteScanDataStore, error) {
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
	for _, opt := range opts {
		opt(store)
	}

	if err := store.initializeDatabase(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// NewSqliteScanDataStoreReader creates a new SQLite-based scan data store for reading
func NewSqliteScanDataStoreReader(filePath string) (*SqliteScanDataStore, error) {
	sqlDB, err := sql.Open("sqlite", readOnlyDSN(filePath))
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
		"client_version": cnspec.GetVersion(),
		"client_build":   cnspec.GetBuild(),
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
			Checksum: s.rowChecksum("data", codeId, &s.checksumStats.data, func() (uint64, error) {
				return llxchecksum.HashDataRow(codeId, result)
			}),
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
		Checksum: s.rowChecksum("resource", resource.Resource+"/"+resource.Id, &s.checksumStats.resources, func() (uint64, error) {
			return llxchecksum.HashResourceRow(resource)
		}),
	}); err != nil {
		return fmt.Errorf("failed to write resource %s/%s: %w", resource.Resource, resource.Id, err)
	}
	return nil
}

// WriteAssetFilters persists the code_ids of the filters the scanner passed
// to ResolveAndUpdateJobs. Storage is per-row, deduped by code_id (the
// schema's primary key), so repeated calls with overlapping sets are
// idempotent. Captured on every scan that uses the SQLite datalake; the
// server can recover the full Mquery from the code_id at replay time, so
// the on-disk MQL string isn't worth the bytes.
func (s *SqliteScanDataStore) WriteAssetFilters(ctx context.Context, codeIDs []string) error {
	if s.readOnly {
		return fmt.Errorf("cannot write asset filters in read-only mode")
	}
	for _, id := range codeIDs {
		if id == "" {
			continue
		}
		if err := s.queries.InsertAssetFilter(ctx, id); err != nil {
			return fmt.Errorf("failed to write asset filter %s: %w", id, err)
		}
	}
	return nil
}

// GetAssetFilters retrieves the code_ids of the asset filters captured
// during the original scan. Returns policy.ErrAssetFiltersNotFound for
// scan databases that don't carry the optional asset_filters table or
// haven't had any filters written (older databases / non-SQLite paths).
func (s *SqliteScanDataStore) GetAssetFilters(ctx context.Context) ([]string, error) {
	rows, err := s.queries.StreamAssetFilters(ctx)
	if err != nil {
		if isMissingAssetFiltersTable(err) {
			return nil, policy.ErrAssetFiltersNotFound
		}
		return nil, fmt.Errorf("failed to get asset filters: %w", err)
	}
	if len(rows) == 0 {
		return nil, policy.ErrAssetFiltersNotFound
	}
	return rows, nil
}

func isMissingAssetFiltersTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table: asset_filters")
}

// WriteAsset persists the inventory.Asset proto for the scanned asset.
// The asset table is optional and announced by its own presence — the
// stamped schema_version stays "1.0" (see SchemaVersion). It enables
// consumers (e.g. cnspec loadtest) to replay scan databases against
// SynchronizeAssets without out-of-band asset metadata.
func (s *SqliteScanDataStore) WriteAsset(ctx context.Context, asset *inventory.Asset) error {
	if s.readOnly {
		return fmt.Errorf("cannot write asset in read-only mode")
	}
	if asset == nil {
		return fmt.Errorf("cannot write nil asset")
	}
	data, err := proto.Marshal(asset)
	if err != nil {
		return fmt.Errorf("failed to marshal asset: %w", err)
	}
	if err := s.queries.InsertAsset(ctx, data); err != nil {
		return fmt.Errorf("failed to write asset: %w", err)
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
		Checksum: s.rowChecksum("risk", risk.Mrn, &s.checksumStats.risks, func() (uint64, error) {
			return checksum.HashRiskRow(risk)
		}),
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
		Checksum: s.rowChecksum("score", score.QrId, &s.checksumStats.scores, func() (uint64, error) {
			return checksum.HashScoreRow(score)
		}),
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
		ClientVersion:   rawMetadata["client_version"],
		ClientBuild:     rawMetadata["client_build"],
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("score not found: %s", qrId)
		}
		return nil, fmt.Errorf("failed to get score: %w", err)
	}

	row := sqlc.StreamScoresRow(scoreRow)
	return s.convertScore(&row)
}

// GetData retrieves a specific data result by code ID
func (s *SqliteScanDataStore) GetData(ctx context.Context, codeId string) (*llx.Result, error) {
	data, err := s.queries.GetData(ctx, codeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
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

// GetAsset retrieves the inventory.Asset proto for the scanned asset.
// Returns policy.ErrAssetNotFound for older scan databases that lack the
// optional asset table, or when the asset row has not been written yet.
func (s *SqliteScanDataStore) GetAsset(ctx context.Context) (*inventory.Asset, error) {
	data, err := s.queries.GetAsset(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isMissingAssetTable(err) {
			return nil, policy.ErrAssetNotFound
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	asset := &inventory.Asset{}
	if err := proto.Unmarshal(data, asset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset: %w", err)
	}
	return asset, nil
}

// isMissingAssetTable detects the SQLite error returned when reading from a
// database that predates the (optional) asset table. SQLite reports it as
// "no such table: asset"; matching by message keeps us decoupled from the
// driver-specific error type.
func isMissingAssetTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table: asset")
}

// GetResource retrieves a specific resource by name and ID
func (s *SqliteScanDataStore) GetResource(ctx context.Context, resource string, id string) (*llx.ResourceRecording, error) {
	data, err := s.queries.GetResource(ctx, sqlc.GetResourceParams{
		Name: resource,
		ID:   id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, policy.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	result := &llx.ResourceRecording{}
	if err := proto.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource %s/%s: %w", resource, id, err)
	}

	return result, nil
}

// StreamResources reads all resources with a callback function for memory-efficient processing
func (s *SqliteScanDataStore) StreamResources(ctx context.Context, callback func(*llx.ResourceRecording) error) error {
	resourceRows, err := s.queries.StreamResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to query resources: %w", err)
	}

	for _, row := range resourceRows {
		result := &llx.ResourceRecording{}
		if err := proto.Unmarshal(row.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal resource %s/%s: %w", row.Name, row.ID, err)
		}

		if err := callback(result); err != nil {
			return fmt.Errorf("callback error for resource %s/%s: %w", row.Name, row.ID, err)
		}
	}

	return nil
}

// convertScore converts a sqlc-generated score row to a policy.Score. The
// readers select every column except checksum (explicit-column queries are
// what keep old files readable), so sqlc emits per-query row structs;
// GetScoreRow is field-identical and converts to StreamScoresRow at its
// call site.
func (s *SqliteScanDataStore) convertScore(scoreRow *sqlc.StreamScoresRow) (*policy.Score, error) {
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
	sqlDB, err := sql.Open("sqlite", readOnlyDSN(s.filePath))
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
