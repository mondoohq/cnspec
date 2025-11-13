// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v12/policy"
)

// Db is the ent-based Database backend for Mondoo services
// It allows you to interact with the underlying data in Mondoo.
type Db struct {
	cache               kvStore
	services            *policy.LocalServices // bidirectional connection between db + services
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
func NewServices(runtime llx.Runtime) (*Db, *policy.LocalServices, error) {
	var cache kvStore = newKissDb()

	resolvedPolicyCache := NewResolvedPolicyCache(0)

	db := &Db{
		cache:               cache,
		nowProvider:         time.Now,
		resolvedPolicyCache: resolvedPolicyCache,
		writer:              &cacheDataWriter{cache: cache},
	}

	services := policy.NewLocalServices(db, runtime)
	db.services = services // close the connection between db and services

	return db, services, nil
}

func (db *Db) SetDataWriter(writer DataStore) {
	db.writer = writer
}

func WithServices(ctx context.Context, runtime llx.Runtime, assetMrn string, upstreamClient *upstream.UpstreamClient, f func(*policy.LocalServices) error) error {
	_, ls, err := NewServices(runtime)
	if err != nil {
		return err
	}

	var upstream *policy.Services
	if upstreamClient != nil {
		var err error
		upstream, err = policy.NewRemoteServices(upstreamClient.ApiEndpoint, upstreamClient.Plugins, upstreamClient.HttpClient)
		if err != nil {
			return err
		}
	}

	ls.Upstream = upstream

	return f(ls)
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
