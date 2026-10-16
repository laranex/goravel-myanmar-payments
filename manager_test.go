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
	assert.Equal(t, kbzpay.Config{AppID: "kp-app", AppKey: "kbz-secret", MerchantCode: "200001", TimeoutSeconds: 30, APIURL: "https://gateway.test/kbz", PWAURL: "https://gateway.test/kbz-pwa/", TimeToLiveMinutes: 30}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, wavemoney.Config{MerchantID: "wave-merchant", SecretKey: "wave-secret", MerchantName: "Shop", TimeToLiveSeconds: 300, TimeoutSeconds: 30, BaseURL: "https://gateway.test/wave", BaseRedirectURL: "https://gateway.test/wave-pay"}, wave.Config())

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "aya-key", AppSecret: "aya-secret", TimeoutSeconds: 30, BaseURL: "https://gateway.test/aya"}, aya.Config())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, yomammqr.Config{MerchantID: "yoma-merchant", ClientID: "yoma-client", ClientSecret: "yoma-secret", WebhookHashKey: "yoma-hash", APIVersion: "v1rc", TimeoutSeconds: 30, WebhookSecret: "yoma-webhook", BaseURL: "https://gateway.test/yoma"}, yoma.Config())

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, cybersource.Config{ProfileID: "profile", AccessKey: "access", SecretKey: "cyber-secret", BaseURL: "https://gateway.test/cyber"}, cyber.Config())
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

