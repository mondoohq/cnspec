// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"sync"
	"time"

	"go.mondoo.com/cnspec/v9/policy"
	"google.golang.org/protobuf/proto"
)

const ResolvedPolicyCacheTTL = 1 * time.Hour

type cachedResolvedPolicy struct {
	createdOn      time.Time
	lastAccessedOn time.Time
	resolvedPolicy *policy.ResolvedPolicy
	size           int64
}

func (c *cachedResolvedPolicy) isExpired(now time.Time) bool {
	return now.Sub(c.createdOn) > ResolvedPolicyCacheTTL
}

type ResolvedPolicyCache struct {
	mu          sync.Mutex
	data        map[string]*cachedResolvedPolicy
	totalSize   int64
	sizeLimit   int64
	nowProvider func() time.Time
}

// NewResolvedPolicyCache creates a new ResolvedPolicyCache with the given size limit. If the size
// limit is 0, the cache is unlimited.
func NewResolvedPolicyCache(sizeLimit int64) *ResolvedPolicyCache {
	if sizeLimit < 0 {
		panic("sizeLimit must be >= 0")
	}
	return &ResolvedPolicyCache{
		data:        make(map[string]*cachedResolvedPolicy),
		sizeLimit:   sizeLimit,
		nowProvider: time.Now,
	}
}

func (c *ResolvedPolicyCache) Get(key string) (*policy.ResolvedPolicy, bool) {
	c.mu.Lock()
	res, ok := c.data[key]
	defer c.mu.Unlock()
	if !ok {
		return nil, ok
	}

	// If the entry is older than TTL delete it and return nothing.
	if c.nowProvider().Sub(res.createdOn) > ResolvedPolicyCacheTTL {
		delete(c.data, key)
		c.totalSize -= res.size
		return nil, false
	}

	res.lastAccessedOn = c.nowProvider()

	return res.resolvedPolicy, ok
}

func (c *ResolvedPolicyCache) Set(key string, resolvedPolicy *policy.ResolvedPolicy) bool {
	cacheEntry := cachedResolvedPolicy{
		createdOn:      c.nowProvider(),
		lastAccessedOn: c.nowProvider(),
		resolvedPolicy: resolvedPolicy,
		size:           int64(proto.Size(resolvedPolicy)),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.freeSpace(cacheEntry.size)

	// If there is still no space, then return false.
	if c.hasSpace(cacheEntry.size) {
		return false
	}

	// If we are overwriting an entry, remove the old entry's size first.
	if existing, ok := c.data[key]; ok {
		c.totalSize -= existing.size
	}
	c.totalSize += cacheEntry.size
	c.data[key] = &cacheEntry

	return true
}

// freeSpace deletes expired entries and entries starting from the oldest one until the cache has sufficient space
// to accommodate a new entry with the given size. The function doesn't solve thread safety so the caller needs to
// handle that.
func (c *ResolvedPolicyCache) freeSpace(size int64) {
	if len(c.data) == 0 {
		return
	}

	for c.hasSpace(size) {
		// Delete the oldest entry in the cache to make space for the new one
		var oldestEntry *cachedResolvedPolicy
		oldestKey := ""
		oldestTimestamp := c.nowProvider().Add(1 * time.Minute)
		for k, v := range c.data {
			// If the entry is older than TTL delete it.
			if c.nowProvider().Sub(v.createdOn) > ResolvedPolicyCacheTTL {
				delete(c.data, k)
				c.totalSize -= v.size
				continue
			}

			if v.lastAccessedOn.Before(oldestTimestamp) {
				oldestTimestamp = v.lastAccessedOn
				oldestEntry = v
				oldestKey = k
			}
		}

		// Since the loop above also delete expired entries, check whether we have sufficient
		// space now before deleting the oldest entry.
		if c.hasSpace(size) {
			if oldestKey == "" { // If the oldest key is not set, there is nothing to delete.
				return
			}

			// Delete the entry and update the total size
			delete(c.data, oldestKey)
			c.totalSize -= oldestEntry.size
		}
	}
}

func (c *ResolvedPolicyCache) hasSpace(size int64) bool {
	return c.sizeLimit > 0 && c.totalSize+size > c.sizeLimit
}
