package payments

import (
	"testing"
	"time"

	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/stretchr/testify/assert"
)

var _ myanmarpayments.TokenCache = (*TokenCache)(nil)
var _ myanmarpayments.HTTPDoer = (*HTTPClient)(nil)

func TestTokenCache(t *testing.T) {
	config := newConfig(t, nil)
	memory := newMemoryCache(t, config)
	tokens := NewTokenCache(memory)

	_, ok := tokens.Get("token")
	assert.False(t, ok)

	tokens.Set("token", "abc", time.Minute)
	value, ok := tokens.Get("token")
	assert.True(t, ok)
	assert.Equal(t, "abc", value)
	assert.Equal(t, "abc", memory.GetString("token"), "stored in the Goravel cache")

	tokens.Delete("token")
	_, ok = tokens.Get("token")
	assert.False(t, ok)
}

func TestTokenCacheExpiry(t *testing.T) {
	tokens := NewTokenCache(newMemoryCache(t, newConfig(t, nil)))

	tokens.Set("short", "abc", 50*time.Millisecond)
	tokens.Set("forever", "xyz", 0)
	time.Sleep(120 * time.Millisecond)

	_, ok := tokens.Get("short")
	assert.False(t, ok)
	value, ok := tokens.Get("forever")
	assert.True(t, ok)
	assert.Equal(t, "xyz", value)
}
