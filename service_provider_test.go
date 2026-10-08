package payments

import (
	"context"
	"errors"
	nethttp "net/http"
	"net/url"
	"testing"

	"github.com/goravel/framework/cache"
	"github.com/goravel/framework/contracts/binding"
	contractscache "github.com/goravel/framework/contracts/cache"
	"github.com/goravel/framework/contracts/foundation"
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	mockscache "github.com/goravel/framework/mocks/cache"
	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	mocksroute "github.com/goravel/framework/mocks/route"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// singleStoreCache is a cache facade whose only store is a Goravel memory store.
type singleStoreCache struct{ *cache.Memory }

func (c singleStoreCache) Store(string) contractscache.Driver { return c.Memory }

// registerWith runs ServiceProvider.Register on a mock application and returns the
// singleton callback it bound.
func registerWith(t *testing.T, app *mocksfoundation.Application) func(foundation.Application) (any, error) {
	t.Helper()
	var callback func(foundation.Application) (any, error)
	app.EXPECT().Singleton(Binding, mock.AnythingOfType("func(foundation.Application) (interface {}, error)")).
		Run(func(_ any, cb func(foundation.Application) (any, error)) { callback = cb }).Once()

	(&ServiceProvider{}).Register(app)
	require.NotNil(t, callback)

	return callback
}

func TestServiceProviderRelationship(t *testing.T) {
	relationship := (&ServiceProvider{}).Relationship()
	assert.Equal(t, []string{Binding}, relationship.Bindings)
	assert.Equal(t, []string{binding.Config, binding.Cache, binding.Crypt, binding.Http, binding.Route}, relationship.Dependencies)
}

func TestServiceProviderBindsAManagerWiredToGoravel(t *testing.T) {
	t.Cleanup(resetRegistered)
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	memory := newMemoryCache(t, config)
	app := mocksfoundation.NewApplication(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(config).Once()
	app.EXPECT().MakeHttp().Return(factory).Once()
	app.EXPECT().MakeCache().Return(singleStoreCache{memory}).Once()
	app.EXPECT().MakeCrypt().Return(newCrypt(t, config)).Once()

	instance, err := callback(app)
	require.NoError(t, err)
	manager, ok := instance.(*Manager)
	require.True(t, ok)

	// HTTP goes through Goravel's client: a fake answers the KBZ call.
	factory.Fake(map[string]any{"https://gateway.test/kbz/precreate": factory.Response().Json(200, map[string]any{"Response": map[string]any{
		"result": "SUCCESS", "code": "0", "prepay_id": "prepay-1",
	}})}).PreventStrayRequests()
	t.Cleanup(factory.Reset)
	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	payment, err := kbz.PWA(context.Background(), kbzpay.PaymentData{OrderID: "ORDER_1", Amount: myanmarpayments.Kyat(1000), CallbackURL: "https://shop.test/callback"})
	require.NoError(t, err)
	assert.Equal(t, "prepay-1", payment.GatewayReference)

	// Form links use the crypt facade.
	link, err := manager.AutoSubmitURL(&myanmarpayments.FormPayment{OrderID: "1", Action: "https://gateway.test/pay"})
	require.NoError(t, err)
	assert.Contains(t, link, "https://shop.test/myanmar-payments/form?payload=")

	// Tokens go to the Goravel cache.
	manager.tokenCache.Set("probe", "value", 0)
	assert.Equal(t, "value", memory.GetString("probe"))
}

func TestServiceProviderUsesTheConfiguredCacheStoreAndHTTPClient(t *testing.T) {
	t.Cleanup(resetRegistered)
	config := newConfig(t, map[string]any{ConfigKey: map[string]any{
		"cache_store": "redis",
		"http":        map[string]any{"client": "payments", "timeout": 9},
	}})
	app := mocksfoundation.NewApplication(t)
	cache := mockscache.NewCache(t)
	store := mockscache.NewDriver(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(config).Once()
	app.EXPECT().MakeHttp().Return(nil).Once()
	app.EXPECT().MakeCache().Return(cache).Once()
	app.EXPECT().MakeCrypt().Return(nil).Once()
	cache.EXPECT().Store("redis").Return(store).Once()
	store.EXPECT().Forget("probe").Return(true).Once()

	instance, err := callback(app)
	require.NoError(t, err)
	manager := instance.(*Manager)
	manager.tokenCache.Delete("probe")

	client, ok := manager.httpClient.(*HTTPClient)
	require.True(t, ok)
	assert.Equal(t, "payments", client.name)
	assert.Equal(t, 9.0, client.timeout.Seconds())
	assert.Nil(t, manager.crypt)
}

func TestServiceProviderWithoutOptionalFacades(t *testing.T) {
	t.Cleanup(resetRegistered)
	config := newConfig(t, nil)
	app := mocksfoundation.NewApplication(t)
	callback := registerWith(t, app)

	app.EXPECT().MakeConfig().Return(config).Once()
	app.EXPECT().MakeHttp().Return(nil).Once()
	app.EXPECT().MakeCache().Return(nil).Once()
	app.EXPECT().MakeCrypt().Return(nil).Once()

	instance, err := callback(app)
	require.NoError(t, err)
	manager := instance.(*Manager)
	assert.Nil(t, manager.tokenCache, "the SDK falls back to its in-memory cache")
	_, err = manager.AutoSubmitURL(&myanmarpayments.FormPayment{Action: "https://gateway.test"})
	assert.ErrorIs(t, err, ErrCryptNotAvailable)
}

func TestServiceProviderRequiresTheConfigFacade(t *testing.T) {
	t.Cleanup(resetRegistered)
	app := mocksfoundation.NewApplication(t)
	callback := registerWith(t, app)
	app.EXPECT().MakeConfig().Return(nil).Once()

	_, err := callback(app)
	assert.ErrorContains(t, err, "config facade is not registered")
}

func expectPublishes(app *mocksfoundation.Application) {
	app.EXPECT().ConfigPath("myanmar_payments.go").Return("/app/config/myanmar_payments.go").Once()
	app.EXPECT().Publishes(PackageName, map[string]string{
		"config/myanmar_payments.go": "/app/config/myanmar_payments.go",
	}, "goravel-myanmar-payments", "goravel-myanmar-payments-config").Once()
}

func TestServiceProviderBootPublishesConfigAndRegistersTheFormRoute(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	server := newTestServer(config)
	manager := newManager(t, config, nil, nil)
	app := mocksfoundation.NewApplication(t)
	route := mocksroute.NewRoute(t)

	expectPublishes(app)
	app.EXPECT().MakeConfig().Return(config).Once()
	app.EXPECT().MakeRoute().Return(route).Once()
	route.EXPECT().Get("/myanmar-payments/form", mock.Anything).RunAndReturn(func(path string, handler contractshttp.HandlerFunc) contractsroute.Action {
		return server.router.Get(path, handler)
	}).Once()
	(&ServiceProvider{}).Boot(app)

	// The registered handler resolves the manager from the container on each request.
	app.EXPECT().Make(Binding).Return(manager, nil).Once()
	link, err := manager.AutoSubmitURL(&myanmarpayments.FormPayment{OrderID: "1", Action: "https://gateway.test/pay", Fields: []myanmarpayments.FormField{{Name: "a", Value: "b"}}})
	require.NoError(t, err)
	parsed, err := url.Parse(link)
	require.NoError(t, err)
	response := server.do(t, nethttp.MethodGet, parsed.RequestURI(), "", "")
	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	assert.Contains(t, readBody(t, response), `<input type="hidden" name="a" value="b">`)

	app.EXPECT().Make(Binding).Return(nil, errors.New("not bound")).Once()
	response = server.do(t, nethttp.MethodGet, parsed.RequestURI(), "", "")
	assert.Equal(t, nethttp.StatusInternalServerError, response.StatusCode)
}

func TestServiceProviderBootAppliesTheFormRouteMiddleware(t *testing.T) {
	middleware := []contractshttp.Middleware{countingMiddleware{hits: new(int)}}
	config := newConfig(t, map[string]any{ConfigKey: map[string]any{"form_route": map[string]any{"middleware": middleware, "path": "checkout/form"}}})
	app := mocksfoundation.NewApplication(t)
	route := mocksroute.NewRoute(t)
	router := mocksroute.NewRouter(t)
	action := mocksroute.NewAction(t)

	expectPublishes(app)
	app.EXPECT().MakeConfig().Return(config).Once()
	app.EXPECT().MakeRoute().Return(route).Once()
	route.EXPECT().Middleware(middleware[0]).Return(router).Once()
	router.EXPECT().Get("/checkout/form", mock.Anything).Return(action).Once()
	action.EXPECT().Name(FormRouteName).Return(action).Once()

	(&ServiceProvider{}).Boot(app)
}

func TestServiceProviderBootSkipsADisabledFormRoute(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: map[string]any{"form_route": map[string]any{"enabled": false}}})
	app := mocksfoundation.NewApplication(t)

	expectPublishes(app)
	app.EXPECT().MakeConfig().Return(config).Once()

	(&ServiceProvider{}).Boot(app) // MakeRoute is never called
}

func TestServiceProviderBootWithoutConfigOrRoute(t *testing.T) {
	app := mocksfoundation.NewApplication(t)
	expectPublishes(app)
	app.EXPECT().MakeConfig().Return(nil).Once()
	(&ServiceProvider{}).Boot(app)

	app = mocksfoundation.NewApplication(t)
	expectPublishes(app)
	app.EXPECT().MakeConfig().Return(newConfig(t, nil)).Once()
	app.EXPECT().MakeRoute().Return(nil).Once()
	(&ServiceProvider{}).Boot(app)
}

func TestResolve(t *testing.T) {
	manager, err := NewManager(Options{Config: newConfig(t, nil)})
	require.NoError(t, err)

	tests := []struct {
		name     string
		instance any
		err      error
		wantErr  string
	}{
		{"bound manager", manager, nil, ""},
		{"container error", nil, errors.New("not bound"), "not bound"},
		{"wrong type", "nope", nil, "not *payments.Manager"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mocksfoundation.NewApplication(t)
			app.EXPECT().Make(Binding).Return(tt.instance, tt.err).Once()

			got, err := Resolve(app)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Same(t, manager, got)
		})
	}
}

func TestRegisteredAndPackageLevelAutoSubmitURL(t *testing.T) {
	resetRegistered()
	t.Cleanup(resetRegistered)

	_, err := Registered()
	assert.ErrorContains(t, err, "register &payments.ServiceProvider{}")
	_, err = AutoSubmitURL(&myanmarpayments.FormPayment{})
	assert.ErrorContains(t, err, "register &payments.ServiceProvider{}")

	config := newConfig(t, map[string]any{ConfigKey: credentials("")})
	manager := newManager(t, config, nil, nil)
	app := mocksfoundation.NewApplication(t)
	app.EXPECT().Make(Binding).Return(manager, nil).Twice()
	setRegistered(app)

	got, err := Registered()
	require.NoError(t, err)
	assert.Same(t, manager, got)

	link, err := AutoSubmitURL(&myanmarpayments.FormPayment{OrderID: "1", Action: "https://gateway.test/pay"})
	require.NoError(t, err)
	assert.Contains(t, link, "https://shop.test/myanmar-payments/form?payload=")
}
