package payments

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	nethttp "net/http"
	"net/url"
	"strings"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	myanmarpayments "github.com/laranex/go-myanmar-payments/v4"
	"github.com/laranex/go-myanmar-payments/v4/kbzpay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callbackServer registers POST /callback/{gateway} and GET /aya/return the way an
// application would: build the SDK request from the Goravel context, verify it with
// the gateway and acknowledge it.
type callbackServer struct {
	*testServer
	received []*myanmarpayments.PaymentCallback
	requests []*myanmarpayments.CallbackRequest
}

func newCallbackServer(t *testing.T) *callbackServer {
	t.Helper()
	config := newConfig(t, map[string]any{ConfigKey: credentials("https://gateway.test")})
	manager := newManager(t, config, nil, nil)
	server := &callbackServer{testServer: newTestServer(config)}

	server.router.Post("/callback/{gateway}", func(ctx contractshttp.Context) contractshttp.Response {
		request, err := CallbackRequestFromContext(ctx)
		require.NoError(t, err)
		server.requests = append(server.requests, request)

		callback, err := manager.HandleCallback(ctx.Request().Route("gateway"), request)
		if err != nil {
			return ctx.Response().String(nethttp.StatusBadRequest, "invalid signature")
		}
		server.received = append(server.received, callback)

		return Acknowledge(ctx, callback)
	})

	server.router.Get("/aya/return", func(ctx contractshttp.Context) contractshttp.Response {
		request, err := CallbackRequestFromContext(ctx)
		require.NoError(t, err)
		g, err := manager.AyaPay()
		require.NoError(t, err)
		callback, err := g.VerifyRedirect(request)
		if err != nil {
			return ctx.Response().String(nethttp.StatusBadRequest, "invalid signature")
		}

		return ctx.Response().Json(nethttp.StatusOK, map[string]any{"orderId": callback.OrderID, "status": callback.Status})
	})

	return server
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)

	return string(encoded)
}

func hmacHex(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))

	return hex.EncodeToString(mac.Sum(nil))
}

func kbzCallbackBody(t *testing.T, tradeStatus string) string {
	fields := map[string]any{
		"appid": "kp-app", "merch_code": "200001", "merch_order_id": "ORDER_1", "mm_order_id": "MM1",
		"total_amount": "1000", "trans_currency": "MMK", "trade_status": tradeStatus, "notify_time": "1700000000",
		"nonce_str": "abc", "sign_type": "SHA256",
	}
	fields["sign"] = kbzpay.NewSigner("kbz-secret").Sign(fields)

	return mustJSON(t, map[string]any{"Request": fields})
}

func waveCallbackBody(t *testing.T) string {
	payload := map[string]any{
		"status": "PAYMENT_CONFIRMED", "timeToLiveSeconds": 300, "merchantId": "wave-merchant", "orderId": "100", "amount": "1000",
		"backendResultUrl": "https://shop.test/wave/callback", "merchantReferenceId": "ref-001", "initiatorMsisdn": "9791009039",
		"transactionId": "360", "paymentRequestId": "360", "requestTime": nil,
	}
	payload["hashValue"] = hmacHex("wave-secret", "PAYMENT_CONFIRMED300wave-merchant1001000https://shop.test/wave/callbackref-0019791009039360360null")

	return mustJSON(t, payload)
}

func ayaSignedValues(statusCode string) url.Values {
	payload := `{"merchOrderId":"ORD123456","tranId":"TRN0001","amount":"1000","statusCode":"` + statusCode + `","dateTime":"2026-10-07 10:00:00"}`
	return url.Values{
		"payload":  {base64.StdEncoding.EncodeToString([]byte(payload))},
		"checkSum": {hmacHex("aya-secret", "ORD123456:TRN0001:1000:"+statusCode+":2026-10-07 10:00:00")},
	}
}

