// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v12/policy"
)

type wrapAsset struct {
	mrn                   string
	resolvedPolicyVersion string
	ResolvedPolicy        *policy.ResolvedPolicy
}

// EnsureAsset makes sure an asset exists
func (db *Db) EnsureAsset(ctx context.Context, mrn string) error {
	_, _, _, err := db.ensureAsset(ctx, mrn)
	return err
}

func (db *Db) ensureAsset(ctx context.Context, mrn string) (wrapAsset, wrapPolicy, wrapFramework, error) {
	assetw, assetIsNew, err := db.ensureAssetObject(ctx, mrn)
	if err != nil {
		return wrapAsset{}, wrapPolicy{}, wrapFramework{}, err
	}

	var policyw wrapPolicy
	var frameworkw wrapFramework
	createPolicy := true
	createFramework := true

	if !assetIsNew {
		if x, ok := db.cache.Get(dbIDPolicy + mrn); ok {
			policyw = x.(wrapPolicy)
			createPolicy = false
		} else {
			log.Warn().Str("asset", mrn).Msg("assets> asset did not have a policy set, this should not happen, fixing")
		}

		if x, ok := db.cache.Get(dbIDFramework + mrn); ok {
			frameworkw = x.(wrapFramework)
			createFramework = false
		} else {
			log.Warn().Str("asset", mrn).Msg("assets> asset did not have a policy set, this should not happen, fixing")
		}
	}

	if createPolicy {
		policyw, err = db.ensureAssetPolicy(ctx, mrn)
		if err != nil {
			return wrapAsset{}, wrapPolicy{}, wrapFramework{}, err
		}
	}

	if createFramework {
		frameworkw, err = db.ensureAssetFramework(ctx, mrn)
		if err != nil {
			return wrapAsset{}, wrapPolicy{}, wrapFramework{}, err
		}
	}

	return assetw, policyw, frameworkw, nil
}

func (db *Db) ensureAssetPolicy(ctx context.Context, mrn string) (wrapPolicy, error) {
	policyObj := db.services.CreatePolicyObject(mrn, "")
	policyObj, filters, _, err := db.services.PreparePolicy(ctx, policyObj, nil)
	if err != nil {
		return wrapPolicy{}, err
	}

	policyw, err := db.setPolicy(ctx, policyObj, filters)
	if err != nil {
		return wrapPolicy{}, err
	}

	return policyw, nil
}

func (db *Db) ensureAssetFramework(ctx context.Context, mrn string) (wrapFramework, error) {
	obj := db.services.CreateFrameworkObject(mrn, "")
	frameworkw, err := db.setFramework(ctx, obj)
	if err != nil {
		return wrapFramework{}, err
	}

	return frameworkw, nil
}

func (db *Db) ensureAssetObject(ctx context.Context, mrn string) (wrapAsset, bool, error) {
	log.Debug().Str("mrn", mrn).Msg("assets> ensure asset")

	x, ok := db.cache.Get(dbIDAsset + mrn)
	if ok {
		return x.(wrapAsset), false, nil
	}

	assetw := wrapAsset{mrn: mrn}
	ok = db.cache.Set(dbIDAsset+mrn, assetw, 1)
	if !ok {
		return wrapAsset{}, false, errors.New("failed to create asset '" + mrn + "'")
	}

	return assetw, true, nil
}
