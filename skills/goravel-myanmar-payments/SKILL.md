---
name: goravel-myanmar-payments
description: >
  Integrate Myanmar payment gateways (KBZ Pay, Wave Money, AYA Pay, Yoma MMQR, CyberSource) in a Goravel app with github.com/laranex/goravel-myanmar-payments/v4.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Myanmar Payments

## When to use

Use this skill when a Goravel application starts a payment, handles a gateway callback or checks a payment status with KBZ Pay, Wave Money, AYA Payment Gateway, Yoma MMQR or CyberSource. Start payments and verify callbacks with the package's typed API; never build gateway signatures by hand. The package wraps the Go SDK `github.com/laranex/go-myanmar-payments/v4`: gateways, payment data, results, statuses and errors are the SDK's types.

## Install

```bash
go get github.com/laranex/goravel-myanmar-payments/v4
./artisan package:install github.com/laranex/goravel-myanmar-payments/v4
```

Requires Go 1.25+ and Goravel 1.18+. `package:install` registers `&payments.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/myanmar_payments.go` and appends the gateway variables to `.env.example`. Gateway calls go through Goravel's HTTP client, Yoma MMQR tokens through the cache, and auto-submit form links are encrypted with the crypt facade (`APP_KEY`).

## Configure

Set only the env keys of the gateways you use. Every gateway runs against its sandbox until `<PREFIX>_SANDBOX=false`.

- KBZ Pay: `KBZ_PAY_APP_ID`, `KBZ_PAY_APP_KEY`, `KBZ_PAY_MERCHANT_CODE`, `KBZ_PAY_SANDBOX`
- Wave Money: `WAVE_MONEY_MERCHANT_ID`, `WAVE_MONEY_SECRET_KEY`, `WAVE_MONEY_MERCHANT_NAME` (defaults to `APP_NAME`), `WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS`, `WAVE_MONEY_SANDBOX`
- AYA Pay: `AYA_PAY_APP_KEY`, `AYA_PAY_APP_SECRET`, `AYA_PAY_SANDBOX` (`AYA_PGW_*` also read)
- Yoma MMQR: `YOMA_MMQR_MERCHANT_ID`, `YOMA_MMQR_CLIENT_ID`, `YOMA_MMQR_CLIENT_SECRET`, `YOMA_MMQR_WEBHOOK_HASHKEY`, optional `YOMA_MMQR_WEBHOOK_SECRET`, `YOMA_MMQR_SANDBOX`
- CyberSource: `CYBER_SOURCE_PROFILE_ID`, `CYBER_SOURCE_ACCESS_KEY`, `CYBER_SOURCE_SECRET_KEY`, `CYBER_SOURCE_SANDBOX`
- Shared: `MYANMAR_PAYMENTS_HTTP_TIMEOUT` (seconds, default 30), `MYANMAR_PAYMENTS_HTTP_CLIENT` (an `http.clients` name, default client when empty), `MYANMAR_PAYMENTS_CACHE_STORE` (store for the Yoma access token, default store when empty)

Edit `config/myanmar_payments.go` only to change it, for example the auto-submit form route (`form_route.enabled`, `path`, `middleware`, `ttl_minutes`, `base_url`). Without `package:install`, publish it with `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`.

An unconfigured gateway returns a `*myanmarpayments.ConfigurationError` naming the missing key when first requested, not at boot.

## Use

Every gateway is reached through `paymentsfacades.MyanmarPayments()` (`github.com/laranex/goravel-myanmar-payments/v4/facades`): `KbzPay()`, `WaveMoney()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return the SDK gateway and an error, or `Gateway("kbz-pay")` by name (`payments.GatewayNames()` lists `kbz-pay`, `wave-money`, `aya-pay`, `yoma-mmqr`, `cyber-source`). Each gateway is built once and reused. Goravel's `http.Context` is a `context.Context`: pass `ctx` to calls that hit the network.

### Amounts

Pass a `myanmarpayments.Amount` (`myanmarpayments.Kyat(10000)`, `myanmarpayments.ParseAmount("10000.50")`), never a `float64`. Only KBZ Pay (up to 2 decimals) and CyberSource accept decimals. Invalid data returns a `*myanmarpayments.InvalidPaymentDataError`; read the messages from its `Errors`.

### Start a payment

```go
import (
	"fmt"

	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)

kbz, err := paymentsfacades.MyanmarPayments().KbzPay()
if err != nil {
	return ctx.Response().String(http.StatusInternalServerError, "%s", err.Error())
}
data := kbzpay.PaymentData{
	OrderID:     fmt.Sprintf("ORDER_%d", order.ID),
	Amount:      myanmarpayments.Kyat(10000),
	CallbackURL: "https://shop.test/payments/kbz/callback",
}
payment, err := kbz.PWA(ctx, data)
if err != nil {
	return ctx.Response().String(http.StatusBadGateway, "%s", err.Error())
}
return ctx.Response().Redirect(http.StatusFound, payment.URL)
```

- `RedirectPayment` from `KbzPay().PWA()` and `WaveMoney().Initiate()`: redirect to `payment.URL`. Wave takes `&wavemoney.PaymentData{...}` and fills `MerchantReferenceID`; store it with the order.
- `QrPayment` from `KbzPay().QR()` (encode `QRString`) and `YomaMmqr().Initiate()` (`QRImage` as base64, `QRImageDataURI("")`, `ExpiresAt`, `Reference`). Renew an expired Yoma QR with `RenewQR(ctx, orderID)`.
- `AppPayment` from `KbzPay().App()`: return it as JSON to the mobile app.

### Form payments (AYA Pay, CyberSource)

`AyaPay().Initiate()` and `CyberSource().Initiate()` return a `*myanmarpayments.FormPayment` the customer's browser must POST. Redirect to `payments.AutoSubmitURL(form)`: a link to the package's `GET myanmar-payments/form` route, encrypted with `APP_KEY` and valid for `form_route.ttl_minutes` (30); a tampered or expired link answers 410. Or send `form.HTML()` yourself.

AYA Pay needs a channel: list them with `aya.Services(ctx)`, then:

```go
import (
	"fmt"

	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/ayapay"
	payments "github.com/laranex/goravel-myanmar-payments/v4"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)

