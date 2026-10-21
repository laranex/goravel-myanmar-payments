// Package payments is the Goravel integration of Go Myanmar Payments
// (github.com/laranex/go-myanmar-payments/v4): KBZ Pay, Wave Pay, Wave Pay Mini
// App, AYA Payment Gateway, Yoma MMQR and CyberSource Secure Acceptance.
//
// It adds no gateway logic of its own. The ServiceProvider builds each SDK
// gateway from config/myanmar_payments.go (or the SDK's environment variables)
// the first time it is used, sends gateway HTTP calls through Goravel's HTTP
// client so facades.Http().Fake() works in tests, keeps Yoma MMQR and Wave Pay
// Mini App access tokens in Goravel's cache, and registers a route that renders the signed
// AYA Pay and CyberSource forms from an encrypted, expiring link.
//
//	kbz, err := paymentsfacades.MyanmarPayments().KbzPay() // *kbzpay.Gateway
//	payment, err := kbz.PWA(ctx, kbzpay.PaymentData{...})
//
//	request, err := payments.CallbackRequestFromContext(ctx)
//	callback, err := kbz.HandleCallback(request)
//	return payments.Acknowledge(ctx, callback)
//
// One route can serve every gateway by name ("kbz-pay", "wave-pay", ...); it
// verifies Wave Pay Mini App callbacks against the URL they were posted to:
//
//	callback, err := paymentsfacades.MyanmarPayments().
//		HandleCallbackFromContext(ctx.Request().Route("gateway"), ctx)
package payments
