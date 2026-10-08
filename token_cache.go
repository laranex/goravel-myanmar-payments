package payments

import (
	"time"

	"github.com/goravel/framework/contracts/cache"
)

// TokenCache is a myanmarpayments.TokenCache backed by a Goravel cache store, so
// Yoma MMQR access tokens are shared by every process using the same store.
type TokenCache struct {
	store cache.Driver
}

// NewTokenCache returns a TokenCache that stores tokens in store.
func NewTokenCache(store cache.Driver) *TokenCache {
	return &TokenCache{store: store}
}

// Get returns the cached token, if any.
func (c *TokenCache) Get(key string) (string, bool) {
	value := c.store.GetString(key, "")

	return value, value != ""
}

// Set stores value for ttl; a ttl of zero or less never expires.
func (c *TokenCache) Set(key, value string, ttl time.Duration) {
	if ttl <= 0 {
		c.store.Forever(key, value)

		return
	}
	_ = c.store.Put(key, value, ttl)
}

// Delete removes key.
func (c *TokenCache) Delete(key string) {
	c.store.Forget(key)
}
