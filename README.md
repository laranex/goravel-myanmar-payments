# Goravel Myanmar Payments

[![Go Reference](https://pkg.go.dev/badge/github.com/laranex/goravel-myanmar-payments/v4.svg)](https://pkg.go.dev/github.com/laranex/goravel-myanmar-payments/v4)
[![Tests](https://github.com/laranex/goravel-myanmar-payments/actions/workflows/tests.yml/badge.svg)](https://github.com/laranex/goravel-myanmar-payments/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/laranex/goravel-myanmar-payments.svg?style=flat-square)](LICENSE.md)

Goravel integration for Myanmar payment gateways: KBZ Pay, Wave Pay, Wave Pay Mini App, AYA Pay, Yoma MMQR and CyberSource. Built on Go Myanmar Payments. Facade and config, verified callbacks and test fakes. Built for humans and AI agents.

## Why Goravel Myanmar Payments

Every Myanmar payment gateway signs, sends and acknowledges payments its own way, with its own rules for amounts. Goravel Myanmar Payments puts KBZ Pay, Wave Pay, AYA Pay, Yoma MMQR and CyberSource behind one facade. All gateway logic lives in [Go Myanmar Payments](https://laranex.vercel.app/go-myanmar-payments/introduction); this package wires it into Goravel.

- **One facade, every gateway.** `paymentsfacades.MyanmarPayments()` returns a manager with `KbzPay()`, `WavePay()`, `AyaPay()`, `YomaMmqr()` and `CyberSource()`, configured from environment variables.
- **One-command setup.** `package:install` registers the service provider, writes `config/myanmar_payments.go` and adds the gateway variables to `.env.example`.
- **A typed result for each flow.** Every payment returns a `RedirectPayment`, `FormPayment`, `QrPayment` or `AppPayment`, so you always know what to do next. See [Payment Flows](https://laranex.vercel.app/goravel-myanmar-payments/payment-flows).
- **Exact amounts.** `myanmarpayments.Amount` values, never floats, checked against each gateway's documented rules before any request is sent.
- **Verified callbacks.** `HandleCallback` checks the gateway's signature against the exact body it sent and maps its status onto one `PaymentStatus`; `payments.Acknowledge` returns the response the gateway expects.
- **Goravel all the way down.** Gateway calls go through Goravel's HTTP client, Yoma MMQR tokens are shared through your cache store, and AYA Pay and CyberSource checkout forms open from an encrypted link.
- **Easy to test.** Goravel's HTTP client fakes see every gateway call, and signed callbacks can be posted to your own routes. See [Testing](https://laranex.vercel.app/goravel-myanmar-payments/testing).
- **Your webhooks, your storage.** The package never stores webhooks; the docs show the recommended store, acknowledge and process-once flow as sample code. See [Handling Webhooks](https://laranex.vercel.app/goravel-myanmar-payments/webhooks).
- **Ready for AI agents.** An agent skill, installed with `npx skills add`, teaches coding agents to use the package correctly.

## Installation

> **Requires** Go 1.25+ and Goravel 1.18+.

```bash
go get github.com/laranex/goravel-myanmar-payments/v4
./artisan package:install github.com/laranex/goravel-myanmar-payments/v4
```

## Documentation

The full documentation lives at **[laranex.vercel.app/goravel-myanmar-payments](https://laranex.vercel.app/goravel-myanmar-payments)**:

- [Introduction](https://laranex.vercel.app/goravel-myanmar-payments/introduction): what Goravel Myanmar Payments does and why to use it
- [Installation](https://laranex.vercel.app/goravel-myanmar-payments/installation): `go get`, `package:install`, the service provider and what the package provides
- [AI Agent Skill](https://laranex.vercel.app/goravel-myanmar-payments/ai-agent-skill): installing the agent skill with `npx skills add`
- [Configuration](https://laranex.vercel.app/goravel-myanmar-payments/configuration): environment variables, production and UAT endpoints, the HTTP client, the auto-submit form route and the cache
- [Amounts](https://laranex.vercel.app/goravel-myanmar-payments/amounts): exact amounts and each gateway's rules for decimals and minimums
- [Payment Flows](https://laranex.vercel.app/goravel-myanmar-payments/payment-flows): redirect, form, QR and app payments and the typed result each returns
- [Callbacks & Status](https://laranex.vercel.app/goravel-myanmar-payments/callbacks): verifying callbacks, `PaymentStatus`, acknowledging and status checks
- [Handling Webhooks](https://laranex.vercel.app/goravel-myanmar-payments/webhooks): the recommended verify, store, acknowledge and process-once flow
- [Testing](https://laranex.vercel.app/goravel-myanmar-payments/testing): Goravel's HTTP client fakes, signed callbacks, callbacks you build yourself and following form links
- [KBZ Pay](https://laranex.vercel.app/goravel-myanmar-payments/drivers/kbz-pay): PWA redirect, QR and in-app payments, status checks and callbacks
- [Wave Pay](https://laranex.vercel.app/goravel-myanmar-payments/drivers/wave-pay): redirect payments with line items and callbacks
- [Wave Pay Mini App](https://laranex.vercel.app/goravel-myanmar-payments/drivers/wave-pay-mini-app): payments inside the WavePay app with Wave's JavaScript SDK: access tokens, signed payment payloads and callbacks
- [AYA Pay](https://laranex.vercel.app/goravel-myanmar-payments/drivers/aya-pay): the AYA Payment Gateway hosted checkout, channel discovery, status checks and callbacks
- [Yoma MMQR](https://laranex.vercel.app/goravel-myanmar-payments/drivers/yoma-mmqr): QR images, QR renewal, status checks and callbacks
- [CyberSource](https://laranex.vercel.app/goravel-myanmar-payments/drivers/cyber-source): Secure Acceptance card payments with a signed hosted checkout form
- [Results](https://laranex.vercel.app/goravel-myanmar-payments/references/results): every field of `RedirectPayment`, `FormPayment`, `QrPayment`, `AppPayment` and `MiniAppPayment`
- [PaymentCallback & Status](https://laranex.vercel.app/goravel-myanmar-payments/references/payment-callback): every field of `PaymentCallback`, `PaymentStatusResult` and `PaymentStatus`
- [Errors](https://laranex.vercel.app/goravel-myanmar-payments/references/errors): validation, API, signature and configuration errors

## License

MIT. See [LICENSE.md](LICENSE.md).
