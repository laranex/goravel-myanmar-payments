package payments

import (
	"errors"
	"testing"

	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/ayapay"
	"github.com/laranex/go-myanmar-payments/v4/cybersource"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	"github.com/laranex/go-myanmar-payments/v4/wavemoney"
	"github.com/laranex/go-myanmar-payments/v4/yomammqr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManagerRequiresConfig(t *testing.T) {
	_, err := NewManager(Options{})
	assert.ErrorContains(t, err, "config.Config is required")
}

func TestGatewaysAreBuiltFromThePublishedConfig(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)

	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Equal(t, kbzpay.Config{AppID: "kp-app", AppKey: "kbz-secret", MerchantCode: "200001", TimeoutSeconds: 30, APIURL: "https://gateway.test/kbz"}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, wavemoney.Config{MerchantID: "wave-merchant", SecretKey: "wave-secret", MerchantName: "Shop", TimeToLiveSeconds: 300, TimeoutSeconds: 30, BaseURL: "https://gateway.test/wave"}, wave.Config())

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "aya-key", AppSecret: "aya-secret", TimeoutSeconds: 30, BaseURL: "https://gateway.test/aya"}, aya.Config())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, yomammqr.Config{MerchantID: "yoma-merchant", ClientID: "yoma-client", ClientSecret: "yoma-secret", WebhookHashKey: "yoma-hash", APIVersion: "v1rc", TimeoutSeconds: 30, BaseURL: "https://gateway.test/yoma"}, yoma.Config())

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, cybersource.Config{ProfileID: "profile", AccessKey: "access", SecretKey: "cyber-secret"}, cyber.Config())
}

func TestGatewaysAreBuiltOnceAndReused(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)

	first, err := manager.KbzPay()
	require.NoError(t, err)
	second, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Same(t, first, second)

	yoma1, _ := manager.YomaMmqr()
	yoma2, _ := manager.YomaMmqr()
	assert.Same(t, yoma1, yoma2)
}

func TestGatewaysDefaultToTheProductionEndpoints(t *testing.T) {
	section := credentials("")
	for _, gateway := range []string{"wave_money", "aya_pay", "yoma_mmqr", "cyber_source"} {
		section[gateway].(map[string]any)["base_url"] = ""
	}
	section["kbz_pay"].(map[string]any)["api_url"] = ""
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Equal(t, kbzpay.ProductionAPIURL, kbz.Config().ResolvedAPIURL())
	assert.Equal(t, kbzpay.ProductionPWAURL, kbz.Config().ResolvedPWAURL())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, wavemoney.ProductionURL, wave.Config().ResolvedBaseURL())
	assert.Equal(t, wavemoney.ProductionAuthenticateURL, wave.Config().ResolvedAuthenticateURL())

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.ProductionURL, aya.Config().ResolvedBaseURL())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, yomammqr.ProductionURL, yoma.Config().ResolvedBaseURL())

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, cybersource.ProductionURL, cyber.Config().ResolvedBaseURL())
}

func TestEverySettingIsRequired(t *testing.T) {
	tests := []struct {
		gateway, section, key, errorKey string
		value                           any
		invalid                         bool
	}{
		{"kbz_pay", "kbz_pay", "app_id", "app_id", "", false},
		{"kbz_pay", "kbz_pay", "merchant_code", "merchant_code", "", false},
		{"kbz_pay", "http", "timeout", "timeout_in_seconds", "", false},
		{"kbz_pay", "http", "timeout", "timeout_in_seconds", 0, true},
		{"kbz_pay", "http", "timeout", "timeout_in_seconds", "five", true},
		{"wave_money", "wave_money", "merchant_name", "merchant_name", "", false},
		{"wave_money", "wave_money", "time_to_live_in_seconds", "time_to_live_in_seconds", "", false},
		{"wave_money", "wave_money", "time_to_live_in_seconds", "time_to_live_in_seconds", "-5", true},
		{"wave_money", "http", "timeout", "timeout_in_seconds", "", false},
		{"aya_pay", "aya_pay", "app_secret", "app_secret", "", false},
		{"aya_pay", "http", "timeout", "timeout_in_seconds", "", false},
		{"yoma_mmqr", "yoma_mmqr", "webhook_hashkey", "webhook_hashkey", "", false},
		{"yoma_mmqr", "yoma_mmqr", "api_version", "api_version", "", false},
		{"yoma_mmqr", "http", "timeout", "timeout_in_seconds", "1.5", true},
		{"cyber_source", "cyber_source", "secret_key", "secret_key", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.gateway+" "+tt.section+"."+tt.key, func(t *testing.T) {
			section := credentials("https://gateway.test")
			section[tt.section].(map[string]any)[tt.key] = tt.value
			manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

			builds := map[string]func() error{
				"kbz_pay":      func() error { _, err := manager.KbzPay(); return err },
				"wave_money":   func() error { _, err := manager.WaveMoney(); return err },
				"aya_pay":      func() error { _, err := manager.AyaPay(); return err },
				"yoma_mmqr":    func() error { _, err := manager.YomaMmqr(); return err },
				"cyber_source": func() error { _, err := manager.CyberSource(); return err },
			}
			var configurationError *myanmarpayments.ConfigurationError
			require.True(t, errors.As(builds[tt.gateway](), &configurationError))
			assert.Equal(t, myanmarpayments.ConfigurationError{Gateway: tt.gateway, Key: tt.errorKey, Invalid: tt.invalid}, *configurationError)
		})
	}
}

