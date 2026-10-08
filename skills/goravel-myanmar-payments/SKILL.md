---
name: goravel-myanmar-payments
description: >
  Accept KBZ Pay, Wave Money, AYA Pay, Yoma MMQR and CyberSource payments in a Goravel application with github.com/laranex/goravel-myanmar-payments/v4: facade, config, callbacks, the auto-submit form route and HTTP fakes in tests.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Myanmar Payments

## When to use

Use this skill when Goravel code starts a payment, handles a gateway callback or checks a payment status with KBZ Pay, Wave Money, AYA Payment Gateway, Yoma MMQR or CyberSource Secure Acceptance. The package wraps the Go SDK `github.com/laranex/go-myanmar-payments/v4`: gateways, payment data, results and errors are the SDK's types.

## Install

- `go get github.com/laranex/goravel-myanmar-payments/v4` (Go 1.25+, Goravel 1.18+), then `./artisan package:install github.com/laranex/goravel-myanmar-payments/v4`: it registers `&payments.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/myanmar_payments.go` and appends the gateway variables to `.env.example`
- by hand: add the provider and run `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`
- the crypt (`APP_KEY`), cache and HTTP client facades are used when registered

## Configure

- set only the gateways you use in `.env`; names are the SDK's:
  - KBZ Pay: `KBZ_PAY_APP_ID`, `KBZ_PAY_APP_KEY`, `KBZ_PAY_MERCHANT_CODE`
  - Wave Money: `WAVE_MONEY_MERCHANT_ID`, `WAVE_MONEY_SECRET_KEY`, `WAVE_MONEY_MERCHANT_NAME` (defaults to `APP_NAME`)
  - AYA Pay: `AYA_PAY_APP_KEY`, `AYA_PAY_APP_SECRET` (`AYA_PGW_*` also read)
  - Yoma MMQR: `YOMA_MMQR_MERCHANT_ID`, `YOMA_MMQR_CLIENT_ID`, `YOMA_MMQR_CLIENT_SECRET`, `YOMA_MMQR_WEBHOOK_HASHKEY`, optional `YOMA_MMQR_WEBHOOK_SECRET`
  - CyberSource: `CYBER_SOURCE_PROFILE_ID`, `CYBER_SOURCE_ACCESS_KEY`, `CYBER_SOURCE_SECRET_KEY`
- every gateway has `<PREFIX>_SANDBOX` (default `true`); set `false` with production credentials to go live
- `MYANMAR_PAYMENTS_HTTP_TIMEOUT` (seconds, default 30), `MYANMAR_PAYMENTS_HTTP_CLIENT` (an `http.clients` name), `MYANMAR_PAYMENTS_CACHE_STORE` (Yoma token store)
- `myanmar_payments.form_route`: `enabled`, `path` (`myanmar-payments/form`), `middleware` (`[]http.Middleware`), `ttl_minutes` (30), `base_url` (`APP_URL`)

## Use

Imports:

```go
import (
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	payments "github.com/laranex/goravel-myanmar-payments/v4"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)
```

### Get a gateway

