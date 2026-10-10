package payments

import (
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ginpkg "github.com/gin-gonic/gin"
	"github.com/goravel/framework/cache"
	frameworkconfig "github.com/goravel/framework/config"
	contractsconfig "github.com/goravel/framework/contracts/config"
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsclient "github.com/goravel/framework/contracts/http/client"
	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/crypt"
	"github.com/goravel/framework/foundation/json"
	"github.com/goravel/framework/http/client"
	goravelgin "github.com/goravel/gin"
	"github.com/stretchr/testify/require"
)

const testAppKey = "abcdefghijklmnopqrstuvwxyz123456"

// gatewayEnv lists every variable a gateway reads, so tests start from a clean environment.
var gatewayEnv = []string{
	"APP_NAME", "APP_URL",
	"KBZ_PAY_APP_ID", "KBZ_PAY_APP_KEY", "KBZ_PAY_MERCHANT_CODE", "KBZ_PAY_BASE_URL", "KBZ_PAY_PWA_BASE_REDIRECT_URL",
	"WAVE_MONEY_MERCHANT_ID", "WAVE_MONEY_SECRET_KEY", "WAVE_MONEY_MERCHANT_NAME", "WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS", "WAVE_MONEY_BASE_URL", "WAVE_MONEY_AUTHENTICATE_URL",
	"AYA_PAY_APP_KEY", "AYA_PAY_APP_SECRET", "AYA_PAY_BASE_URL", "AYA_PGW_APP_KEY", "AYA_PGW_APP_SECRET", "AYA_PGW_BASE_URL",
	"YOMA_MMQR_MERCHANT_ID", "YOMA_MMQR_CLIENT_ID", "YOMA_MMQR_CLIENT_SECRET", "YOMA_MMQR_WEBHOOK_HASHKEY", "YOMA_MMQR_WEBHOOK_SECRET", "YOMA_MMQR_BASE_URL", "YOMA_MMQR_API_VERSION",
	"CYBER_SOURCE_PROFILE_ID", "CYBER_SOURCE_ACCESS_KEY", "CYBER_SOURCE_SECRET_KEY", "CYBER_SOURCE_BASE_URL",
	"MYANMAR_PAYMENTS_HTTP_CLIENT", "MYANMAR_PAYMENTS_HTTP_TIMEOUT", "MYANMAR_PAYMENTS_CACHE_STORE", "MYANMAR_PAYMENTS_FORM_TTL_MINUTES",
}

// newConfig returns a real Goravel config with app.key set and every gateway variable unset.
// Tests using it cannot run in parallel (t.Setenv).
func newConfig(t *testing.T, sections map[string]any) contractsconfig.Config {
	t.Helper()
	t.Setenv("APP_KEY", testAppKey)
	for _, name := range gatewayEnv {
		t.Setenv(name, "")
	}

	config := frameworkconfig.NewApplication("")
	config.Add("app", map[string]any{"key": testAppKey, "name": "Goravel"})
	config.Add("cache", map[string]any{"prefix": "test"})
	for name, section := range sections {
		config.Add(name, section)
	}

	return config
}

// credentials is a published myanmar_payments section with every gateway configured.
func credentials(baseURL string) map[string]any {
	return map[string]any{
		"kbz_pay":      map[string]any{"app_id": "kp-app", "app_key": "kbz-secret", "merchant_code": "200001", "api_url": baseURL + "/kbz", "pwa_url": ""},
		"wave_money":   map[string]any{"merchant_id": "wave-merchant", "secret_key": "wave-secret", "merchant_name": "Shop", "time_to_live_in_seconds": 300, "base_url": baseURL + "/wave"},
		"aya_pay":      map[string]any{"app_key": "aya-key", "app_secret": "aya-secret", "base_url": baseURL + "/aya"},
		"yoma_mmqr":    map[string]any{"merchant_id": "yoma-merchant", "client_id": "yoma-client", "client_secret": "yoma-secret", "webhook_hashkey": "yoma-hash", "webhook_secret": "", "base_url": baseURL + "/yoma", "api_version": "v1rc"},
		"cyber_source": map[string]any{"profile_id": "profile", "access_key": "access", "secret_key": "cyber-secret", "base_url": ""},
		"http":         map[string]any{"timeout": 30},
		"form_route":   map[string]any{"enabled": true, "path": "myanmar-payments/form", "ttl_minutes": 30, "base_url": "https://shop.test"},
	}
}

func newCrypt(t *testing.T, config contractsconfig.Config) *crypt.AES {
	t.Helper()
	aes, err := crypt.NewAES(config, json.New())
	require.NoError(t, err)

	return aes
}

func newHTTPFactory(t *testing.T, config contractsconfig.Config) *client.Factory {
	t.Helper()
	factory, err := client.NewFactory(&client.FactoryConfig{
		Default: "default",
		Clients: map[string]client.Config{"default": {Timeout: 5 * time.Second}},
	}, config, json.New(), nil)
	require.NoError(t, err)

	return factory
}

func newMemoryCache(t *testing.T, config contractsconfig.Config) *cache.Memory {
	t.Helper()
	memory, err := cache.NewMemory(config)
	require.NoError(t, err)

	return memory
}

// newManager returns a Manager wired like the ServiceProvider wires it, with Goravel's
// HTTP client factory, memory cache and AES crypt.
func newManager(t *testing.T, config contractsconfig.Config, factory *client.Factory, now func() time.Time) *Manager {
	t.Helper()
	s := settings{config: config}
	var httpFactory contractsclient.Factory
	if factory != nil {
		httpFactory = factory
	}
	manager, err := NewManager(Options{
		Config:     config,
		HTTPClient: NewHTTPClient(httpFactory, s.HTTPClientName(), s.HTTPTimeout()),
		TokenCache: NewTokenCache(newMemoryCache(t, config)),
		Crypt:      newCrypt(t, config),
		Now:        now,
	})
	require.NoError(t, err)

	return manager
}

// testServer is a real Goravel router (gin driver) behind an httptest server.
type testServer struct {
	router contractsroute.Router
	engine *ginpkg.Engine
}

func newTestServer(config contractsconfig.Config) *testServer {
	ginpkg.SetMode(ginpkg.ReleaseMode)
	engine := ginpkg.New()
	router := goravelgin.NewGroup(config, engine.Group("/"), "", []contractshttp.Middleware{}, []contractshttp.Middleware{goravelgin.ResponseMiddleware()})

	return &testServer{router: router, engine: engine}
}

func (s *testServer) do(t *testing.T, method, target, contentType, body string) *nethttp.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	s.engine.ServeHTTP(recorder, request)

	return recorder.Result()
}

func readBody(t *testing.T, response *nethttp.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return string(body)
}
