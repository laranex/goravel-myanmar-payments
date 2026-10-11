# Changelog

All notable changes to `goravel-myanmar-payments` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-myanmar-payments`, built on `github.com/laranex/go-myanmar-payments/v4`. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-myanmar-payments/v4`.

### Added
- `ServiceProvider` binding a `*payments.Manager` singleton, and `facades.MyanmarPayments()`. `KbzPay()`, `WaveMoney()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return the SDK's own gateway types, built on first use; a missing credential returns the SDK's `*ConfigurationError`.
- `config/myanmar_payments.go` with the same keys and environment variables as laravel-myanmar-payments (`KBZ_PAY_*`, `WAVE_MONEY_*`, `AYA_PAY_*` with the `AYA_PGW_*` fallback, `YOMA_MMQR_*`, `CYBER_SOURCE_*`, `MYANMAR_PAYMENTS_HTTP_TIMEOUT`, `MYANMAR_PAYMENTS_FORM_TTL_MINUTES`, `MYANMAR_PAYMENTS_HTTP_CLIENT`, `MYANMAR_PAYMENTS_CACHE_STORE`), publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`. Without the published file the SDK's environment variables are read directly.
- Settings follow the SDK: every gateway uses its production endpoints (the URL settings are optional overrides); the HTTP timeout (`http.timeout`, `MYANMAR_PAYMENTS_HTTP_TIMEOUT`) defaults to 30 seconds; `KBZ_PAY_TIME_TO_LIVE_IN_MINUTES` (1 to 120) and `WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS` are optional defaults that a payment's own time to live overrides, and a payment with neither returns `*myanmarpayments.InvalidPaymentDataError` before any request. Every other setting of a gateway you use is required, including `MYANMAR_PAYMENTS_FORM_TTL_MINUTES` for the form link; a missing or invalid setting returns the SDK's `*myanmarpayments.ConfigurationError` when the gateway is first used.
- `HTTPClient`: gateway calls go through Goravel's HTTP client, so `facades.Http().Fake()` and `AssertSent` work for them.
- `TokenCache`: Yoma MMQR access tokens live in a Goravel cache store and are shared between processes.
- `CallbackRequestFromContext(ctx)` and `Acknowledge(ctx, callback)` for gateway callbacks in Goravel handlers.
- `Manager.Gateway(name)` and `Manager.HandleCallback(name, request)` resolve a gateway by its callback route name (`GatewayKbzPay` = `"kbz-pay"`, `"wave-money"`, `"aya-pay"`, `"yoma-mmqr"`, `"cyber-source"`, all listed by `GatewayNames()`), so one route can serve every gateway, like laravel-myanmar-payments and nestjs-myanmar-payments; an unknown name returns `ErrUnknownGateway`.
- The auto-submit form route (`myanmar_payments.form_route`, default `GET /myanmar-payments/form`) and `AutoSubmitURL(form)` for AYA Pay and CyberSource: the signed form travels in an encrypted (APP_KEY), expiring link.
- `setup` program for `./artisan package:install github.com/laranex/goravel-myanmar-payments/v4`.
- Agent skill in `skills/goravel-myanmar-payments`; install it with `npx skills add laranex/goravel-myanmar-payments`.
