// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"time"

	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnspec/v10/policy"
)

// Db is the ent-based Database backend for Mondoo services
// It allows you to interact with the underlying data in Mondoo.
type Db struct {
	cache               kvStore
	services            *policy.LocalServices // bidirectional connection between db + services
	uuid                string                // used for all object identifiers to prevent clashes (eg in-memory pubsub)
	nowProvider         func() time.Time
	resolvedPolicyCache *ResolvedPolicyCache
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
	}

	services := policy.NewLocalServices(db, db.uuid, runtime)
	db.services = services // close the connection between db and services

	return db, services, nil
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
	dbIDAssetRisk      = "ar\x00"
	dbIDFramework      = "f\x00"
	dbIDFrameworkMap   = "fm\x00"
)

func (db *Db) SetNowProvider(f func() time.Time) {
	db.nowProvider = f
}
