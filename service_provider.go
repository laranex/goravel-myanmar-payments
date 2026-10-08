package payments

import (
	"fmt"
	"sync/atomic"

	"github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
)

const (
	// Binding is the container key of the *Manager.
	Binding = "laranex.myanmar_payments"
	// PackageName is the name used by `./artisan vendor:publish --package=...`.
	PackageName = "github.com/laranex/goravel-myanmar-payments/v4"
)

type appHolder struct{ app foundation.Application }

var registeredApp atomic.Pointer[appHolder]

// ServiceProvider registers the payments *Manager with a Goravel application.
// Add &payments.ServiceProvider{} to bootstrap/providers.go.
type ServiceProvider struct{}

// Relationship declares the binding. The cache, crypt, HTTP client and route
// facades are optional: when they are registered, this provider boots after them.
func (r *ServiceProvider) Relationship() binding.Relationship {
	return binding.Relationship{
		Bindings:     []string{Binding},
		Dependencies: []string{binding.Config, binding.Cache, binding.Crypt, binding.Http, binding.Route},
	}
}

// Register binds the *Manager as a singleton. Its gateways send HTTP requests through
// Goravel's HTTP client (myanmar_payments.http.client, default client when empty) with
// myanmar_payments.http.timeout, keep Yoma MMQR tokens in the cache store
// myanmar_payments.cache_store (default store when empty) and sign form links with
// the crypt facade.
func (r *ServiceProvider) Register(app foundation.Application) {
	registeredApp.Store(&appHolder{app: app})

	app.Singleton(Binding, func(app foundation.Application) (any, error) {
		config := app.MakeConfig()
		if config == nil {
			return nil, fmt.Errorf("goravel-myanmar-payments: the config facade is not registered")
		}
		s := settings{config: config}

		options := Options{Config: config}
		options.HTTPClient = NewHTTPClient(app.MakeHttp(), s.HTTPClientName(), s.HTTPTimeout())
		if cache := app.MakeCache(); cache != nil {
			if store := s.CacheStore(); store != "" {
				options.TokenCache = NewTokenCache(cache.Store(store))
			} else {
				options.TokenCache = NewTokenCache(cache)
			}
		}
		if crypt := app.MakeCrypt(); crypt != nil {
			options.Crypt = crypt
		}

		return NewManager(options)
	})
}

// Boot registers config/myanmar_payments.go for `./artisan vendor:publish`
// (--package=github.com/laranex/goravel-myanmar-payments/v4) and, unless
// myanmar_payments.form_route.enabled is false, the auto-submit form route.
func (r *ServiceProvider) Boot(app foundation.Application) {
	app.Publishes(PackageName, map[string]string{
		"config/myanmar_payments.go": app.ConfigPath("myanmar_payments.go"),
	}, "goravel-myanmar-payments", "goravel-myanmar-payments-config")

	config := app.MakeConfig()
	if config == nil {
		return
	}
	s := settings{config: config}
	if !s.FormRouteEnabled() {
		return
	}
	router := app.MakeRoute()
	if router == nil {
		return
	}

	registerFormRoute(router, s, func(ctx http.Context) http.Response {
		manager, err := Resolve(app)
		if err != nil {
			return ctx.Response().String(500, "%s", err.Error())
		}

		return manager.ServeForm(ctx)
	})
}

// Resolve returns the *Manager bound in app's container.
func Resolve(app foundation.Application) (*Manager, error) {
	instance, err := app.Make(Binding)
	if err != nil {
		return nil, err
	}
	manager, ok := instance.(*Manager)
	if !ok {
		return nil, fmt.Errorf("goravel-myanmar-payments: binding %q is %T, not *payments.Manager", Binding, instance)
	}

	return manager, nil
}

// Registered returns the *Manager of the application the ServiceProvider was
// registered with. It returns an error when the provider is not registered.
func Registered() (*Manager, error) {
	holder := registeredApp.Load()
	if holder == nil {
		return nil, fmt.Errorf("goravel-myanmar-payments: register &payments.ServiceProvider{} in bootstrap/providers.go")
	}

	return Resolve(holder.app)
}

// AutoSubmitURL is Manager.AutoSubmitURL on the registered application's manager.
func AutoSubmitURL(form *myanmarpayments.FormPayment) (string, error) {
	manager, err := Registered()
	if err != nil {
		return "", err
	}

	return manager.AutoSubmitURL(form)
}
