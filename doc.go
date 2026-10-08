// Package payments is the Goravel integration of Go Myanmar Payments
// (github.com/laranex/go-myanmar-payments/v4): KBZ Pay, Wave Money, AYA Payment
// Gateway, Yoma MMQR and CyberSource Secure Acceptance.
//
// It adds no gateway logic of its own. The ServiceProvider builds each SDK
// gateway from config/myanmar_payments.go (or the SDK's environment variables)
// the first time it is used, sends gateway HTTP calls through Goravel's HTTP
// client so facades.Http().Fake() works in tests, keeps Yoma MMQR access
// tokens in Goravel's cache, and registers a route that renders the signed
// AYA Pay and CyberSource forms from an encrypted, expiring link.
//
//	kbz, err := paymentsfacades.MyanmarPayments().KbzPay() // *kbzpay.Gateway
//	payment, err := kbz.PWA(ctx, kbzpay.PaymentData{...})
//
//	request, err := payments.CallbackRequestFromContext(ctx)
//	callback, err := kbz.HandleCallback(request)
//	return payments.Acknowledge(ctx, callback)
package payments
