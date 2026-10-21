---
name: goravel-myanmar-payments
description: >
  Integrate Myanmar payment gateways (KBZ Pay, Wave Pay, Wave Pay Mini App, AYA Pay, Yoma MMQR, CyberSource) in a Goravel app with github.com/laranex/goravel-myanmar-payments/v4.
license: MIT
metadata:
  author: Nay Thu Khant
---

# Goravel Myanmar Payments

## When to use

Use this skill when a Goravel application starts a payment, handles a gateway callback or checks a payment status with KBZ Pay, Wave Pay, Wave Pay Mini App, AYA Payment Gateway, Yoma MMQR or CyberSource. Start payments and verify callbacks with the package's typed API; never build gateway signatures by hand. The package wraps the Go SDK `github.com/laranex/go-myanmar-payments/v4`: gateways, payment data, results, statuses and errors are the SDK's types.

## Install

```bash
go get github.com/laranex/goravel-myanmar-payments/v4
./artisan package:install github.com/laranex/goravel-myanmar-payments/v4
```

Requires Go 1.25+ and Goravel 1.18+. `package:install` registers `&payments.ServiceProvider{}` in `bootstrap/providers.go`, writes `config/myanmar_payments.go` and appends the gateway variables to `.env.example`. Gateway calls go through Goravel's HTTP client, Yoma MMQR and Wave Pay Mini App tokens through the cache, and auto-submit form links are encrypted with the crypt facade (`APP_KEY`).

## Configure

Set the env keys of the gateways you use. Every setting of those gateways is required and has no default, URLs included: set production's URLs, or the gateway's UAT URLs to test. Only the HTTP timeout has a default (30 seconds).