func TestCyberSourceDoesNotNeedTheHTTPTimeout(t *testing.T) {
	section := credentials("")
	delete(section, "http")
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

	_, err := manager.CyberSource()
	require.NoError(t, err)
}

func TestWaveMoneyMerchantNameDoesNotFallBackToTheAppName(t *testing.T) {
	section := credentials("")
	section["wave_money"].(map[string]any)["merchant_name"] = ""
	config := newConfig(t, map[string]any{ConfigKey: section})
	t.Setenv("APP_NAME", "Goravel")
	manager := newManager(t, config, nil, nil)

	_, err := manager.WaveMoney()
	assert.EqualError(t, err, "myanmarpayments: The wave_money configuration is missing [merchant_name].")
}

func TestGatewaysReadTheEnvironmentWithoutAPublishedConfig(t *testing.T) {
	config := newConfig(t, nil)
	for name, value := range map[string]string{
		"KBZ_PAY_APP_ID": "env-app", "KBZ_PAY_APP_KEY": "env-key", "KBZ_PAY_MERCHANT_CODE": "env-merchant",
		"WAVE_MONEY_MERCHANT_ID": "env-wave", "WAVE_MONEY_SECRET_KEY": "env-wave-secret", "WAVE_MONEY_MERCHANT_NAME": "Env Shop", "WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS": "120",
		"AYA_PGW_APP_KEY": "legacy-key", "AYA_PGW_APP_SECRET": "legacy-secret", "AYA_PGW_BASE_URL": "https://legacy.test/",
		"YOMA_MMQR_MERCHANT_ID": "y-merchant", "YOMA_MMQR_CLIENT_ID": "y-client", "YOMA_MMQR_CLIENT_SECRET": "y-secret", "YOMA_MMQR_WEBHOOK_HASHKEY": "y-hash", "YOMA_MMQR_WEBHOOK_SECRET": "y-webhook", "YOMA_MMQR_API_VERSION": "v1",
		"CYBER_SOURCE_PROFILE_ID": "c-profile", "CYBER_SOURCE_ACCESS_KEY": "c-access", "CYBER_SOURCE_SECRET_KEY": "c-secret",
		"MYANMAR_PAYMENTS_HTTP_TIMEOUT": "15",
	} {
		t.Setenv(name, value)
	}
	manager := newManager(t, config, nil, nil)

	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Equal(t, kbzpay.Config{AppID: "env-app", AppKey: "env-key", MerchantCode: "env-merchant", TimeoutSeconds: 15}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, "Env Shop", wave.Config().MerchantName)
	assert.Equal(t, 120, wave.Config().TimeToLiveSeconds)
	assert.Equal(t, 15, wave.Config().TimeoutSeconds)

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "legacy-key", AppSecret: "legacy-secret", TimeoutSeconds: 15, BaseURL: "https://legacy.test/"}, aya.Config())
	assert.Equal(t, "https://legacy.test", aya.Config().ResolvedBaseURL())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, "y-webhook", yoma.Config().WebhookSecret)
	assert.Equal(t, "v1", yoma.Config().APIVersion)

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, "c-profile", cyber.Config().ProfileID)
}

func TestMissingCredentialsAreNotCached(t *testing.T) {
	config := newConfig(t, nil)
	manager := newManager(t, config, nil, nil)

	tests := map[string]func() error{
		"kbz_pay":      func() error { _, err := manager.KbzPay(); return err },
		"wave_money":   func() error { _, err := manager.WaveMoney(); return err },
		"aya_pay":      func() error { _, err := manager.AyaPay(); return err },
		"yoma_mmqr":    func() error { _, err := manager.YomaMmqr(); return err },
		"cyber_source": func() error { _, err := manager.CyberSource(); return err },
	}
	for gateway, build := range tests {
		var configurationError *myanmarpayments.ConfigurationError
		require.True(t, errors.As(build(), &configurationError), gateway)
		assert.Equal(t, gateway, configurationError.Gateway)
	}

	t.Setenv("CYBER_SOURCE_PROFILE_ID", "p")
	t.Setenv("CYBER_SOURCE_ACCESS_KEY", "a")
	t.Setenv("CYBER_SOURCE_SECRET_KEY", "s")
	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, "p", cyber.Config().ProfileID)
}
