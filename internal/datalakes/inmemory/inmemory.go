// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnspec/v12/policy"
)

// Db is the ent-based Database backend for Mondoo services
// It allows you to interact with the underlying data in Mondoo.
type Db struct {
	cache               kvStore
	services            *policy.LocalServices // bidirectional connection between db + services
	uuid                string                // used for all object identifiers to prevent clashes (eg in-memory pubsub)
	nowProvider         func() time.Time
	resolvedPolicyCache *ResolvedPolicyCache
	writer              DataStore
}

type DataStore interface {
	WriteScore(ctx context.Context, assetMrn string, scores *policy.Score) error
	GetScore(ctx context.Context, assetMrn string, scoreID string) (*policy.Score, error)
	WriteData(ctx context.Context, assetMrn string, data *llx.Result) error
	GetData(ctx context.Context, assetMrn string, qrId string) (*llx.Result, error)
	WriteRisk(ctx context.Context, assetMrn string, risk *policy.ScoredRiskFactor) error
	GetRisk(ctx context.Context, assetMrn string, riskMrn string) (*policy.ScoredRiskFactor, error)
	StreamRisks(ctx context.Context, assetMrn string, f func(risk *policy.ScoredRiskFactor) error) error
	WriteResource(ctx context.Context, assetMrn string, resource *llx.ResourceRecording) error
}

type cacheDataWriter struct {
	cache kvStore
}

func (c *cacheDataWriter) WriteScore(ctx context.Context, assetMrn string, score *policy.Score) error {
	ok := c.cache.Set(dbIDScore+assetMrn+"\x00"+score.QrId, *score, 1)
	if !ok {
		return errors.New("failed to set score for asset '" + assetMrn + "' with ID '" + score.QrId + "'")
	}
	return nil
}

func (c *cacheDataWriter) GetScore(ctx context.Context, assetMrn string, scoreID string) (*policy.Score, error) {
	x, ok := c.cache.Get(dbIDScore + assetMrn + "\x00" + scoreID)
	if !ok {
		return &policy.Score{}, errors.New("cannot find score")
	}
	s := x.(policy.Score)
	return s.CloneVT(), nil
}

func (n *cacheDataWriter) WriteData(ctx context.Context, assetMrn string, value *llx.Result) error {
	id := dbIDData + assetMrn + "\x00" + value.CodeId
	ok := n.cache.Set(id, value, 1)
	if !ok {
		return errors.New("failed to save asset data for asset '" + assetMrn + "' and checksum '" + value.CodeId + "'")
	}
	return nil
}

func (n *cacheDataWriter) GetData(ctx context.Context, assetMrn string, qrId string) (*llx.Result, error) {
	x, ok := n.cache.Get(dbIDData + assetMrn + "\x00" + qrId)
	if !ok {
		return nil, errors.New("cannot find data")
	}
	return x.(*llx.Result), nil
}

func (n *cacheDataWriter) WriteRisk(ctx context.Context, assetMrn string, risk *policy.ScoredRiskFactor) error {
	var existingRisks map[string]*policy.ScoredRiskFactor

	dbID := dbIDAssetRisk + assetMrn
	raw, ok := n.cache.Get(dbID)
	if ok {
		existingRisks = raw.(map[string]*policy.ScoredRiskFactor)
	} else {
		existingRisks = map[string]*policy.ScoredRiskFactor{}
	}
	existingRisks[risk.Mrn] = risk
	n.cache.Set(dbID, existingRisks, 1)
	return nil
}

func (n *cacheDataWriter) GetRisk(ctx context.Context, assetMrn string, riskMrn string) (*policy.ScoredRiskFactor, error) {
	raw, ok := n.cache.Get(dbIDAssetRisk + assetMrn)
	if !ok {
		return nil, policy.ErrRiskNotFound
	}
	existingRisks := raw.(map[string]*policy.ScoredRiskFactor)
	risk, ok := existingRisks[riskMrn]
	if !ok {
		return nil, policy.ErrRiskNotFound
	}
	return risk, nil
}

func (n *cacheDataWriter) StreamRisks(ctx context.Context, assetMrn string, f func(risk *policy.ScoredRiskFactor) error) error {
	raw, ok := n.cache.Get(dbIDAssetRisk + assetMrn)
	if !ok {
		return nil
	}
	existingRisks := raw.(map[string]*policy.ScoredRiskFactor)
	for _, risk := range existingRisks {
		if err := f(risk); err != nil {
			return err
		}
	}
	return nil
}

func (n *cacheDataWriter) WriteResource(ctx context.Context, assetMrn string, resource *llx.ResourceRecording) error {
	return nil
}

// NewServices creates a new set of policy services
func NewServices(runtime llx.Runtime, resolvedPolicyCache *ResolvedPolicyCache) (*Db, *policy.LocalServices, error) {
	var cache kvStore = newKissDb()

	if resolvedPolicyCache == nil {
		resolvedPolicyCache = NewResolvedPolicyCache(0)
	}

	db := &Db{
		cache:               cache,
		uuid:                uuid.New().String(),
		nowProvider:         time.Now,
		resolvedPolicyCache: resolvedPolicyCache,
		writer:              &cacheDataWriter{cache: cache},
	}

	services := policy.NewLocalServices(db, db.uuid, runtime)
	db.services = services // close the connection between db and services

	return db, services, nil
}

func (db *Db) SetDataWriter(writer DataStore) {
	db.writer = writer
}

// WithDb creates a new set of policy services and closes everything out once the function is done
func WithDb(runtime llx.Runtime, resolvedPolicyCache *ResolvedPolicyCache, f func(*Db, *policy.LocalServices) error) error {
	db, ls, err := NewServices(runtime, resolvedPolicyCache)
	if err != nil {
		return err
	}

	return f(db, ls)
}

// Prefixes for all keys that are stored in the cache.
// Prevent collisions by creating namespaces for different types of data.
const (
	dbIDQuery          = "q\x00"
	dbIDProp           = "qp\x00"
	dbIDPolicy         = "p\x00"
	dbIDBundle         = "b\x00"
	dbIDListPolicies   = "pl\x00"
	dbIDScore          = "s\x00"
	dbIDData           = "d\x00"
	dbIDAsset          = "a\x00"
	dbIDResolvedPolicy = "rp\x00"
	dbIDRiskFactor     = "rf\x00"
	dbIDAssetRisk      = "ar\x00"
	dbIDFramework      = "f\x00"
	dbIDFrameworkMap   = "fm\x00"
)

func (db *Db) SetNowProvider(f func() time.Time) {
	db.nowProvider = f
}