- KBZ Pay: `KBZ_PAY_APP_ID`, `KBZ_PAY_APP_KEY`, `KBZ_PAY_MERCHANT_CODE`, `KBZ_PAY_BASE_URL`, `KBZ_PAY_PWA_BASE_REDIRECT_URL`, `KBZ_PAY_TIME_TO_LIVE_IN_MINUTES` (1 to 120; a payment's `TimeToLiveMinutes` overrides it)
- Wave Pay: `WAVE_PAY_MERCHANT_ID`, `WAVE_PAY_SECRET_KEY`, `WAVE_PAY_MERCHANT_NAME`, `WAVE_PAY_BASE_URL` (payments are requested at its `/payment`), `WAVE_PAY_BASE_REDIRECT_URL` (the customer is sent to its `/authenticate`), `WAVE_PAY_TIME_TO_LIVE_IN_SECONDS` (greater than 0; a payment's `TimeToLiveSeconds` overrides it)
- Wave Pay Mini App: `WAVE_PAY_MINI_APP_CLIENT_ID`, `WAVE_PAY_MINI_APP_CLIENT_SECRET`, `WAVE_PAY_MINI_APP_MINI_APP_ID`, `WAVE_PAY_MINI_APP_SECRET_KEY`, `WAVE_PAY_MINI_APP_BASE_URL` (production `https://api.wavemoney.io:8100`, `wavepayminiapp.ProductionURL`; UAT `https://preprodnewapi.wavemoney.io`)
- AYA Pay: `AYA_PAY_APP_KEY`, `AYA_PAY_APP_SECRET`, `AYA_PAY_BASE_URL`
- Yoma MMQR: `YOMA_MMQR_MERCHANT_ID`, `YOMA_MMQR_CLIENT_ID`, `YOMA_MMQR_CLIENT_SECRET`, `YOMA_MMQR_WEBHOOK_HASHKEY`, `YOMA_MMQR_API_VERSION`, `YOMA_MMQR_WEBHOOK_SECRET` (every callback must carry it in `X-Webhook-Secret`), `YOMA_MMQR_BASE_URL`
- CyberSource: `CYBER_SOURCE_PROFILE_ID`, `CYBER_SOURCE_ACCESS_KEY`, `CYBER_SOURCE_SECRET_KEY`, `CYBER_SOURCE_BASE_URL`
- Form link lifetime in minutes, required to create an auto-submit form link: `MYANMAR_PAYMENTS_FORM_TTL_MINUTES`
- Optional: `MYANMAR_PAYMENTS_HTTP_TIMEOUT` (seconds before a KBZ Pay, Wave Pay, Wave Pay Mini App, AYA Pay or Yoma MMQR call gives up, 30 when empty; one shared value, redirects are never followed), `MYANMAR_PAYMENTS_HTTP_CLIENT` (an `http.clients` name, default client when empty), `MYANMAR_PAYMENTS_CACHE_STORE` (store for the Yoma MMQR and Wave Pay Mini App access tokens, default store when empty)

Edit `config/myanmar_payments.go` only to change it, for example the auto-submit form route (`form_route.enabled`, `path`, `middleware`, `ttl_minutes`, `base_url`, `nonce`). Without `package:install`, publish it with `./artisan vendor:publish --package=github.com/laranex/goravel-myanmar-payments/v4`.

A gateway with a missing setting returns a `*myanmarpayments.ConfigurationError` naming the key (`The kbz_pay configuration is missing [app_key].`) when first requested, not at boot. A time setting that is not a whole number greater than 0 returns the same error with `Invalid` set.

## Use

Every gateway is reached through `paymentsfacades.MyanmarPayments()` (`github.com/laranex/goravel-myanmar-payments/v4/facades`): `KbzPay()`, `WavePay()`, `WavePayMiniApp()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()` return the SDK gateway and an error, or `Gateway("kbz-pay")` by name (`payments.GatewayNames()` lists `kbz-pay`, `wave-pay`, `wave-pay-mini-app`, `aya-pay`, `yoma-mmqr`, `cyber-source`). Each gateway is built once and reused. Goravel's `http.Context` is a `context.Context`: pass `ctx` to calls that hit the network.

### Amounts

Pass a `myanmarpayments.Amount` (`myanmarpayments.Kyat(10000)`, `myanmarpayments.ParseAmount("10000.50")`), never a `float64`. Wave Pay Mini App takes whole kyat. Only KBZ Pay (up to 2 decimals) and CyberSource accept decimals. Invalid data returns a `*myanmarpayments.InvalidPaymentDataError`; read the messages from its `Errors`. Compare a gateway's amount by value with `amount.Equals(callback.Amount)`.

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

- `RedirectPayment` from `KbzPay().PWA()` and `WavePay().Initiate()`: redirect to `payment.URL`. Wave takes `&wavepay.PaymentData{...}` and fills `MerchantReferenceID`; store it with the order. Its optional `MerchantName` (`string`, empty uses `WAVE_PAY_MERCHANT_NAME`) and `TimeToLiveSeconds` (`*int`, nil uses `WAVE_PAY_TIME_TO_LIVE_IN_SECONDS`, 0 is rejected) override the configured values for that payment only. Optional `AdditionalField3`, `AdditionalField4` and `AdditionalField5` (at most 50 characters each) are echoed in the callback. `OrderID` and `MerchantReferenceID` take at most 100 characters, `CallbackURL`, `ReturnURL` and `Description` at most 255.
- `QrPayment` from `KbzPay().QR()` (encode `QRString`) and `YomaMmqr().Initiate()` (`QRImage` as base64, `QRImageDataURI("")`, `ExpiresAt`, `Reference`). Renew an expired Yoma QR with `RenewQR(ctx, orderID)`.
- `AppPayment` from `KbzPay().App()`: return it as JSON to the mobile app.

### Mini app payments (Wave Pay Mini App)

The customer pays inside the WavePay app, on your page loaded in its webview with Wave's JavaScript SDK. Your server gives the page an access token and a signed payment payload; the page calls `WaveJsSDK.authModule.loginWithAccessToken(token)` and then `WaveJsSDK.paymentModule.makePayment(payload)`. Never call Wave's token endpoint from the page: it needs the client secret.

```go
import (
	"fmt"

	"github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/wavepayminiapp"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)

miniApp, err := paymentsfacades.MyanmarPayments().WavePayMiniApp()
// handle err
token, err := miniApp.AccessToken(ctx) // cached in the configured cache store
// handle err
data := &wavepayminiapp.PaymentData{
	OrderID:     fmt.Sprintf("ORDER_%d", order.ID),
	CallbackURL: "https://shop.test/payments/callback/wave-pay-mini-app",
	Items:       []wavepayminiapp.Item{{Name: "Ticket", Amount: myanmarpayments.Kyat(5000)}},
	// Amount defaults to the sum of the items
}
payment, err := miniApp.Payment(data) // no request is sent; fills data.MerchantReferenceID
// handle err
return ctx.Response().Json(http.StatusOK, map[string]any{"token": token, "payment": payment})
```

- `payment` is a `*myanmarpayments.MiniAppPayment`; it encodes as exactly Wave's makePayment object (`amount`, `miniAppID`, `merchantRefID`, `orderID`, `callBackUrl`, `hashValue`, `items`), also as `payment.JSON()`. Pass it to the page unchanged.
- Store `data.MerchantReferenceID` (32 hex characters when you leave it empty) with the order; the callback's `merchantRefId` names the attempt that was paid.
- The hash signs the callback URL, so the callback must be verified with the same URL (see below). `ForgetToken()` drops the cached token, e.g. after rotating the client secret.

### Form payments (AYA Pay, CyberSource)

`AyaPay().Initiate()` and `CyberSource().Initiate()` return a `*myanmarpayments.FormPayment` the customer's browser must POST. Redirect to `payments.AutoSubmitURL(form)`: a link to the package's `GET myanmar-payments/form` route, encrypted with `APP_KEY` and valid for `form_route.ttl_minutes` (`MYANMAR_PAYMENTS_FORM_TTL_MINUTES`, required: missing returns `The form_route configuration is missing [ttl_minutes].`); a tampered or expired link answers 410. The page is sent with `Cache-Control: no-store` and `Referrer-Policy: strict-origin`, so the gateway never sees the encrypted link. Under a nonce-based Content-Security-Policy, set `form_route.nonce` to a `func(contractshttp.Context) string` returning the request's nonce; the page then renders `form.HTMLWithNonce(nonce)`. Or send `form.HTML()` yourself.

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

CyberSource's `cybersource.PaymentData` also needs `Currency` (e.g. `"MMK"`), `TransactionType` (e.g. `cybersource.Sale`) and `Locale` (e.g. `"en-us"`); there are no defaults.

`Initiate()` never needs the form route; `AutoSubmitURL` checks, in order: the route is on (`payments.ErrFormRouteDisabled`), `ttl_minutes` is set (`*myanmarpayments.ConfigurationError`), the crypt facade exists (`payments.ErrCryptNotAvailable`).

### Handle the callback

Register a POST route without CSRF or auth middleware. `HandleCallbackFromContext(gateway, ctx)` resolves the gateway, reads the request and verifies the signature, returning a `*myanmarpayments.PaymentCallback`, or a `*myanmarpayments.SignatureVerificationError`:

```go
import (
	"github.com/goravel/framework/contracts/http"
	payments "github.com/laranex/goravel-myanmar-payments/v4"
	paymentsfacades "github.com/laranex/goravel-myanmar-payments/v4/facades"
)

func KbzCallback(ctx http.Context) http.Response {
	callback, err := paymentsfacades.MyanmarPayments().
		HandleCallbackFromContext(payments.GatewayKbzPay, ctx)
	if err != nil {
		return ctx.Response().String(http.StatusBadRequest, "invalid signature")
	}
	if callback.IsSuccessful() {
		// check order.Amount.Equals(callback.Amount), then fulfill once
	}
	return payments.Acknowledge(ctx, callback) // KBZ Pay: plain "success"
}
```

- `payments.Acknowledge(ctx, callback)` returns the reply each gateway expects so it stops retrying; with a nil callback it is an empty 200.
- One route for every gateway: `HandleCallbackFromContext(ctx.Request().Route("gateway"), ctx)`; an unknown name returns `payments.ErrUnknownGateway`. To keep the raw request (e.g. to store it), build it with `payments.CallbackRequestFromContext(ctx)` and call `HandleCallback(gateway, request)`.
- Wave Pay Mini App callbacks are signed with the payment's callback URL, which the callback does not carry. `HandleCallbackFromContext("wave-pay-mini-app", ctx)` verifies against the URL the callback was posted to (`payments.CallbackURLFromContext(ctx)`: scheme, host and path, no query; `X-Forwarded-Proto`/`X-Forwarded-Host` are not read), so sign payments with that route's URL. Behind a proxy that changes the scheme or host, or for a stored request, call `WavePayMiniApp().HandleCallback(request, callbackURL)` with the URL the payment was signed with. `HandleCallback("wave-pay-mini-app", request)` returns `payments.ErrCallbackURLRequired`. Statuses: `SUCCESSFUL` is paid, `PROCESSING` pending, `FAILED` failed; the acknowledgement is an empty 200; `merchantRefId` is in `callback.Raw`.
- Check AYA's browser return with `aya.VerifyRedirect(request)`; it is never proof of payment.
- `HandleCallbackFromContext` and `CallbackRequestFromContext` return `payments.ErrCallbackBodyTooLarge` for a body over `payments.MaxCallbackBodyBytes` (64 KiB).
- Yoma MMQR callbacks sign only the order number and status and carry no amount: match the order number to your stored order (its QR was generated for the stored amount) and, to confirm, call `YomaMmqr().Status(ctx, reference)` with that order's QR reference (`refLabel`). The status result has no amount either, so trust your stored amount.
- For production, store the verified call, acknowledge immediately and process it once in a queued job; the docs show this flow as app code (the package stores nothing).

### Check status and handle errors

- `Status(ctx, orderID)` on KBZ Pay and AYA Pay and `Status(ctx, reference)` on Yoma MMQR return a `*myanmarpayments.PaymentStatusResult` with `Status` and `IsSuccessful()`. Wave Pay, Wave Pay Mini App and CyberSource have no status API.
- Statuses are `myanmarpayments.PaymentStatus` values: `StatusSuccessful`, `StatusPending`, `StatusFailed`, `StatusCanceled`, `StatusExpired`, `StatusUnknown`. `myanmarpayments.PaymentStatuses()` lists them.
- Gateway errors return a `*myanmarpayments.APIError` (`GatewayCode`, `GatewayMessage`, `HTTPStatus`, `Raw`); match every package error with `errors.As`. All SDK errors implement `myanmarpayments.PaymentError`.
- For a KBZ Pay call the package does not wrap, sign the fields with `kbz.Signer()`.

## Test your app

- Gateway calls go through Goravel's HTTP client: fake them by URL (the `KBZ_PAY_BASE_URL` of your test config, and so on) on the `client.Factory` from `facades.App().MakeHttp()` (the generated `facades.Http()` returns a `client.Request`, which cannot fake) and call `PreventStrayRequests()` so nothing reaches a real gateway.

```go
fake := facades.App().MakeHttp()
fake.Fake(map[string]any{
	"<KBZ_PAY_BASE_URL>/precreate": fake.Response().Json(200, map[string]any{
		"Response": map[string]any{"result": "SUCCESS", "code": "0", "prepay_id": "PREPAY1"},
	}),
}).PreventStrayRequests()
defer fake.Reset()
```

- Fake the Wave Pay Mini App token at `<WAVE_PAY_MINI_APP_BASE_URL>/auth/v1/external-utility/token` (`{"access_token": "...", "expires_in": 3600}`); `Payment()` sends no request.
- Post correctly signed payloads to your callback route, signed with the secret from your test configuration as each gateway page describes (`kbzpay.NewSigner(appKey).Sign(fields)` signs KBZ Pay fields).
- To test your own handling without signed payloads, pass a `*myanmarpayments.PaymentCallback` you build yourself to the function that fulfills the order.
- Follow the `AutoSubmitURL` link with a GET to assert the auto-submitting form; tampered or expired links answer 410.

## Avoid

- Fulfilling orders from return pages or query strings; fulfill only from a verified callback or a status check.
- Treating `StatusPending` or `StatusUnknown` as paid, or skipping the amount check.
- Passing floats as amounts.
- Calling Yoma `Initiate()` twice for one order (use `RenewQR()`), or reusing a Wave or Wave Pay Mini App `MerchantReferenceID`.
- Sending the Wave Pay Mini App client secret or secret key to the page, or changing the signed payload there.
- Putting callback routes behind CSRF or authentication middleware, or the form route behind authentication; gateways and redirected browsers have no session.
- Building gateways with the SDK's `New` inside handlers, or reading the callback body before `CallbackRequestFromContext`; use the facade so HTTP fakes, the shared token cache and config apply.
