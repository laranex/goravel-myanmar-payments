package payments

import (
	"context"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contractsclient "github.com/goravel/framework/contracts/http/client"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/ayapay"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	"github.com/laranex/go-myanmar-payments/v4/wavemoney"
	"github.com/laranex/go-myanmar-payments/v4/yomammqr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKbzPayCallsGoThroughGoravelFakes(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	factory.Fake(map[string]any{
		"https://gateway.test/kbz/precreate": factory.Response().Json(200, map[string]any{"Response": map[string]any{
			"result": "SUCCESS", "code": "0", "prepay_id": "prepay-1", "qrCode": "kbz-qr-payload", "nonce_str": "n", "sign_type": "SHA256",
		}}),
	}).PreventStrayRequests()
	t.Cleanup(factory.Reset)

	kbz, err := newManager(t, config, factory, nil).KbzPay()
	require.NoError(t, err)

	payment, err := kbz.QR(context.Background(), kbzpay.PaymentData{OrderID: "ORDER_1", Amount: myanmarpayments.Kyat(1000), CallbackURL: "https://shop.test/callback"})
	require.NoError(t, err)
	assert.Equal(t, "kbz-qr-payload", payment.QRString)
	assert.Equal(t, "prepay-1", payment.Reference)
	assert.True(t, factory.AssertSent(func(request contractsclient.Request) bool {
		return request.Url() == "https://gateway.test/kbz/precreate" && request.Method() == nethttp.MethodPost &&
			strings.Contains(request.Body(), `"merch_order_id":"ORDER_1"`)
	}))
	assert.True(t, factory.AssertSentCount(1))
}

func TestWaveMoneyInitiateWithAFake(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	factory.Fake(map[string]any{
		"https://gateway.test/wave/payment": factory.Response().Json(200, map[string]any{"message": "success", "transaction_id": "tx-9"}),
	}).PreventStrayRequests()
	t.Cleanup(factory.Reset)

	wave, err := newManager(t, config, factory, nil).WaveMoney()
	require.NoError(t, err)

	payment, err := wave.Initiate(context.Background(), &wavemoney.PaymentData{
		OrderID: "100", CallbackURL: "https://shop.test/wave/callback", ReturnURL: "https://shop.test/done", Description: "Order 100",
		Items: []wavemoney.Item{{Name: "Tea", Amount: myanmarpayments.Kyat(1000)}},
	})
	require.NoError(t, err)
	assert.Equal(t, "tx-9", payment.GatewayReference)
	assert.Contains(t, payment.URL, "/authenticate?transaction_id=tx-9")
	assert.True(t, factory.AssertSent(func(request contractsclient.Request) bool {
		return request.Header("Content-Type") == "application/x-www-form-urlencoded" && strings.Contains(request.Body(), "order_id=100")
	}))
}

func TestGatewayErrorsFromAFakeAreAPIErrors(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	factory.Fake(map[string]any{
		"https://gateway.test/aya/v1/payment/enquiry": factory.Response().Json(200, map[string]any{"status": "09", "message": "Order not found"}),
	})
	t.Cleanup(factory.Reset)

	aya, err := newManager(t, config, factory, nil).AyaPay()
	require.NoError(t, err)

	_, err = aya.Status(context.Background(), "ORDER_404")
	var apiError *myanmarpayments.APIError
	require.True(t, errors.As(err, &apiError))
	assert.Equal(t, "09", apiError.GatewayCode)
}

func TestFakesRegisteredAfterTheGatewayWasBuiltStillApply(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	aya, err := newManager(t, config, factory, nil).AyaPay()
	require.NoError(t, err)

	factory.Fake(map[string]any{
		"https://gateway.test/aya/v1/payment/services": factory.Response().Json(200, map[string]any{"status": "00", "data": []any{
			map[string]any{"key": "aya_pay", "name": "AYA Pay", "methods": []any{"QR", "WEB"}},
		}}),
	}).PreventStrayRequests()
	t.Cleanup(factory.Reset)

	services, err := aya.Services(context.Background())
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.True(t, services[0].Supports(ayapay.MethodQR))
}

func TestYomaTokensAreSharedThroughTheGoravelCache(t *testing.T) {
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	factory := newHTTPFactory(t, config)
	factory.Fake(map[string]any{
		"https://gateway.test/yoma/token": factory.Response().Json(200, map[string]any{"access_token": "token-1", "expires_in": 3600}),
		"https://gateway.test/yoma/payment-gateway/v1rc/api/payment/check-status": factory.Response().Json(200, map[string]any{
			"paymentStatus": "SUCCESS", "refLabel": "ref-1",
		}),
	}).PreventStrayRequests()
	t.Cleanup(factory.Reset)

	memory := newMemoryCache(t, config)
	build := func() *yomammqr.Gateway {
		manager, err := NewManager(Options{Config: config, HTTPClient: NewHTTPClient(factory, "", time.Second), TokenCache: NewTokenCache(memory)})
		require.NoError(t, err)
		yoma, err := manager.YomaMmqr()
		require.NoError(t, err)
		return yoma
	}

	first, second := build(), build()
	result, err := first.Status(context.Background(), "ref-1")
	require.NoError(t, err)
	assert.True(t, result.IsSuccessful())
	_, err = second.Status(context.Background(), "ref-1")
	require.NoError(t, err)

	assert.True(t, factory.AssertSentCount(3), "one token request and two status checks")
	assert.True(t, factory.AssertSent(func(request contractsclient.Request) bool {
		return request.Header("Authorization") == "Bearer token-1"
	}))
}

func TestHTTPClientWithoutAGoravelFactoryUsesAPlainClient(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		_, _ = w.Write([]byte("pong"))
	}))
	t.Cleanup(server.Close)

	client := NewHTTPClient(nil, "", time.Second)
	request, err := nethttp.NewRequest(nethttp.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	assert.Equal(t, "pong", readBody(t, response))
}

func TestHTTPClientAppliesItsTimeout(t *testing.T) {
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(server.Close)

	config := newConfig(t, nil)
	client := NewHTTPClient(newHTTPFactory(t, config), "", 50*time.Millisecond)
	request, err := nethttp.NewRequest(nethttp.MethodGet, server.URL, nil)
	require.NoError(t, err)
	_, err = client.Do(request)
	assert.ErrorContains(t, err, "Client.Timeout")
}

func TestHTTPClientRejectsAnUnknownGoravelClient(t *testing.T) {
	config := newConfig(t, nil)
	client := NewHTTPClient(newHTTPFactory(t, config), "missing", time.Second)
	request, err := nethttp.NewRequest(nethttp.MethodGet, "https://gateway.test", nil)
	require.NoError(t, err)

	_, err = client.Do(request)
	assert.ErrorContains(t, err, `HTTP client "missing" is not configured`)
}
