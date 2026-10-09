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

	assert.Equal(t, []string{"kbz-pay", "wave-money", "aya-pay", "yoma-mmqr", "cyber-source"}, GatewayNames())

	kbz, _ := manager.KbzPay()
	wave, _ := manager.WaveMoney()
	aya, _ := manager.AyaPay()
	yoma, _ := manager.YomaMmqr()
	cyber, _ := manager.CyberSource()
	expected := []CallbackHandler{kbz, wave, aya, yoma, cyber}

	for i, name := range GatewayNames() {
		gateway, err := manager.Gateway(name)
		require.NoError(t, err)
		assert.Same(t, expected[i], gateway, name)
	}
}

func TestGatewayRejectsAnUnknownName(t *testing.T) {
	manager := newManager(t, newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")}), nil, nil)

	gateway, err := manager.Gateway("paypal")
	assert.Nil(t, gateway)
	require.ErrorIs(t, err, ErrUnknownGateway)
	assert.EqualError(t, err, "goravel-myanmar-payments: unknown payment gateway [paypal]; use one of kbz-pay, wave-money, aya-pay, yoma-mmqr, cyber-source")

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
		{GatewayWaveMoney, "application/json", waveCallbackBody(t), ""},
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
	response := server.do(t, nethttp.MethodPost, "/webhooks/payments/wave-money", "application/json", yomaCallbackBody(t))
	assert.Equal(t, nethttp.StatusBadRequest, response.StatusCode)
	assert.True(t, strings.Contains(readBody(t, response), "invalid signature"))
}
