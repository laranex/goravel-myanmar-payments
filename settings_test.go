package payments

import (
	"os"
	"regexp"
	"testing"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopMiddleware struct{}

func (noopMiddleware) Signature() string { return "noop" }

func (noopMiddleware) Handle(ctx contractshttp.Context) { ctx.Request().Next() }

func TestGetenvPrefersThePublishedConfig(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	t.Setenv("KBZ_PAY_APP_ID", "from-env")
	s := settings{config: config}

	assert.Equal(t, "kp-app", s.Getenv("KBZ_PAY_APP_ID"))
	assert.Equal(t, "https://gateway.test/kbz", s.Getenv("KBZ_PAY_BASE_URL"))
	assert.Equal(t, "30", s.Getenv("MYANMAR_PAYMENTS_HTTP_TIMEOUT"))
	assert.Equal(t, "30", s.Getenv("MYANMAR_PAYMENTS_FORM_TTL_MINUTES"))
	assert.Equal(t, "300", s.Getenv("WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS"))
}

func TestGetenvFallsBackToTheEnvironment(t *testing.T) {
	config := newConfig(t, nil)
	t.Setenv("KBZ_PAY_APP_ID", " from-env ")
	t.Setenv("AYA_PGW_APP_KEY", "legacy-key")
	s := settings{config: config}

	assert.Equal(t, "from-env", s.Getenv("KBZ_PAY_APP_ID"))
	assert.Equal(t, "", s.Getenv("AYA_PAY_APP_KEY"))
	assert.Equal(t, "legacy-key", s.Getenv("AYA_PGW_APP_KEY"))
	assert.Equal(t, "", s.Getenv("APP_NAME"), "app.name is not a fallback for any setting")
	assert.Equal(t, "", s.Getenv("UNKNOWN_VARIABLE"))
}

func TestGetenvFallsBackWhenAPublishedValueIsEmpty(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: map[string]any{"kbz_pay": map[string]any{"app_id": ""}}})
	t.Setenv("KBZ_PAY_APP_ID", "from-env")

	assert.Equal(t, "from-env", settings{config: config}.Getenv("KBZ_PAY_APP_ID"))
}

func TestHTTPAndCacheSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		s := settings{config: newConfig(t, nil)}
		assert.Equal(t, "", s.HTTPClientName())
		assert.Equal(t, time.Duration(0), s.HTTPTimeout(), "no default timeout")
		assert.Equal(t, "", s.CacheStore())
	})
	t.Run("config", func(t *testing.T) {
		s := settings{config: newConfig(t, map[string]any{ConfigKey: map[string]any{
			"http":        map[string]any{"client": "payments", "timeout": 12},
			"cache_store": "redis",
		}})}
		assert.Equal(t, "payments", s.HTTPClientName())
		assert.Equal(t, 12*time.Second, s.HTTPTimeout())
		assert.Equal(t, "redis", s.CacheStore())
	})
	t.Run("environment", func(t *testing.T) {
		config := newConfig(t, nil)
		t.Setenv("MYANMAR_PAYMENTS_HTTP_TIMEOUT", "7")
		t.Setenv("MYANMAR_PAYMENTS_CACHE_STORE", "file")
		s := settings{config: config}
		assert.Equal(t, 7*time.Second, s.HTTPTimeout())
		assert.Equal(t, "file", s.CacheStore())
	})
	t.Run("invalid timeout", func(t *testing.T) {
		for _, value := range []string{"soon", "0", "-3", "1.5", "99999999999999999999999"} {
			config := newConfig(t, nil)
			t.Setenv("MYANMAR_PAYMENTS_HTTP_TIMEOUT", value)
			assert.Equal(t, time.Duration(0), settings{config: config}.HTTPTimeout(), value)
		}
	})
}

