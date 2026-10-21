package payments

import (
	"errors"
	nethttp "net/http"
	"strings"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayResolvesEveryName(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)

	assert.Equal(t, []string{"kbz-pay", "wave-pay", "wave-pay-mini-app", "aya-pay", "yoma-mmqr", "cyber-source"}, GatewayNames())

	kbz, _ := manager.KbzPay()
	wave, _ := manager.WavePay()
	aya, _ := manager.AyaPay()
	yoma, _ := manager.YomaMmqr()
	cyber, _ := manager.CyberSource()
	expected := map[string]CallbackHandler{GatewayKbzPay: kbz, GatewayWavePay: wave, GatewayAyaPay: aya, GatewayYomaMmqr: yoma, GatewayCyberSource: cyber}

	for _, name := range GatewayNames() {
		gateway, err := manager.Gateway(name)
		require.NoError(t, err)
		if name == GatewayWavePayMiniApp {
			continue
		}
		assert.Same(t, expected[name], gateway, name)
	}

	miniApp, _ := manager.WavePayMiniApp()
	gateway, err := manager.Gateway(GatewayWavePayMiniApp)
	require.NoError(t, err)
	handler, ok := gateway.(*WavePayMiniAppCallbackHandler)
	require.True(t, ok)
	assert.Same(t, miniApp, handler.Gateway)
	assert.Empty(t, handler.CallbackURL)
}

func TestGatewayRejectsAnUnknownName(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)

	gateway, err := manager.Gateway("paypal")
	assert.Nil(t, gateway)
	require.ErrorIs(t, err, ErrUnknownGateway)
	assert.EqualError(t, err, "goravel-myanmar-payments: unknown payment gateway [paypal]; use one of kbz-pay, wave-pay, wave-pay-mini-app, aya-pay, yoma-mmqr, cyber-source")

	_, err = manager.HandleCallback("paypal", &myanmarpayments.CallbackRequest{})
	assert.ErrorIs(t, err, ErrUnknownGateway)
}

func TestGatewayReturnsConfigurationErrors(t *testing.T) {
	manager := newManager(t, newConfig(t, nil), nil, nil)

	for _, name := range GatewayNames() {
		gateway, err := manager.Gateway(name)
		assert.Nil(t, gateway, name)
		var configuration *myanmarpayments.ConfigurationError
		assert.True(t, errors.As(err, &configuration), name)
	}
}

func TestHandleCallbackByNameFromOneRoute(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	manager := newManager(t, config, nil, nil)
	server := newTestServer(config)
	server.router.Post("/webhooks/payments/{gateway}", func(ctx contractshttp.Context) contractshttp.Response {
		request, err := CallbackRequestFromContext(ctx)
		require.NoError(t, err)
		callback, err := manager.HandleCallback(ctx.Request().Route("gateway"), request)
		if err != nil {
			return ctx.Response().String(nethttp.StatusBadRequest, "invalid signature")
		}

		return Acknowledge(ctx, callback)
	})

	tests := []struct {
		gateway     string
		contentType string
		body        string
		ack         string
	}{
		{GatewayKbzPay, "application/json", kbzCallbackBody(t, "PAY_SUCCESS"), "success"},
		{GatewayWavePay, "application/json", waveCallbackBody(t), ""},
		{GatewayAyaPay, "application/x-www-form-urlencoded", ayaSignedValues("00").Encode(), ""},
		{GatewayYomaMmqr, "application/json", yomaCallbackBody(t), ""},
		{GatewayCyberSource, "application/x-www-form-urlencoded", cyberSourceValues().Encode(), ""},
	}
	for _, tt := range tests {
		response := server.do(t, nethttp.MethodPost, "/webhooks/payments/"+tt.gateway, tt.contentType, tt.body)
		assert.Equal(t, nethttp.StatusOK, response.StatusCode, tt.gateway)
		assert.Equal(t, tt.ack, readBody(t, response), tt.gateway)
	}

	// A body signed for another gateway does not verify.
	response := server.do(t, nethttp.MethodPost, "/webhooks/payments/wave-pay", "application/json", yomaCallbackBody(t))
	assert.Equal(t, nethttp.StatusBadRequest, response.StatusCode)
	assert.True(t, strings.Contains(readBody(t, response), "invalid signature"))
}