func TestEverySettingIsRequired(t *testing.T) {
	tests := []struct {
		gateway, section, key, errorKey string
		value                           any
		invalid                         bool
	}{
		{"kbz_pay", "kbz_pay", "app_id", "app_id", "", false},
		{"kbz_pay", "kbz_pay", "merchant_code", "merchant_code", "", false},
		{"kbz_pay", "kbz_pay", "api_url", "api_url", "", false},
		{"kbz_pay", "kbz_pay", "pwa_url", "pwa_url", "", false},
		{"kbz_pay", "kbz_pay", "time_to_live_in_minutes", "time_to_live_in_minutes", "", false},
		{"kbz_pay", "http", "timeout", "timeout_in_seconds", 0, true},
		{"kbz_pay", "http", "timeout", "timeout_in_seconds", "five", true},
		{"wave_money", "wave_money", "merchant_name", "merchant_name", "", false},
		{"wave_money", "wave_money", "time_to_live_in_seconds", "time_to_live_in_seconds", "", false},
		{"wave_money", "wave_money", "time_to_live_in_seconds", "time_to_live_in_seconds", "-5", true},
		{"wave_money", "wave_money", "base_url", "base_url", "", false},
		{"wave_money", "wave_money", "base_redirect_url", "base_redirect_url", "", false},
		{"wave_money", "http", "timeout", "timeout_in_seconds", "-30", true},
		{"aya_pay", "aya_pay", "app_secret", "app_secret", "", false},
		{"aya_pay", "aya_pay", "base_url", "base_url", "", false},
		{"aya_pay", "http", "timeout", "timeout_in_seconds", "2.5", true},
		{"yoma_mmqr", "yoma_mmqr", "webhook_hashkey", "webhook_hashkey", "", false},
		{"yoma_mmqr", "yoma_mmqr", "api_version", "api_version", "", false},
		{"yoma_mmqr", "yoma_mmqr", "webhook_secret", "webhook_secret", "", false},
		{"yoma_mmqr", "yoma_mmqr", "base_url", "base_url", "", false},
		{"yoma_mmqr", "http", "timeout", "timeout_in_seconds", "1.5", true},
		{"cyber_source", "cyber_source", "secret_key", "secret_key", "", false},
		{"cyber_source", "cyber_source", "base_url", "base_url", "", false},
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

func TestTheHTTPTimeoutDefaultsTo30Seconds(t *testing.T) {
	for _, value := range []any{"", "   ", nil} {
		section := credentials("https://gateway.test")
		section["http"] = map[string]any{"timeout": value}
		manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

		kbz, err := manager.KbzPay()
		require.NoError(t, err)
		assert.Equal(t, 30, kbz.Config().TimeoutSeconds, value)
		wave, err := manager.WaveMoney()
		require.NoError(t, err)
		assert.Equal(t, 30, wave.Config().TimeoutSeconds, value)
		aya, err := manager.AyaPay()
		require.NoError(t, err)
		assert.Equal(t, 30, aya.Config().TimeoutSeconds, value)
		yoma, err := manager.YomaMmqr()
		require.NoError(t, err)
		assert.Equal(t, 30, yoma.Config().TimeoutSeconds, value)
	}
}

func TestCyberSourceDoesNotNeedTheHTTPTimeout(t *testing.T) {
	section := credentials("https://gateway.test")
	delete(section, "http")
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

	_, err := manager.CyberSource()
	require.NoError(t, err)
}

func TestWaveMoneyMerchantNameDoesNotFallBackToTheAppName(t *testing.T) {
	section := credentials("https://gateway.test")
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
		"KBZ_PAY_APP_ID": "env-app", "KBZ_PAY_APP_KEY": "env-key", "KBZ_PAY_MERCHANT_CODE": "env-merchant", "KBZ_PAY_BASE_URL": "https://kbz.test", "KBZ_PAY_PWA_BASE_REDIRECT_URL": "https://kbz.test/pwa/", "KBZ_PAY_TIME_TO_LIVE_IN_MINUTES": "45",
		"WAVE_MONEY_MERCHANT_ID": "env-wave", "WAVE_MONEY_SECRET_KEY": "env-wave-secret", "WAVE_MONEY_MERCHANT_NAME": "Env Shop", "WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS": "120", "WAVE_MONEY_BASE_URL": "https://wave.test", "WAVE_MONEY_BASE_REDIRECT_URL": "https://wave-pay.test",
		"AYA_PAY_APP_KEY": "env-aya", "AYA_PAY_APP_SECRET": "env-aya-secret", "AYA_PAY_BASE_URL": "https://aya.test/",
		"YOMA_MMQR_MERCHANT_ID": "y-merchant", "YOMA_MMQR_CLIENT_ID": "y-client", "YOMA_MMQR_CLIENT_SECRET": "y-secret", "YOMA_MMQR_WEBHOOK_HASHKEY": "y-hash", "YOMA_MMQR_WEBHOOK_SECRET": "y-webhook", "YOMA_MMQR_API_VERSION": "v1", "YOMA_MMQR_BASE_URL": "https://yoma.test",
		"CYBER_SOURCE_PROFILE_ID": "c-profile", "CYBER_SOURCE_ACCESS_KEY": "c-access", "CYBER_SOURCE_SECRET_KEY": "c-secret", "CYBER_SOURCE_BASE_URL": "https://cyber.test",
		"MYANMAR_PAYMENTS_HTTP_TIMEOUT": "15",
	} {
		t.Setenv(name, value)
	}
	manager := newManager(t, config, nil, nil)

	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Equal(t, kbzpay.Config{AppID: "env-app", AppKey: "env-key", MerchantCode: "env-merchant", TimeoutSeconds: 15, APIURL: "https://kbz.test", PWAURL: "https://kbz.test/pwa/", TimeToLiveMinutes: 45}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, "Env Shop", wave.Config().MerchantName)
	assert.Equal(t, 120, wave.Config().TimeToLiveSeconds)
	assert.Equal(t, "https://wave-pay.test", wave.Config().BaseRedirectURL)
	assert.Equal(t, 15, wave.Config().TimeoutSeconds)

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "env-aya", AppSecret: "env-aya-secret", TimeoutSeconds: 15, BaseURL: "https://aya.test/"}, aya.Config())
	assert.Equal(t, "https://aya.test", aya.Config().ResolvedBaseURL())

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
	t.Setenv("CYBER_SOURCE_BASE_URL", "https://cyber.test")
	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, "p", cyber.Config().ProfileID)
}

func TestTheKbzPayTimeToLiveIsReadFromTheConfig(t *testing.T) {
	section := credentials("https://gateway.test")
	section["kbz_pay"].(map[string]any)["time_to_live_in_minutes"] = "45"
	kbz, err := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil).KbzPay()
	require.NoError(t, err)
	assert.Equal(t, 45, kbz.Config().TimeToLiveMinutes)

	section["kbz_pay"].(map[string]any)["time_to_live_in_minutes"] = "121"
	_, err = newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil).KbzPay()
	var configurationError *myanmarpayments.ConfigurationError
	require.True(t, errors.As(err, &configurationError))
	assert.Equal(t, "time_to_live_in_minutes", configurationError.Key)
	assert.Contains(t, configurationError.Error(), "must be a whole number from 1 to 120.")
}
