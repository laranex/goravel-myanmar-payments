# Changelog

All notable changes to `goravel-myanmar-payments` will be documented in this file.

## v4.0.0 - Unreleased

Initial release, the Goravel counterpart of `laranex/laravel-myanmar-payments`, built on `github.com/laranex/go-myanmar-payments/v4`. The version number matches the other Laranex packages, so the module path carries the major version: import `github.com/laranex/goravel-myanmar-payments/v4`.

### Added
- `ServiceProvider` binding a `*payments.Manager` singleton, and `facades.MyanmarPayments()`. `KbzPay()`, `WavePay()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return the SDK's own gateway types, built on first use; a missing credential returns the SDK's `*ConfigurationError`.
- `config/myanmar_payments.go` with the same keys and environment variables as laravel-myanmar-payments (`KBZ_PAY_*`, `WAVE_PAY_*`, `AYA_PAY_*`, `YOMA_MMQR_*`, `CYBER_SOURCE_*`, `MYANMAR_PAYMENTS_HTTP_TIMEOUT`, `MYANMAR_PAYMENTS_FORM_TTL_MINUTES`, `MYANMAR_PAYMENTS_HTTP_CLIENT`, `MYANMAR_PAYMENTS_CACHE_STORE`), publishable with `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`. Without the published file the SDK's environment variables are read directly.
- Settings follow the SDK: every setting of a gateway you use is required and has no default, except the HTTP timeout (`http.timeout`, `MYANMAR_PAYMENTS_HTTP_TIMEOUT`, 30 seconds by default). That includes every gateway URL (`KBZ_PAY_BASE_URL`, `KBZ_PAY_PWA_BASE_REDIRECT_URL`, `WAVE_PAY_BASE_URL`, `WAVE_PAY_BASE_REDIRECT_URL`, `AYA_PAY_BASE_URL`, `YOMA_MMQR_BASE_URL`, `CYBER_SOURCE_BASE_URL`; production's or the gateway's UAT URLs), `KBZ_PAY_TIME_TO_LIVE_IN_MINUTES` (1 to 120) and `WAVE_PAY_TIME_TO_LIVE_IN_SECONDS` (a payment's own time to live overrides them), `YOMA_MMQR_WEBHOOK_SECRET` (every Yoma callback must carry it in `X-Webhook-Secret`) and `MYANMAR_PAYMENTS_FORM_TTL_MINUTES` for the form link; a missing or invalid setting returns the SDK's `*myanmarpayments.ConfigurationError` when the gateway is first used.
- `HTTPClient`: gateway calls go through Goravel's HTTP client, so `facades.Http().Fake()` and `AssertSent` work for them. Redirects are never followed, like every Laranex SDK; one shared timeout applies to every gateway.
- `TokenCache`: Yoma MMQR access tokens live in a Goravel cache store and are shared between processes.
- `CallbackRequestFromContext(ctx)` and `Acknowledge(ctx, callback)` for gateway callbacks in Goravel handlers; bodies over 64 KiB (`MaxCallbackBodyBytes`) return `ErrCallbackBodyTooLarge`.
- `Manager.HandleCallbackFromContext(name, ctx)` resolves the gateway (unknown name or configuration error), then reads the request (body limit), then verifies it, in the same order as laravel-myanmar-payments and nestjs-myanmar-payments.
- `Manager.Gateway(name)` and `Manager.HandleCallback(name, request)` resolve a gateway by its callback route name (`GatewayKbzPay` = `"kbz-pay"`, `"wave-pay"`, `"aya-pay"`, `"yoma-mmqr"`, `"cyber-source"`, all listed by `GatewayNames()`), so one route can serve every gateway, like laravel-myanmar-payments and nestjs-myanmar-payments; an unknown name returns `ErrUnknownGateway`.
- The auto-submit form route (`myanmar_payments.form_route`, default `GET /myanmar-payments/form`) and `AutoSubmitURL(form)` for AYA Pay and CyberSource: the signed form travels in an encrypted (APP_KEY), expiring link. The page is served with `Cache-Control: no-store` and `Referrer-Policy: strict-origin`; `form_route.nonce` (a `func(http.Context) string`) puts a CSP nonce on its inline script. `AutoSubmitURL` checks the route is enabled, then `ttl_minutes`, then the crypt facade; `Initiate()` never needs the route.
- `setup` program for `./artisan package:install github.com/laranex/goravel-myanmar-payments/v4`.
- Agent skill in `skills/goravel-myanmar-payments`; install it with `npx skills add laranex/goravel-myanmar-payments`.
