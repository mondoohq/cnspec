package inmemory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mondoo.com/cnspec/policy"
)

// kvStore is an abstraction on top of ristretto
type kvStore interface {
	Get(key interface{}) (interface{}, bool)
	Set(key interface{}, value interface{}, cost int64) bool
	Del(key interface{})
}

// Db is the ent-based Database backend for Mondoo services
// It allows you to interact with the underlying data in Mondoo.
type Db struct {
	cache               kvStore
	services            *policy.LocalServices
	uuid                string // used for all object identifiers to prevent clashes (eg in-memory pubsub)
	nowProvider         func() time.Time
	resolvedPolicyCache *ResolvedPolicyCache
}

// NewServices creates a new set of policy services
func NewServices(ctx context.Context, resolvedPolicyCache *ResolvedPolicyCache) (*Db, *policy.LocalServices, error) {
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

	services := policy.NewLocalServices(db, db.uuid)
	db.services = services

	return db, services, nil
}

// WithDb creates a new set of policy services and closes everything out once the function is done
func WithDb(ctx context.Context, resolvedPolicyCache *ResolvedPolicyCache, f func(*Db, *policy.LocalServices) error) error {
	db, ls, err := NewServices(ctx, resolvedPolicyCache)
	if err != nil {
		return err
	}

	return f(db, ls)
}

// Prefixes for all keys that are stored in the cache.
// Prevent collissions by creating namespaces for different types of data.
const (
	dbIDQuery          = "q\x00"
	dbIDPolicy         = "p\x00"
	dbIDBundle         = "b\x00"
	dbIDListPolicies   = "pl\x00"
	dbIDScore          = "s\x00"
	dbIDData           = "d\x00"
	dbIDAsset          = "a\x00"
	dbIDResolvedPolicy = "rp\x00"
)

func (db *Db) SetNowProvider(f func() time.Time) {
	db.nowProvider = f
}
