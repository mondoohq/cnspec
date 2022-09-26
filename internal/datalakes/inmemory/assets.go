package inmemory

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
)

type wrapAsset struct {
	mrn                   string
	resolvedPolicyVersion string
	ResolvedPolicy        *policy.ResolvedPolicy
}

// EnsureAsset makes sure an asset exists
func (db *Db) EnsureAsset(ctx context.Context, mrn string) error {
	_, _, err := db.ensureAsset(ctx, mrn)
	return err
}

func (db *Db) ensureAsset(ctx context.Context, mrn string) (wrapAsset, wrapPolicy, error) {
	assetw, created, err := db.ensureAssetObject(ctx, mrn)
	if err != nil {
		return wrapAsset{}, wrapPolicy{}, err
	}

	var policyw wrapPolicy
	if !created {
		x, ok := db.cache.Get(dbIDPolicy + mrn)
		if ok {
			return assetw, policyw, nil
		}
		policyw = x.(wrapPolicy)

		log.Warn().Str("asset", mrn).Msg("asset did not have a policy set, this should not happen, fixing")
	}

	policyw, err = db.ensureAssetPolicy(ctx, mrn)
	if err != nil {
		return wrapAsset{}, wrapPolicy{}, err
	}

	return assetw, policyw, nil
}

func (db *Db) ensureAssetPolicy(ctx context.Context, mrn string) (wrapPolicy, error) {
	policyObj := db.services.CreatePolicyObject(mrn, "")
	policyObj, filters, err := db.services.PreparePolicy(ctx, policyObj, nil)
	if err != nil {
		return wrapPolicy{}, err
	}

	policyw, err := db.setPolicy(ctx, policyObj, filters)
	if err != nil {
		return wrapPolicy{}, err
	}

	return policyw, nil
}

func (db *Db) ensureAssetObject(ctx context.Context, mrn string) (wrapAsset, bool, error) {
	log.Debug().Str("mrn", mrn).Msg("distributor> ensure asset")

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
