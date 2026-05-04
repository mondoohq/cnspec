// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scandb

import (
	"context"
	"fmt"

	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13/llx"
)

// ScanDataStoreWrapper wraps a ScanDataStore to implement the DataStore interface
// from internal/datalakes/inmemory. It validates that the provided asset MRN
// matches the one stored in the scan data store.
//
// Usage example:
//
//	store, err := NewSqliteScanDataStore("scan.db", assetMrn, sessionId)
//	if err != nil { ... }
//	defer store.Close()
//
//	wrapper := NewScanDataStoreWrapper(store, assetMrn)
//	db.SetDataWriter(wrapper) // db is *inmemory.Db
type ScanDataStoreWrapper struct {
	store    ScanDataStore
	assetMrn string
}

// NewScanDataStoreWrapper creates a wrapper around a ScanDataStore that validates asset MRN
func NewScanDataStoreWrapper(store ScanDataStore, expectedAssetMrn string) *ScanDataStoreWrapper {
	return &ScanDataStoreWrapper{
		store:    store,
		assetMrn: expectedAssetMrn,
	}
}

func (w *ScanDataStoreWrapper) validate(assetMrn string) error {
	if assetMrn != w.assetMrn {
		return fmt.Errorf("asset MRN mismatch: expected %s, got %s", w.assetMrn, assetMrn)
	}
	return nil
}

// WriteScore writes a single score, verifying the asset MRN matches
func (w *ScanDataStoreWrapper) WriteScore(ctx context.Context, assetMrn string, score *policy.Score) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}

	return w.store.WriteScores(ctx, []*policy.Score{score})
}

// GetScore retrieves a score by ID, verifying the asset MRN matches
func (w *ScanDataStoreWrapper) GetScore(ctx context.Context, assetMrn string, scoreID string) (*policy.Score, error) {
	if err := w.validate(assetMrn); err != nil {
		return nil, err
	}

	return w.store.GetScore(ctx, scoreID)
}

// WriteData writes a single data result, verifying the asset MRN matches
func (w *ScanDataStoreWrapper) WriteData(ctx context.Context, assetMrn string, data *llx.Result) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}

	return w.store.WriteData(ctx, []*llx.Result{data})
}

func (w *ScanDataStoreWrapper) WriteResource(ctx context.Context, assetMrn string, resource *llx.ResourceRecording) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}

	return w.store.WriteResource(ctx, resource)
}

// GetData retrieves data by query ID, verifying the asset MRN matches
func (w *ScanDataStoreWrapper) GetData(ctx context.Context, assetMrn string, qrId string) (*llx.Result, error) {
	if err := w.validate(assetMrn); err != nil {
		return nil, err
	}

	return w.store.GetData(ctx, qrId)
}

func (w *ScanDataStoreWrapper) WriteRisk(ctx context.Context, assetMrn string, risk *policy.ScoredRiskFactor) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}
	return w.store.WriteRisk(ctx, risk)
}

func (w *ScanDataStoreWrapper) GetRisk(ctx context.Context, assetMrn string, riskID string) (*policy.ScoredRiskFactor, error) {
	if err := w.validate(assetMrn); err != nil {
		return nil, err
	}

	return w.store.GetRisk(ctx, riskID)
}

func (w *ScanDataStoreWrapper) GetResource(ctx context.Context, assetMrn string, resource string, id string) (*llx.ResourceRecording, error) {
	if err := w.validate(assetMrn); err != nil {
		return nil, err
	}

	return w.store.GetResource(ctx, resource, id)
}

func (w *ScanDataStoreWrapper) StreamResources(ctx context.Context, assetMrn string, f func(resource *llx.ResourceRecording) error) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}

	return w.store.StreamResources(ctx, f)
}

func (w *ScanDataStoreWrapper) StreamRisks(ctx context.Context, assetMrn string, f func(risk *policy.ScoredRiskFactor) error) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}

	err := w.store.StreamRisks(ctx, func(risk *policy.ScoredRiskFactor) error {
		return f(risk)
	})
	return err
}

// Finalize optimizes the underlying store and returns the database path
func (w *ScanDataStoreWrapper) Finalize() (string, error) {
	return w.store.Finalize()
}

// SetAssetFilters records the code_ids of the filters the scanner sent to
// ResolveAndUpdateJobs into the scan database. Verifies the asset MRN
// matches and pulls just the code_ids from the proto — no MQL/text is
// persisted, since downstream replay can recover the filter from its
// code_id alone.
func (w *ScanDataStoreWrapper) SetAssetFilters(ctx context.Context, assetMrn string, filters *policy.Mqueries) error {
	if err := w.validate(assetMrn); err != nil {
		return err
	}
	if filters == nil {
		return nil
	}
	codeIDs := make([]string, 0, len(filters.Items))
	for _, f := range filters.Items {
		if f.CodeId != "" {
			codeIDs = append(codeIDs, f.CodeId)
		}
	}
	if len(codeIDs) == 0 {
		return nil
	}
	store, ok := w.store.(*SqliteScanDataStore)
	if !ok {
		return nil
	}
	return store.WriteAssetFilters(ctx, codeIDs)
}