- `paymentsfacades.MyanmarPayments()` returns the `*payments.Manager`; `KbzPay()`, `WaveMoney()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return `(*kbzpay.Gateway, error)` and so on
- the error is a `*myanmarpayments.ConfigurationError` naming the missing setting; handle it, the facade does not panic for it
- gateways are built once and reused; pass `ctx` (Goravel's `http.Context` is a `context.Context`) to calls that hit the network

### Start a payment

```go
kbz, err := paymentsfacades.MyanmarPayments().KbzPay()
if err != nil {
	return ctx.Response().String(http.StatusInternalServerError, "%s", err.Error())
}
payment, err := kbz.PWA(ctx, kbzpay.PaymentData{
	OrderID:     "ORDER_1",
	Amount:      myanmarpayments.Kyat(1000),
	CallbackURL: "https://shop.test/payments/callback/kbzpay",
})
if err != nil {
	return ctx.Response().String(http.StatusBadGateway, "%s", err.Error())
}
return ctx.Response().Redirect(http.StatusFound, payment.URL)
```

- KBZ Pay: `PWA` (redirect), `QR` (`QRString`), `App` (`AppPayment` for the mobile SDK), `Status(ctx, orderID)`
- Wave Money: `Initiate(ctx, &wavemoney.PaymentData{...})` (pointer; it fills `MerchantReferenceID`), redirect to `payment.URL`; no status API
- Yoma MMQR: `Initiate(ctx, yomammqr.PaymentData{...})` returns a `QrPayment` with `QRImageDataURI("")` and `Reference`; `RenewQR`, `Status(ctx, reference)`
- amounts are `myanmarpayments.Amount`: `Kyat(1000)`, `ParseAmount("1000.50")`; never `float64`

### Form payments (AYA Pay, CyberSource)

```go
aya, err := paymentsfacades.MyanmarPayments().AyaPay()
form, err := aya.Initiate(ayapay.PaymentData{OrderID: "ORD123456", Amount: myanmarpayments.Kyat(1000), Channel: "aya_pay", Method: ayapay.MethodQR, ReturnURL: "https://shop.test/payments/aya-pay/done"})
link, err := payments.AutoSubmitURL(form) // or paymentsfacades.MyanmarPayments().AutoSubmitURL(form)
return ctx.Response().Redirect(http.StatusFound, link)
```

- the link points at the form route, carries the signed form encrypted with `APP_KEY` and expires after `ttl_minutes`; an invalid or expired link answers 410
- `AutoSubmitURL` returns `payments.ErrFormRouteDisabled` when the route is off and `payments.ErrCryptNotAvailable` without the crypt facade; you can still render `form.HTML()` yourself
- AYA's return URL: verify with `aya.VerifyRedirect(request)` to show the result; fulfill orders from the backend callback

### Handle callbacks

```go
func (c *PaymentController) KbzCallback(ctx http.Context) http.Response {
	request, err := payments.CallbackRequestFromContext(ctx)
	if err != nil {
		return ctx.Response().String(http.StatusBadRequest, "bad request")
	}
	kbz, err := paymentsfacades.MyanmarPayments().KbzPay()
	if err != nil {
		return ctx.Response().String(http.StatusInternalServerError, "not configured")
	}
	callback, err := kbz.HandleCallback(request)
	if err != nil { // *myanmarpayments.SignatureVerificationError
		return ctx.Response().String(http.StatusBadRequest, "invalid signature")
	}
	if callback.IsSuccessful() {
		// compare callback.Amount with the order, then fulfill callback.OrderID once
	}
	return payments.Acknowledge(ctx, callback) // KBZ Pay gets its plain "success"
}
```

- register callback routes as `POST` without CSRF or auth middleware; gateways retry until acknowledged, so make fulfillment idempotent
- `callback.Status` is gateway-independent (`myanmarpayments.StatusSuccessful`, `StatusPending`, `StatusFailed`, `StatusCancelled`, `StatusExpired`, `StatusUnknown`); `GatewayStatus` and `Raw` keep the gateway's values
- `Status(ctx, ...)` on KBZ Pay, AYA Pay and Yoma MMQR returns a `*myanmarpayments.PaymentStatusResult` for polling

## Test your app

- gateway HTTP calls go through Goravel's HTTP client, so fake them by URL on the `client.Factory` (`facades.Http()`; if your app's `facades.Http()` returns `client.Request`, use `facades.App().MakeHttp()`):

```go
facades.Http().Fake(map[string]any{
	"http://api-uat.kbzpay.com/payment/gateway/uat/precreate": facades.Http().Response().Json(200, map[string]any{
		"Response": map[string]any{"result": "SUCCESS", "code": "0", "prepay_id": "prepay-1"},
	}),
}).PreventStrayRequests()
defer facades.Http().Reset()
```

- assert with `facades.Http().AssertSent(func(r client.Request) bool { ... })`
- test callbacks by posting a correctly signed payload to your route; `kbzpay.NewSigner(appKey).Sign(fields)` signs KBZ Pay fields
- CyberSource signs forms locally and makes no HTTP calls

## Avoid

- do not build gateways with the SDK's `New` inside handlers; use the facade so HTTP fakes, the shared token cache and config apply
- do not read the callback body before `CallbackRequestFromContext` or rebuild it from `ctx.Request().All()`; signatures need the original values
- do not fulfill an order from the browser return URL alone or without comparing `callback.Amount`
- do not return your own response body to KBZ Pay callbacks; use `payments.Acknowledge`
- do not log or commit gateway secrets
