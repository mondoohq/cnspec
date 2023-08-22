// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import "sync"

// kvStore is an general-purpose abstraction for key-value stores
type kvStore interface {
	Get(key interface{}) (interface{}, bool)
	Set(key interface{}, value interface{}, cost int64) bool
	Del(key interface{})
}

// kissDb for ristretto-like behavior; works synchronously
type kissDb struct {
	mu   sync.Mutex
	data map[string]interface{}
}

func newKissDb() *kissDb {
	return &kissDb{
		data: map[string]interface{}{},
	}
}

func (c *kissDb) Get(key interface{}) (interface{}, bool) {
	k, ok := key.(string)
	if !ok {
		panic("cannot map key to string for kissDB")
	}

	c.mu.Lock()
	res, ok := c.data[k]
	c.mu.Unlock()

	return res, ok
}

func (c *kissDb) Set(key interface{}, value interface{}, cost int64) bool {
	k, ok := key.(string)
	if !ok {
		panic("cannot map key to string for kissDB")
	}

	c.mu.Lock()
	c.data[k] = value
	c.mu.Unlock()

	return true
}

func (c *kissDb) Del(key interface{}) {
	k, ok := key.(string)
	if !ok {
		panic("cannot map key to string for kissDB")
	}

	c.mu.Lock()
	delete(c.data, k)
	c.mu.Unlock()
}
