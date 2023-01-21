// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memcache

import (
	"context"
	"fmt"
	"strings"
	"time"

	cache "github.com/beego/beego-cache/v2"
	berror "github.com/beego/beego-error/v2"
	"github.com/bradfitz/gomemcache/memcache"
)

// Cache Memcache adapter.
type Cache struct {
	conn     *memcache.Client
	conninfo []string
}

type CacheOptions func(c *Cache)

// CacheWithConninfo configures conninfo for memcache
func CacheWithConninfo(conninfo []string) CacheOptions {
	return func(c *Cache) {
		c.conninfo = conninfo
	}
}

// NewMemCache creates a new memcache adapter.
func NewMemCache() cache.Cache {
	return &Cache{}
}

// NewMemCacheV2 creates new memcache adapter.
func NewMemCacheV2(conn *memcache.Client, opts ...CacheOptions) cache.Cache {
	res := &Cache{
		conn: conn,
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

// Get get value from memcache.
func (rc *Cache) Get(ctx context.Context, key string) (interface{}, error) {
	if item, err := rc.conn.Get(key); err == nil {
		return item.Value, nil
	} else {
		return nil, berror.Wrapf(err, cache.MemCacheCurdFailed,
			"could not read data from memcache, please check your key, network and connection. Root cause: %s",
			err.Error())
	}
}

// GetMulti gets a value from a key in memcache.
func (rc *Cache) GetMulti(ctx context.Context, keys []string) ([]interface{}, error) {
	rv := make([]interface{}, len(keys))

	mv, err := rc.conn.GetMulti(keys)
	if err != nil {
		return rv, berror.Wrapf(err, cache.MemCacheCurdFailed,
			"could not read multiple key-values from memcache, "+
				"please check your keys, network and connection. Root cause: %s",
			err.Error())
	}

	keysErr := make([]string, 0)
	for i, ki := range keys {
		if _, ok := mv[ki]; !ok {
			keysErr = append(keysErr, fmt.Sprintf("key [%s] error: %s", ki, "key not exist"))
			continue
		}
		rv[i] = mv[ki].Value
	}

	if len(keysErr) == 0 {
		return rv, nil
	}
	return rv, berror.Error(cache.MultiGetFailed, strings.Join(keysErr, "; "))
}

// Put puts a value into memcache.
func (rc *Cache) Put(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	item := memcache.Item{Key: key, Expiration: int32(timeout / time.Second)}
	if v, ok := val.([]byte); ok {
		item.Value = v
	} else if str, ok := val.(string); ok {
		item.Value = []byte(str)
	} else {
		return berror.Errorf(cache.InvalidMemCacheValue,
			"the value must be string or byte[]. key: %s, value:%v", key, val)
	}
	return berror.Wrapf(rc.conn.Set(&item), cache.MemCacheCurdFailed,
		"could not put key-value to memcache, key: %s", key)
}

// Delete deletes a value in memcache.
func (rc *Cache) Delete(ctx context.Context, key string) error {
	return berror.Wrapf(rc.conn.Delete(key), cache.MemCacheCurdFailed,
		"could not delete key-value from memcache, key: %s", key)
}

// Incr increases counter.
func (rc *Cache) Incr(ctx context.Context, key string) error {
	_, err := rc.conn.Increment(key, 1)
	return berror.Wrapf(err, cache.MemCacheCurdFailed,
		"could not increase value for key: %s", key)
}

// Decr decreases counter.
func (rc *Cache) Decr(ctx context.Context, key string) error {
	_, err := rc.conn.Decrement(key, 1)
	return berror.Wrapf(err, cache.MemCacheCurdFailed,
		"could not decrease value for key: %s", key)
}

// IsExist checks if a value exists in memcache.
func (rc *Cache) IsExist(ctx context.Context, key string) (bool, error) {
	_, err := rc.Get(ctx, key)
	return err == nil, err
}

// ClearAll clears all cache in memcache.
func (rc *Cache) ClearAll(context.Context) error {
	return berror.Wrap(rc.conn.FlushAll(), cache.MemCacheCurdFailed,
		"try to clear all key-value pairs failed")
}