func cyberSourceValues() url.Values {
	values := url.Values{
		"signed_field_names":   {"decision,req_reference_number,req_amount,transaction_id,signed_field_names"},
		"decision":             {"ACCEPT"},
		"req_reference_number": {"ORD-1"},
		"req_amount":           {"1000.00"},
		"transaction_id":       {"tx-1"},
	}
	names := strings.Split(values.Get("signed_field_names"), ",")
	pairs := make([]string, len(names))
	for i, name := range names {
		pairs[i] = name + "=" + values.Get(name)
	}
	mac := hmac.New(sha256.New, []byte("cyber-secret"))
	mac.Write([]byte(strings.Join(pairs, ",")))
	values.Set("signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	return values
}

func yomaCallbackBody(t *testing.T) string {
	return mustJSON(t, map[string]any{
		"orderNumber": "ORD-1", "status": "SUCCESS",
		"hashValue": hmacHex("ORD-1"+"yoma-hash", "orderNumber=ORD-1&status=SUCCESS"),
	})
}

func TestCallbackRoundTripThroughAGoravelRoute(t *testing.T) {
	tests := []struct {
		name        string
		gateway     string
		contentType string
		body        func(t *testing.T) string
		orderID     string
		status      myanmarpayments.PaymentStatus
		ackBody     string
	}{
		{"kbz pay json", "kbz-pay", "application/json", func(t *testing.T) string { return kbzCallbackBody(t, "PAY_SUCCESS") }, "ORDER_1", myanmarpayments.StatusSuccessful, "success"},
		{"wave money json", "wave-money", "application/json", waveCallbackBody, "100", myanmarpayments.StatusSuccessful, ""},
		{"aya pay form", "aya-pay", "application/x-www-form-urlencoded", func(*testing.T) string { return ayaSignedValues("00").Encode() }, "ORD123456", myanmarpayments.StatusSuccessful, ""},
		{"aya pay json", "aya-pay", "application/json", func(t *testing.T) string {
			values := ayaSignedValues("02")
			return mustJSON(t, map[string]string{"payload": values.Get("payload"), "checkSum": values.Get("checkSum")})
		}, "ORD123456", myanmarpayments.StatusFailed, ""},
		{"yoma mmqr json", "yoma-mmqr", "application/json", yomaCallbackBody, "ORD-1", myanmarpayments.StatusSuccessful, ""},
		{"cyber source form", "cyber-source", "application/x-www-form-urlencoded", func(*testing.T) string { return cyberSourceValues().Encode() }, "ORD-1", myanmarpayments.StatusSuccessful, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newCallbackServer(t)
			response := server.do(t, nethttp.MethodPost, "/callback/"+tt.gateway+"?source=test", tt.contentType, tt.body(t))

			assert.Equal(t, nethttp.StatusOK, response.StatusCode)
			assert.Equal(t, tt.ackBody, readBody(t, response))
			assert.True(t, strings.HasPrefix(response.Header.Get("Content-Type"), "text/plain"))
			require.Len(t, server.received, 1)
			assert.Equal(t, tt.orderID, server.received[0].OrderID)
			assert.Equal(t, tt.status, server.received[0].Status)
			assert.Equal(t, "test", server.requests[0].Query.Get("source"))
		})
	}
}

func TestCallbackRequestKeepsJSONBodiesByteForByte(t *testing.T) {
	server := newCallbackServer(t)
	body := kbzCallbackBody(t, "PAY_SUCCESS")
	server.do(t, nethttp.MethodPost, "/callback/kbzpay", "application/json", body)

	require.Len(t, server.requests, 1)
	assert.Equal(t, body, string(server.requests[0].Body))
	assert.Equal(t, "application/json", server.requests[0].HeaderValue("content-type"))
}

