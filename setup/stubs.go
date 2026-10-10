package main

import "strings"

// configStub returns config/myanmar_payments.go for an application whose config
// package is configPackage and whose facades live at facadesImport. It is the same
// file as config/myanmar_payments.go in this module.
func configStub(configPackage, facadesImport string) string {
	return strings.NewReplacer("DummyPackage", configPackage, "DummyFacades", facadesImport).Replace(`package DummyPackage

import (
	contractshttp "github.com/goravel/framework/contracts/http"
	"DummyFacades"
)

func init() {
	config := facades.Config()
	config.Add("myanmar_payments", map[string]any{
		// Gateways
		//
		// Only the gateways you call need settings, and every setting of a
		// gateway you call is required. Each gateway calls its production
		// endpoints; to test against UAT or go through a proxy, set the URL
		// overrides. The environment variable names are the ones the Go SDK
		// reads.
		"kbz_pay": map[string]any{
			"app_id":        config.Env("KBZ_PAY_APP_ID", ""),
			"app_key":       config.Env("KBZ_PAY_APP_KEY", ""),
			"merchant_code": config.Env("KBZ_PAY_MERCHANT_CODE", ""),
			"api_url":       config.Env("KBZ_PAY_BASE_URL", ""),
			"pwa_url":       config.Env("KBZ_PAY_PWA_BASE_REDIRECT_URL", ""),
		},

		"wave_money": map[string]any{
			"merchant_id":             config.Env("WAVE_MONEY_MERCHANT_ID", ""),
			"secret_key":              config.Env("WAVE_MONEY_SECRET_KEY", ""),
			"merchant_name":           config.Env("WAVE_MONEY_MERCHANT_NAME", ""),
			"time_to_live_in_seconds": config.Env("WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS", ""),
			"base_url":                config.Env("WAVE_MONEY_BASE_URL", ""),
			"authenticate_url":        config.Env("WAVE_MONEY_AUTHENTICATE_URL", ""),
		},

		"aya_pay": map[string]any{
			"app_key":    config.Env("AYA_PAY_APP_KEY", config.Env("AYA_PGW_APP_KEY", "")),
			"app_secret": config.Env("AYA_PAY_APP_SECRET", config.Env("AYA_PGW_APP_SECRET", "")),
			"base_url":   config.Env("AYA_PAY_BASE_URL", config.Env("AYA_PGW_BASE_URL", "")),
		},

		"yoma_mmqr": map[string]any{
			"merchant_id":     config.Env("YOMA_MMQR_MERCHANT_ID", ""),
			"client_id":       config.Env("YOMA_MMQR_CLIENT_ID", ""),
			"client_secret":   config.Env("YOMA_MMQR_CLIENT_SECRET", ""),
			"webhook_hashkey": config.Env("YOMA_MMQR_WEBHOOK_HASHKEY", ""),
			"webhook_secret":  config.Env("YOMA_MMQR_WEBHOOK_SECRET", ""),
			"api_version":     config.Env("YOMA_MMQR_API_VERSION", ""),
			"base_url":        config.Env("YOMA_MMQR_BASE_URL", ""),
		},

		"cyber_source": map[string]any{
			"profile_id": config.Env("CYBER_SOURCE_PROFILE_ID", ""),
			"access_key": config.Env("CYBER_SOURCE_ACCESS_KEY", ""),
			"secret_key": config.Env("CYBER_SOURCE_SECRET_KEY", ""),
			"base_url":   config.Env("CYBER_SOURCE_BASE_URL", ""),
		},

		// HTTP
		//
		// Gateway calls go through Goravel's HTTP client, so facades.Http().Fake()
		// intercepts them in tests. "client" names an entry of http.clients
		// (empty means the default client). "timeout" is in seconds and is
		// required by KBZ Pay, Wave Money, AYA Pay and Yoma MMQR.
		"http": map[string]any{
			"client":  config.Env("MYANMAR_PAYMENTS_HTTP_CLIENT", ""),
			"timeout": config.Env("MYANMAR_PAYMENTS_HTTP_TIMEOUT", ""),
		},

		// Cache
		//
		// The cache store that keeps Yoma MMQR access tokens between requests
		// and processes. Empty uses your default store.
		"cache_store": config.Env("MYANMAR_PAYMENTS_CACHE_STORE", ""),

		// Auto-submit Form Route
		//
		// AYA Pay and CyberSource need the customer's browser to POST a signed
		// form. This route renders that form and submits it, so a handler can
		// redirect to payments.AutoSubmitURL(form). Links are encrypted with
		// APP_KEY and expire after "ttl_minutes" (required to create a link).
		// "base_url" starts the links.
		"form_route": map[string]any{
			"enabled":     true,
			"path":        "myanmar-payments/form",
			"middleware":  []contractshttp.Middleware{},
			"ttl_minutes": config.Env("MYANMAR_PAYMENTS_FORM_TTL_MINUTES", ""),
			"base_url":    config.Env("APP_URL", "http://localhost"),
		},
	})
}
`)
}

// envStub is appended to .env.example: every required setting, plus the
// optional Yoma MMQR webhook secret. The URL overrides are left out: every
// gateway calls its production endpoints unless you set them.
func envStub() string {
	return `
MYANMAR_PAYMENTS_HTTP_TIMEOUT=
MYANMAR_PAYMENTS_FORM_TTL_MINUTES=

KBZ_PAY_APP_ID=
KBZ_PAY_APP_KEY=
KBZ_PAY_MERCHANT_CODE=

WAVE_MONEY_MERCHANT_ID=
WAVE_MONEY_SECRET_KEY=
WAVE_MONEY_MERCHANT_NAME=
WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS=

AYA_PAY_APP_KEY=
AYA_PAY_APP_SECRET=

YOMA_MMQR_MERCHANT_ID=
YOMA_MMQR_CLIENT_ID=
YOMA_MMQR_CLIENT_SECRET=
YOMA_MMQR_WEBHOOK_HASHKEY=
YOMA_MMQR_WEBHOOK_SECRET=
YOMA_MMQR_API_VERSION=

CYBER_SOURCE_PROFILE_ID=
CYBER_SOURCE_ACCESS_KEY=
CYBER_SOURCE_SECRET_KEY=
`
}