aya, err := paymentsfacades.MyanmarPayments().AyaPay()
// handle err
form, err := aya.Initiate(ayapay.PaymentData{
	OrderID: fmt.Sprintf("ORDER_%d", order.ID),
	Amount:  myanmarpayments.Kyat(10000),
	Channel: "kbz_pay",
	Method:  ayapay.MethodQR,
})
// handle err
link, err := payments.AutoSubmitURL(form)
// handle err
return ctx.Response().Redirect(http.StatusFound, link)
```

`AutoSubmitURL` returns `payments.ErrFormRouteDisabled` when the route is off and `payments.ErrCryptNotAvailable` without the crypt facade.

### Handle the callback

Register a POST route without CSRF or auth middleware. `payments.CallbackRequestFromContext(ctx)` reads the request, `HandleCallback` verifies the signature and returns a `*myanmarpayments.PaymentCallback`, or a `*myanmarpayments.SignatureVerificationError`:

```go
import (
	"github.com/goravel/framework/contracts/http"
	payments "github.com/laranex/goravel-myanmar-payments/v4"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)

func KbzCallback(ctx http.Context) http.Response {
	request, err := payments.CallbackRequestFromContext(ctx)
	if err != nil {
		return ctx.Response().String(http.StatusBadRequest, "bad request")
	}
	callback, err := paymentsfacades.MyanmarPayments().
		HandleCallback(payments.GatewayKbzPay, request)
	if err != nil {
		return ctx.Response().String(http.StatusBadRequest, "invalid signature")
	}
	if callback.IsSuccessful() {
		// compare callback.Amount with the order, then fulfill callback.OrderID once
	}
	return payments.Acknowledge(ctx, callback) // KBZ Pay: plain "success"
}
```

- `payments.Acknowledge(ctx, callback)` returns the reply each gateway expects so it stops retrying; with a nil callback it is an empty 200.
- One route for every gateway: `HandleCallback(ctx.Request().Route("gateway"), request)`; an unknown name returns `payments.ErrUnknownGateway`.
- Check AYA's browser return with `aya.VerifyRedirect(request)`; it is never proof of payment.
- For production, store the verified call, acknowledge immediately and process it once in a queued job; the docs show this flow as app code (the package stores nothing).

### Check status and handle errors

- `Status(ctx, orderID)` on KBZ Pay and AYA Pay and `Status(ctx, reference)` on Yoma MMQR return a `*myanmarpayments.PaymentStatusResult` with `Status` and `IsSuccessful()`. Wave Money and CyberSource have no status API.
- Statuses are `myanmarpayments.PaymentStatus` values: `StatusSuccessful`, `StatusPending`, `StatusFailed`, `StatusCanceled`, `StatusExpired`, `StatusUnknown`.
- Gateway errors return a `*myanmarpayments.APIError` (`GatewayCode`, `GatewayMessage`, `HTTPStatus`, `Raw`); match every package error with `errors.As`.

## Test your app

- Gateway calls go through Goravel's HTTP client: fake them by URL on the `client.Factory` from `facades.App().MakeHttp()` (the generated `facades.Http()` returns a `client.Request`, which cannot fake) and call `PreventStrayRequests()` so nothing reaches a real gateway.

```go
fake := facades.App().MakeHttp()
fake.Fake(map[string]any{
	"http://api-uat.kbzpay.com/payment/gateway/uat/precreate": fake.Response().Json(200, map[string]any{
		"Response": map[string]any{"result": "SUCCESS", "code": "0", "prepay_id": "PREPAY1"},
	}),
}).PreventStrayRequests()
defer fake.Reset()
```

- Post correctly signed payloads to your callback route, signed with the secret from your test configuration as each gateway page describes (`kbzpay.NewSigner(appKey).Sign(fields)` signs KBZ Pay fields).
- To test your own handling without signed payloads, pass a `*myanmarpayments.PaymentCallback` you build yourself to the function that fulfills the order.
- Follow the `AutoSubmitURL` link with a GET to assert the auto-submitting form; tampered or expired links answer 410.

## Avoid

- Fulfilling orders from return pages or query strings; fulfill only from a verified callback or a status check.
- Treating `StatusPending` or `StatusUnknown` as paid, or skipping the amount check.
- Passing floats as amounts.
- Calling Yoma `Initiate()` twice for one order (use `RenewQR()`), or reusing a Wave `MerchantReferenceID`.
- Putting callback routes behind CSRF or authentication middleware, or the form route behind authentication; gateways and redirected browsers have no session.
- Building gateways with the SDK's `New` inside handlers, or reading the callback body before `CallbackRequestFromContext`; use the facade so HTTP fakes, the shared token cache and config apply.
