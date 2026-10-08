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
	assert.Equal(t, kbzpay.Config{AppID: "kp-app", AppKey: "kbz-secret", MerchantCode: "200001", APIURL: "https://gateway.test/kbz"}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, wavemoney.Config{MerchantID: "wave-merchant", SecretKey: "wave-secret", MerchantName: "Shop", TimeToLiveSeconds: 300, BaseURL: "https://gateway.test/wave"}, wave.Config())

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "aya-key", AppSecret: "aya-secret", BaseURL: "https://gateway.test/aya"}, aya.Config())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, yomammqr.Config{MerchantID: "yoma-merchant", ClientID: "yoma-client", ClientSecret: "yoma-secret", WebhookHashKey: "yoma-hash", BaseURL: "https://gateway.test/yoma", APIVersion: "v1rc"}, yoma.Config())

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.Equal(t, cybersource.Config{ProfileID: "profile", AccessKey: "access", SecretKey: "cyber-secret"}, cyber.Config())
	assert.Equal(t, cybersource.SandboxURL, cyber.Config().ResolvedBaseURL())
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

func TestSandboxFlags(t *testing.T) {
	tests := []struct {
		name       string
		sandbox    any
		production bool
	}{
		{"bool true", true, false},
		{"bool false", false, true},
		{"string false", "false", true},
		{"string 0", "0", true},
		{"string true", "true", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := credentials("")
			for _, gateway := range []string{"kbz_pay", "wave_money", "aya_pay", "yoma_mmqr", "cyber_source"} {
				section[gateway].(map[string]any)["sandbox"] = tt.sandbox
				section[gateway].(map[string]any)["base_url"] = ""
			}
			section["kbz_pay"].(map[string]any)["api_url"] = ""
			manager := newManager(t, newConfig(t, map[string]any{ConfigKey: section}), nil, nil)

			kbz, err := manager.KbzPay()
			require.NoError(t, err)
			wave, err := manager.WaveMoney()
			require.NoError(t, err)
			aya, err := manager.AyaPay()
			require.NoError(t, err)
			yoma, err := manager.YomaMmqr()
			require.NoError(t, err)
			cyber, err := manager.CyberSource()
			require.NoError(t, err)

			for _, production := range []bool{kbz.Config().Production, wave.Config().Production, aya.Config().Production, yoma.Config().Production, cyber.Config().Production} {
				assert.Equal(t, tt.production, production)
			}
			if tt.production {
				assert.Equal(t, kbzpay.ProductionAPIURL, kbz.Config().ResolvedAPIURL())
				assert.Equal(t, ayapay.ProductionURL, aya.Config().ResolvedBaseURL())
			} else {
				assert.Equal(t, kbzpay.SandboxAPIURL, kbz.Config().ResolvedAPIURL())
				assert.Equal(t, yomammqr.SandboxURL, yoma.Config().ResolvedBaseURL())
			}
		})
	}
}

func TestGatewaysReadTheEnvironmentWithoutAPublishedConfig(t *testing.T) {
	config := newConfig(t, nil)
	for name, value := range map[string]string{
		"KBZ_PAY_APP_ID": "env-app", "KBZ_PAY_APP_KEY": "env-key", "KBZ_PAY_MERCHANT_CODE": "env-merchant", "KBZ_PAY_SANDBOX": "false",
		"WAVE_MONEY_MERCHANT_ID": "env-wave", "WAVE_MONEY_SECRET_KEY": "env-wave-secret", "WAVE_MONEY_TIME_TO_LIVE_IN_SECONDS": "120",
		"AYA_PGW_APP_KEY": "legacy-key", "AYA_PGW_APP_SECRET": "legacy-secret", "AYA_PGW_BASE_URL": "https://legacy.test/",
		"YOMA_MMQR_MERCHANT_ID": "y-merchant", "YOMA_MMQR_CLIENT_ID": "y-client", "YOMA_MMQR_CLIENT_SECRET": "y-secret", "YOMA_MMQR_WEBHOOK_HASHKEY": "y-hash", "YOMA_MMQR_WEBHOOK_SECRET": "y-webhook",
		"CYBER_SOURCE_PROFILE_ID": "c-profile", "CYBER_SOURCE_ACCESS_KEY": "c-access", "CYBER_SOURCE_SECRET_KEY": "c-secret", "CYBER_SOURCE_SANDBOX": "true",
	} {
		t.Setenv(name, value)
	}
	manager := newManager(t, config, nil, nil)

	kbz, err := manager.KbzPay()
	require.NoError(t, err)
	assert.Equal(t, kbzpay.Config{AppID: "env-app", AppKey: "env-key", MerchantCode: "env-merchant", Production: true}, kbz.Config())

	wave, err := manager.WaveMoney()
	require.NoError(t, err)
	assert.Equal(t, "Goravel", wave.Config().MerchantName, "merchant name falls back to app.name")
	assert.Equal(t, 120, wave.Config().TimeToLiveSeconds)

	aya, err := manager.AyaPay()
	require.NoError(t, err)
	assert.Equal(t, ayapay.Config{AppKey: "legacy-key", AppSecret: "legacy-secret", BaseURL: "https://legacy.test/"}, aya.Config())
	assert.Equal(t, "https://legacy.test", aya.Config().ResolvedBaseURL())

	yoma, err := manager.YomaMmqr()
	require.NoError(t, err)
	assert.Equal(t, "y-webhook", yoma.Config().WebhookSecret)

	cyber, err := manager.CyberSource()
	require.NoError(t, err)
	assert.False(t, cyber.Config().Production)
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
