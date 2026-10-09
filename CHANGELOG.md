# Changelog

All notable changes to `goravel-myanmar-payments` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-myanmar-payments`, built on `github.com/laranex/go-myanmar-payments/v4`. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-myanmar-payments/v4`.

### Added
- `ServiceProvider` binding a `*payments.Manager` singleton, and `facades.MyanmarPayments()`. `KbzPay()`, `WaveMoney()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return the SDK's own gateway types, built on first use; a missing credential returns the SDK's `*ConfigurationError`.
- `config/myanmar_payments.go` with the same keys and environment variables as laravel-myanmar-payments (`KBZ_PAY_*`, `WAVE_MONEY_*`, `AYA_PAY_*` with the `AYA_PGW_*` fallback, `YOMA_MMQR_*`, `CYBER_SOURCE_*`, `*_SANDBOX`, `MYANMAR_PAYMENTS_HTTP_TIMEOUT`, `MYANMAR_PAYMENTS_HTTP_CLIENT`, `MYANMAR_PAYMENTS_CACHE_STORE`), publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`. Without the published file the SDK's environment variables are read directly.
- `HTTPClient`: gateway calls go through Goravel's HTTP client, so `facades.Http().Fake()` and `AssertSent` work for them.
- `TokenCache`: Yoma MMQR access tokens live in a Goravel cache store and are shared between processes.
- `CallbackRequestFromContext(ctx)` and `Acknowledge(ctx, callback)` for gateway callbacks in Goravel handlers.
- The auto-submit form route (`myanmar_payments.form_route`, default `GET /myanmar-payments/form`) and `AutoSubmitURL(form)` for AYA Pay and CyberSource: the signed form travels in an encrypted (APP_KEY), expiring link.
- `setup` program for `./artisan package:install github.com/laranex/goravel-myanmar-payments/v4`.
- Agent skill in `skills/goravel-myanmar-payments`; install it with `npx skills add laranex/goravel-myanmar-payments`.

### Changed since the pre-releases
- Built on a go-myanmar-payments version where `myanmarpayments.StatusCancelled` (`"cancelled"`) is renamed to `StatusCanceled` (`"canceled"`); update code or stored statuses from `v4.0.0-alpha.1`.
