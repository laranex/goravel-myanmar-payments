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
		// Only the gateways you call need credentials. "sandbox" picks each
		// gateway's UAT endpoints; set it to false and use production
		// credentials when you go live. URL overrides are optional. The
		// environment variable names are the ones the Go SDK reads.
		"kbz_pay": map[string]any{
			"sandbox":       config.Env("KBZ_PAY_SANDBOX", true),
			"app_id":        config.Env("KBZ_PAY_APP_ID", ""),
			"app_key":       config.Env("KBZ_PAY_APP_KEY", ""),
			"merchant_code": config.Env("KBZ_PAY_MERCHANT_CODE", ""),
			"api_url":       config.Env("KBZ_PAY_BASE_URL", ""),
			"pwa_url":       config.Env("KBZ_PAY_PWA_BASE_REDIRECT_URL", ""),
		},

		"wave_money": map[string]any{
			"sandbox":                 config.Env("WAVE_MONEY_SANDBOX", true),
			"merchant_id":             config.Env("WAVE_MONEY_MERCHANT_ID", ""),
			"secret_key":              config.Env("WAVE_MONEY_SECRET_KEY", ""),
			"merchant_name":           config.Env("WAVE_MONEY_MERCHANT_NAME", config.Env("APP_NAME", "")),
			"time_to_live_in_seconds": config.Env("WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS", 300),
			"base_url":                config.Env("WAVE_MONEY_BASE_URL", ""),
			"authenticate_url":        config.Env("WAVE_MONEY_AUTHENTICATE_URL", ""),
		},

		"aya_pay": map[string]any{
			"sandbox":    config.Env("AYA_PAY_SANDBOX", true),
			"app_key":    config.Env("AYA_PAY_APP_KEY", config.Env("AYA_PGW_APP_KEY", "")),
			"app_secret": config.Env("AYA_PAY_APP_SECRET", config.Env("AYA_PGW_APP_SECRET", "")),
			"base_url":   config.Env("AYA_PAY_BASE_URL", config.Env("AYA_PGW_BASE_URL", "")),
		},

		"yoma_mmqr": map[string]any{
			"sandbox":         config.Env("YOMA_MMQR_SANDBOX", true),
			"merchant_id":     config.Env("YOMA_MMQR_MERCHANT_ID", ""),
			"client_id":       config.Env("YOMA_MMQR_CLIENT_ID", ""),
			"client_secret":   config.Env("YOMA_MMQR_CLIENT_SECRET", ""),
			"webhook_hashkey": config.Env("YOMA_MMQR_WEBHOOK_HASHKEY", ""),
			"webhook_secret":  config.Env("YOMA_MMQR_WEBHOOK_SECRET", ""),
			"base_url":        config.Env("YOMA_MMQR_BASE_URL", ""),
			"api_version":     config.Env("YOMA_MMQR_API_VERSION", "v1rc"),
		},

		"cyber_source": map[string]any{
			"sandbox":    config.Env("CYBER_SOURCE_SANDBOX", true),
			"profile_id": config.Env("CYBER_SOURCE_PROFILE_ID", ""),
			"access_key": config.Env("CYBER_SOURCE_ACCESS_KEY", ""),
			"secret_key": config.Env("CYBER_SOURCE_SECRET_KEY", ""),
			"base_url":   config.Env("CYBER_SOURCE_BASE_URL", ""),
		},

		// HTTP
		//
		// Gateway calls go through Goravel's HTTP client, so facades.Http().Fake()
		// intercepts them in tests. "client" names an entry of http.clients
		// (empty means the default client); "timeout" is in seconds.
		"http": map[string]any{
			"client":  config.Env("MYANMAR_PAYMENTS_HTTP_CLIENT", ""),
			"timeout": config.Env("MYANMAR_PAYMENTS_HTTP_TIMEOUT", 30),
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
		// APP_KEY and expire after "ttl_minutes". "base_url" starts the links.
		"form_route": map[string]any{
			"enabled":     true,
			"path":        "myanmar-payments/form",
			"middleware":  []contractshttp.Middleware{},
			"ttl_minutes": 30,
			"base_url":    config.Env("APP_URL", "http://localhost"),
		},
	})
}
`)
}

// envStub is appended to .env.example: every variable the gateways read.
func envStub() string {
	return `
KBZ_PAY_SANDBOX=true
KBZ_PAY_APP_ID=
KBZ_PAY_APP_KEY=
KBZ_PAY_MERCHANT_CODE=

WAVE_MONEY_SANDBOX=true
WAVE_MONEY_MERCHANT_ID=
WAVE_MONEY_SECRET_KEY=
WAVE_MONEY_MERCHANT_NAME=

AYA_PAY_SANDBOX=true
AYA_PAY_APP_KEY=
AYA_PAY_APP_SECRET=

YOMA_MMQR_SANDBOX=true
YOMA_MMQR_MERCHANT_ID=
YOMA_MMQR_CLIENT_ID=
YOMA_MMQR_CLIENT_SECRET=
YOMA_MMQR_WEBHOOK_HASHKEY=
YOMA_MMQR_WEBHOOK_SECRET=

CYBER_SOURCE_SANDBOX=true
CYBER_SOURCE_PROFILE_ID=
CYBER_SOURCE_ACCESS_KEY=
CYBER_SOURCE_SECRET_KEY=
`
}