func TestFormRouteSettings(t *testing.T) {
	t.Run("defaults without a published config", func(t *testing.T) {
		config := newConfig(t, nil)
		t.Setenv("APP_URL", "https://env.test/")
		s := settings{config: config}
		assert.True(t, s.FormRouteEnabled())
		assert.Equal(t, "/myanmar-payments/form", s.FormRoutePath())
		assert.Empty(t, s.FormRouteMiddleware())
		assert.Equal(t, "https://env.test", s.FormRouteBaseURL())
	})
	t.Run("http.url is used before APP_URL", func(t *testing.T) {
		s := settings{config: newConfig(t, map[string]any{"http": map[string]any{"url": "https://http.test"}})}
		assert.Equal(t, "https://http.test", s.FormRouteBaseURL())
	})
	t.Run("published values", func(t *testing.T) {
		middleware := []contractshttp.Middleware{noopMiddleware{}}
		s := settings{config: newConfig(t, map[string]any{ConfigKey: map[string]any{"form_route": map[string]any{
			"enabled": false, "path": "/pay/form/", "ttl_minutes": 5, "middleware": middleware, "base_url": "https://shop.test/",
		}}})}
		assert.False(t, s.FormRouteEnabled())
		assert.Equal(t, "/pay/form", s.FormRoutePath())
		ttl, err := s.FormRouteTTL()
		require.NoError(t, err)
		assert.Equal(t, 5*time.Minute, ttl)
		assert.Equal(t, middleware, s.FormRouteMiddleware())
		assert.Equal(t, "https://shop.test", s.FormRouteBaseURL())
	})
	t.Run("blank path falls back", func(t *testing.T) {
		s := settings{config: newConfig(t, map[string]any{ConfigKey: map[string]any{"form_route": map[string]any{"path": "/"}}})}
		assert.Equal(t, "/myanmar-payments/form", s.FormRoutePath())
	})
	t.Run("ttl from the environment", func(t *testing.T) {
		config := newConfig(t, nil)
		t.Setenv("MYANMAR_PAYMENTS_FORM_TTL_MINUTES", " 15 ")
		ttl, err := settings{config: config}.FormRouteTTL()
		require.NoError(t, err)
		assert.Equal(t, 15*time.Minute, ttl)
	})
	t.Run("ttl is required", func(t *testing.T) {
		for value, invalid := range map[any]bool{nil: false, "": false, " ": false, 0: true, -5: true, "five": true, "1.5": true} {
			s := settings{config: newConfig(t, map[string]any{ConfigKey: map[string]any{"form_route": map[string]any{"ttl_minutes": value}}})}
			_, err := s.FormRouteTTL()
			assert.Equal(t, &myanmarpayments.ConfigurationError{Gateway: "form_route", Key: "ttl_minutes", Invalid: invalid}, err, value)
		}
	})
}

// TestEnvKeysMatchThePublishedConfig keeps the env-to-config table in sync with
// config/myanmar_payments.go: every config.Env("NAME") entry maps to its own key.
func TestEnvKeysMatchThePublishedConfig(t *testing.T) {
	source, err := os.ReadFile("config/myanmar_payments.go")
	require.NoError(t, err)

	section := regexp.MustCompile(`^\t\t"(\w+)": map\[string\]any\{`)
	entry := regexp.MustCompile(`^\t+"(\w+)":\s+config\.Env\("(\w+)"`)
	found := map[string]string{}
	current := ""
	for _, line := range regexp.MustCompile("\n").Split(string(source), -1) {
		if match := section.FindStringSubmatch(line); match != nil {
			current = match[1] + "."
			continue
		}
		if match := entry.FindStringSubmatch(line); match != nil {
			key := match[1]
			if match[1] != "cache_store" {
				key = current + match[1]
			}
			if match[2] == "APP_URL" {
				continue // form_route.base_url is read directly, not through the SDK
			}
			found[match[2]] = key
		}
	}

	assert.Equal(t, envKeys, found)
}

// TestPublicNames pins the names and defaults the documentation lists.
func TestPublicNames(t *testing.T) {
	assert.Equal(t, "laranex.myanmar_payments", Binding)
	assert.Equal(t, "github.com/laranex/goravel-myanmar-payments/v4", PackageName)
	assert.Equal(t, "myanmar_payments", ConfigKey)
	assert.Equal(t, "myanmar-payments.form", FormRouteName)
	assert.Equal(t, "myanmar-payments/form", DefaultFormRoutePath)
}