func TestHandleCallbackByNameNeedsTheMiniAppCallbackURL(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)
	callbackURL := "https://shop.test/payments/callback/wave-pay-mini-app"
	request := myanmarpayments.NewCallbackRequest([]byte(miniAppCallbackBody(t, callbackURL, "SUCCESSFUL")), nil, nil)

	_, err := manager.HandleCallback(GatewayWavePayMiniApp, request)
	require.ErrorIs(t, err, ErrCallbackURLRequired)
	assert.EqualError(t, err, "goravel-myanmar-payments: the wave-pay-mini-app callback needs the callback URL the payment was signed with; use HandleCallbackFromContext or WavePayMiniApp().HandleCallback(request, callbackURL)")

	_, err = newManager(t, newConfig(t, nil), nil, nil).HandleCallback(GatewayWavePayMiniApp, request)
	var configuration *myanmarpayments.ConfigurationError
	assert.True(t, errors.As(err, &configuration), "the gateway is resolved first")

	gateway, err := manager.Gateway(GatewayWavePayMiniApp)
	require.NoError(t, err)
	handler := gateway.(*WavePayMiniAppCallbackHandler)
	handler.CallbackURL = callbackURL
	callback, err := handler.HandleCallback(request)
	require.NoError(t, err)
	assert.Equal(t, myanmarpayments.StatusSuccessful, callback.Status)

	miniApp, err := manager.WavePayMiniApp()
	require.NoError(t, err)
	callback, err = miniApp.HandleCallback(request, callbackURL)
	require.NoError(t, err)
	assert.Equal(t, "ORDER123", callback.OrderID)
	assert.Equal(t, "TXN-1", callback.GatewayReference)
	assert.Equal(t, "5000", callback.Amount)
}

func TestHandleCallbackFromContext(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	manager := newManager(t, config, nil, nil)
	unconfigured := newManager(t, newConfig(t, nil), nil, nil)
	server := newTestServer(config)
	var errs []error
	route := func(m *Manager) contractshttp.HandlerFunc {
		return func(ctx contractshttp.Context) contractshttp.Response {
			callback, err := m.HandleCallbackFromContext(ctx.Request().Route("gateway"), ctx)
			errs = append(errs, err)
			if err != nil {
				return ctx.Response().String(nethttp.StatusBadRequest, "rejected")
			}

			return Acknowledge(ctx, callback)
		}
	}
	server.router.Post("/webhooks/{gateway}", route(manager))
	server.router.Post("/unconfigured/{gateway}", route(unconfigured))

	response := server.do(t, nethttp.MethodPost, "/webhooks/kbz-pay", "application/json", kbzCallbackBody(t, "PAY_SUCCESS"))
	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	assert.Equal(t, "success", readBody(t, response))
	require.NoError(t, errs[0])

	tooLarge := `{"a":"` + strings.Repeat("b", MaxCallbackBodyBytes) + `"}`
	server.do(t, nethttp.MethodPost, "/webhooks/paypal", "application/json", tooLarge)
	assert.ErrorIs(t, errs[1], ErrUnknownGateway, "the gateway is resolved before the body is read")

	server.do(t, nethttp.MethodPost, "/unconfigured/kbz-pay", "application/json", tooLarge)
	var configuration *myanmarpayments.ConfigurationError
	assert.True(t, errors.As(errs[2], &configuration), "configuration errors come before the body limit")

	server.do(t, nethttp.MethodPost, "/webhooks/kbz-pay", "application/json", tooLarge)
	assert.ErrorIs(t, errs[3], ErrCallbackBodyTooLarge)

	server.do(t, nethttp.MethodPost, "/webhooks/wave-pay", "application/json", yomaCallbackBody(t))
	var signature *myanmarpayments.SignatureVerificationError
	assert.True(t, errors.As(errs[4], &signature))

	_, err := manager.HandleCallbackFromContext(GatewayKbzPay, nil)
	assert.ErrorContains(t, err, "no request")
}

func TestHandleCallbackFromContextVerifiesMiniAppCallbacksAgainstTheRequestURL(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	manager := newManager(t, config, nil, nil)
	server := newTestServer(config)
	var errs []error
	var callbacks []*myanmarpayments.PaymentCallback
	server.router.Post("/payments/callback/{gateway}", func(ctx contractshttp.Context) contractshttp.Response {
		callback, err := manager.HandleCallbackFromContext(ctx.Request().Route("gateway"), ctx)
		errs = append(errs, err)
		if err != nil {
			return ctx.Response().String(nethttp.StatusBadRequest, "rejected")
		}
		callbacks = append(callbacks, callback)

		return Acknowledge(ctx, callback)
	})

	// httptest requests are http://example.com/...; the query string is not part of the URL.
	signedURL := "http://example.com/payments/callback/wave-pay-mini-app"
	response := server.do(t, nethttp.MethodPost, "/payments/callback/wave-pay-mini-app?attempt=1", "application/json", miniAppCallbackBody(t, signedURL, "SUCCESSFUL"))
	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	assert.Equal(t, "", readBody(t, response), "Wave defines no response body")
	require.NoError(t, errs[0])
	assert.Equal(t, myanmarpayments.StatusSuccessful, callbacks[0].Status)
	assert.Equal(t, "ORDER123", callbacks[0].OrderID)
	assert.Equal(t, "MERCHANT789", callbacks[0].Raw["merchantRefId"])

	// A callback signed for another URL does not verify.
	response = server.do(t, nethttp.MethodPost, "/payments/callback/wave-pay-mini-app", "application/json", miniAppCallbackBody(t, "https://shop.test/payments/callback/wave-pay-mini-app", "SUCCESSFUL"))
	assert.Equal(t, nethttp.StatusBadRequest, response.StatusCode)
	var signature *myanmarpayments.SignatureVerificationError
	require.True(t, errors.As(errs[1], &signature))
	assert.Equal(t, "Wave Pay Mini App callback hash verification failed.", signature.Message)
}