func TestCallbackRequestRebuildsMultipartForms(t *testing.T) {
	server := newCallbackServer(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, values := range cyberSourceValues() {
		require.NoError(t, writer.WriteField(name, values[0]))
	}
	require.NoError(t, writer.Close())

	response := server.do(t, nethttp.MethodPost, "/callback/cyber-source", writer.FormDataContentType(), body.String())

	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	require.Len(t, server.received, 1)
	assert.Equal(t, "tx-1", server.received[0].GatewayReference)
	assert.Equal(t, "application/x-www-form-urlencoded", server.requests[0].HeaderValue("Content-Type"))
}

func TestTamperedCallbacksAreRejected(t *testing.T) {
	tests := map[string]struct {
		gateway, contentType, body string
	}{
		"kbz pay":      {"kbz-pay", "application/json", strings.Replace(kbzCallbackBody(t, "PAY_SUCCESS"), `"total_amount":"1000"`, `"total_amount":"1"`, 1)},
		"wave money":   {"wave-money", "application/json", strings.Replace(waveCallbackBody(t), `"amount":"1000"`, `"amount":"1"`, 1)},
		"aya pay":      {"aya-pay", "application/x-www-form-urlencoded", url.Values{"payload": ayaSignedValues("00")["payload"], "checkSum": {"bad"}}.Encode()},
		"yoma mmqr":    {"yoma-mmqr", "application/json", strings.Replace(yomaCallbackBody(t), "SUCCESS", "FAILED", 1)},
		"cyber source": {"cyber-source", "application/x-www-form-urlencoded", strings.Replace(cyberSourceValues().Encode(), "ACCEPT", "DECLINE", 1)},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := newCallbackServer(t)
			response := server.do(t, nethttp.MethodPost, "/callback/"+tt.gateway, tt.contentType, tt.body)

			assert.Equal(t, nethttp.StatusBadRequest, response.StatusCode)
			assert.Empty(t, server.received)
		})
	}
}

func TestAyaRedirectIsVerifiedFromTheQueryString(t *testing.T) {
	server := newCallbackServer(t)

	response := server.do(t, nethttp.MethodGet, "/aya/return?"+ayaSignedValues("00").Encode(), "", "")
	assert.Equal(t, nethttp.StatusOK, response.StatusCode)
	assert.JSONEq(t, `{"orderId":"ORD123456","status":"successful"}`, readBody(t, response))

	response = server.do(t, nethttp.MethodGet, "/aya/return?payload=x&checkSum=y", "", "")
	assert.Equal(t, nethttp.StatusBadRequest, response.StatusCode)
}

func TestAcknowledgeWritesStatusBodyAndHeaders(t *testing.T) {
	config := newConfig(t, nil)
	server := newTestServer(config)
	callbacks := map[string]*myanmarpayments.PaymentCallback{
		"custom": {Acknowledgement: myanmarpayments.Acknowledgement{Status: nethttp.StatusAccepted, Body: `{"ok":true}`, Headers: map[string]string{"Content-Type": "application/json", "X-Ack": "1"}}},
		"zero":   {},
		"nil":    nil,
	}
	server.router.Get("/ack/{name}", func(ctx contractshttp.Context) contractshttp.Response {
		return Acknowledge(ctx, callbacks[ctx.Request().Route("name")])
	})

	response := server.do(t, nethttp.MethodGet, "/ack/custom", "", "")
	assert.Equal(t, nethttp.StatusAccepted, response.StatusCode)
	assert.Equal(t, `{"ok":true}`, readBody(t, response))
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, "1", response.Header.Get("X-Ack"))

	for _, name := range []string{"zero", "nil"} {
		response = server.do(t, nethttp.MethodGet, "/ack/"+name, "", "")
		assert.Equal(t, nethttp.StatusOK, response.StatusCode, name)
		assert.Equal(t, "", readBody(t, response), name)
		assert.True(t, strings.HasPrefix(response.Header.Get("Content-Type"), "text/plain"), name)
	}
}

func TestCallbackRequestFromContextWithoutARequest(t *testing.T) {
	_, err := CallbackRequestFromContext(nil)
	assert.ErrorContains(t, err, "no request")
}
