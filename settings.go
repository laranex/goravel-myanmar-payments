package payments

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/config"
	contractshttp "github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
)

// ConfigKey is the name of the configuration added by config/myanmar_payments.go.
const ConfigKey = "myanmar_payments"

// DefaultFormRoutePath is the form route path used when
// myanmar_payments.form_route.path is empty or config/myanmar_payments.go is not
// published.
const DefaultFormRoutePath = "myanmar-payments/form"

// envKeys maps every environment variable the SDK's ConfigFromEnv functions read to
// the config key that holds it, so a published config file wins over the
// environment and an unpublished one falls back to the SDK's own variable names.
var envKeys = map[string]string{
	"KBZ_PAY_APP_ID":                "kbz_pay.app_id",
	"KBZ_PAY_APP_KEY":               "kbz_pay.app_key",
	"KBZ_PAY_MERCHANT_CODE":         "kbz_pay.merchant_code",
	"KBZ_PAY_BASE_URL":              "kbz_pay.api_url",
	"KBZ_PAY_PWA_BASE_REDIRECT_URL": "kbz_pay.pwa_url",

	"WAVE_MONEY_MERCHANT_ID":             "wave_money.merchant_id",
	"WAVE_MONEY_SECRET_KEY":              "wave_money.secret_key",
	"WAVE_MONEY_MERCHANT_NAME":           "wave_money.merchant_name",
	"WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS": "wave_money.time_to_live_in_seconds",
	"WAVE_MONEY_BASE_URL":                "wave_money.base_url",
	"WAVE_MONEY_AUTHENTICATE_URL":        "wave_money.authenticate_url",

	"AYA_PAY_APP_KEY":    "aya_pay.app_key",
	"AYA_PAY_APP_SECRET": "aya_pay.app_secret",
	"AYA_PAY_BASE_URL":   "aya_pay.base_url",

	"YOMA_MMQR_MERCHANT_ID":     "yoma_mmqr.merchant_id",
	"YOMA_MMQR_CLIENT_ID":       "yoma_mmqr.client_id",
	"YOMA_MMQR_CLIENT_SECRET":   "yoma_mmqr.client_secret",
	"YOMA_MMQR_WEBHOOK_HASHKEY": "yoma_mmqr.webhook_hashkey",
	"YOMA_MMQR_WEBHOOK_SECRET":  "yoma_mmqr.webhook_secret",
	"YOMA_MMQR_BASE_URL":        "yoma_mmqr.base_url",
	"YOMA_MMQR_API_VERSION":     "yoma_mmqr.api_version",

	"CYBER_SOURCE_PROFILE_ID": "cyber_source.profile_id",
	"CYBER_SOURCE_ACCESS_KEY": "cyber_source.access_key",
	"CYBER_SOURCE_SECRET_KEY": "cyber_source.secret_key",
	"CYBER_SOURCE_BASE_URL":   "cyber_source.base_url",

	"MYANMAR_PAYMENTS_HTTP_CLIENT":  "http.client",
	"MYANMAR_PAYMENTS_HTTP_TIMEOUT": "http.timeout",
	"MYANMAR_PAYMENTS_CACHE_STORE":  "cache_store",

	"MYANMAR_PAYMENTS_FORM_TTL_MINUTES": "form_route.ttl_minutes",
}

// settings reads the myanmar_payments configuration.
type settings struct {
	config config.Config
}

// Getenv returns the lookup function handed to the SDK's ConfigFromEnv functions: the
// published config value when it is set, otherwise the environment variable.
func (s settings) Getenv(name string) string {
	if key, ok := envKeys[name]; ok {
		if value := s.string(ConfigKey + "." + key); value != "" {
			return value
		}
	}

	return strings.TrimSpace(s.config.EnvString(name))
}

func (s settings) string(key string) string {
	switch value := s.config.Get(key).(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

// HTTPClientName is the Goravel HTTP client the gateways use; empty means the default client.
func (s settings) HTTPClientName() string { return s.Getenv("MYANMAR_PAYMENTS_HTTP_CLIENT") }

// HTTPTimeout is the timeout of every gateway call, or 0 when
// myanmar_payments.http.timeout is missing or not a whole number greater than 0.
// There is no default: the SDK then returns a *myanmarpayments.ConfigurationError
// for timeout_in_seconds when a gateway that calls an API is built.
func (s settings) HTTPTimeout() time.Duration {
	seconds, ok := wholeNumber(s.Getenv("MYANMAR_PAYMENTS_HTTP_TIMEOUT"))
	if !ok {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

// CacheStore is the cache store that keeps Yoma MMQR tokens; empty means the default store.
func (s settings) CacheStore() string { return s.Getenv("MYANMAR_PAYMENTS_CACHE_STORE") }

// FormRouteEnabled reports whether the auto-submit form route is registered.
func (s settings) FormRouteEnabled() bool {
	return s.config.GetBool(ConfigKey+".form_route.enabled", true)
}

// FormRoutePath is the route path, always starting with "/".
func (s settings) FormRoutePath() string {
	path := strings.Trim(s.config.GetString(ConfigKey+".form_route.path", DefaultFormRoutePath), "/ ")
	if path == "" {
		path = DefaultFormRoutePath
	}

	return "/" + path
}

// FormRouteTTL is how long a form link stays valid. It has no default: a missing
// or blank myanmar_payments.form_route.ttl_minutes (MYANMAR_PAYMENTS_FORM_TTL_MINUTES)
// returns a *myanmarpayments.ConfigurationError, and a value that is not a whole
// number greater than 0 returns one with Invalid set.
func (s settings) FormRouteTTL() (time.Duration, error) {
	minutes, ok := wholeNumber(s.Getenv("MYANMAR_PAYMENTS_FORM_TTL_MINUTES"))
	if !ok {
		return 0, &myanmarpayments.ConfigurationError{
			Gateway: "form_route",
			Key:     "ttl_minutes",
			Invalid: minutes < 0,
		}
	}

	return time.Duration(minutes) * time.Minute, nil
}

// wholeNumber parses a time setting the way the SDK does: blank text is missing
// (0, false); a whole number greater than 0 is valid; anything else is invalid
// (-1, false).
func wholeNumber(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if !wholeNumberPattern.MatchString(value) {
		return -1, false
	}
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return -1, false
	}

	return number, true
}

var wholeNumberPattern = regexp.MustCompile(`^[+-]?[0-9]+$`)

// FormRouteMiddleware is the middleware applied to the form route.
func (s settings) FormRouteMiddleware() []contractshttp.Middleware {
	middleware, _ := s.config.Get(ConfigKey + ".form_route.middleware").([]contractshttp.Middleware)

	return middleware
}

// FormRouteBaseURL is the absolute URL the form links start with: form_route.base_url,
// then http.url, then APP_URL.
func (s settings) FormRouteBaseURL() string {
	for _, key := range []string{ConfigKey + ".form_route.base_url", "http.url", "app.url"} {
		if value := s.string(key); value != "" {
			return strings.TrimRight(value, "/")
		}
	}

	return strings.TrimRight(s.config.EnvString("APP_URL", "http://localhost"), "/")
}
